package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const cmdPrefix = "!"

const helpText = `Commands:
!g <query>					- Search Google for <query>
!weather <location>			- Show current weather for <location>
!joke						- Tell a random joke
!calc <expr>					- Evaluate a math expression (e.g. 2+3*4)
!roulette						- Russian roulette (1/6 chance of dying)
!remindme in <dur> <msg>		- Remind you after a duration, e.g. in 15m take a break
!remindme list				- List pending reminders
!remindme cancel <id>		- Cancel a reminder
!quote N <comment?>			- Quote the last N messages, optionally with a comment
!help						- Show this help message
`

type Message struct {
	Sender string
	Body   string
}

var msgHistory = make(map[id.RoomID][]Message)

func parseMessage(cli *mautrix.Client, cfg *Config, st time.Time) func(context.Context, *event.Event) {
	return func(ctx context.Context, evt *event.Event) {
		msg := evt.Content.AsMessage()
		raw := strings.TrimSpace(msg.Body)
		nick := parseNick(string(evt.Sender))
		hist := msgHistory[evt.RoomID]
		hist = append(hist, Message{Sender: nick, Body: raw})
		if len(hist) > 100 {
			hist = hist[len(hist)-100:]
		}
		msgHistory[evt.RoomID] = hist

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

		cmd, args := fields[0], fields[1:]
		switch cmd {
		case "g":
			handleGoogle(ctx, cli, cfg, evt.RoomID, args)

		case "weather":
			handleWeather(ctx, cli, cfg, evt.RoomID, args)

		case "joke":
			handleJoke(ctx, cli, evt.RoomID)

		case "calc":
			handleCalc(ctx, cli, evt.RoomID, args)

		case "roulette":
			handleRoulette(ctx, cli, evt.RoomID, evt.Sender)

		case "remindme":
			if len(args) == 1 && args[0] == "list" {
				handleRemindList(ctx, cli, evt.RoomID, evt.Sender)
			} else if len(args) == 2 && args[0] == "cancel" {
				handleRemindCancel(ctx, cli, evt.RoomID, evt.Sender, args[1])
			} else {
				handleRemindIn(ctx, cli, evt.RoomID, evt.Sender, args)
			}

		case "quote":
			handleQuote(ctx, cli, evt.RoomID, args)

		case "help":
			cli.SendText(ctx, evt.RoomID, helpText)
		}
	}
}

func messageWithMention(
	ctx context.Context,
	cli *mautrix.Client,
	roomID id.RoomID,
	mentioned id.UserID,
	msg string,
) {
	mention := fmt.Sprintf(
		`<a href="https://matrix.to/#/%s">%s</a>`, mentioned, mentioned,
	)
	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          fmt.Sprintf("%s: %s", mentioned, msg),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("%s: %s", mention, msg),
	}
	if _, err := cli.SendMessageEvent(ctx, roomID, event.EventMessage, content); err != nil {
		log.Printf("SendMessageEvent error: %v", err)
	}
}

func parseNick(name string) string {
	if i := strings.Index(name, ":"); i > 0 {
		nick := strings.Clone(name)
		nick = nick[1:i]
		return nick
	}
	return name
}

func handleCalc(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, args []string) {
	if len(args) < 1 {
		cli.SendText(ctx, roomID, "Usage: !calc <expression>")
		return
	}
	expr := strings.Join(args, " ")
	e, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		cli.SendText(ctx, roomID, "Invalid expression")
		return
	}
	res, err := e.Evaluate(nil)
	if err != nil {
		cli.SendText(ctx, roomID, fmt.Sprintf("Error: %v", err))
		return
	}
	cli.SendText(ctx, roomID, fmt.Sprintf("%v", res))
}

func handleGoogle(ctx context.Context, cli *mautrix.Client, cfg *Config, roomID id.RoomID, args []string) {
	if len(args) == 0 {
		cli.SendText(ctx, roomID, "Usage: !g <query>")
		return
	}
	query := strings.Join(args, " ")
	log.Printf("Searching Google for: %s", query)
	title, link, err := searchGoogle(cfg, query)

	var reply string
	switch {
	case err != nil:
		reply = fmt.Sprintf("error: %v", err)
	case title == "":
		reply = "No results found."
	default:
		reply = fmt.Sprintf("%s\n\n%s", title, link)
	}

	if _, err := cli.SendText(ctx, roomID, reply); err != nil {
		log.Printf("SendText error (google): %v", err)
	}
}

func handleWeather(ctx context.Context, cli *mautrix.Client, cfg *Config, roomID id.RoomID, args []string) {
	if len(args) == 0 {
		cli.SendText(ctx, roomID, "Usage: !weather <location>")
		return
	}
	loc := strings.Join(args, " ")
	reply, err := searchWeather(cfg, loc)
	if err != nil {
		cli.SendText(ctx, roomID, fmt.Sprintf("️error: %v", err))
		return
	}
	cli.SendText(ctx, roomID, reply)
}

func handleJoke(ctx context.Context, cli *mautrix.Client, roomID id.RoomID) {
	joke, err := fetchJoke()
	if err != nil {
		cli.SendText(ctx, roomID, fmt.Sprintf("error: %v", err))
		return
	}
	cli.SendText(ctx, roomID, joke)
}
