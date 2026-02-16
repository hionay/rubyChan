package roulette

import (
	"context"
	"fmt"
	"html"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/hionay/rubyChan/state"
)

type RouletteCmd struct {
	Store *state.Namespace
	mu    sync.Map
}

func (*RouletteCmd) Name() string      { return "roulette" }
func (*RouletteCmd) Aliases() []string { return nil }
func (*RouletteCmd) Usage() string     { return "!roulette - Play Russian Roulette | !roulette stats" }

type roundState struct {
	Click   int `json:"click"`
	Chamber int `json:"chamber"`
}

type statsState struct {
	TotalPulls    int `json:"total_pulls"`
	TotalDeaths   int `json:"total_deaths"`
	CurrentStreak int `json:"current_streak"`
	LongestStreak int `json:"longest_streak"`

	DeathsByUser   map[string]int `json:"deaths_by_user"`
	SurvivesByUser map[string]int `json:"survives_by_user"`
}

func (c *RouletteCmd) roomLock(roomID string) *sync.Mutex {
	v, _ := c.mu.LoadOrStore(roomID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (c *RouletteCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	roomID := evt.RoomID.String()
	lock := c.roomLock(roomID)
	lock.Lock()
	defer lock.Unlock()

	if len(args) > 0 && strings.EqualFold(args[0], "stats") {
		c.sendStats(ctx, cli, evt)
		return
	}

	c.play(ctx, cli, evt)
}

func (c *RouletteCmd) play(ctx context.Context, cli *mautrix.Client, evt *event.Event) {
	roomID := evt.RoomID.String()
	roundKey := "round:" + roomID
	statsKey := "stats:" + roomID

	rs := &roundState{}
	if err := c.Store.GetJSON(roundKey, rs); err != nil {
		log.Printf("roulette: error loading round state: %v", err)
		cli.SendText(ctx, evt.RoomID, "Internal error")
		return
	}

	ss := &statsState{}
	if err := c.Store.GetJSON(statsKey, ss); err != nil {
		log.Printf("roulette: error loading stats state: %v", err)
		cli.SendText(ctx, evt.RoomID, "Internal error")
		return
	}
	if ss.DeathsByUser == nil {
		ss.DeathsByUser = make(map[string]int)
	}
	if ss.SurvivesByUser == nil {
		ss.SurvivesByUser = make(map[string]int)
	}

	if rs.Chamber == 0 {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		rs.Chamber = rnd.Intn(6) + 1
	}

	rs.Click++
	ss.TotalPulls++

	sender := evt.Sender.String()
	var reply string

	if rs.Click == rs.Chamber {
		reply = fmt.Sprintf("(%d/6) 💥 Bang! You’re dead.", rs.Click)

		ss.TotalDeaths++
		ss.DeathsByUser[sender]++

		if ss.CurrentStreak > ss.LongestStreak {
			ss.LongestStreak = ss.CurrentStreak
		}
		ss.CurrentStreak = 0

		if err := c.Store.Delete(roundKey); err != nil {
			log.Printf("roulette: error deleting round state: %v", err)
		}
	} else {
		reply = fmt.Sprintf("(%d/6) click... you survived.", rs.Click)

		ss.CurrentStreak++
		ss.SurvivesByUser[sender]++

		if err := c.Store.PutJSON(roundKey, rs); err != nil {
			log.Printf("roulette: error saving round state: %v", err)
		}
	}

	if err := c.Store.PutJSON(statsKey, ss); err != nil {
		log.Printf("roulette: error saving stats state: %v", err)
	}

	sendMentionReply(ctx, cli, evt, sender, reply)
}

func (c *RouletteCmd) sendStats(ctx context.Context, cli *mautrix.Client, evt *event.Event) {
	roomID := evt.RoomID.String()
	roundKey := "round:" + roomID
	statsKey := "stats:" + roomID

	rs := &roundState{}
	_ = c.Store.GetJSON(roundKey, rs)

	ss := &statsState{}
	if err := c.Store.GetJSON(statsKey, ss); err != nil {
		log.Printf("roulette: error loading stats: %v", err)
		cli.SendText(ctx, evt.RoomID, "Internal error")
		return
	}
	if ss.DeathsByUser == nil {
		ss.DeathsByUser = map[string]int{}
	}
	if ss.SurvivesByUser == nil {
		ss.SurvivesByUser = map[string]int{}
	}

	currentLine := "Current round: none (start with !roulette)"
	if rs.Chamber != 0 {
		currentLine = fmt.Sprintf("Current round: %d/6 pulls (still alive)", rs.Click)
	}

	deathRate := 0.0
	if ss.TotalPulls > 0 {
		deathRate = (float64(ss.TotalDeaths) / float64(ss.TotalPulls)) * 100
	}

	topDeaths := topN(ss.DeathsByUser, 3)
	topSurv := topN(ss.SurvivesByUser, 3)

	plainMsg := fmt.Sprintf(
		"Roulette stats (this room)\n%s\nAll-time: %d pulls • %d deaths • %.1f%% death rate\nLongest streak: %d survivals\nMost deaths: %s\nMost survivals: %s",
		currentLine,
		ss.TotalPulls, ss.TotalDeaths, deathRate,
		ss.LongestStreak,
		formatTopPlain(ctx, cli, evt.RoomID, topDeaths),
		formatTopPlain(ctx, cli, evt.RoomID, topSurv),
	)

	htmlMsg := fmt.Sprintf(
		`<b>Roulette stats</b> (this room)<br>%s<br>`+
			`All-time: %d pulls • %d deaths • %.1f%% death rate<br>`+
			`Longest streak: %d survivals<br>`+
			`Most deaths: %s<br>`+
			`Most survivals: %s`,
		html.EscapeString(currentLine),
		ss.TotalPulls, ss.TotalDeaths, deathRate,
		ss.LongestStreak,
		formatTopHTML(ctx, cli, evt.RoomID, topDeaths),
		formatTopHTML(ctx, cli, evt.RoomID, topSurv),
	)

	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          plainMsg,
		Format:        event.FormatHTML,
		FormattedBody: htmlMsg,
	}

	if _, err := cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content); err != nil {
		log.Printf("roulette: failed to send stats: %v", err)
	}
}

