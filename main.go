package main

import (
	"fmt"
	"os"

	stdlog "log"

	"github.com/alecthomas/kingpin"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/tlWatchFolderAggregator/internal"
	"github.com/gorilla/handlers"
  "github.com/gorilla/mux"
	"github.com/tlWatchFolderAggregator/elasticSearch"
	"github.com/tlWatchFolderAggregator/rabbitMQ"

  "net/http"

)

const (
	defaultRabbitMqHost     = "localhost"
	defaultRabbitMqPort     = "5672"
	defaultRabbitMqUser     = "rabbitmq"
	defaultRabbitMqPassword = "rabbitmq"
	defaultEsHost           = "localhost"
	defaultEsPort           = "3000"
	defaultEsIndex          = "tl-watch"
	defaultAPIPort          = "3001"
)

var (
	dev              = kingpin.Flag("dev", "Run app in development mode, no-dev for production").Default("true").Envar("DEV").Bool()
	verbose          = kingpin.Flag("verbose", "Enable verbose mode").Envar("VERBOSE").Bool()
	rabbitMqHost     = kingpin.Flag("rabbit-mq-host", "").Envar("RABBITMQ_HOST").Default(defaultRabbitMqHost).String()
	rabbitMqPort     = kingpin.Flag("rabbit-mq-port", "").Envar("RABBITMQ_PORT").Default(defaultRabbitMqPort).String()
	rabbitMqUser     = kingpin.Flag("rabbit-mq-user", "").Envar("RABBITMQ_USER").Default(defaultRabbitMqUser).String()
	rabbitMqPassword = kingpin.Flag("rabbit-mq-password", "").Envar("RABBITMQ_PASSWORD").Default(defaultRabbitMqPassword).String()
	elasticHost      = kingpin.Flag("es-host", "ElasticSearch hostname").Short('h').Envar("ES_HOST").Default(defaultEsHost).String()
  elasticIndex     = kingpin.Flag("es-index", "ElasticSearch index").Short('i').Envar("ES_INDEX").Default(defaultEsIndex).String()
  elasticPort      = kingpin.Flag("es-port", "ElasticSearch port").Envar("ES_PORT").Short('p').Default(defaultEsPort).String()
	apiPort          = kingpin.Flag("api-port", "REST API port").Envar("API_PORT").Short('a').Default(defaultAPIPort).String()
)

func init() {
	// Only log the warning severity or above.
	log.Level(zerolog.WarnLevel)
}

func server(esApp *elasticSearch.App) {
  router := mux.NewRouter().StrictSlash(true)

	// routes we're going to handle
	router.Handle("/get", internal.GetAll(esApp)).Methods("GET")

	host := fmt.Sprintf(":%s", *apiPort)
  log.Printf("Listening on %s...\n", host)
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
  stdlog.Fatal(http.ListenAndServe(host, loggedRouter))
}

func main() {
	// parse the command line arguments
	// kingpin.Version(version.Get())
	kingpin.Parse()

	// Initialise Logging
	if *dev {
		log.Level(zerolog.InfoLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	if *verbose {
		log.Level(zerolog.DebugLevel)
		log.Debug().Msg("Set logging to verbose")
	}

	// Initialise Rabbit MQ
	rabbitMqClient, err := rabbitMQ.NewClient(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMqClient.Close()

	// Initialise elastic search
  esApp, err := elasticSearch.New(*verbose, *elasticHost, *elasticIndex, *elasticPort)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to ElasticSearch")
	}
	// defer esApp.Client.Close .es.Close()

	// Initialise the router so we can serve API requests
	server(esApp)

	// start listening to the RabbitMQ queue & processing the folder / file messages
  // TODO

}
