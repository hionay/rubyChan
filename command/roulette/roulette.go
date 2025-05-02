package roulette

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type RouletteCmd struct {
	mu      sync.Mutex
	rnd     *rand.Rand
	click   int
	chamber int
}

func (*RouletteCmd) Name() string      { return "roulette" }
func (*RouletteCmd) Aliases() []string { return []string{} }
func (*RouletteCmd) Usage() string     { return "!roulette - Play Russian Roulette" }

func (rc *RouletteCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, _ []string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.rnd == nil {
		rc.rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
		rc.click = 0
		rc.chamber = rc.rnd.Intn(6) + 1
	}
	rc.click++

	var reply string
	if rc.chamber == rc.click {
		reply = fmt.Sprintf(`(%d/6) 💥 Bang! You’re dead.`, rc.click)
		rc.rnd = nil
	} else {
		reply = fmt.Sprintf("(%d/6) click... you survived.", rc.click)
	}
	mention := fmt.Sprintf(
		`<a href="https://matrix.to/#/%s">%s</a>`, evt.Sender, evt.Sender,
	)
	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          fmt.Sprintf("%s: %s", evt.Sender, reply),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("%s: %s", mention, reply),
	}
	cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content)
}
