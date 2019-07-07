package elasticSearch

import (
  "context"
  "encoding/json"
  log "github.com/rs/zerolog/log"
)
// FsNode represents a file server node
type FsNode struct {
  ID string `json:"id"`
  Name string `json:"name"`
  IsDir string `json:"isDir"`
  Path string `json:"path"`
}
const docType = "document"

// Save - saves the document
func (app *App) Save(fsNode FsNode) error {
	ctx := context.Background()
	response, err := app.Client.Index().Index(app.Index).
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
  ctx := context.Background()
  query := `"match_all": {}`
  results, err := app.Client.Search().Index(app.Index).Source(query).Do(ctx)
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
