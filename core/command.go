package core

import (
	"time"
)

type Context struct {
	UserID    string
	ChannelID string
	Now       time.Time
	MentionFn func(userID string) string
}

func (c *Context) Mention(userID string) string {
	if c.MentionFn != nil {
		return c.MentionFn(userID)
	}
	return userID
}

type Response struct {
	Text          string
	HTML          string
	EventTypeName string
	Content       any
}

type Command interface {
	Name() string
	Aliases() []string
	Usage() string
	Run(ctx Context, args []string) (*Response, error)
}
