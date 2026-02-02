package dispatch

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/agent"
	"github.com/openclaw/openclaw-go/internal/gateway"
	"github.com/openclaw/openclaw-go/internal/inbound"
	"github.com/openclaw/openclaw-go/internal/llm"
)

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
// llmPlugin 来自 Runtime.LLM，可为 nil（则 agent 回显占位）。defaultModel 来自配置，可为空。
func DispatchInbound(ctx context.Context, msgCtx *inbound.MsgContext, dispatcher gateway.Dispatcher, llmPlugin llm.Plugin, defaultModel string) error {
	msgCtx.Finalize()
	if strings.TrimSpace(msgCtx.BodyForCommands) == "" {
		slog.Debug("dispatch: empty body, skip")
		return nil
	}

	reply, err := agent.Run(ctx, msgCtx, llmPlugin, defaultModel)
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
