package dispatch

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/agent"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

// Dispatcher sends replies to a channel (in-process, function call).
type Dispatcher interface {
	SendFinal(ctx context.Context, channelID, text string) error
}

// DiscordDispatcher sends replies via Discord API.
type DiscordDispatcher struct {
	Session   *discordgo.Session
	ChannelID string
}

// SendFinal sends a final reply.
func (d *DiscordDispatcher) SendFinal(ctx context.Context, channelID, text string) error {
	if d.Session == nil {
		slog.Info("dispatch: no session, would send", "channel", channelID, "text", truncateStr(text, 50))
		return nil
	}
	_, err := d.Session.ChannelMessageSend(channelID, text)
	return err
}

// DispatchInbound processes the message via in-process agent and dispatches reply.
func DispatchInbound(ctx context.Context, msgCtx *inbound.MsgContext, dispatcher Dispatcher) error {
	msgCtx.Finalize()
	if strings.TrimSpace(msgCtx.BodyForCommands) == "" {
		slog.Debug("dispatch: empty body, skip")
		return nil
	}

	reply, err := agent.Run(ctx, msgCtx)
	if err != nil {
		return err
	}
	if reply == "" {
		return nil
	}

	target := msgCtx.ReplyChannelID
	if target == "" {
		target = msgCtx.OriginatingTo
	}
	if dispatcher != nil && target != "" {
		slog.Info("dispatch: inbound message",
			"sessionKey", msgCtx.SessionKey,
			"from", msgCtx.From,
			"body", truncateStr(msgCtx.BodyForCommands, 100))
		return dispatcher.SendFinal(ctx, target, reply)
	}
	return nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
