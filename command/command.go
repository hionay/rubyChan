package command

import (
	"context"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type Command interface {
	Name() string
	Aliases() []string
	Usage() string
	Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string)
}

var Registry []Command

func Register(cmd ...Command) {
	if len(cmd) == 0 {
		return
	}
	Registry = append(Registry, cmd...)
}

type HelpCmd struct{}

func (h *HelpCmd) Name() string      { return "help" }
func (h *HelpCmd) Aliases() []string { return []string{} }
func (h *HelpCmd) Usage() string     { return "!help - Show this help message" }

func (h *HelpCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) > 0 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+h.Usage())
		return
	}

	var helpMsg string
	for _, cmd := range Registry {
		helpMsg += cmd.Usage() + "\n"
	}
	if _, err := cli.SendText(ctx, evt.RoomID, helpMsg); err != nil {
		cli.SendText(ctx, evt.RoomID, "Error sending help message")
	}
}
