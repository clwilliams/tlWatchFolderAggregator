package rabbitMQ

import (
	"fmt"
	"github.com/streadway/amqp"
)

// Client wrapper for rabbitMQ
type Client struct {
	*amqp.Connection
}

// NewClient -
func NewClient(rabbitMqHost, rabbitMqPort, rabbitMqUser, rabbitMqPassword *string) (*Client, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", *rabbitMqUser, *rabbitMqPassword, *rabbitMqHost, *rabbitMqPort))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn,
	}, nil
}
