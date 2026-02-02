package gateway

import (
	"context"

	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/inbound"
	"github.com/openclaw/openclaw-go/internal/llm"
)

// Runtime is the gateway runtime passed to channel plugins (like TS PluginRuntime).
// Channels use it to dispatch inbound messages and get config.
type Runtime struct {
	Config *config.Config
	// LLM 为当前使用的 LLM 插件，由 main 根据配置注入；nil 时不调用大模型（如 echo 占位）。
	LLM llm.Plugin
	// DispatchInbound is called when a channel receives a message. The dispatcher
	// sends replies back to the originating channel (in-process function call).
	DispatchInbound func(ctx context.Context, msgCtx *inbound.MsgContext, dispatcher Dispatcher) error
}

// Dispatcher sends replies to a channel. Implemented per-channel (e.g. DiscordDispatcher).
type Dispatcher interface {
	SendFinal(ctx context.Context, channelID, text string) error
}
