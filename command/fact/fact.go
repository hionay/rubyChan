package fact

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hionay/rubyChan/core"
)

var _ core.Command = (*FactCmd)(nil)

type FactCmd struct{}

func (*FactCmd) Name() string      { return "fact" }
func (*FactCmd) Aliases() []string { return []string{} }
func (*FactCmd) Usage() string     { return "!fact - Get today's useless fact" }

func (*FactCmd) Run(ctx core.Context, args []string) (*core.Response, error) {
	fact, err := fetchFact()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fact: %w", err)
	}
	return &core.Response{Text: fact}, nil
}

func fetchFact() (string, error) {
	const url = "https://uselessfacts.jsph.pl/api/v2/facts/today?language=en"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch fact: %s", resp.Status)
	}

	var factResponse struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&factResponse); err != nil {
		return "", err
	}
	return factResponse.Text, nil
}
