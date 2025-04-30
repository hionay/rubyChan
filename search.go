package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type googleResponse struct {
	Items []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	} `json:"items"`
}

func searchGoogle(cfg *Config, query string) (title, link string, err error) {
	if cfg.GoogleAPIKey == "" || cfg.GoogleCX == "" {
		return "", "", fmt.Errorf("Google API key or CX not set")
	}

	q := strings.ReplaceAll(query, " ", "+")
	url := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?q=%s&key=%s&cx=%s&num=1",
		q, cfg.GoogleAPIKey, cfg.GoogleCX,
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
