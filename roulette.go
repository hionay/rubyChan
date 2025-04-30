package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

var (
	rouletteMu      sync.Mutex
	rouletteRnd     *rand.Rand
	rouletteCount   = 0
	rouletteChamber = 0
)

func handleRoulette(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID) {
	rouletteMu.Lock()
	defer rouletteMu.Unlock()

	if rouletteRnd == nil {
		rouletteRnd = rand.New(rand.NewSource(time.Now().UnixNano()))
		rouletteCount = 0
		rouletteChamber = rouletteRnd.Intn(6) + 1
	}
	rouletteCount++

	var reply string
	if rouletteChamber == rouletteCount {
		reply = fmt.Sprintf(`(%d/6) 💥 Bang! You’re dead.`, rouletteCount)
		rouletteRnd = nil
	} else {
		reply = fmt.Sprintf("(%d/6) click... you survived.", rouletteCount)
	}
	messageWithMention(ctx, cli, roomID, sender, reply)
}
