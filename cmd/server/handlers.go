package main

import (
	"fmt"

	"github.com/EthanColbert8/pub-sub-peril/internal/gamelogic"
	"github.com/EthanColbert8/pub-sub-peril/internal/pubsub"
	"github.com/EthanColbert8/pub-sub-peril/internal/routing"
)

func handlerGameLogs(log routing.GameLog) pubsub.AckType {
	defer fmt.Print("> ")

	err := gamelogic.WriteLog(log)
	if err != nil {
		fmt.Printf("Failed to write game log: %v\n", err)
		return pubsub.NACK_DISCARD
	}

	return pubsub.ACK
}
