package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/discord"
	"github.com/openclaw/openclaw-go/internal/dispatch"
	"github.com/openclaw/openclaw-go/internal/gateway"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

func main() {
	token := flag.String("token", "", "Discord bot token (or DISCORD_TOKEN env)")
	configPath := flag.String("config", "", "Config file path (or OPENCLAW_CONFIG env)")
	flag.Parse()

	if *token == "" {
		*token = os.Getenv("DISCORD_TOKEN")
	}
	if *token == "" {
		slog.Error("discord token required (--token or DISCORD_TOKEN)")
		os.Exit(1)
	}

	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = config.ResolveConfigPath()
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("load config", "path", cfgPath, "err", err)
		os.Exit(1)
	}

	s, err := discordgo.New("Bot " + *token)
	if err != nil {
		slog.Error("create discord session", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent | discordgo.IntentsGuilds

	handler := &discord.MessageHandler{
		Cfg:            cfg,
		DiscordCfg:     &discord.DiscordConfig{AllowBots: false, DMPolicy: "open"},
		AccountID:      "default",
		BotUserID:      "", // set after Open
		DMEnabled:      true,
		GroupDMEnabled: true,
		GuildEntries:   nil,
		DispatchInbound: func(ctx context.Context, msgCtx *inbound.MsgContext, d gateway.Dispatcher) error {
			return dispatch.DispatchInbound(ctx, msgCtx, d)
		},
	}

	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handler.Handle(s, m)
	})

	if err := s.Open(); err != nil {
		slog.Error("open discord connection", "err", err)
		os.Exit(1)
	}
	if s.State != nil && s.State.User != nil {
		handler.BotUserID = s.State.User.ID
		slog.Info("logged in", "bot_id", handler.BotUserID)
	}

	slog.Info("openclaw-go discord monitor running. Press Ctrl+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
}
