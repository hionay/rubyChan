package matrixutil

import (
	"cmp"
	"context"
	"fmt"
	"html"
	"sort"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

type KV[V cmp.Ordered] struct {
	K string
	V V
}

func TopN[V cmp.Ordered](m map[string]V, n int) []KV[V] {
	out := make([]KV[V], 0, len(m))
	for k, v := range m {
		out = append(out, KV[V]{K: k, V: v})
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

func DisplayNick(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, mxid string) string {
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

func MentionNickHTML(ctx context.Context, cli *mautrix.Client, roomID id.RoomID, mxid string) string {
	nick := DisplayNick(ctx, cli, roomID, mxid)
	return fmt.Sprintf(`<a href="https://matrix.to/#/%s">@%s</a>`,
		html.EscapeString(mxid), html.EscapeString(nick))
}
