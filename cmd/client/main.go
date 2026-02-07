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

	fmt.Println("Message broker connection established! Starting Peril server...")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println(err)
		return
	}

	amqpChannel, amqpQueue, err := pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, userName),
		routing.PauseKey,
		pubsub.TRANSIENT,
	)
	if err != nil {
		fmt.Printf("Failed to declare and bind queue: %v\n", err)
		return
	}
	defer amqpChannel.Close()

	fmt.Printf("Successfully created queue: %s\n", amqpQueue.Name)

	gameState := gamelogic.NewGameState(userName)

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			{
				err := gameState.CommandSpawn(words)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}

		case "move":
			{
				_, err := gameState.CommandMove(words)
				if err != nil {
					fmt.Println(err)
					continue
				}
				//fmt.Printf("%d units moved to %s\n", len(move.Units), move.ToLocation)
			}

		case "status":
			{
				gameState.CommandStatus()
			}

		case "help":
			{
				gamelogic.PrintClientHelp()
			}

		case "spam":
			{
				fmt.Println("Spamming not allowed yet!")
			}

		case "quit":
			{
				gamelogic.PrintQuit()
				return
			}

		default:
			{
				fmt.Printf("Unknown command: %s\n", words[0])
			}
		}
	}
}
