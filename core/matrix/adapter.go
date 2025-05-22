package adapter

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/hionay/rubyChan/core"
)

type MatrixAdapter struct {
	Client   *mautrix.Client
	Commands []core.Command
	Prefix   string
}

func NewMatrixAdapter(cli *mautrix.Client, cmds []core.Command, prefix string) *MatrixAdapter {
	return &MatrixAdapter{Client: cli, Commands: cmds, Prefix: prefix}
}

func (m *MatrixAdapter) Start(ctx context.Context) error {
	syncer := m.Client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, m.onMessage)
	return m.Client.SyncWithContext(ctx)
}

var (
	nextRemID int64 = 1
	remMu     sync.Mutex
	reminders = make(map[int64]*scheduledReminder)
)

type scheduledReminder struct {
	id      int64
	room    id.RoomID
	user    string
	message string
	timer   *time.Timer
}

func (m *MatrixAdapter) onMessage(ctx context.Context, evt *event.Event) {
	if evt.Content.AsMessage().MsgType != event.MsgText {
		return
	}
	raw := strings.TrimSpace(evt.Content.AsMessage().Body)
	if !strings.HasPrefix(raw, m.Prefix) {
		return
	}
	parts := strings.Fields(raw[len(m.Prefix):])
	if len(parts) == 0 {
		return
	}
	name, args := parts[0], parts[1:]

	coreCtx := core.Context{
		UserID:    evt.Sender.String(),
		ChannelID: evt.RoomID.String(),
		Now:       time.Now(),
		MentionFn: func(userID string) string {
			return fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, userID, userID)
		},
	}

	for _, cmd := range m.Commands {
		if name == cmd.Name() || slices.Contains(cmd.Aliases(), name) {
			resp, err := cmd.Run(coreCtx, args)
			if err != nil {
				m.Client.SendText(ctx, evt.RoomID, "❗ "+err.Error())
				return
			}
			if m.handleReminder(ctx, evt, resp) {
				return
			}
			if resp.EventTypeName != "" && resp.Content != nil {
				evType := event.NewEventType(resp.EventTypeName)
				m.Client.SendMessageEvent(ctx, evt.RoomID, evType, resp.Content)
				return
			}
			if resp.HTML != "" {
				content := event.MessageEventContent{
					MsgType:       event.MsgText,
					Body:          resp.Text,
					Format:        event.FormatHTML,
					FormattedBody: resp.HTML,
				}
				m.Client.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content)
				return
			}
			m.Client.SendText(ctx, evt.RoomID, resp.Text)
			return
		}
	}
}

func (m *MatrixAdapter) handleReminder(ctx context.Context, evt *event.Event, resp core.Response) bool {
	room := evt.RoomID
	user := evt.Sender.String()
	if resp.ListReminders {
		remMu.Lock()
		defer remMu.Unlock()
		var lines []string
		for _, r := range reminders {
			if r.room == room && r.user == user {
				lines = append(lines, fmt.Sprintf("#%d: %s", r.id, r.message))
			}
		}
		if len(lines) == 0 {
			m.Client.SendText(ctx, room, "You have no pending reminders.")
		} else {
			m.Client.SendText(ctx, room, "Your reminders:\n"+strings.Join(lines, "\n"))
		}
		return true
	}
	if resp.CancelID != nil {
		rid := *resp.CancelID
		remMu.Lock()
		r, ok := reminders[rid]
		if ok && r.room == room && r.user == user {
			r.timer.Stop()
			delete(reminders, rid)
			m.Client.SendText(ctx, room, fmt.Sprintf("Canceled reminder #%d", rid))
		} else {
			m.Client.SendText(ctx, room, fmt.Sprintf("No reminder #%d found", rid))
		}
		remMu.Unlock()
		return true
	}
	if resp.Schedule != nil {
		req := resp.Schedule
		rid := atomic.AddInt64(&nextRemID, 1)
		msg := req.Message
		dur := req.Duration
		timer := time.AfterFunc(dur, func() {
			mention := core.Context{MentionFn: m.buildMentionFn()}.Mention(user)
			m.Client.SendText(context.Background(), room,
				fmt.Sprintf("%s ⏰ Reminder #%d: %s", mention, rid, msg))
			remMu.Lock()
			delete(reminders, rid)
			remMu.Unlock()
		})
		remMu.Lock()
		reminders[rid] = &scheduledReminder{id: rid, room: room, user: user, message: msg, timer: timer}
		remMu.Unlock()
		m.Client.SendText(ctx, room, resp.Text)
		return true
	}
	return false
}

func (m *MatrixAdapter) buildMentionFn() func(string) string {
	return func(userID string) string {
		return fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, userID, userID)
	}
}
