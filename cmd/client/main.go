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

	gameState := gamelogic.NewGameState(userName)

	// amqpChannel, amqpQueue, err := pubsub.DeclareAndBind(
	// 	connection,
	// 	routing.ExchangePerilDirect,
	// 	fmt.Sprintf("%s.%s", routing.PauseKey, userName),
	// 	routing.PauseKey,
	// 	pubsub.TRANSIENT,
	// )
	// if err != nil {
	// 	fmt.Printf("Failed to declare and bind queue: %v\n", err)
	// 	return
	// }
	// defer amqpChannel.Close()
	directChannel, err := pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, userName),
		routing.PauseKey,
		pubsub.TRANSIENT,
		handlerPause(gameState),
	)
	if err != nil {
		fmt.Printf("Failed to subscribe to queue: %v\n", err)
		return
	}
	defer directChannel.Close()

	topicChannel, err := pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, userName),
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		pubsub.TRANSIENT,
		handlerArmyMoves(gameState),
	)
	if err != nil {
		fmt.Printf("Failed to subscribe to queue: %v\n", err)
		return
	}
	defer topicChannel.Close()

	//fmt.Printf("Successfully created queue: %s\n", amqpQueue.Name)

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
				move, err := gameState.CommandMove(words)
				if err != nil {
					fmt.Println(err)
					continue
				}

				err = pubsub.PublishJSON(topicChannel, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, move.Player.Username), move)
				if err != nil {
					fmt.Printf("Failed to publish army move: %v\n", err) // Should we do something else here?
					continue
				}
				fmt.Println("Published move successfully.")
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
