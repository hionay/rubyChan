package gif

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type GifCmd struct {
	APIKey string
}

func (*GifCmd) Name() string      { return "gif" }
func (*GifCmd) Aliases() []string { return nil }
func (*GifCmd) Usage() string     { return "!gif <search terms> — Fetch a GIF from Tenor" }

func (c *GifCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) < 1 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+c.Usage())
		return
	}
	if c.APIKey == "" {
		cli.SendText(ctx, evt.RoomID, "TENOR_API_KEY not configured")
		return
	}

	query := strings.Join(args, " ")
	gifURL, err := fetchGif(c.APIKey, query)
	if err != nil {
		cli.SendText(ctx, evt.RoomID, fmt.Sprintf("Error fetching GIF: %v", err))
		return
	}
	if gifURL == "" {
		cli.SendText(ctx, evt.RoomID, "No GIFs found.")
		return
	}

	mention := fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, evt.Sender, evt.Sender)
	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          fmt.Sprintf("%s: %s", evt.Sender, gifURL),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("%s: <a href=%q>%s</a>", mention, gifURL, gifURL),
	}
	cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content)
}

func fetchGif(apiKey, query string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://tenor.googleapis.com/v2/search?q=%s&key=%s&limit=1",
		url.QueryEscape(query),
		apiKey,
	)
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Tenor API returned status %d", resp.StatusCode)
	}

	var tr struct {
		Results []struct {
			MediaFormats struct {
				GIF struct {
					URL string `json:"url"`
				} `json:"gif"`
			} `json:"media_formats"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	if len(tr.Results) == 0 {
		return "", nil
	}
	return tr.Results[0].MediaFormats.GIF.URL, nil
}
