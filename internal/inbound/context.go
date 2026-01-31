package inbound

import (
	"strings"
)

// MsgContext is the finalized inbound message context (from FinalizedMsgContext in TS).
type MsgContext struct {
	Body              string
	RawBody           string
	CommandBody       string
	BodyForAgent      string
	BodyForCommands   string
	From              string
	To                string
	SessionKey        string
	AccountID         string
	ChatType          string // "direct" or "channel"
	ConversationLabel string
	SenderName        string
	SenderId          string
	SenderUsername    string
	Provider          string
	Surface           string
	WasMentioned      bool
	MessageSid        string
	Timestamp         int64
	CommandAuthorized bool
	OriginatingChannel string
	OriginatingTo     string
	// ReplyChannelID is the Discord channel ID to send reply (for Discord dispatcher).
	ReplyChannelID string
}

// Finalize normalizes and finalizes the context.
func (c *MsgContext) Finalize() {
	c.Body = normalizeNewlines(c.Body)
	c.RawBody = normalizeTextField(c.RawBody)
	c.CommandBody = normalizeTextField(c.CommandBody)
	if c.BodyForAgent == "" {
		c.BodyForAgent = c.Body
	}
	c.BodyForAgent = normalizeNewlines(c.BodyForAgent)
	if c.BodyForCommands == "" {
		c.BodyForCommands = c.CommandBody
		if c.BodyForCommands == "" {
			c.BodyForCommands = c.RawBody
		}
		if c.BodyForCommands == "" {
			c.BodyForCommands = c.Body
		}
	}
	c.BodyForCommands = normalizeNewlines(c.BodyForCommands)
}

func normalizeTextField(s string) string {
	s = strings.TrimSpace(s)
	return normalizeNewlines(s)
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n")
}
