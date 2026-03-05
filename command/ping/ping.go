package ping

import (
	"context"
	"fmt"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type PingCmd struct{}

func (*PingCmd) Name() string      { return "ping" }
func (*PingCmd) Aliases() []string { return []string{} }
func (*PingCmd) Usage() string     { return "!ping - Check bot latency" }

func (c *PingCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	latency := time.Since(time.UnixMilli(evt.Timestamp))
	_, _ = cli.SendText(ctx, evt.RoomID, fmt.Sprintf("pong! latency: %s", formatLatency(latency)))
}

func formatLatency(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
