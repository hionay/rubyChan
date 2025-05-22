package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hionay/rubyChan/core"
)

type HistoryMessage struct {
	Sender string
	Body   string
}

type HistoryFetcher interface {
	GetLast(channelID string, n int) []HistoryMessage
}

type QuoteCmd struct {
	Fetcher HistoryFetcher
	APIURL  string // "https://quotes.halil.io"
}

func (*QuoteCmd) Name() string      { return "quote" }
func (*QuoteCmd) Aliases() []string { return []string{"q"} }
func (*QuoteCmd) Usage() string {
	return "quote <n> [comment] — Quote the last n messages with optional comment"
}

// Run executes the quote command.
func (c *QuoteCmd) Run(ctx core.Context, args []string) (*core.Response, error) {
	if len(args) < 1 {
		return &core.Response{Text: "Usage: " + c.Usage()}, nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return nil, fmt.Errorf("invalid number of messages to quote: %s", args[0])
	}
	comment := ""
	if len(args) > 1 {
		comment = strings.Join(args[1:], " ")
	}

	history := c.Fetcher.GetLast(ctx.ChannelID, n+1)
	if len(history) <= 1 {
		return nil, fmt.Errorf("not enough messages to quote")
	}
	history = history[:len(history)-1]
	if n > len(history) {
		n = len(history)
	}
	slice := history[len(history)-n:]

	lines := make([]string, len(slice))
	for i, msg := range slice {
		body := msg.Body
		parts := strings.Fields(body)
		for j, p := range parts {
			if strings.HasPrefix(p, "@") {
				parts[j] = p[1:]
				if j == 0 {
					parts[j] += ":"
				}
			}
		}
		lines[i] = fmt.Sprintf("<%s> %s", msg.Sender, strings.Join(parts, " "))
	}
	quoteText := strings.Join(lines, "\n")

	fullLink, err := postQuote(c.APIURL, quoteText, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to post quote: %w", err)
	}

	userMention := ctx.Mention(ctx.UserID)
	text := fmt.Sprintf("%s: Quoted %d messages: %s", userMention, n, fullLink)
	html := fmt.Sprintf("%s: <a href=\"%s\">%s</a>", userMention, fullLink, fullLink)
	return &core.Response{Text: text, HTML: html}, nil
}

func postQuote(apiURL, quoteText, comment string) (string, error) {
	resp, err := http.PostForm(apiURL+"/add", url.Values{
		"quote":   {quoteText},
		"comment": {comment},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	html := string(body)
	marker := `class="text-[#b4a6c6] text-sm hover:underline"`
	idx := strings.Index(html, marker)
	if idx < 0 {
		return "", fmt.Errorf("quote link not found in response")
	}
	hrefIdx := strings.LastIndex(html[:idx], `<a href="`)
	if hrefIdx < 0 {
		return "", fmt.Errorf("quote link not found in response")
	}
	start := hrefIdx + len(`<a href="`)
	end := strings.Index(html[start:], `"`)
	if end < 0 {
		return "", fmt.Errorf("quote link not found in response")
	}
	path := html[start : start+end]
	return apiURL + path, nil
}
