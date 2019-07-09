package internal

import (
	"context"
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tlWatchFolderAggregator/elasticSearch"

	"github.com/tlCommonMessaging/rabbitMQ"
)

// HandleFolderWatchUpdate -
func HandleFolderWatchUpdate(config *elasticSearch.App) func(context.Context, []byte) error {
	return func(ctx context.Context, msg []byte) error {
		folderWatchMsg := rabbitMQ.FolderWatchMessage{}
		if err := json.Unmarshal(msg, &folderWatchMsg); err != nil {
			log.Errorf("Can't unmarshal FolderWatch update %v : %v", err, string(msg))
			return nil
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
			// we shouldn't have any unhandled case as the watcher is configured to
			// report the above 4 event types
		}
		return nil
	}
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
	pathParts := strings.Split(folderWatchMsg.Path, "/")
	name := folderWatchMsg.Path
	if len(pathParts) > 1 {
		name = pathParts[len(pathParts)-1]
	}

	isWatchFolder := "false"
	if folderWatchMsg.WatchFolder == folderWatchMsg.Path {
		isWatchFolder = "true"
	}

	fsNode := elasticSearch.FsNode{
		Name:          name,
		IsDir:         folderWatchMsg.IsDir,
		Path:          folderWatchMsg.Path,
		IsWatchFolder: isWatchFolder,
	}
	err := config.Save(fsNode)
	if err != nil {
		return err
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
	err := config.Delete(folderWatchMsg.Path)
	if err != nil {
		return err
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
*/
func handleRename(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatchMessage) error {
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
*/
func handleMove(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatchMessage) error {
	return nil
}
