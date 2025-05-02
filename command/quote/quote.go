package quote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/hionay/rubyChan/history"
)

const quoteWebsite = "https://quotes.halil.io"

type HistoryFetcher interface {
	GetLast(roomID id.RoomID, n int) []history.HistoryMessage
}

type QuoteCmd struct {
	History HistoryFetcher
}

func (*QuoteCmd) Name() string      { return "quote" }
func (*QuoteCmd) Aliases() []string { return []string{"q"} }
func (*QuoteCmd) Usage() string {
	return "!quote <n> [comment] - Quote the last n messages with optional comment"
}

func (q *QuoteCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) < 1 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+q.Usage())
		return
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		cli.SendText(ctx, evt.RoomID, "Invalid number of lines")
		return
	}
	comment := ""
	if len(args) > 1 {
		comment = strings.Join(args[1:], " ")
	}

	hist := q.History.GetLast(evt.RoomID, n+1)
	if len(hist) <= 1 {
		cli.SendText(ctx, evt.RoomID, "No messages to quote.")
		return
	}
	hist = hist[:len(hist)-1]

	if n > len(hist) {
		n = len(hist)
	}
	slice := hist[len(hist)-n:]

	lines := make([]string, len(slice))
	for i, m := range slice {
		body := m.Body
		parts := strings.Fields(body)
		for j, p := range parts {
			if strings.HasPrefix(p, "@") {
				parts[j] = parseNick(p)
				if j == 0 {
					parts[j] += ":"
				}
			}
		}
		lines[i] = fmt.Sprintf("<%s> %s", m.Sender, strings.Join(parts, " "))
	}
	quoteText := strings.Join(lines, "\n")
	fullLink, err := postQuote(quoteText, comment)
	if err != nil {
		cli.SendText(ctx, evt.RoomID, "Failed to post quote: "+err.Error())
		return
	}
	reply := fmt.Sprintf("Quoted %d messages: %s", n, fullLink)
	cli.SendText(ctx, evt.RoomID, reply)
}

func postQuote(quoteText, comment string) (string, error) {
	resp, err := http.PostForm(quoteWebsite+"/add", url.Values{
		"quote":   {quoteText},
		"comment": {comment},
	})
	if err != nil {
		return "", fmt.Errorf("failed to post quote: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	html := string(b)
	marker := `class="text-[#b4a6c6] text-sm hover:underline"`
	idx := strings.Index(html, marker)
	if idx < 0 {
		return "", fmt.Errorf("failed to find quote link in response")
	}
	hrefIdx := strings.LastIndex(html[:idx], `<a href="`)
	if hrefIdx < 0 {
		return "", fmt.Errorf("failed to find quote link in response")
	}
	start := hrefIdx + len(`<a href="`)
	end := strings.Index(html[start:], `"`)
	if end < 0 {
		return "", fmt.Errorf("failed to find quote link in response")
	}
	linkPath := html[start : start+end]
	fullLink := quoteWebsite + linkPath
	return fullLink, nil
}

func parseNick(name string) string {
	if i := strings.Index(name, ":"); i > 0 {
		nick := strings.Clone(name)
		nick = nick[1:i]
		return nick
	}
	return name
}
