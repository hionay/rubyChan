package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

type Reminder struct {
	ID      int64
	Timer   *time.Timer
	RoomID  id.RoomID
	Sender  id.UserID
	Message string
	Due     time.Time
}

var (
	reminders   = make(map[int64]*Reminder)
	remindersMu sync.Mutex
	nextRemID   int64
)

func handleRemindIn(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID, args []string) {
	if len(args) < 3 || args[0] != "in" {
		cli.SendText(ctx, roomID, "Usage: !remindme in <duration> <message>")
		return
	}
	d, err := time.ParseDuration(args[1])
	if err != nil {
		cli.SendText(ctx, roomID, "Invalid duration (e.g. 15m, 1h30m)")
		return
	}
	msg := strings.Join(args[2:], " ")
	id := atomic.AddInt64(&nextRemID, 1)
	due := time.Now().Add(d)

	timer := time.AfterFunc(d, func() {
		m := fmt.Sprintf("⏰ Reminder #%d: %s", id, msg)
		messageWithMention(ctx, cli, roomID, sender, m)
		remindersMu.Lock()
		delete(reminders, id)
		remindersMu.Unlock()
	})

	rem := &Reminder{ID: id, Timer: timer, RoomID: roomID, Sender: sender, Message: msg, Due: due}
	remindersMu.Lock()
	reminders[id] = rem
	remindersMu.Unlock()

	cli.SendText(ctx, roomID, fmt.Sprintf("Reminder #%d set for %s", id, due.Format(time.RFC1123)))
}

func handleRemindList(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID) {
	remindersMu.Lock()
	defer remindersMu.Unlock()

	var lines []string
	for _, r := range reminders {
		if r.RoomID == roomID && r.Sender == sender {
			lines = append(lines,
				fmt.Sprintf("#%d at %s: %s",
					r.ID,
					r.Due.Format("15:04:05 Jan 02"),
					r.Message,
				),
			)
		}
	}
	if len(lines) == 0 {
		cli.SendText(ctx, roomID, "You have no pending reminders.")
	} else {
		cli.SendText(ctx, roomID, "Your reminders:\n"+strings.Join(lines, "\n"))
	}
}

func handleRemindCancel(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID, idArg string) {
	id, err := strconv.ParseInt(idArg, 10, 64)
	if err != nil {
		cli.SendText(ctx, roomID, "Invalid reminder ID")
		return
	}

	remindersMu.Lock()
	rem, ok := reminders[id]
	if ok && rem.RoomID == roomID && rem.Sender == sender {
		rem.Timer.Stop()
		delete(reminders, id)
	}
	remindersMu.Unlock()

	if ok {
		cli.SendText(ctx, roomID, fmt.Sprintf("Canceled reminder #%d", id))
	} else {
		cli.SendText(ctx, roomID, fmt.Sprintf("No reminder #%d found", id))
	}
}
