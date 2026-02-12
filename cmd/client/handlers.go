package main

import (
	"fmt"

	"github.com/EthanColbert8/pub-sub-peril/internal/gamelogic"
	"github.com/EthanColbert8/pub-sub-peril/internal/pubsub"
	"github.com/EthanColbert8/pub-sub-peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(state routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(state)
		return pubsub.ACK
	}
}

func handlerArmyMoves(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.ACK

		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NACK_DISCARD

		case gamelogic.MoveOutcomeMakeWar:
			payload := gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.GetPlayerSnap(),
			}

			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.Player.Username), payload)
			if err != nil {
				fmt.Printf("Failed to publish recognition of war: %v\n", err)
				return pubsub.NACK_REQUEUE
			}

			// return pubsub.NACK_REQUEUE // I was promised this would be fun...
			return pubsub.ACK // ... it wasn't fun.

		default:
			return pubsub.NACK_DISCARD
		}
	}
}

func handlerWarRecognitions(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(row gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")

		outcome, _, _ := gs.HandleWar(row)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NACK_REQUEUE

		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NACK_DISCARD

		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.ACK

		case gamelogic.WarOutcomeYouWon:
			return pubsub.ACK

		case gamelogic.WarOutcomeDraw:
			return pubsub.ACK

		default:
			fmt.Println("Invalid outcome to war.")
			return pubsub.NACK_DISCARD
		}
	}
}
