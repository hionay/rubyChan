package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type WebhookRequest struct {
	Room    string `json:"room"`
	Message string `json:"message"`
}

func newWebhookServer(cli *mautrix.Client, addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req WebhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: invalid JSON", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		roomID, err := findRoomIDByName(ctx, cli, req.Room)
		if err != nil {
			http.Error(w, fmt.Sprintf("room not found: %v", err), http.StatusNotFound)
			return
		}
		if _, err := cli.SendText(ctx, roomID, req.Message); err != nil {
			http.Error(w, fmt.Sprintf("send failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func findRoomIDByName(ctx context.Context, cli *mautrix.Client, name string) (id.RoomID, error) {
	jr, err := cli.JoinedRooms(ctx)
	if err != nil {
		return "", err
	}
	for _, rid := range jr.JoinedRooms {
		var ev struct {
			Name string `json:"name"`
		}
		if err := cli.StateEvent(ctx, rid, event.StateRoomName, "", &ev); err != nil {
			continue
		}
		if ev.Name == name {
			return rid, nil
		}
	}
	return "", fmt.Errorf("no joined room with name '%s'", name)
}
