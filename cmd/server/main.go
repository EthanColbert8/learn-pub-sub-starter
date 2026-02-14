package main

import (
	"fmt"

	"github.com/EthanColbert8/pub-sub-peril/internal/gamelogic"
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

	publishChannel, err := connection.Channel()
	if err != nil {
		fmt.Printf("Failed to open channel: %v\n", err)
		return
	}
	defer publishChannel.Close()

	fmt.Println("Message broker connection established! Starting Peril server...")

	gamelogsChannel, err := pubsub.SubscribeGob(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.DURABLE,
		handlerGameLogs,
	)
	if err != nil {
		fmt.Printf("Failed to subscribe to game logs: %v\n", err)
		return
	}
	defer gamelogsChannel.Close()

	pausedState := routing.PlayingState{IsPaused: true}

	gamelogic.PrintServerHelp()

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			{
				pausedState.IsPaused = true
				fmt.Println("Pausing game...")
				err = pubsub.PublishJSON(publishChannel, routing.ExchangePerilDirect, routing.PauseKey, pausedState)
				if err != nil {
					fmt.Printf("Failed to publish pause message: %v\n", err)
				}
			}

		case "resume":
			{
				pausedState.IsPaused = false
				fmt.Println("Resuming game...")
				err = pubsub.PublishJSON(publishChannel, routing.ExchangePerilDirect, routing.PauseKey, pausedState)
				if err != nil {
					fmt.Printf("Failed to publish resume message: %v\n", err)
				}
			}

		case "help":
			{
				gamelogic.PrintServerHelp()
			}

		case "quit":
			{
				fmt.Println("Shutting down server...")
				return
			}

		default:
			{
				fmt.Println("Unknown command.")
			}
		}
	}
}
