package main

import (
	"fmt"
	"time"

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

func handlerWarRecognitions(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(row gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")

		outcome, winner, loser := gs.HandleWar(row)

		var logMsg string
		var publishLog bool = false
		var response pubsub.AckType

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			response = pubsub.NACK_REQUEUE

		case gamelogic.WarOutcomeNoUnits:
			response = pubsub.NACK_DISCARD

		case gamelogic.WarOutcomeOpponentWon:
			logMsg = fmt.Sprintf("%s won a war against %s", winner, loser)
			publishLog = true
			response = pubsub.ACK

		case gamelogic.WarOutcomeYouWon:
			logMsg = fmt.Sprintf("%s won a war against %s", winner, loser)
			publishLog = true
			response = pubsub.ACK

		case gamelogic.WarOutcomeDraw:
			logMsg = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			publishLog = true
			response = pubsub.ACK

		default:
			fmt.Println("Invalid outcome to war.")
			response = pubsub.NACK_DISCARD
		}

		if publishLog {
			log := routing.GameLog{
				CurrentTime: time.Now(),
				Message:     logMsg,
				Username:    gs.Player.Username,
			}

			err := publishGameLog(ch, log)
			if err != nil {
				fmt.Printf("Failed to publish game log: %v\n", err)
				response = pubsub.NACK_REQUEUE
			}
		}

		return response
	}
}

func publishGameLog(ch *amqp.Channel, log routing.GameLog) error {
	return pubsub.PublishGob(
		ch,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.GameLogSlug, log.Username),
		log,
	)
}
