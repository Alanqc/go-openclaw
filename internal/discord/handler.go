package discord

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/dispatch"
	"github.com/openclaw/openclaw-go/internal/gateway"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

// MessageHandler handles incoming Discord messages (debounce + preflight + process).
type MessageHandler struct {
	Cfg             *config.Config
	DiscordCfg      *DiscordConfig
	AccountID       string
	BotUserID       string
	DMEnabled       bool
	GroupDMEnabled  bool
	GuildEntries    map[string]GuildEntry
	// DispatchInbound is called to process the message (from gateway runtime).
	// If nil, messages are not dispatched.
	DispatchInbound func(ctx context.Context, msgCtx *inbound.MsgContext, d gateway.Dispatcher) error
}

// Handle is called for each MessageCreate event.
func (h *MessageHandler) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()

	pre := Preflight(PreflightParams{
		Cfg:            h.Cfg,
		DiscordCfg:     h.DiscordCfg,
		AccountID:      h.AccountID,
		BotUserID:      h.BotUserID,
		Data:           m,
		DMEnabled:      h.DMEnabled,
		GroupDMEnabled: h.GroupDMEnabled,
		GuildEntries:   h.GuildEntries,
	})
	if pre == nil {
		return
	}
	if h.DispatchInbound == nil {
		slog.Debug("discord: no DispatchInbound, skip")
		return
	}

	disp := &dispatch.DiscordDispatcher{Session: s, ChannelID: pre.ChannelID}
	err := ProcessMessage(ctx, pre, ProcessOpts{
		DispatchInbound: h.DispatchInbound,
		Dispatcher:      disp,
	})
	if err != nil {
		slog.Error("discord process failed", "err", err, "msgId", pre.Message.ID)
	}
}
