package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

const quoteWebsite = "https://quotes.halil.io"

func handleQuote(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, args []string) {
	if len(args) < 1 {
		cli.SendText(ctx, roomID, "Usage: !quote N <optional comment>")
		return
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		cli.SendText(ctx, roomID, "Invalid number of lines")
		return
	}
	comment := ""
	if len(args) > 1 {
		comment = strings.Join(args[1:], " ")
	}

	hist := msgHistory[roomID]
	if len(hist) <= 1 {
		cli.SendText(ctx, roomID, "No messages to quote.")
		return
	}
	hist = hist[:len(hist)-1] // exclude the last command message (!quote N)
	if n > len(hist) {
		n = len(hist)
	}
	slice := hist[len(hist)-n:]
	lines := make([]string, n)
	for i, m := range slice {
		lines[i] = fmt.Sprintf("%s: %s", m.Sender, m.Body)
	}
	quoteText := strings.Join(lines, "\n")

	resp, err := http.PostForm(quoteWebsite+"/add", url.Values{
		"quote":   {quoteText},
		"comment": {comment},
	})
	if err != nil {
		cli.SendText(ctx, roomID, fmt.Sprintf("Error posting quote: %v", err))
		return
	}
	defer resp.Body.Close()

	// this is pain in the ass, i wish i could respond in a JSON from the quotes API
	b, _ := io.ReadAll(resp.Body)
	html := string(b)
	marker := `class="text-[#b4a6c6] text-sm hover:underline"`
	idx := strings.Index(html, marker)
	if idx < 0 {
		cli.SendText(ctx, roomID, "Error adding quote")
		return
	}
	hrefIdx := strings.LastIndex(html[:idx], `<a href="`)
	if hrefIdx < 0 {
		cli.SendText(ctx, roomID, "Error adding quote")
		return
	}
	start := hrefIdx + len(`<a href="`)
	end := strings.Index(html[start:], `"`)
	if end < 0 {
		cli.SendText(ctx, roomID, "Error adding quote")
		return
	}
	linkPath := html[start : start+end]
	fullLink := quoteWebsite + linkPath
	cli.SendText(ctx, roomID, fmt.Sprintf("Quote added: %s", fullLink))
}
