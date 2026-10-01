package teams

import (
	"time"

	"github.com/glenthomas/microsoft-teams-cli/internal/graph"
)

// Reaction is a simplified reaction to a message.
type Reaction struct {
	Type            string    `json:"type"`
	DisplayName     string    `json:"displayName,omitempty"`
	User            string    `json:"user,omitempty"`
	UserID          string    `json:"userId,omitempty"`
	CreatedDateTime time.Time `json:"createdDateTime"`
}

// Mention is a simplified @mention in a message.
type Mention struct {
	Text string `json:"text"`
	Kind string `json:"kind,omitempty"` // user, application, device, conversation or tag
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

func identity(set *graph.IdentitySet) (name, id string) {
	if set == nil {
		return "", ""
	}
	for _, i := range []*graph.Identity{set.User, set.Application, set.Device} {
		if i != nil {
			return i.DisplayName, i.ID
		}
	}
	return "", ""
}

func newReactions(rs []graph.Reaction) []Reaction {
	var out []Reaction
	for _, r := range rs {
		name, id := identity(r.User)
		out = append(out, Reaction{
			Type: r.ReactionType, DisplayName: r.DisplayName,
			User: name, UserID: id, CreatedDateTime: r.CreatedDateTime,
		})
	}
	return out
}

func newMentions(ms []graph.Mention) []Mention {
	var out []Mention
	for _, m := range ms {
		mention := Mention{Text: m.MentionText}
		if set := m.Mentioned; set != nil {
			for _, c := range []struct {
				kind string
				id   *graph.Identity
			}{
				{"user", set.User}, {"application", set.Application}, {"device", set.Device},
				{"conversation", set.Conversation}, {"tag", set.Tag},
			} {
				if c.id != nil {
					mention.Kind, mention.Name, mention.ID = c.kind, c.id.DisplayName, c.id.ID
					break
				}
			}
		}
		out = append(out, mention)
	}
	return out
}
