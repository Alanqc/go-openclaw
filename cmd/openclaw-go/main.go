package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/openclaw/openclaw-go/internal/channels"
	"github.com/openclaw/openclaw-go/internal/channels/discord"
	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/dispatch"
	"github.com/openclaw/openclaw-go/internal/gateway"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

func main() {
	tokenFlag := flag.String("token", "", "Discord bot token (or DISCORD_TOKEN env)")
	configPath := flag.String("config", "", "Config file path (or OPENCLAW_CONFIG env)")
	flag.Parse()

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("DISCORD_TOKEN")
	}
	if token == "" {
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

	// Gateway as main process: create runtime, register plugins, start channels.
	rt := &gateway.Runtime{
		Config: cfg,
		DispatchInbound: func(ctx context.Context, msgCtx *inbound.MsgContext, d gateway.Dispatcher) error {
			return dispatch.DispatchInbound(ctx, msgCtx, d)
		},
	}
	channels.Register(discord.Plugin{})

	plugin := channels.Get(discord.Plugin{}.ID())
	if plugin == nil {
		slog.Error("discord plugin not found")
		os.Exit(1)
	}

	ctx := channels.StartAccountContext{
		Cfg:       cfg,
		AccountID: "default",
		Account:   &discord.DiscordAccount{Token: token},
		Runtime:   rt,
	}

	if err := plugin.StartAccount(ctx); err != nil {
		slog.Error("discord exited", "err", err)
		os.Exit(1)
	}
}
