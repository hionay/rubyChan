// core/calc.go

package core

import (
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"

	"github.com/hionay/rubyChan/core"
)

var _ core.Command = (*CalcCmd)(nil)

type CalcCmd struct{}

func (*CalcCmd) Name() string      { return "calc" }
func (*CalcCmd) Aliases() []string { return nil }
func (*CalcCmd) Usage() string     { return "calc <expr> — Evaluate a math expression" }

func (c *CalcCmd) Run(ctx core.Context, args []string) (*core.Response, error) {
	if len(args) < 1 {
		return &core.Response{Text: "Usage: " + c.Usage()}, nil
	}
	expr := strings.Join(args, " ")
	e, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid expression: %w", err)
	}
	res, err := e.Evaluate(nil)
	if err != nil {
		return nil, err
	}
	return &core.Response{Text: fmt.Sprintf("%v", res)}, nil
}
