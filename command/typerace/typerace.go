package typerace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/hionay/rubyChan/internal/matrixutil"
	"github.com/hionay/rubyChan/state"
)

const quotesURL = "https://dummyjson.com/quotes/random"

type TypeRaceCmd struct {
	store  *state.Namespace
	mu     sync.Mutex
	active map[id.RoomID]*race
	client *http.Client
}

type race struct {
	prompt    string
	startedAt time.Time
	startedBy id.UserID
}

type statsState struct {
	TotalRaces    int            `json:"total_races"`
	BestWPMByUser map[string]int `json:"best_wpm_by_user"`
	WinsByUser    map[string]int `json:"wins_by_user"`
}

func NewTypeRaceCmd(store *state.Namespace) *TypeRaceCmd {
	return &TypeRaceCmd{
		store:  store,
		active: make(map[id.RoomID]*race),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (*TypeRaceCmd) Name() string      { return "typerace" }
func (*TypeRaceCmd) Aliases() []string { return []string{"t"} }
func (*TypeRaceCmd) Usage() string {
	return "!typerace - First to type the prompt wins | !typerace stats"
}

func (c *TypeRaceCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) > 0 && strings.ToLower(args[0]) == "stats" {
		c.sendStats(ctx, cli, evt)
		return
	}

	c.mu.Lock()
	if _, ongoing := c.active[evt.RoomID]; ongoing {
		c.mu.Unlock()
		_, _ = cli.SendText(ctx, evt.RoomID, "a race is already in progress!")
		return
	}
	c.mu.Unlock()

	prompt, err := c.fetchPrompt(ctx)
	if err != nil {
		_, _ = cli.SendText(ctx, evt.RoomID, fmt.Sprintf("fetch error: %v", err))
		return
	}

	startedAt := time.Now()

	c.mu.Lock()
	if _, ongoing := c.active[evt.RoomID]; ongoing {
		c.mu.Unlock()
		_, _ = cli.SendText(ctx, evt.RoomID, "a race is already in progress!")
		return
	}
	c.active[evt.RoomID] = &race{
		prompt:    prompt,
		startedAt: startedAt,
		startedBy: evt.Sender,
	}
	c.mu.Unlock()

	_, _ = cli.SendText(ctx, evt.RoomID, fmt.Sprintf("type this:\n\n%s", prompt))

	_ = time.AfterFunc(60*time.Second, func() {
		c.mu.Lock()
		r, still := c.active[evt.RoomID]
		if !still || r.startedAt != startedAt {
			c.mu.Unlock()
			return
		}
		delete(c.active, evt.RoomID)
		c.mu.Unlock()
		_, _ = cli.SendText(ctx, evt.RoomID, "time's up! no one finished the race.")
	})
}

func (c *TypeRaceCmd) HandleMessage(ctx context.Context, cli *mautrix.Client, evt *event.Event) {
	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	if !ok || content.MsgType != event.MsgText {
		return
	}
	if evt.Sender == cli.UserID {
		return
	}

	c.mu.Lock()
	r, ongoing := c.active[evt.RoomID]
	if !ongoing {
		c.mu.Unlock()
		return
	}
	if strings.TrimSpace(content.Body) != r.prompt {
		c.mu.Unlock()
		return
	}

	elapsed := time.Since(r.startedAt)
	delete(c.active, evt.RoomID)
	c.mu.Unlock()

	wpm := calculateWPM(r.prompt, elapsed)
	c.recordWin(evt.RoomID, evt.Sender.String(), wpm)

	_, _ = cli.SendText(ctx, evt.RoomID, fmt.Sprintf(
		"%s wins! finished in %s (%d wpm)",
		evt.Sender,
		formatDuration(elapsed),
		wpm,
	))
}

func (c *TypeRaceCmd) recordWin(roomID id.RoomID, sender string, wpm int) {
	key := "stats:" + roomID.String()
	ss := c.loadStats(key)

	ss.TotalRaces++
	ss.WinsByUser[sender]++
	if wpm > ss.BestWPMByUser[sender] {
		ss.BestWPMByUser[sender] = wpm
	}

	if err := c.store.PutJSON(key, ss); err != nil {
		log.Printf("typerace: error saving stats: %v", err)
	}
}

func (c *TypeRaceCmd) loadStats(key string) *statsState {
	ss := &statsState{}
	if err := c.store.GetJSON(key, ss); err != nil {
		log.Printf("typerace: error loading stats: %v", err)
	}
	if ss.BestWPMByUser == nil {
		ss.BestWPMByUser = make(map[string]int)
	}
	if ss.WinsByUser == nil {
		ss.WinsByUser = make(map[string]int)
	}
	return ss
}

func (c *TypeRaceCmd) sendStats(ctx context.Context, cli *mautrix.Client, evt *event.Event) {
	key := "stats:" + evt.RoomID.String()
	ss := c.loadStats(key)

	if ss.TotalRaces == 0 {
		_, _ = cli.SendText(ctx, evt.RoomID, "no races have been completed in this room yet.")
		return
	}

	top := matrixutil.TopN(ss.BestWPMByUser, 5)

	plainMsg := fmt.Sprintf(
		"TypeRace stats (this room)\nTotal races: %d\nTop 5 by best WPM:\n%s",
		ss.TotalRaces,
		formatTopPlain(ctx, cli, evt.RoomID, top, ss.WinsByUser),
	)
	htmlMsg := fmt.Sprintf(
		"<b>TypeRace stats</b> (this room)<br>Total races: %d<br><b>Top 5 by best WPM:</b><br>%s",
		ss.TotalRaces,
		formatTopHTML(ctx, cli, evt.RoomID, top, ss.WinsByUser),
	)

	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          plainMsg,
		Format:        event.FormatHTML,
		FormattedBody: htmlMsg,
	}
	if _, err := cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content); err != nil {
		log.Printf("typerace: failed to send stats: %v", err)
	}
}

func (c *TypeRaceCmd) fetchPrompt(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, quotesURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("quotes api returned %d", resp.StatusCode)
	}
	var result struct {
		Quote string `json:"quote"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Quote == "" {
		return "", errors.New("empty quote in response")
	}
	return strings.ToLower(result.Quote), nil
}

func formatTopHTML(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, items []matrixutil.KV, wins map[string]int) string {
	if len(items) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(items))
	for i, it := range items {
		parts = append(parts, fmt.Sprintf("%d. %s — %d wpm (%d wins)",
			i+1, matrixutil.MentionNickHTML(ctx, cli, roomID, it.K), it.V, wins[it.K]))
	}
	return strings.Join(parts, "<br>")
}

func formatTopPlain(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, items []matrixutil.KV, wins map[string]int) string {
	if len(items) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(items))
	for i, it := range items {
		parts = append(parts, fmt.Sprintf("%d. @%s — %d wpm (%d wins)",
			i+1, matrixutil.DisplayNick(ctx, cli, roomID, it.K), it.V, wins[it.K]))
	}
	return strings.Join(parts, "\n")
}

func calculateWPM(text string, d time.Duration) int {
	words := len(strings.Fields(text))
	minutes := d.Minutes()
	if minutes == 0 {
		return 0
	}
	return int(float64(words) / minutes)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	default:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}
