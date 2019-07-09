package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	stdlog "log"

	"github.com/alecthomas/kingpin"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	"github.com/tlCommonMessaging/rabbitMQ"
	"github.com/tlWatchFolderAggregator/elasticSearch"
	"github.com/tlWatchFolderAggregator/internal"

	"net/http"
)

const (
	defaultRabbitMqHost       = "localhost"
	defaultRabbitMqPort       = "5672"
	defaultRabbitMqUser       = "rabbitmq"
	defaultRabbitMqPassword   = "rabbitmq"
	defaultRabbitMqExchange   = "thirdlight"
	defaultRabbitMqQueue      = "watcher"
	defaultRabbitMqRoutingKey = "crud"
	defaultEsURL              = "http://localhost:9200"
	defaultEsIndex            = "tl-watch"
	defaultAPIPort            = "3001"
	defaultHandlerTimeout     = "50000"
)

var (
	dev                = kingpin.Flag("dev", "Run app in development mode, no-dev for production").Default("true").Envar("DEV").Bool()
	verbose            = kingpin.Flag("verbose", "Enable verbose mode").Envar("VERBOSE").Bool()
	rabbitMqHost       = kingpin.Flag("rabbit-mq-host", "").Envar("RABBITMQ_HOST").Default(defaultRabbitMqHost).String()
	rabbitMqPort       = kingpin.Flag("rabbit-mq-port", "").Envar("RABBITMQ_PORT").Default(defaultRabbitMqPort).String()
	rabbitMqUser       = kingpin.Flag("rabbit-mq-user", "").Envar("RABBITMQ_USER").Default(defaultRabbitMqUser).String()
	rabbitMqPassword   = kingpin.Flag("rabbit-mq-password", "").Envar("RABBITMQ_PASSWORD").Default(defaultRabbitMqPassword).String()
	rabbitMqExchange   = kingpin.Flag("rabbit-mq-exchange", "").Default(defaultRabbitMqExchange).String()
	rabbitMqQueue      = kingpin.Flag("rabbit-mq-queue", "").Default(defaultRabbitMqQueue).String()
	rabbitMqRoutingKey = kingpin.Flag("rabbit-mq-routing-key", "").Default(defaultRabbitMqRoutingKey).String()
	elasticURL         = kingpin.Flag("es-url", "ElasticSearch URL").Short('u').Envar("ES_URL").Default(defaultEsURL).String()
	elasticIndex       = kingpin.Flag("es-index", "ElasticSearch index").Short('i').Envar("ES_INDEX").Default(defaultEsIndex).String()
	apiPort            = kingpin.Flag("api-port", "REST API port").Envar("API_PORT").Short('a').Default(defaultAPIPort).String()
	handlerTimeout     = kingpin.Flag("handler-timeout", "Timeout in milliseconds for handling a message").Default(defaultHandlerTimeout).Int()
)

func init() {
	// Only log the warning severity or above.
	log.Level(zerolog.WarnLevel)
}

func server(esApp *elasticSearch.App) {
	router := mux.NewRouter()

	// routes we're going to handle
	router.Handle("/all", internal.GetAll(esApp)).Methods("GET")
	router.Handle("/watch", internal.GetFsNodesForWatchFolder(esApp)).Methods("GET")

	host := fmt.Sprintf(":%s", *apiPort)
	log.Printf("Listening on %s...\n", host)
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
	stdlog.Fatal(http.ListenAndServe(":8000", loggedRouter))
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
	esApp, err := elasticSearch.Connect(*verbose, *elasticURL, *elasticIndex)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to ElasticSearch")
	}
	// defer esApp.Client.Close .es.Close()

	// Initialise Rabbit MQ
	rabbitMQClient := rabbitMQ.MessageClient{}
	err = rabbitMQClient.Connect(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMQClient.Connection.Close()

	// (b) channel
	rabbitMqChannel, err := rabbitMQClient.Connection.Channel()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ Channel")
	}
	defer rabbitMqChannel.Close()

	// (c) exchange
	if err = rabbitMqChannel.ExchangeDeclare(*rabbitMqExchange, "topic", true, false, false, false, nil); err != nil {
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

	for _, binding := range []bind{
		{*rabbitMqQueue, *rabbitMqRoutingKey, internal.HandleFolderWatchUpdate(esApp)},
	} {
		if _, err := rabbitMqChannel.QueueDeclare(binding.queue, true, false, false, false, nil); err != nil {
			log.Error().Err(err).Str(binding.queue, binding.queue).Msg("Problem declaring queue")
		}
		if err := rabbitMqChannel.QueueBind(binding.queue, binding.key, *rabbitMqExchange, false, nil); err != nil {
			log.Error().Err(err).Str(binding.queue, binding.queue).Msg("Problem binding")
		}
		if err := rabbitMqChannel.Qos(3, 0, false); err != nil {
			log.Error().Err(err).Msg("Problem setting QOS")
		}
		deliveries, err := rabbitMqChannel.Consume(binding.queue, "", false, false, false, false, nil)
		if err != nil {
			log.Error().Err(err).Str(binding.queue, binding.queue).Msg("Problem setting consumer for")
		}

		wg.Add(1)
		go func(binding bind) {
			defer wg.Done()
			for delivery := range deliveries {
				log.Printf("Reading on %v", binding.queue)
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*handlerTimeout)*time.Millisecond)
				if err := binding.handler(ctx, delivery.Body); err != nil {
					log.Printf("ErrLogging")
					/*
					errLog <- err
					if err := delivery.Nack(false, false); err != nil { // TODO - requeue?
						errLog <- err
					}
					log.Printf("Aborting %v", binding.queue)
					*/
				}
				if err := delivery.Ack(false); err != nil {
					errLog <- err
				}
				cancel()
			}
		}(binding)
	}
/*
	go func() {
		log.Printf("reading error log")
		for err := range errLog {
			log.Error().Err(err).Msg("ERROR :")
		}
	}()
	log.Printf("waiting...")
	wg.Wait()
	log.Printf("done.")
	*/

	// Lastly initialise the router so we can serve API requests
	server(esApp)
}
