package teams

import (
	"context"
	"sync"
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

// nameLookupWorkers bounds concurrent directory lookups for reacting users.
const nameLookupWorkers = 8

// NameReactors fills in Reaction.User for channel messages. Graph usually
// returns only the reacting user's ID, so names come from authors and
// mentions already in msgs, then from directory lookups.
func (s *Service) NameReactors(ctx context.Context, msgs []Message) {
	var reactions []*Reaction
	for i := range msgs {
		m := &msgs[i]
		s.remember(m.AuthorID, m.Author)
		for _, mention := range m.Mentions {
			if mention.Kind == "user" {
				s.remember(mention.ID, mention.Name)
			}
		}
		for j := range m.Reactions {
			reactions = append(reactions, &m.Reactions[j])
		}
	}
	s.nameReactions(ctx, reactions)
}

// NameChatReactors is NameReactors for chat messages.
func (s *Service) NameChatReactors(ctx context.Context, msgs []ChatMessage) {
	var reactions []*Reaction
	for i := range msgs {
		m := &msgs[i]
		s.remember(m.AuthorID, m.Author)
		for _, mention := range m.Mentions {
			if mention.Kind == "user" {
				s.remember(mention.ID, mention.Name)
			}
		}
		for j := range m.Reactions {
			reactions = append(reactions, &m.Reactions[j])
		}
	}
	s.nameReactions(ctx, reactions)
}

func (s *Service) remember(id, name string) {
	if id == "" || name == "" {
		return
	}
	if s.names == nil {
		s.names = map[string]string{}
	}
	s.names[id] = name
}

// nameReactions resolves unnamed reacting users. Lookups are best effort: a
// user that cannot be found (for example a deleted or external account) stays
// unnamed, and any other failure stops further lookups without failing the
// command.
func (s *Service) nameReactions(ctx context.Context, reactions []*Reaction) {
	for _, r := range reactions {
		s.remember(r.UserID, r.User)
	}
	var missing []string
	seen := map[string]bool{}
	for _, r := range reactions {
		if r.User != "" || r.UserID == "" || seen[r.UserID] {
			continue
		}
		seen[r.UserID] = true
		if _, ok := s.names[r.UserID]; !ok {
			missing = append(missing, r.UserID)
		}
	}
	if len(missing) > 0 && s.Graph != nil {
		s.lookupNames(ctx, missing)
	}
	for _, r := range reactions {
		if r.User == "" {
			r.User = s.names[r.UserID]
		}
	}
}

func (s *Service) lookupNames(ctx context.Context, ids []string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := 0; w < min(nameLookupWorkers, len(ids)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				name, err := s.Graph.UserDisplayName(ctx, id)
				if err != nil && !graph.IsStatus(err, 404) {
					cancel()
					continue
				}
				mu.Lock()
				if s.names == nil {
					s.names = map[string]string{}
				}
				s.names[id] = name // "" records a user that does not exist
				mu.Unlock()
			}
		}()
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		jobs <- id
	}
	close(jobs)
	wg.Wait()
}
