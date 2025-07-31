package roulette

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"

	"github.com/hionay/rubyChan/state"
)

type RouletteCmd struct {
	Store *state.Namespace
}

func (*RouletteCmd) Name() string      { return "roulette" }
func (*RouletteCmd) Aliases() []string { return nil }
func (*RouletteCmd) Usage() string     { return "!roulette - Play Russian Roulette" }

type roomState struct {
	Click   int `json:"click"`
	Chamber int `json:"chamber"`
}

func (c *RouletteCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, _ []string) {
	st := &roomState{}
	if err := c.Store.GetJSON(evt.RoomID.String(), st); err != nil {
		log.Printf("roulette: error loading state: %v", err)
		cli.SendText(ctx, evt.RoomID, "Internal error")
		return
	}
	if st.Chamber == 0 {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		st.Chamber = rnd.Intn(6) + 1
	}

	st.Click++
	var reply string

	if st.Click == st.Chamber {
		reply = fmt.Sprintf("(%d/6) 💥 Bang! You’re dead.", st.Click)
		if err := c.Store.Delete(evt.RoomID.String()); err != nil {
			log.Printf("roulette: error deleting state: %v", err)
		}
	} else {
		reply = fmt.Sprintf("(%d/6) click... you survived.", st.Click)
		if err := c.Store.PutJSON(evt.RoomID.String(), st); err != nil {
			log.Printf("roulette: error saving state: %v", err)
		}
	}

	mention := fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, evt.Sender, evt.Sender)
	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          fmt.Sprintf("%s: %s", evt.Sender, reply),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("%s: %s", mention, reply),
	}
	if _, err := cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content); err != nil {
		log.Printf("roulette: failed to send reply: %v", err)
	}
}
