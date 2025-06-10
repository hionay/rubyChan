package main

import (
	"context"
	"slices"
	"strings"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"

	"github.com/hionay/rubyChan/command"
	"github.com/hionay/rubyChan/history"
)

const cmdPrefix = "!"

func parseMessage(cli *mautrix.Client, store *history.HistoryStore) func(context.Context, *event.Event) {
	st := time.Now()
	return func(ctx context.Context, evt *event.Event) {
		raw := strings.TrimSpace(evt.Content.AsMessage().Body)
		nick := evt.Sender.Localpart()
		store.Add(evt.RoomID, history.HistoryMessage{
			Sender:    nick,
			Body:      raw,
			Timestamp: evt.Timestamp,
		})

		// Ignore commands from the history
		ts := time.UnixMilli(evt.Timestamp)
		if ts.Before(st) || evt.Sender == cli.UserID {
			return
		}

		body, ok := strings.CutPrefix(raw, cmdPrefix)
		if !ok {
			return
		}
		fields := strings.Fields(body)
		if len(fields) == 0 {
			return
		}

		name, args := fields[0], fields[1:]
		for _, cmd := range command.Registry {
			if name == cmd.Name() || slices.Contains(cmd.Aliases(), name) {
				cmd.Execute(ctx, cli, evt, args)
				return
			}
		}
	}
}
