package roulette

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type roomState struct {
	click   int
	chamber int
}

type RouletteCmd struct {
	mu         sync.Mutex
	roomStates map[id.RoomID]*roomState
}

func (c *RouletteCmd) Name() string      { return "roulette" }
func (c *RouletteCmd) Aliases() []string { return nil }
func (c *RouletteCmd) Usage() string     { return "!roulette - Play Russian Roulette" }

func (c *RouletteCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, _ []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.roomStates == nil {
		c.roomStates = make(map[id.RoomID]*roomState)
	}
	st, ok := c.roomStates[evt.RoomID]
	if !ok {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		st = &roomState{
			click:   0,
			chamber: rnd.Intn(6) + 1,
		}
		c.roomStates[evt.RoomID] = st
	}
	st.click++
	var reply string
	if st.click == st.chamber {
		reply = fmt.Sprintf("(%d/6) 💥 Bang! You’re dead.", st.click)
		delete(c.roomStates, evt.RoomID)
	} else {
		reply = fmt.Sprintf("(%d/6) click... you survived.", st.click)
	}
	mention := fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, evt.Sender, evt.Sender)
	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          fmt.Sprintf("%s: %s", evt.Sender, reply),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("%s: %s", mention, reply),
	}
	cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content)
}
