package elasticSearch

import (
	"context"
	"encoding/json"
	"fmt"

  "github.com/olivere/elastic"
	log "github.com/rs/zerolog/log"
)

// FsNode represents a file server node
type FsNode struct {
	Name          string `json:"name"`
	IsDir         string `json:"isDir"`
	FullPath      string `json:"fullPath"`
	IsWatchFolder string `json:"isWatchFolder"`
}

const docType = "doc"

// GenerateUniqueID -
func (fsNode *FsNode) GenerateUniqueID() string {
	typePrefix := "file"
	if fsNode.IsDir == "true" {
		typePrefix = "dir"
	}
	return fmt.Sprintf("%s_%s", typePrefix, fsNode.FullPath)
}

// Save - saves the document
func (app *App) Save(fsNode FsNode) error {
	ctx := context.Background()
	response, err := app.Client.Index().
		Index(app.Index).
		Type(docType).
		Id(fsNode.GenerateUniqueID()).
		BodyJson(fsNode).
		Do(ctx)
	if err != nil {
		return err
	}
	log.Printf("Indexed fsNode %s to index %s\n", response.Id, response.Index)
	return nil
}

// Delete - deletes a document from the index given its id
func (app *App) Delete(id string) error {
	ctx := context.Background()
	_, err := app.Client.Delete().
		Index(app.Index).
		Type(docType).
		Id(id).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

// GetAllFsNodes returns a list of all FsNodes
func (app *App) GetAllFsNodes() ([]FsNode, int64, error) {
	log.Debug().Msg("START - elasticSearch.GetAllFsNodes")
	ctx := context.Background()
	results, err := app.Client.
	  Search().
		FetchSourceContext(elastic.NewFetchSourceContext(true).Include("fullPath")).
	  Index(app.Index).
		Source(elastic.NewMatchAllQuery()).
		Sort("path", true).
		Do(ctx)
	if err != nil {
		return nil, 0, err
	}

	// process results
	var fsNodes []FsNode
	for _, hit := range results.Hits.Hits {
		var fsn FsNode
		json.Unmarshal(*hit.Source, &fsn)
		fsNodes = append(fsNodes, fsn)
	}

	log.Debug().Msg("END - elasticSearch.GetAllFsNodes")
	return fsNodes, results.Hits.TotalHits, nil
}

// GetFsNodesForWatchFolder -
func (app *App) GetFsNodesForWatchFolder(folderPath string) ([]FsNode, int64, error) {
	ctx := context.Background()
/*
	ss := elastic.NewSearchSource().Query(
		elastic.NewPrefixQuery("path", folderPath))
data, _ := json.Marshal(ss.Source())
fmt.Printf("%s", string(data))
*/

	q := elastic.NewPrefixQuery("fullPath", folderPath)
	// q = q.QueryName("my_query_name")

	results, err := app.Client.Search().
		FetchSourceContext(elastic.NewFetchSourceContext(true).Include("fullPath")).
		Index(app.Index).
		Query(q).
		//Source(elastic.NewPrefixQuery("path", folderPath)).
		//Sort("path", true).
		Pretty(true).
		Do(ctx)
	if err != nil {
		return nil, 0, err
	}

	// process results
	var fsNodes []FsNode
	for _, hit := range results.Hits.Hits {
		var fsn FsNode
		json.Unmarshal(*hit.Source, &fsn)
		fsNodes = append(fsNodes, fsn)
	}

	return fsNodes, results.Hits.TotalHits, nil
}