type kv struct {
	K string
	V int
}

func topN(m map[string]int, n int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{K: k, V: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].V == out[j].V {
			return out[i].K < out[j].K
		}
		return out[i].V > out[j].V
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func formatTopHTML(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, items []kv) string {
	if len(items) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s (%d)", mentionNickHTML(ctx, cli, roomID, it.K), it.V))
	}
	return strings.Join(parts, ", ")
}

func formatTopPlain(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, items []kv) string {
	if len(items) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("@%s (%d)", displayNick(ctx, cli, roomID, it.K), it.V))
	}
	return strings.Join(parts, ", ")
}

func mentionNickHTML(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, mxid string) string {
	nick := displayNick(ctx, cli, roomID, mxid)
	return fmt.Sprintf(
		`<a href="https://matrix.to/#/%s">@%s</a>`,
		html.EscapeString(mxid),
		html.EscapeString(nick),
	)
}

func displayNick(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, mxid string) string {
	if cli.StateStore != nil {
		if member, err := cli.StateStore.GetMember(ctx, roomID, id.UserID(mxid)); err == nil && member != nil {
			if member.Displayname != "" {
				return member.Displayname
			}
		}
	}

	if i := strings.IndexByte(mxid, ':'); i > 1 && mxid[0] == '@' {
		return mxid[1:i]
	}
	return mxid
}

func sendMentionReply(ctx context.Context, cli *mautrix.Client, evt *event.Event, senderMXID, reply string) {
	nick := displayNick(ctx, cli, evt.RoomID, senderMXID)

	escMXID := html.EscapeString(senderMXID)
	escNick := html.EscapeString(nick)
	escReply := html.EscapeString(reply)

	mention := fmt.Sprintf(`<a href="https://matrix.to/#/%s">@%s</a>`, escMXID, escNick)

	content := event.MessageEventContent{
		MsgType:       event.MsgText,
		Body:          fmt.Sprintf("%s: %s", senderMXID, reply),
		Format:        event.FormatHTML,
		FormattedBody: fmt.Sprintf("%s: %s", mention, escReply),
	}
	if _, err := cli.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, content); err != nil {
		log.Printf("roulette: failed to send reply: %v", err)
	}
}
