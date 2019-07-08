package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	stdlog "log"

	"github.com/alecthomas/kingpin"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/tlWatchFolderAggregator/internal"
	"github.com/gorilla/handlers"
  "github.com/gorilla/mux"
	"github.com/tlWatchFolderAggregator/elasticSearch"
	"github.com/streadway/amqp"

  "net/http"

)

const (
	defaultRabbitMqHost     = "localhost"
	defaultRabbitMqPort     = "5672"
	defaultRabbitMqUser     = "rabbitmq"
	defaultRabbitMqPassword = "rabbitmq"
	defaultEsURL            = "http://localhost:9200"
	defaultEsIndex          = "tl-watch"
	defaultAPIPort          = "3001"
	defaultHandlerTimeout   = "50000"
)

var (
	dev              = kingpin.Flag("dev", "Run app in development mode, no-dev for production").Default("true").Envar("DEV").Bool()
	verbose          = kingpin.Flag("verbose", "Enable verbose mode").Envar("VERBOSE").Bool()
	rabbitMqHost     = kingpin.Flag("rabbit-mq-host", "").Envar("RABBITMQ_HOST").Default(defaultRabbitMqHost).String()
	rabbitMqPort     = kingpin.Flag("rabbit-mq-port", "").Envar("RABBITMQ_PORT").Default(defaultRabbitMqPort).String()
	rabbitMqUser     = kingpin.Flag("rabbit-mq-user", "").Envar("RABBITMQ_USER").Default(defaultRabbitMqUser).String()
	rabbitMqPassword = kingpin.Flag("rabbit-mq-password", "").Envar("RABBITMQ_PASSWORD").Default(defaultRabbitMqPassword).String()
	elasticURL       = kingpin.Flag("es-host", "ElasticSearch URL").Short('u').Envar("ES_URL").Default(defaultEsURL).String()
  elasticIndex     = kingpin.Flag("es-index", "ElasticSearch index").Short('i').Envar("ES_INDEX").Default(defaultEsIndex).String()
	apiPort          = kingpin.Flag("api-port", "REST API port").Envar("API_PORT").Short('a').Default(defaultAPIPort).String()
	handlerTimeout   = kingpin.Flag("handler-timeout", "Timeout in milliseconds for handling a message").Default(defaultHandlerTimeout).Int()
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

	// Initialise elastic search
  esApp, err := elasticSearch.New(*verbose, *elasticURL, *elasticIndex)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to ElasticSearch")
	}
	// defer esApp.Client.Close .es.Close()

	// Initialise the router so we can serve API requests
	server(esApp)


	// Initialise Rabbit MQ
	/*
	rabbitMqClient, err := rabbitMQ.NewClient(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMqClient.Close()*/

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", *rabbitMqUser, *rabbitMqPassword, *rabbitMqHost, *rabbitMqPort))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer conn.Close()
	conn.Channel()

	c, err := conn.Channel()
	defer c.Close()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ Channel")
	}
	if err = c.ExchangeDeclare("thirdlight", "topic", true, false, false, false, nil); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ Topic")
	}

	errLog := make(chan error)
	defer close(errLog)

	var wg sync.WaitGroup
	type bind struct {
		queue   string
		key     string
		handler func(context.Context, []byte) error
	}
// func HandleFolderWatchUpdate(config *elasticSearch.App, folderWatchMsg *rabbitMQ.FolderWatch) error {
	for _, binding := range []bind{
		{"thirdlight", "watcher_update", internal.HandleFolderWatchUpdate(esApp)},
	} {
		if _, err := c.QueueDeclare(binding.queue, true, false, false, false, nil); err != nil {
			log.Error().Err(err).Str(binding.queue,binding.queue).Msg("Problem declaring queue")
			// log.Errorf("Problem declaring queue %s: %v", binding.queue, err)
		}
		if err := c.QueueBind(binding.queue, binding.key, "thirdlight", false, nil); err != nil {
			log.Error().Err(err).Str(binding.queue,binding.queue).Msg("Problem binding")
		}
		if err := c.Qos(3, 0, false); err != nil {
			log.Error().Err(err).Msg("Problem setting QOS")
		}

		deliveries, err := c.Consume(binding.queue, "", false, false, false, false, nil)
		if err != nil {
			log.Error().Err(err).Str(binding.queue,binding.queue).Msg("Problem setting consumer for")
		}

		wg.Add(1)
		go func(binding bind) {
			defer wg.Done()
			for delivery := range deliveries {
				log.Printf("Reading on %v", binding.queue)
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*handlerTimeout)*time.Millisecond)
				if err := binding.handler(ctx, delivery.Body); err != nil {
					log.Printf("ErrLogging")
					errLog <- err
					if err := delivery.Nack(false, false); err != nil { // TODO - requeue?
						errLog <- err
					}
					log.Printf("Aborting %v", binding.queue)
				}
				if err := delivery.Ack(false); err != nil {
					errLog <- err
				}
				cancel()
			}
		}(binding)
	}

	go func() {
		log.Printf("reading error log")
		for err := range errLog {
			log.Error().Err(err).Msg("ERROR :")
		}
	}()
	log.Printf("waiting...")
	wg.Wait()
	log.Printf("done.")



	// start listening to the RabbitMQ queue & processing the folder / file messages
  // TODO

}
