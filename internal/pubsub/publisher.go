package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	messageData, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msgToPublish := amqp.Publishing{
		ContentType: "application/json",
		Body:        messageData,
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msgToPublish)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
