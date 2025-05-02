package history

import (
	"sync"

	"maunium.net/go/mautrix/id"
)

type HistoryMessage struct {
	Sender    string
	Body      string
	Timestamp int64
}

type HistoryStore struct {
	mu    sync.Mutex
	data  map[id.RoomID][]HistoryMessage
	limit int
}

func NewHistoryStore(limit int) *HistoryStore {
	return &HistoryStore{
		data:  make(map[id.RoomID][]HistoryMessage),
		limit: limit,
	}
}

func (hs *HistoryStore) Add(roomID id.RoomID, msg HistoryMessage) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hist := hs.data[roomID]
	hist = append(hist, msg)
	if len(hist) > hs.limit {
		hist = hist[len(hist)-hs.limit:]
	}
	hs.data[roomID] = hist
}

func (hs *HistoryStore) GetLast(roomID id.RoomID, n int) []HistoryMessage {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hist := hs.data[roomID]
	if len(hist) < n {
		n = len(hist)
	}
	return append([]HistoryMessage(nil), hist[len(hist)-n:]...)
}
