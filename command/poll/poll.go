package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hionay/rubyChan/core"
)

var _ core.Command = (*PollCmd)(nil)

// PollCmd creates a native MSC3381 poll in Matrix (with fallback text).
// Note: Requires your Matrix adapter to honor EventType and Content fields.
type PollCmd struct{}

func (*PollCmd) Name() string      { return "poll" }
func (*PollCmd) Aliases() []string { return nil }
func (*PollCmd) Usage() string {
	return "poll <question> | <option1> | <option2> [| …] — Create a poll"
}

func (c *PollCmd) Run(ctx core.Context, args []string) (*core.Response, error) {
	raw := strings.Join(args, " ")
	parts := strings.Split(raw, "|")
	if len(parts) < 3 {
		return &core.Response{Text: "Usage: " + c.Usage()}, nil
	}

	question := strings.TrimSpace(parts[0])
	opts := parts[1:]
	for i := range opts {
		opts[i] = strings.TrimSpace(opts[i])
	}

	answers := make([]map[string]interface{}, len(opts))
	for i, opt := range opts {
		uid := strconv.FormatInt(time.Now().UnixNano()+int64(i), 36)
		answers[i] = map[string]interface{}{
			"id":                      uid,
			"org.matrix.msc1767.text": opt,
		}
	}

	var sb strings.Builder
	sb.WriteString(question)
	for i, opt := range opts {
		sb.WriteString(fmt.Sprintf("\n%d. %s", i+1, opt))
	}
	fallback := sb.String()

	content := map[string]interface{}{
		"org.matrix.msc3381.poll.start": map[string]interface{}{
			"question": map[string]interface{}{
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

	return &core.Response{
		Text:          fallback,
		EventTypeName: "org.matrix.msc3381.poll.start",
		Content:       content,
	}, nil
}
