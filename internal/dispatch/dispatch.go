package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/openclaw/openclaw-go/internal/inbound"
)

// Dispatcher sends replies to Discord (or other channels).
type Dispatcher interface {
	SendFinal(ctx context.Context, channelID, text string) error
}

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

// GatewayClient calls the OpenClaw gateway HTTP API (optional).
type GatewayClient struct {
	BaseURL string
	Client  *http.Client
}

// AgentRequest for gateway agent endpoint.
type AgentRequest struct {
	Message   string `json:"message"`
	SessionKey string `json:"sessionKey"`
	Channel   string `json:"channel"`
	AccountID string `json:"accountId"`
	Deliver   bool   `json:"deliver"`
}

// AgentResponse from gateway.
type AgentResponse struct {
	RunID string `json:"runId,omitempty"`
}

// DispatchInbound processes the message and dispatches to agent.
func DispatchInbound(ctx context.Context, msgCtx *inbound.MsgContext, cfg interface{}, dispatcher Dispatcher) error {
	msgCtx.Finalize()

	if msgCtx.BodyForCommands == "" {
		slog.Debug("dispatch: empty body, skip")
		return nil
	}

	// Option 1: Call gateway HTTP API
	if gc, ok := cfg.(*GatewayClient); ok && gc != nil && gc.BaseURL != "" {
		return dispatchViaGateway(ctx, msgCtx, gc, dispatcher)
	}

	// Option 2: Placeholder - log and optionally echo
	slog.Info("dispatch: inbound message",
		"sessionKey", msgCtx.SessionKey,
		"from", msgCtx.From,
		"body", truncateStr(msgCtx.BodyForCommands, 100))

	// For demo: echo back
	reply := fmt.Sprintf("Received: %s (session: %s)", truncateStr(msgCtx.BodyForCommands, 80), msgCtx.SessionKey)
	target := msgCtx.ReplyChannelID
	if target == "" {
		target = msgCtx.OriginatingTo
	}
	if dispatcher != nil && target != "" {
		_ = dispatcher.SendFinal(ctx, target, reply)
	}
	return nil
}

func dispatchViaGateway(ctx context.Context, msgCtx *inbound.MsgContext, gc *GatewayClient, dispatcher Dispatcher) error {
	reqBody := AgentRequest{
		Message:    msgCtx.BodyForCommands,
		SessionKey: msgCtx.SessionKey,
		Channel:    msgCtx.Provider,
		AccountID:  msgCtx.AccountID,
		Deliver:    false, // we'll deliver ourselves
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	client := gc.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, "POST", gc.BaseURL+"/agent", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var agentResp AgentResponse
	_ = json.NewDecoder(resp.Body).Decode(&agentResp)
	slog.Info("dispatch: gateway response", "runId", agentResp.RunID)

	// In real impl, would wait for run completion and get reply
	reply := "Agent request queued (runId: " + agentResp.RunID + "). Full reply delivery requires gateway integration."
	target := msgCtx.ReplyChannelID
	if target == "" {
		target = msgCtx.OriginatingTo
	}
	if dispatcher != nil && target != "" {
		_ = dispatcher.SendFinal(ctx, target, reply)
	}
	return nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
