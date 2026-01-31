package routing

import (
	"regexp"
	"strings"
)

const (
	DefaultAgentID   = "main"
	DefaultMainKey   = "main"
	DefaultAccountID = "default"
)

var (
	validIDRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	invalidChars = regexp.MustCompile(`[^a-z0-9_-]+`)
)

// NormalizeToken trims and lowercases.
func NormalizeToken(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// NormalizeID trims.
func NormalizeID(s string) string {
	return strings.TrimSpace(s)
}

// NormalizeAgentID sanitizes agent ID.
func NormalizeAgentId(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultAgentID
	}
	s = strings.ToLower(s)
	if validIDRe.MatchString(s) {
		return s
	}
	s = invalidChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 64 {
		s = s[:64]
	}
	if s == "" {
		return DefaultAgentID
	}
	return s
}

// SanitizeAgentID same as NormalizeAgentId.
func SanitizeAgentId(s string) string {
	return NormalizeAgentId(s)
}

// NormalizeAccountID sanitizes account ID.
func NormalizeAccountID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultAccountID
	}
	s = strings.ToLower(s)
	if validIDRe.MatchString(s) {
		return s
	}
	s = invalidChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 64 {
		s = s[:64]
	}
	if s == "" {
		return DefaultAccountID
	}
	return s
}

// BuildAgentMainSessionKey builds "agent:{agentId}:{mainKey}".
func BuildAgentMainSessionKey(agentId, mainKey string) string {
	a := NormalizeAgentId(agentId)
	m := strings.TrimSpace(strings.ToLower(mainKey))
	if m == "" {
		m = DefaultMainKey
	}
	return "agent:" + a + ":" + m
}

// BuildAgentPeerSessionKey builds session key for peer-scoped sessions.
func BuildAgentPeerSessionKey(p PeerSessionKeyParams) string {
	agentId := NormalizeAgentId(p.AgentID)
	channel := NormalizeToken(p.Channel)
	if channel == "" {
		channel = "unknown"
	}
	accountId := NormalizeAccountID(p.AccountID)

	peerKind := p.PeerKind
	if peerKind == "" {
		peerKind = "dm"
	}

	dmScope := p.DMScope
	if dmScope == "" {
		dmScope = "main"
	}

	peerId := strings.TrimSpace(p.PeerID)
	if dmScope == "main" {
		return BuildAgentMainSessionKey(agentId, DefaultMainKey)
	}

	// per-peer or per-channel-peer
	parts := []string{"agent", agentId}
	if dmScope == "per-channel-peer" || dmScope == "per-account-channel-peer" {
		parts = append(parts, channel, accountId)
	}
	parts = append(parts, peerKind)
	if peerId != "" {
		parts = append(parts, peerId)
	} else {
		parts = append(parts, "unknown")
	}
	return strings.ToLower(strings.Join(parts, ":"))
}

// PeerSessionKeyParams for BuildAgentPeerSessionKey.
type PeerSessionKeyParams struct {
	AgentID    string
	Channel    string
	AccountID  string
	PeerKind   string // dm, group, channel
	PeerID     string
	DMScope    string
	MainKey    string
}
