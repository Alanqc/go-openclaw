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
	"github.com/openclaw/openclaw-go/internal/llm"
	"github.com/openclaw/openclaw-go/internal/llm/kimi"
)

func main() {
	tokenFlag := flag.String("token", "", "Discord bot token (or DISCORD_TOKEN env)")
	configPath := flag.String("config", "", "Config file path (or OPENCLAW_CONFIG env)")
	flag.Parse()

	// 本地认证文件：必须存在，否则提示并退出（该文件不提交、不 push）
	secretsPath := config.ResolveSecretsPath()
	if err := config.LoadSecrets(secretsPath); err != nil {
		if os.IsNotExist(err) {
			slog.Error("secrets file not found",
				"path", secretsPath,
				"hint", "create goopenclaw.secrets from goopenclaw.secrets.example and fill in DISCORD_TOKEN, MOONSHOT_API_KEY etc.")
			os.Exit(1)
		}
		slog.Error("load secrets", "path", secretsPath, "err", err)
		os.Exit(1)
	}

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("DISCORD_TOKEN")
	}
	if token == "" {
		slog.Error("discord token required (--token or DISCORD_TOKEN, or set in goopenclaw.secrets)")
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

	// 注册 LLM 插件（与 channel 插件解耦，后续换大模型只需换插件）
	llm.Register(&kimi.Plugin{})

	var llmPlugin llm.Plugin
	if pid := cfg.Agents.Defaults.LLMProvider; pid != "" {
		llmPlugin = llm.Get(llm.ProviderID(pid))
		if llmPlugin == nil {
			slog.Warn("llm provider not found, agent will echo only", "provider", pid)
		}
	}
	defaultModel := ""
	if cfg != nil {
		defaultModel = cfg.Agents.Defaults.DefaultModel
	}

	// Gateway as main process: create runtime, register plugins, start channels.
	rt := &gateway.Runtime{
		Config: cfg,
		LLM:    llmPlugin,
		DispatchInbound: func(ctx context.Context, msgCtx *inbound.MsgContext, d gateway.Dispatcher) error {
			return dispatch.DispatchInbound(ctx, msgCtx, d, llmPlugin, defaultModel)
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
