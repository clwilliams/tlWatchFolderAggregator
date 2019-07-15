package elasticSearch

import (
	"context"

	es "github.com/olivere/elastic"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
)

type hash map[string]interface{}

// App - represents the es configuration for the folder aggregation client
type App struct {
	Verbose          bool
	ElasticSearchURL string
	Index            string
	Client           *es.Client
}

// Connect - connects to the es client, & creates the index if needed
func Connect(verbose bool, esURL, esIndex string) (*App, error) {

	ctx := context.Background()

	// Initialise elastic search
	var esTraceLog es.Logger
	if verbose {
		esTraceLog = elasticLog{log.Logger}
	}

	// connect to the elastic search client
	client, err := es.NewClient(
		es.SetURL(esURL),
		es.SetErrorLog(elasticLog{log.Logger}),
		es.SetTraceLog(esTraceLog),
	)
	if err != nil {
		return nil, err
	}

	// ensure the index exists, if not create it
	_, err = ensureIndexExists(ctx, client, esIndex, tlFolderWatchMapping)
	if err != nil {
		return nil, err
	}

	app := &App{
		Client:           client,
		Verbose:          verbose,
		Index:            esIndex,
		ElasticSearchURL: esURL,
	}

	return app, nil
}

// ensureIndexExists - checks whether the given index exists, if not creates it
func ensureIndexExists(ctx context.Context, client *es.Client, indexName, mapping string) (bool, error) {
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		return false, err
	}
	if !exists {
		err := createIndex(ctx, client, indexName, mapping)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// createIndex - creates an index
func createIndex(ctx context.Context, client *es.Client, indexName, mapping string) error {
	createIndex, err := client.CreateIndex(indexName).BodyString(mapping).Do(ctx)
	if err != nil {
		return err
	}
	if !createIndex.Acknowledged {
		// Not acknowledged
	}
	return nil
}

type elasticLog struct {
	zerolog zerolog.Logger
}

func (el elasticLog) Printf(format string, vals ...interface{}) {
	el.zerolog.Printf(format, vals...)
}
