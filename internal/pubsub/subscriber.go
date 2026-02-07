package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	DURABLE   = SimpleQueueType("durable")
	TRANSIENT = SimpleQueueType("transient")
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("failed to open channel: %w", err)
	}

	// Since we only have transient and durable, we don'y need to worry about other types
	isTransient := queueType == TRANSIENT

	queue, err := channel.QueueDeclare(queueName, !isTransient, isTransient, isTransient, false, nil)
	if err != nil {
		channel.Close()
		return nil, amqp.Queue{}, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = channel.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		channel.Close()
		return nil, amqp.Queue{}, fmt.Errorf("failed to bind queue: %w", err)
	}

	return channel, queue, nil
}

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T)) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("failed to bind to queue: %w", err)
	}
	// defer channel.Close()

	queueChannel, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to consume from queue: %w", err)
	}

	go func(qch <-chan amqp.Delivery) {
		for msg := range qch {
			var data T
			err := json.Unmarshal(msg.Body, &data)
			if err != nil {
				fmt.Printf("failed to unmarshal message: %v\n", err)
				err = msg.Nack(false, false)
				if err != nil {
					fmt.Printf("negative message acknowledgement failed: %v\n", err)
				}
			}

			handler(data)

			err = msg.Ack(false)
			if err != nil {
				fmt.Printf("message acknowledgement failed: %v\n", err)
			}
		}
	}(queueChannel)

	return nil
}
