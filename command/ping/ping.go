package ping

import (
	"context"
	"fmt"
	"log"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type PingCmd struct{}

func (*PingCmd) Name() string      { return "ping" }
func (*PingCmd) Aliases() []string { return []string{} }
func (*PingCmd) Usage() string     { return "!ping - Check bot latency" }

func (c *PingCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	eventAge := time.Since(time.UnixMilli(evt.Timestamp))

	sendStart := time.Now()
	resp, err := cli.SendText(ctx, evt.RoomID, fmt.Sprintf(
		"pong! event age: %s | send: measuring...",
		formatLatency(eventAge),
	))
	sendLatency := time.Since(sendStart)

	if err != nil {
		log.Printf("cli.SendText error: %v", err)
		return
	}

	finalText := fmt.Sprintf("pong! event age: %s | send: %s",
		formatLatency(eventAge),
		formatLatency(sendLatency),
	)

	_, err = cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, &event.MessageEventContent{
		MsgType: event.MsgText,
		Body:    "* " + finalText,
		NewContent: &event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    finalText,
		},
		RelatesTo: &event.RelatesTo{
			Type:    event.RelReplace,
			EventID: resp.EventID,
		},
	})
	if err != nil {
		log.Printf("cli.SendMessageEvent error: %v", err)
	}
}

func formatLatency(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
