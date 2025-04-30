package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		cancel()
		log.Fatalf("run(): %v", err)
	}
}

func run(ctx context.Context) error {
	cfg, err := NewConfig()
	if err != nil {
		return fmt.Errorf("NewConfig(): %w", err)
	}

	cli, err := mautrix.NewClient(cfg.MatrixServer, "", "")
	if err != nil {
		return fmt.Errorf("mautrix.NewClient(%q): %w", cfg.MatrixServer, err)
	}

	resp, err := cli.Login(ctx, &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			User: cfg.MatrixUsername,
			Type: mautrix.IdentifierTypeUser,
		},
		Password:         cfg.MatrixPassword,
		StoreCredentials: true,
	})
	if err != nil {
		return fmt.Errorf("cli.Login(): %w", err)
	}
	log.Printf("Logged in as %s", resp.UserID)

	startTime := time.Now()
	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, parseMessage(cli, cfg, startTime))
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == cli.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := cli.JoinRoomByID(ctx, evt.RoomID)
			if err != nil {
				log.Printf("JoinRoomByID error: %v", err)
				return
			}
			var content struct {
				Name string `json:"name,omitempty"`
			}
			err = cli.StateEvent(ctx, evt.RoomID, event.StateRoomName, "", &content)
			if err != nil {
				log.Printf("StateEvent error: %v", err)
				return
			}
			log.Printf("Joined room %s: %s", evt.RoomID, content.Name)
		}
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cli.SyncWithContext(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("Sync canceled")
				return
			}
			log.Printf("Sync error: %v", err)
		}
	}()
	wg.Wait()
	return nil
}
