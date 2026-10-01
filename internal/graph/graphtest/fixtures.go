package graphtest

import (
	"time"

	"github.com/glenthomas/microsoft-teams-cli/internal/graph"
)

// Msg builds a channel message created at t with an HTML body.
func Msg(id string, t time.Time, author, html string, replies ...graph.ChatMessage) graph.ChatMessage {
	for i := range replies {
		replies[i].ReplyToID = id
	}
	return graph.ChatMessage{
		ID:              id,
		MessageType:     "message",
		CreatedDateTime: t,
		From:            &graph.IdentitySet{User: &graph.Identity{ID: "u-" + author, DisplayName: author}},
		Body:            graph.ItemBody{ContentType: "html", Content: html},
		WebURL:          "https://teams.microsoft.com/l/message/" + id,
		Replies:         replies,
	}
}

// Seed populates s with two teams:
//
//	Platform (team-1): "Platform Engineering" (19:pe), "General" (19:gen1)
//	Data (team-2):     "General" (19:gen2)
//
// Messages in "Platform Engineering" are relative to now, newest thread first.
func (s *Server) Seed(now time.Time) {
	day := 24 * time.Hour
	s.Me = graph.User{ID: "me-1", DisplayName: "Test User", UserPrincipalName: "test@example.com"}
	s.Users["u-Dave"] = graph.User{ID: "u-Dave", DisplayName: "Dave"}
	s.Teams = []graph.Team{{ID: "team-1", DisplayName: "Platform"}, {ID: "team-2", DisplayName: "Data"}}
	s.Channels["team-1"] = []graph.Channel{
		{ID: "19:pe@thread.tacv2", DisplayName: "Platform Engineering", MembershipType: "standard"},
		{ID: "19:gen1@thread.tacv2", DisplayName: "General", MembershipType: "standard"},
	}
	s.Channels["team-2"] = []graph.Channel{
		{ID: "19:gen2@thread.tacv2", DisplayName: "General", MembershipType: "standard"},
	}
	system := Msg("sys", now.Add(-1*day), "", "<systemEventMessage/>")
	system.MessageType = "systemEventMessage"
	deleted := Msg("del", now.Add(-2*day), "Bob", "private endpoints secret")
	deletedAt := now.Add(-2 * day)
	deleted.DeletedDateTime = &deletedAt
	r1 := Msg("r1", now.Add(-2*day), "Bob", "<p>Yes, private endpoints plus DNS zones, <at id=\"0\">Carol</at>.</p>")
	r1.Mentions = []graph.Mention{{ID: 0, MentionText: "Carol", Mentioned: &graph.MentionedIdentitySet{User: &graph.Identity{ID: "u-Carol", DisplayName: "Carol"}}}}
	r1.Reactions = []graph.Reaction{
		{ReactionType: "like", CreatedDateTime: now.Add(-2*day + time.Minute), User: &graph.IdentitySet{User: &graph.Identity{ID: "u-Alice"}}},
		{ReactionType: "❤️", DisplayName: "Heart", CreatedDateTime: now.Add(-2*day + 2*time.Minute), User: &graph.IdentitySet{User: &graph.Identity{ID: "u-Carol", DisplayName: "Carol"}}},
		{ReactionType: "laugh", CreatedDateTime: now.Add(-2*day + 3*time.Minute), User: &graph.IdentitySet{User: &graph.Identity{ID: "u-Dave"}}},
		{ReactionType: "like", CreatedDateTime: now.Add(-2*day + 4*time.Minute), User: &graph.IdentitySet{User: &graph.Identity{ID: "u-Gone"}}},
	}
	s.Messages["19:pe@thread.tacv2"] = []graph.ChatMessage{
		system,
		Msg("m1", now.Add(-3*day), "Alice", "<p>Should we use <b>Private Endpoints</b> for ACR?</p>",
			r1,
			Msg("r2", now.Add(-2*day+time.Hour), "Carol", "<p>Agreed</p>"),
		),
		deleted,
		Msg("m2", now.Add(-10*day), "Dave", "<p>Cluster upgrade tonight</p>"),
		// Old thread with a recent reply mentioning the search terms.
		Msg("m3", now.Add(-60*day), "Erin", "<p>Networking notes</p>",
			Msg("r3", now.Add(-5*day), "Frank", "<p>Update: private endpoints are now enabled</p>"),
		),
		Msg("m4", now.Add(-45*day), "Gina", "<p>Old private endpoints discussion</p>"),
		Msg("m5", now.Add(-90*day), "Hank", "<p>Very old private endpoints</p>"),
		Msg("m6", now.Add(-120*day), "Ivan", "<p>Ancient private endpoints</p>"),
		Msg("m7", now.Add(-150*day), "Judy", "<p>Prehistoric private endpoints</p>"),
	}
	s.Messages["19:gen1@thread.tacv2"] = []graph.ChatMessage{
		Msg("g1", now.Add(-1*day), "Alice", "<p>Welcome! private endpoints FAQ is pinned.</p>"),
	}
	s.Messages["19:gen2@thread.tacv2"] = []graph.ChatMessage{}
}
