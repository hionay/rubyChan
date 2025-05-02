package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type SearchCmd struct {
	GoogleAPIKey string
	GoogleCX     string
}

func (*SearchCmd) Name() string      { return "google" }
func (*SearchCmd) Aliases() []string { return []string{"g"} }
func (*SearchCmd) Usage() string {
	return "!g <query> - Search Google for <query>"
}

func (sc *SearchCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) == 0 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+sc.Usage())
		return
	}
	query := strings.Join(args, " ")
	title, link, err := sc.searchGoogle(query)

	var reply string
	switch {
	case err != nil:
		reply = fmt.Sprintf("error: %v", err)
	case title == "":
		reply = "No results found."
	default:
		reply = fmt.Sprintf("%s\n\n%s", title, link)
	}

	if _, err := cli.SendText(ctx, evt.RoomID, reply); err != nil {
		log.Printf("SendText error (google): %v", err)
	}
}

type googleResponse struct {
	Items []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	} `json:"items"`
}

func (sc *SearchCmd) searchGoogle(query string) (title, link string, err error) {
	if sc.GoogleAPIKey == "" || sc.GoogleCX == "" {
		return "", "", fmt.Errorf("Google API key or CX not set")
	}

	q := strings.ReplaceAll(query, " ", "+")
	url := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?q=%s&key=%s&cx=%s&num=1",
		q, sc.GoogleAPIKey, sc.GoogleCX,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Google API returned status %d", resp.StatusCode)
	}

	var gr googleResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", "", err
	}
	if len(gr.Items) == 0 {
		return "", "", nil
	}
	return gr.Items[0].Title, gr.Items[0].Link, nil
}
