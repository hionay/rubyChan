package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"modernc.org/sqlite"

	"github.com/hionay/rubyChan/command"
	"github.com/hionay/rubyChan/command/calc"
	"github.com/hionay/rubyChan/command/fact"
	"github.com/hionay/rubyChan/command/gif"
	"github.com/hionay/rubyChan/command/joke"
	"github.com/hionay/rubyChan/command/poll"
	"github.com/hionay/rubyChan/command/quote"
	"github.com/hionay/rubyChan/command/reminder"
	"github.com/hionay/rubyChan/command/repo"
	"github.com/hionay/rubyChan/command/roulette"
	"github.com/hionay/rubyChan/command/search"
	"github.com/hionay/rubyChan/command/weather"
	"github.com/hionay/rubyChan/history"
	"github.com/hionay/rubyChan/state"
)

func init() {
	sql.Register("sqlite3-fk-wal", &sqlite.Driver{})
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		cancel()
		log.Fatalf("run(): %v", err)
	}
}

func run(ctx context.Context) error {
	storePath := "bot_state.db"
	if v := os.Getenv("BOT_STATE_DB_PATH"); v != "" {
		storePath = v
	}
	store, err := state.NewStore(storePath)
	if err != nil {
		return fmt.Errorf("state.NewStore(): %w", err)
	}
	defer store.Close()

	rouletteNS, err := store.Namespace("roulette")
	if err != nil {
		return fmt.Errorf("store.Namespace(roulette): %w", err)
	}
	weatherNS, err := store.Namespace("weather")
	if err != nil {
		return fmt.Errorf("store.Namespace(weather): %w", err)
	}

	cfg, err := NewConfig()
	if err != nil {
		return fmt.Errorf("NewConfig(): %w", err)
	}

	cli, err := mautrix.NewClient(cfg.MatrixServer, "", "")
	if err != nil {
		return fmt.Errorf("mautrix.NewClient(%q): %w", cfg.MatrixServer, err)
	}

	historyStore := history.NewHistoryStore(100)
	command.Register(
		&calc.CalcCmd{},
		&command.HelpCmd{},
		&joke.JokeCmd{},
		&quote.QuoteCmd{History: historyStore},
		&reminder.RemindMeCmd{},
		&roulette.RouletteCmd{Store: rouletteNS},
		&search.SearchCmd{GoogleAPIKey: cfg.GoogleAPIKey, GoogleCX: cfg.GoogleCX},
		&weather.WeatherCmd{Store: weatherNS, WeatherAPIKey: cfg.WeatherAPIKey},
		&repo.RepoCmd{},
		&fact.FactCmd{},
		&poll.PollCmd{},
		&gif.GifCmd{APIKey: cfg.TenorAPIKey},
	)

	syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, parseMessage(cli, historyStore))
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

	cryptoHelper, err := cryptohelper.NewCryptoHelper(cli, []byte("onay"), "db/crypto.db")
	if err != nil {
		return fmt.Errorf("cryptohelper.NewCryptoHelper(): %w", err)
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			User: cfg.MatrixUsername,
			Type: mautrix.IdentifierTypeUser,
		},
		Password:         cfg.MatrixPassword,
		StoreCredentials: true,
	}
	if err := cryptoHelper.Init(ctx); err != nil {
		return fmt.Errorf("cryptoHelper.Init(): %w", err)
	}
	cli.Crypto = cryptoHelper
	log.Printf("Logged in as %s", cli.UserID)

	srv := newWebhookServer(cli, cfg.WebhookAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Println("Server closed")
				return
			}
			log.Printf("Server error: %v", err)
		}
	}()

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

	<-ctx.Done()
	sCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(sCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	wg.Wait()
	return nil
}
