package pubsub

import (
	"bytes"
	"encoding/gob"
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

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) AckType) (*amqp.Channel, error) {
	jsonUnmarshal := func(data []byte) (T, error) {
		var result T
		err := json.Unmarshal(data, &result)
		return result, err
	}

	return subscribe(conn, exchange, queueName, key, queueType, handler, jsonUnmarshal)
}

func SubscribeGob[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) AckType) (*amqp.Channel, error) {
	gobUnmarshal := func(data []byte) (T, error) {
		payload := bytes.NewReader(data)
		dec := gob.NewDecoder(payload)

		var result T
		err := dec.Decode(&result)
		return result, err
	}

	return subscribe(conn, exchange, queueName, key, queueType, handler, gobUnmarshal)
}

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

	queueTable := make(amqp.Table)
	queueTable["x-dead-letter-exchange"] = "peril_dlx" // dead letter exchange name

	queue, err := channel.QueueDeclare(queueName, !isTransient, isTransient, isTransient, false, queueTable)
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

func subscribe[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) (*amqp.Channel, error) {
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
			data, err := unmarshaller(msg.Body)
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
