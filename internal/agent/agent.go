package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/openclaw/openclaw-go/internal/inbound"
)

// Run processes a message and returns the reply text (in-process, no HTTP).
// This mirrors the TS agentCommand flow; for now it echoes as placeholder.
func Run(ctx context.Context, msgCtx *inbound.MsgContext) (string, error) {
	msgCtx.Finalize()
	if strings.TrimSpace(msgCtx.BodyForCommands) == "" {
		return "", nil
	}
	// Placeholder: echo back. Will be replaced with LLM call later.
	body := msgCtx.BodyForCommands
	if len(body) > 80 {
		body = body[:80] + "..."
	}
	return fmt.Sprintf("Received: %s (session: %s)", body, msgCtx.SessionKey), nil
}
