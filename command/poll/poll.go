package poll

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type PollCmd struct{}

func (*PollCmd) Name() string      { return "poll" }
func (*PollCmd) Aliases() []string { return []string{} }
func (*PollCmd) Usage() string {
	return "!poll <question> | <option1> | <option2> [| …] — Create a poll"
}

func (c *PollCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	raw := strings.Join(args, " ")
	parts := strings.Split(raw, "|")
	if len(parts) < 3 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+c.Usage())
		return
	}

	question := strings.TrimSpace(parts[0])
	opts := parts[1:]
	for i := range opts {
		opts[i] = strings.TrimSpace(opts[i])
	}

	answers := make([]map[string]any, len(opts))
	for i, opt := range opts {
		uid := strconv.FormatInt(time.Now().UnixNano()+int64(i), 36)
		answers[i] = map[string]any{
			"id":                      uid,
			"org.matrix.msc1767.text": opt,
		}
	}

	var b strings.Builder
	b.WriteString(question)
	for i, opt := range opts {
		b.WriteString(fmt.Sprintf("\n%d. %s", i+1, opt))
	}
	fallback := b.String()

	content := map[string]any{
		"org.matrix.msc3381.poll.start": map[string]any{
			"question": map[string]any{
				"org.matrix.msc1767.text": question,
				"body":                    question,
				"msgtype":                 "m.text",
			},
			"kind":           "org.matrix.msc3381.poll.disclosed",
			"max_selections": 1,
			"answers":        answers,
		},
		"org.matrix.msc1767.text": fallback,
	}

	_, err := cli.SendMessageEvent(
		ctx,
		evt.RoomID,
		event.EventUnstablePollStart,
		content,
	)
	if err != nil {
		log.Printf("Poll send error: %v", err)
	}
}
