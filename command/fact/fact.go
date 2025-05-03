package fact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type FactCmd struct{}

func (*FactCmd) Name() string      { return "fact" }
func (*FactCmd) Aliases() []string { return []string{} }
func (*FactCmd) Usage() string     { return "!fact - Get today's useless fact" }

func (*FactCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, _ []string) {
	fact, err := fetchFact()
	if err != nil {
		cli.SendText(ctx, evt.RoomID, "Error fetching fact: "+err.Error())
		return
	}
	cli.SendText(ctx, evt.RoomID, fact)
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
