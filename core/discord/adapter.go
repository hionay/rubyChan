// discord/adapter.go

package discord

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/hionay/rubyChan/core"
)

type DiscordBot struct {
	Session  *discordgo.Session
	Commands []core.Command
}

func NewDiscordBot(sess *discordgo.Session, cmds []core.Command) *DiscordBot {
	bot := &DiscordBot{Session: sess, Commands: cmds}
	sess.AddHandler(bot.onMessage)
	return bot
}

func (b *DiscordBot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}
	raw := strings.TrimSpace(m.Content)
	if !strings.HasPrefix(raw, "!") {
		return
	}
	parts := strings.Fields(raw[1:])
	if len(parts) == 0 {
		return
	}
	name, args := parts[0], parts[1:]

	c := core.Context{
		UserID:    m.Author.ID,
		ChannelID: m.ChannelID,
		Now:       time.Now(),
		MentionFn: func(userID string) string {
			return fmt.Sprintf("<@%s>", userID)
		},
	}

	for _, cmd := range b.Commands {
		if name == cmd.Name() || slices.Contains(cmd.Aliases(), name) {
			resp, err := cmd.Run(c, args)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			} else {
				s.ChannelMessageSend(m.ChannelID, resp.Text)
			}
			break
		}
	}
}
