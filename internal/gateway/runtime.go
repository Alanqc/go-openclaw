package gateway

import (
	"context"

	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

// Runtime is the gateway runtime passed to channel plugins (like TS PluginRuntime).
// Channels use it to dispatch inbound messages and get config.
type Runtime struct {
	Config *config.Config
	// DispatchInbound is called when a channel receives a message. The dispatcher
	// sends replies back to the originating channel (in-process function call).
	DispatchInbound func(ctx context.Context, msgCtx *inbound.MsgContext, dispatcher Dispatcher) error
}

// Dispatcher sends replies to a channel. Implemented per-channel (e.g. DiscordDispatcher).
type Dispatcher interface {
	SendFinal(ctx context.Context, channelID, text string) error
}
