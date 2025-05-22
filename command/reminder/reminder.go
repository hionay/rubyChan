package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hionay/rubyChan/core"
)

type ScheduleRequest struct {
	Duration time.Duration // how long to wait
	Message  string        // message to send after duration
	ID       int64         // unique reminder ID (0 for new reminders)
}

type Response struct {
	Text          string           // plain-text reply
	HTML          string           // optional rich reply
	Schedule      *ScheduleRequest // non-nil to schedule a new reminder
	CancelID      *int64           // non-nil to cancel existing reminder
	ListReminders bool             // true if adapter should list
}

type RemindMeCmd struct{}

func (*RemindMeCmd) Name() string      { return "remindme" }
func (*RemindMeCmd) Aliases() []string { return nil }
func (*RemindMeCmd) Usage() string {
	return "remindme in <duration> <message> | remindme list | remindme cancel <id>"
}

func (c *RemindMeCmd) Run(ctx core.Context, args []string) (*core.Response, error) {
	if len(args) == 1 && args[0] == "list" {
		return &core.Response{ListReminders: true}, nil
	}
	if len(args) == 2 && args[0] == "cancel" {
		id, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid reminder ID: %s", args[1])
		}
		return &core.Response{CancelID: &id}, nil
	}
	// schedule
	if len(args) < 3 || args[0] != "in" {
		return &core.Response{Text: "Usage: " + c.Usage()}, nil
	}
	d, err := time.ParseDuration(args[1])
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %s", args[1])
	}
	msg := strings.Join(args[2:], " ")
	req := &ScheduleRequest{Duration: d, Message: msg, ID: 0}
	text := fmt.Sprintf("Okay! I will remind you in %s: %s", args[1], msg)
	return &core.Response{Text: text, Schedule: req}, nil
}
