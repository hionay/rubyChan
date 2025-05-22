package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hionay/rubyChan/core"
)

var _ core.Command = (*GifCmd)(nil)

type GifCmd struct {
	APIKey string
}

func (*GifCmd) Name() string      { return "gif" }
func (*GifCmd) Aliases() []string { return nil }
func (*GifCmd) Usage() string     { return "gif <search terms> — Fetch a GIF from Tenor" }

func (c *GifCmd) Run(ctx core.Context, args []string) (*core.Response, error) {
	if len(args) < 1 {
		return &core.Response{Text: "Usage: " + c.Usage()}, nil
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("API key not set")
	}

	query := strings.Join(args, " ")
	gifURL, err := fetchGif(c.APIKey, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GIF: %w", err)
	}
	if gifURL == "" {
		return nil, fmt.Errorf("no GIF found for query: %s", query)
	}

	userMention := ctx.Mention(ctx.UserID)
	text := fmt.Sprintf("%s: %s", userMention, gifURL)
	html := fmt.Sprintf("%s: <a href=\"%s\">%s</a>", userMention, gifURL, gifURL)

	return &core.Response{Text: text, HTML: html}, nil
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

	var result struct {
		Results []struct {
			MediaFormats struct {
				GIF struct {
					URL string `json:"url"`
				} `json:"gif"`
			} `json:"media_formats"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Results) == 0 {
		return "", nil
	}
	return result.Results[0].MediaFormats.GIF.URL, nil
}
