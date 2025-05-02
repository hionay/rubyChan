package calc

import (
	"context"
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type CalcCmd struct{}

func (*CalcCmd) Name() string      { return "calc" }
func (*CalcCmd) Aliases() []string { return []string{} }
func (*CalcCmd) Usage() string     { return "!calc <expr> - Evaluate a math expression" }

func (c *CalcCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) < 1 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+c.Usage())
		return
	}
	expr := strings.Join(args, " ")
	e, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		cli.SendText(ctx, evt.RoomID, "Invalid expression")
		return
	}
	res, err := e.Evaluate(nil)
	if err != nil {
		cli.SendText(ctx, evt.RoomID, fmt.Sprintf("Error: %v", err))
		return
	}
	cli.SendText(ctx, evt.RoomID, fmt.Sprintf("%v", res))
}
