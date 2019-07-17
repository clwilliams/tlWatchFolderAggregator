package main

import (
	"context"
	"fmt"
	"os"
	"time"

	stdlog "log"

	"github.com/alecthomas/kingpin"
	"github.com/clwilliams/tlCommonMessaging/rabbitMQ"
	"github.com/clwilliams/tlWatchFolderAggregator/elasticSearch"
	"github.com/clwilliams/tlWatchFolderAggregator/internal"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"

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
	rabbitMqExchange   = kingpin.Flag("rabbit-mq-exchange", "").Envar("RABBITMQ_EXCHANGE").Default(defaultRabbitMqExchange).String()
	rabbitMqQueue      = kingpin.Flag("rabbit-mq-queue", "").Envar("RABBITMQ_QUEUE").Default(defaultRabbitMqQueue).String()
	rabbitMqRoutingKey = kingpin.Flag("rabbit-mq-routing-key", "").Envar("RABBITMQ_ROUTING_KEY").Default(defaultRabbitMqRoutingKey).String()
	elasticURL         = kingpin.Flag("es-url", "ElasticSearch URL").Short('u').Envar("ES_URL").Default(defaultEsURL).String()
	elasticIndex       = kingpin.Flag("es-index", "ElasticSearch index").Short('i').Envar("ES_INDEX").Default(defaultEsIndex).String()
	apiPort            = kingpin.Flag("api-port", "REST API port").Envar("API_PORT").Short('a').Default(defaultAPIPort).String()
	handlerTimeout     = kingpin.Flag("handler-timeout", "Timeout in milliseconds for message handler").Default(defaultHandlerTimeout).Int()
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
	kingpin.Parse()

	// Initialise Logging
	// by default set to warn level
	// if we're in development mode, default to info level
	// if the verbose flag is set, set to the verbose level
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

	// initialise connection to elastic search, which will also ensure the index
	// that we want to use exixts. If not it will create it
	esApp, err := elasticSearch.Connect(*verbose, *elasticURL, *elasticIndex)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to ElasticSearch")
	}
	defer esApp.Client.Stop()

	// Initialise Rabbit MQ
	rabbitMQClient := rabbitMQ.MessageClient{}
	err = rabbitMQClient.Connect(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMQClient.Connection.Close()

	// configure the RabbitMQ channel and exchange
	err = rabbitMQClient.ConfigureChannelAndExchange(rabbitMqExchange)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to configure RabbitMQ Channel / Exchange")
	}
	defer rabbitMQClient.Channel.Close()

	// create an error channel to receive any errors when processing the messages
	errLog := make(chan error)
	defer close(errLog)

	type bind struct {
		queue   string
		key     string
		handler func(context.Context, []byte) error
	}

	// configure the queue / routing key to a message handler
	// (potential to configure many if needed)
	for _, binding := range []bind{
		{*rabbitMqQueue, *rabbitMqRoutingKey, internal.HandleFolderWatchUpdate(esApp)},
	} {
		if _, err := rabbitMQClient.Channel.QueueDeclare(binding.queue, true, false, false, false, nil); err != nil {
			log.Error().Err(err).Str(binding.queue, binding.queue).Msg("Problem declaring queue")
		}
		if err := rabbitMQClient.Channel.QueueBind(binding.queue, binding.key, *rabbitMqExchange, false, nil); err != nil {
			log.Error().Err(err).Str(binding.queue, binding.queue).Msg("Problem binding")
		}
		if err := rabbitMQClient.Channel.Qos(3, 0, false); err != nil {
			log.Error().Err(err).Msg("Problem setting QOS")
		}
		deliveries, err := rabbitMQClient.Channel.Consume(binding.queue, "", false, false, false, false, nil)
		if err != nil {
			log.Error().Err(err).Str(binding.queue, binding.queue).Msg("Problem setting consumer for")
		}

		// start a thread for the queue / handler
		go func(binding bind) {
			// listen for messages being delivered
			for delivery := range deliveries {
				log.Printf("Reading on %v", binding.queue)
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*handlerTimeout)*time.Millisecond)
				if err := binding.handler(ctx, delivery.Body); err != nil {
					// there has been an error processing the message
					log.Printf("ErrLogging")
					// add to the error log
					errLog <- err
					// and negatively acknowledge message processing
					if err := delivery.Nack(false, false); err != nil {
						errLog <- err
					}
					log.Printf("Aborting %v", binding.queue)
				}
				// successfully processed message - acknowledge
				if err := delivery.Ack(false); err != nil {
					errLog <- err
				}
				// release resources if slowOperation completes before timeout elapses
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
	log.Printf("done.")

	// Lastly initialise the router so we can serve API requests
	server(esApp)
}
