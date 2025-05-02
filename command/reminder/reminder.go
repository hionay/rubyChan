package reminder

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type RemindMeCmd struct{}

func (*RemindMeCmd) Name() string      { return "remindme" }
func (*RemindMeCmd) Aliases() []string { return nil }
func (*RemindMeCmd) Usage() string {
	return `!remindme in <duration> <message> | !remindme list | !remindme cancel <id>`
}

func (rc *RemindMeCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) == 1 && args[0] == "list" {
		list(ctx, cli, evt.RoomID, evt.Sender)
	} else if len(args) == 2 && args[0] == "cancel" {
		cancel(ctx, cli, evt.RoomID, evt.Sender, args[1])
	} else {
		rc.schedule(ctx, cli, evt.RoomID, evt.Sender, args)
	}
}

type reminder struct {
	ID      int64
	Timer   *time.Timer
	RoomID  id.RoomID
	Sender  id.UserID
	Message string
	Due     time.Time
}

var (
	reminders   = make(map[int64]*reminder)
	remindersMu sync.Mutex
	nextRemID   int64
)

func (rc *RemindMeCmd) schedule(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID, args []string) {
	if len(args) < 3 || args[0] != "in" {
		cli.SendText(ctx, roomID, rc.Usage())
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
		mention := fmt.Sprintf(
			`<a href="https://matrix.to/#/%s">%s</a>`, sender, sender,
		)
		content := event.MessageEventContent{
			MsgType:       event.MsgText,
			Body:          fmt.Sprintf("%s: %s", sender, m),
			Format:        event.FormatHTML,
			FormattedBody: fmt.Sprintf("%s: %s", mention, m),
		}
		cli.SendMessageEvent(ctx, roomID, event.EventMessage, content)
		remindersMu.Lock()
		delete(reminders, id)
		remindersMu.Unlock()
	})

	rem := &reminder{ID: id, Timer: timer, RoomID: roomID, Sender: sender, Message: msg, Due: due}
	remindersMu.Lock()
	reminders[id] = rem
	remindersMu.Unlock()

	cli.SendText(ctx, roomID, fmt.Sprintf("Reminder #%d set for %s", id, due.Format(time.RFC1123)))
}

func list(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID) {
	remindersMu.Lock()
	defer remindersMu.Unlock()

	var lines []string
	for _, r := range reminders {
		if r.RoomID == roomID && r.Sender == sender {
			lines = append(lines,
				fmt.Sprintf("#%d at %s: %s", r.ID, r.Due.Format("15:04:05 Jan 02"), r.Message),
			)
		}
	}
	if len(lines) == 0 {
		cli.SendText(ctx, roomID, "You have no pending reminders.")
	} else {
		cli.SendText(ctx, roomID, "Your reminders:\n"+strings.Join(lines, "\n"))
	}
}

func cancel(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, sender id.UserID, idArg string) {
	rid, err := strconv.ParseInt(idArg, 10, 64)
	if err != nil {
		cli.SendText(ctx, roomID, "Invalid reminder ID")
		return
	}

	remindersMu.Lock()
	rem, ok := reminders[rid]
	if ok && rem.RoomID == roomID && rem.Sender == sender {
		rem.Timer.Stop()
		delete(reminders, rid)
	}
	remindersMu.Unlock()

	if ok {
		cli.SendText(ctx, roomID, fmt.Sprintf("Canceled reminder #%d", rid))
	} else {
		cli.SendText(ctx, roomID, fmt.Sprintf("No reminder #%d found", rid))
	}
}
