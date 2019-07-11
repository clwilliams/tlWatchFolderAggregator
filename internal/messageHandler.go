package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tlWatchFolderAggregator/elasticSearch"

	"github.com/tlCommonMessaging/rabbitMQ"
)

// HandleFolderWatchUpdate - given the message body from RabbitMQ, marshall
// into the folder watch message entity & based on the action, send to the
// appropriate method for handling the message
func HandleFolderWatchUpdate(config *elasticSearch.App) func(context.Context, []byte) error {
	return func(ctx context.Context, msg []byte) error {

		folderWatchMsg := rabbitMQ.FolderWatchMessage{}
		if err := json.Unmarshal(msg, &folderWatchMsg); err != nil {
			log.Errorf("Can't unmarshal FolderWatch update %v : %v", err, string(msg))
			return nil
		}

		if config.Verbose {
			log.Infof("HandleFolderWatchUpdate for %#v", folderWatchMsg)
		}

		switch folderWatchMsg.Action {
		case rabbitMQ.CreateAction:
			{
				return handleCreate(config, &folderWatchMsg)
			}
		case rabbitMQ.DeleteAction:
			{
				return handleDelete(config, &folderWatchMsg)
			}
		case rabbitMQ.RenameAction:
			{
				return handleRename(config, &folderWatchMsg)
			}
		case rabbitMQ.MoveAction:
			{
				return handleMove(config, &folderWatchMsg)
			}
		default:
			// we shouldn't have any unhandled case as the watcher is configured to
			// report the above 4 actions, but should something else arrive, raise an
			// error so we report it's not something we currently handle
			return fmt.Errorf("This message handler doesn't support action %s. Message: %#v",
				folderWatchMsg.Action, folderWatchMsg)
		}
	}
}

func retrieveName(folderPath string) string {
	pathParts := strings.Split(folderPath, "/")
	name := folderPath
	if len(pathParts) >= 1 {
		name = pathParts[len(pathParts)-1]
	}
	return name
}

/*
  handleCreate
  example msg {
    XMLName:xml.Name{Space:"", Local:""},
    Action:"CREATE",
    Path:"/Users/clairew/watch_me/2019/04 April/05 May/de Gournay Chinoiserie C076 Chatsworth.pdf",
    IsDir:"false"
  }
*/
func handleCreate(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatchMessage) error {
	id := generateUniqueID(folderWatchMsg.Path, folderWatchMsg.IsDir)
	// retrieve the name from the full folder path
	name := retrieveName(folderWatchMsg.Path)

	// set a boolean value to state whether or not this is the parent watch folder
	isWatchFolder := "false"
	if folderWatchMsg.WatchFolder == folderWatchMsg.Path {
		isWatchFolder = "true"
	}

	// initialise the data that we will store in elastic search
	fsNode := elasticSearch.FsNode{
		Name:          name,
		IsDir:         folderWatchMsg.IsDir,
		FullPath:      folderWatchMsg.Path,
		IsWatchFolder: isWatchFolder,
	}

	// & save
	err := config.Save(fsNode, id)
	if err != nil {
		return fmt.Errorf("Error storing in elastic search %#v", err)
	}
	return nil
}

/*
  handleDelete - handler for a remove message
  example msg {
    XMLName:xml.Name{Space:"", Local:""},
    Action:"REMOVE",
    Path:"/Users/clairew/watch_me/2019/April/May/.DS_Store",
    IsDir:"false"
  }
*/
func handleDelete(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatchMessage) error {
	id := generateUniqueID(folderWatchMsg.Path, folderWatchMsg.IsDir)
	if config.Verbose {
		log.Infof("handleDelete for id %#v", id)
	}
	err := config.Delete(id)
	if err != nil {
		return fmt.Errorf("Error deleting document with ID %s %v", id, err)
	}
	return nil
}

/*
  handleRename - handler for a rename message
  example msg {
    XMLName:xml.Name{Space:"", Local:""},
    Action:"RENAME",
    Path:"/Users/clairew/watch_me/2019/April/de Gournay Chinoiserie C075 Jardinières & Citrus Trees.pdf -> /Users/clairew/watch_me/2019/April/de Gournay Chinoiserie cC075 Jardinières & Citrus Trees.pdf",
    IsDir:"false"
  }
	because the full folder path is used to generate the id for storing in elastic search
	we don't just perform an update to the existing document, we remove it and add a new one
	with the updated folder path / name and new id
*/
func handleRename(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatchMessage) error {
	paths := strings.Split(folderWatchMsg.Path, " -> ")
	if len(paths) != 2 {
		return fmt.Errorf("Rename operation needs to have the old and new names in order to process change. %#v", folderWatchMsg)
	}
	oldFullPath := paths[0]
	newFullPath := paths[1]

	// retrieve the original document from elastic search
	originalID := generateUniqueID(oldFullPath, folderWatchMsg.IsDir)
	originalDoc, err := config.Get(originalID)
	if err != nil {
		return fmt.Errorf("Error renaming: error retrieve original document with ID %s %v",
			originalID, err)
	}

	// delete the original
	err = config.Delete(originalID)
	if err != nil {
		return fmt.Errorf("Error renaming: can't delete original document with ID %s %v",
			originalID, err)
	}

	// then apply the new name & full path to the document and save
	originalDoc.Name = retrieveName(newFullPath)
	originalDoc.FullPath = newFullPath
	newID := generateUniqueID(newFullPath, folderWatchMsg.IsDir)
	err = config.Save(originalDoc, newID)
	if err != nil {
		return fmt.Errorf("Error renaming: can't save renamed document with ID %s %v",
			newID, err)
	}

	return nil
}

/*
  handleMove - handler for a move action
  example msg {
    XMLName:xml.Name{Space:"", Local:""},
    Action:"MOVE",
    Path:"/Users/clairew/watch_me/2019/04 April/05 May/de Gournay Chinoiserie C076 Chatsworth.pdf -> /Users/clairew/watch_me/2019/04 April/de Gournay Chinoiserie C076 Chatsworth.pdf",
    IsDir:"false"
  }
	moving is the same as a rename operation in that it will result in a id, name and full path change,
	so just call the handleRename
*/
func handleMove(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatchMessage) error {
	return handleRename(config, folderWatchMsg)
}

// GenerateUniqueID - calculate the ID for storing document in elastic search
func generateUniqueID(fullPath, isDir string) string {
	typePrefix := "file"
	if isDir == "true" {
		typePrefix = "dir"
	}
	return fmt.Sprintf("%s_%s", typePrefix, fullPath)
}
