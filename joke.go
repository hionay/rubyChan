package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func fetchJoke() (string, error) {
	const url = "https://v2.jokeapi.dev/joke/Programming"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("joke API returned status %d", resp.StatusCode)
	}

	var jr struct {
		Error    bool   `json:"error"`
		Type     string `json:"type"`
		Joke     string `json:"joke"`
		Setup    string `json:"setup,omitempty"`
		Delivery string `json:"delivery"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jr); err != nil {
		return "", err
	}
	if jr.Error {
		return "", fmt.Errorf("joke API error")
	}

	if jr.Type == "single" {
		return jr.Joke, nil
	}
	return fmt.Sprintf("%s\n\n%s", jr.Setup, jr.Delivery), nil
}
