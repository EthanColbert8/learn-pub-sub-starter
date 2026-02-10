package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	DURABLE   = SimpleQueueType(0)
	TRANSIENT = SimpleQueueType(1)
)

type AckType int

const (
	ACK          = AckType(0)
	NACK_REQUEUE = AckType(1)
	NACK_DISCARD = AckType(2)
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

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) AckType) (*amqp.Channel, error) {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to queue: %w", err)
	}

	queueChannel, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		channel.Close()
		return nil, fmt.Errorf("failed to consume from queue: %w", err)
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

			response := handler(data)

			switch response {
			case ACK:
				err = msg.Ack(false)
				fmt.Println("\nAcknowledging message")
			case NACK_REQUEUE:
				err = msg.Nack(false, true)
				fmt.Println("\nNacking message, requeueing")
			case NACK_DISCARD:
				err = msg.Nack(false, false)
				fmt.Println("\nNacking message, discarding")
			}
			if err != nil {
				fmt.Printf("message acknowledgement failed: %v\n", err)
			}
			fmt.Print("> ")
		}
	}(queueChannel)

	return channel, nil
}
