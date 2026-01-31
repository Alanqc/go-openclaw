package discord

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/dispatch"
)

// MessageHandler handles incoming Discord messages (debounce + preflight + process).
type MessageHandler struct {
	Cfg            *config.Config
	DiscordCfg     *DiscordConfig
	AccountID      string
	BotUserID      string
	DMEnabled      bool
	GroupDMEnabled bool
	GuildEntries   map[string]GuildEntry
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

	disp := &dispatch.DiscordDispatcher{}
	disp.Session = s
	disp.ChannelID = pre.ChannelID

	err := ProcessMessage(ctx, pre, ProcessOpts{
		DiscordDispatcher: disp,
	})
	if err != nil {
		slog.Error("discord process failed", "err", err, "msgId", pre.Message.ID)
	}
}
