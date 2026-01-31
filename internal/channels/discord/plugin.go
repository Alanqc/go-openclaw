package discord

import (
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/channels"
	discordpkg "github.com/openclaw/openclaw-go/internal/discord"
)

const pluginID = channels.ChannelId("discord")

// DiscordAccount holds Discord account config (token).
type DiscordAccount struct {
	Token string
}

// Plugin implements ChannelPlugin for Discord.
type Plugin struct{}

// ID returns the channel id.
func (Plugin) ID() channels.ChannelId {
	return pluginID
}

// StartAccount runs the Discord bot. Blocks until abort.
func (Plugin) StartAccount(ctx channels.StartAccountContext) error {
	acc, ok := ctx.Account.(*DiscordAccount)
	if !ok || acc == nil || acc.Token == "" {
		slog.Error("discord: account missing or invalid")
		return nil
	}
	token := strings.TrimSpace(acc.Token)
	if token != "" && !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	s, err := discordgo.New(token)
	if err != nil {
		slog.Error("discord: create session", "err", err)
		return err
	}
	defer s.Close()

	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent | discordgo.IntentsGuilds

	handler := &discordpkg.MessageHandler{
		Cfg:             ctx.Cfg,
		DiscordCfg:      &discordpkg.DiscordConfig{AllowBots: false, DMPolicy: "open"},
		AccountID:       ctx.AccountID,
		BotUserID:       "",
		DMEnabled:       true,
		GroupDMEnabled:  true,
		GuildEntries:    nil,
		DispatchInbound: ctx.Runtime.DispatchInbound,
	}

	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handler.Handle(s, m)
	})

	if err := s.Open(); err != nil {
		slog.Error("discord: open connection", "err", err)
		return err
	}
	if s.State != nil && s.State.User != nil {
		handler.BotUserID = s.State.User.ID
		slog.Info("discord: logged in", "account", ctx.AccountID, "bot_id", handler.BotUserID)
	}

	slog.Info("discord: monitor running, press Ctrl+C to stop")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("discord: shutting down")
	return nil
}
