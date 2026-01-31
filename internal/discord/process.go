package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/gateway"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

// ProcessMessage builds inbound context and dispatches to agent.
func ProcessMessage(ctx context.Context, pre *PreflightContext, opts ProcessOpts) error {
	if pre == nil {
		return nil
	}

	msg := pre.Message
	text := pre.MessageText
	if strings.TrimSpace(text) == "" {
		return nil
	}

	fromLabel := buildFromLabel(pre)
	senderLabel := buildSenderLabel(pre)

	msgCtx := &inbound.MsgContext{
		Body:               text,
		RawBody:            pre.BaseText,
		CommandBody:        pre.BaseText,
		From:               fromLabel,
		To:                 pre.ChannelID,
		SessionKey:         pre.Route.SessionKey,
		AccountID:          pre.AccountID,
		ChatType:           chatType(pre),
		ConversationLabel:  fromLabel,
		SenderName:         senderLabel,
		SenderId:           pre.Author.ID,
		SenderUsername:     pre.Author.Username,
		Provider:           "discord",
		Surface:            "discord",
		WasMentioned:       pre.WasMentioned,
		MessageSid:         msg.ID,
		CommandAuthorized:  pre.CommandAuthorized,
		OriginatingChannel: "discord",
		OriginatingTo:      buildReplyTarget(pre),
		ReplyChannelID:     pre.ChannelID,
	}

	if opts.Dispatcher == nil || opts.DispatchInbound == nil {
		return nil
	}
	return opts.DispatchInbound(ctx, msgCtx, opts.Dispatcher)
}

// ProcessOpts holds options for ProcessMessage.
type ProcessOpts struct {
	DispatchInbound func(ctx context.Context, msgCtx *inbound.MsgContext, d gateway.Dispatcher) error
	Dispatcher      gateway.Dispatcher
}

func buildFromLabel(pre *PreflightContext) string {
	if pre.IsDirectMessage {
		return formatUserTag(pre.Author)
	}
	if pre.ChannelName != "" {
		return fmt.Sprintf("#%s", pre.ChannelName)
	}
	return pre.ChannelID
}

func buildSenderLabel(pre *PreflightContext) string {
	author := pre.Author
	display := author.Username
	if pre.Message != nil && pre.Message.Member != nil && pre.Message.Member.Nick != "" {
		display = pre.Message.Member.Nick
	}
	tag := formatUserTag(author)
	if display != "" && tag != "" && display != tag {
		return fmt.Sprintf("%s (%s)", display, tag)
	}
	if display != "" {
		return display
	}
	return tag
}

func formatUserTag(u *discordgo.User) string {
	if u == nil {
		return ""
	}
	if u.GlobalName != "" && u.GlobalName != u.Username {
		return u.GlobalName + " (" + u.Username + ")"
	}
	return u.Username
}

func chatType(pre *PreflightContext) string {
	if pre.IsDirectMessage {
		return "direct"
	}
	return "channel"
}

func buildReplyTarget(pre *PreflightContext) string {
	if pre.IsDirectMessage {
		return "user:" + pre.Author.ID
	}
	return "channel:" + pre.ChannelID
}
