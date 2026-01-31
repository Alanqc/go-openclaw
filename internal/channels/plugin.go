package channels

import (
	"github.com/openclaw/openclaw-go/internal/config"
	"github.com/openclaw/openclaw-go/internal/gateway"
)

// ChannelId identifies a channel plugin (e.g. "discord").
type ChannelId string

// StartAccountContext is passed to StartAccount (like TS ChannelGatewayContext).
type StartAccountContext struct {
	Cfg        *config.Config
	AccountID  string
	Account    interface{} // channel-specific account config
	Runtime    *gateway.Runtime
	AbortSignal <-chan struct{}
}

// ChannelPlugin is the interface channel plugins implement (like TS ChannelPlugin).
// Gateway calls StartAccount to run the channel monitor within the process.
type ChannelPlugin interface {
	ID() ChannelId
	// StartAccount runs the channel monitor (e.g. Discord bot). Blocks until abort.
	// When messages arrive, the plugin calls Runtime.DispatchInbound with a Dispatcher.
	StartAccount(ctx StartAccountContext) error
}
