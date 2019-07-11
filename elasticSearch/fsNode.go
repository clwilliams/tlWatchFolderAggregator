package elasticSearch

import (
	"context"
	"encoding/json"

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

// Save - saves the document to elastic search
func (app *App) Save(fsNode FsNode, id string) error {
	ctx := context.Background()
	response, err := app.Client.Index().
		Index(app.Index).
		Type(docType).
		Id(id).
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

// Get - gets a document from the index given its id
func (app *App) Get(id string) (FsNode, error) {
	ctx := context.Background()
	doc, err := app.Client.Get().
		Index(app.Index).
		Type(docType).
		Id(id).
		Do(ctx)
	if err != nil {
		return FsNode{}, err
	}
	var fsNode FsNode
	json.Unmarshal(*doc.Source, &fsNode)
	return fsNode, nil
}

// GetAllFsNodes returns a list of all FsNodes, ordered by folder path
func (app *App) GetAllFsNodes() ([]FsNode, int64, error) {
	ctx := context.Background()
	q := elastic.NewMatchAllQuery()
	results, err := app.Client.
		Search().
		Index(app.Index).
		Query(q).
		Sort("fullPath.keyword", true).
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

// GetFsNodesForWatchFolder - given the start of a folder path, returns all
// documents that start with that folderpath, ordered by folder path
func (app *App) GetFsNodesForWatchFolder(folderPath string) ([]FsNode, int64, error) {
	ctx := context.Background()

	// full path is stroed in elastic search using path_hierarchy tokeniser, see
	// https://www.elastic.co/guide/en/elasticsearch/reference/current/analysis-pathhierarchy-tokenizer.html
	q := elastic.NewPrefixQuery("fullPath.tree", folderPath)
	results, err := app.Client.Search().
		Index(app.Index).
		Query(q).
		Sort("fullPath.keyword", true).
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
