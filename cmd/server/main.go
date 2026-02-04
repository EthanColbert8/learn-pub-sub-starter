package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/EthanColbert8/pub-sub-peril/internal/pubsub"
	"github.com/EthanColbert8/pub-sub-peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connectionString := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		fmt.Printf("Failed to connect to message broker: %v\n", err)
		return
	}
	defer connection.Close()

	fmt.Println("Message broker connection established! Starting Peril server...")

	amqpChannel, err := connection.Channel()
	if err != nil {
		fmt.Printf("Failed to open a channel: %v\n", err)
		return
	}
	defer amqpChannel.Close()

	// Sending a test message to the broker
	pausedState := routing.PlayingState{IsPaused: true}
	err = pubsub.PublishJSON(amqpChannel, routing.ExchangePerilDirect, routing.PauseKey, pausedState)

	/************************************************
	 * Just wait for an interrupt signal to shut down
	 */

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	// blocks the main thread until an interrupt
	<-signalChannel
	fmt.Println("\nShutdown signal received, closing server.")

}
