package routing

import (
	"github.com/openclaw/openclaw-go/internal/config"
	"strings"
)

// RoutePeerKind is dm, group, or channel.
type RoutePeerKind string

const (
	PeerDM      RoutePeerKind = "dm"
	PeerGroup   RoutePeerKind = "group"
	PeerChannel RoutePeerKind = "channel"
)

// RoutePeer identifies the message origin.
type RoutePeer struct {
	Kind RoutePeerKind
	ID   string
}

// ResolvedAgentRoute is the result of route resolution.
type ResolvedAgentRoute struct {
	AgentID      string
	Channel      string
	AccountID    string
	SessionKey   string
	MainSessionKey string
	MatchedBy    string // binding.peer, binding.guild, binding.team, binding.account, binding.channel, default
}

// ResolveAgentRouteInput for route resolution.
type ResolveAgentRouteInput struct {
	Cfg       *config.Config
	Channel   string
	AccountID string
	Peer      *RoutePeer
	GuildID   string
	TeamID    string
}

func matchesAccountId(matchAccountId, actual string) bool {
	m := strings.TrimSpace(matchAccountId)
	if m == "" {
		return actual == DefaultAccountID
	}
	if m == "*" {
		return true
	}
	return m == actual
}

func matchesChannel(match *config.BindingMatch, channel string) bool {
	if match == nil {
		return false
	}
	k := NormalizeToken(match.Channel)
	if k == "" {
		return false
	}
	return k == channel
}

func matchesPeer(match *config.BindingMatch, peer RoutePeer) bool {
	if match == nil || match.Peer == nil {
		return false
	}
	m := match.Peer
	kind := NormalizeToken(m.Kind)
	id := NormalizeID(m.ID)
	if kind == "" || id == "" {
		return false
	}
	return kind == string(peer.Kind) && id == peer.ID
}

func matchesGuild(match *config.BindingMatch, guildId string) bool {
	if match == nil {
		return false
	}
	id := NormalizeID(match.GuildID)
	if id == "" {
		return false
	}
	return id == guildId
}

func matchesTeam(match *config.BindingMatch, teamId string) bool {
	if match == nil {
		return false
	}
	id := NormalizeID(match.TeamID)
	if id == "" {
		return false
	}
	return id == teamId
}

func resolveDefaultAgentId(cfg *config.Config) string {
	if cfg == nil || cfg.Agents.Defaults.DefaultModel == "" {
		return DefaultAgentID
	}
	for _, a := range cfg.Agents.List {
		if strings.TrimSpace(a.ID) != "" {
			return SanitizeAgentId(a.ID)
		}
	}
	return DefaultAgentID
}

func pickFirstExistingAgentId(cfg *config.Config, agentId string) string {
	a := strings.TrimSpace(agentId)
	if a == "" {
		return SanitizeAgentId(resolveDefaultAgentId(cfg))
	}
	norm := NormalizeAgentId(a)
	for _, entry := range cfg.Agents.List {
		if NormalizeAgentId(entry.ID) == norm && strings.TrimSpace(entry.ID) != "" {
			return SanitizeAgentId(strings.TrimSpace(entry.ID))
		}
	}
	return SanitizeAgentId(a)
}

// ResolveAgentRoute resolves which agent handles the message.
func ResolveAgentRoute(input ResolveAgentRouteInput) ResolvedAgentRoute {
	if input.Cfg == nil {
		input.Cfg = &config.Config{}
	}
	channel := NormalizeToken(input.Channel)
	accountId := NormalizeAccountID(input.AccountID)

	var peer *RoutePeer
	if input.Peer != nil {
		peer = &RoutePeer{
			Kind: input.Peer.Kind,
			ID:   NormalizeID(input.Peer.ID),
		}
	}

	guildId := NormalizeID(input.GuildID)
	teamId := NormalizeID(input.TeamID)

	choose := func(agentId, matchedBy string) ResolvedAgentRoute {
		resolved := pickFirstExistingAgentId(input.Cfg, agentId)
		dmScope := "main"
		if input.Cfg != nil && input.Cfg.Session.DMScope != "" {
			dmScope = input.Cfg.Session.DMScope
		}
		sessionKey := BuildAgentSessionKey(BuildAgentSessionKeyParams{
			AgentID:   resolved,
			Channel:   channel,
			AccountID: accountId,
			Peer:      peer,
			DMScope:   dmScope,
		})
		mainKey := BuildAgentMainSessionKey(resolved, DefaultMainKey)
		return ResolvedAgentRoute{
			AgentID:        resolved,
			Channel:        channel,
			AccountID:      accountId,
			SessionKey:     strings.ToLower(sessionKey),
			MainSessionKey: strings.ToLower(mainKey),
			MatchedBy:      matchedBy,
		}
	}

	for _, b := range input.Cfg.Bindings {
		if !matchesChannel(&b.Match, channel) {
			continue
		}
		if !matchesAccountId(b.Match.AccountID, accountId) {
			continue
		}

		if peer != nil {
			if matchesPeer(&b.Match, *peer) {
				return choose(b.AgentID, "binding.peer")
			}
		}
		if guildId != "" {
			if matchesGuild(&b.Match, guildId) {
				return choose(b.AgentID, "binding.guild")
			}
		}
		if teamId != "" {
			if matchesTeam(&b.Match, teamId) {
				return choose(b.AgentID, "binding.team")
			}
		}

		// Account match (no peer/guild/team)
		m := b.Match
		if m.AccountID != "" && m.AccountID != "*" && m.Peer == nil && m.GuildID == "" && m.TeamID == "" {
			return choose(b.AgentID, "binding.account")
		}
		if m.AccountID == "*" && m.Peer == nil && m.GuildID == "" && m.TeamID == "" {
			return choose(b.AgentID, "binding.channel")
		}
	}

	return choose(resolveDefaultAgentId(input.Cfg), "default")
}

// BuildAgentSessionKeyParams for building session key.
type BuildAgentSessionKeyParams struct {
	AgentID   string
	Channel   string
	AccountID string
	Peer      *RoutePeer
	DMScope   string
}

// BuildAgentSessionKey builds the full session key.
func BuildAgentSessionKey(p BuildAgentSessionKeyParams) string {
	peerKind := "dm"
	if p.Peer != nil {
		peerKind = string(p.Peer.Kind)
	}
	peerId := ""
	if p.Peer != nil && p.Peer.ID != "" {
		peerId = p.Peer.ID
	} else {
		peerId = "unknown"
	}
	return BuildAgentPeerSessionKey(PeerSessionKeyParams{
		AgentID:   p.AgentID,
		Channel:   p.Channel,
		AccountID: p.AccountID,
		PeerKind:  peerKind,
		PeerID:    peerId,
		DMScope:   p.DMScope,
	})
}
