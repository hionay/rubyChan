package repo

import (
	"context"
	"log"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type RepoCmd struct{}

func (*RepoCmd) Name() string      { return "repo" }
func (*RepoCmd) Aliases() []string { return []string{} }
func (*RepoCmd) Usage() string     { return "!repo - Display Github Repo for the codebase" }

func (c *RepoCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, _ []string) {

	_, err := cli.SendText(ctx, evt.RoomID, "https://github.com/hionay/rubyChan fork me daddy")

	if err != nil {
		log.Println("SendText error ", err)
	}
}
