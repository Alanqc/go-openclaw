package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/openclaw/openclaw-go/internal/inbound"
	"github.com/openclaw/openclaw-go/internal/llm"
)

// Run processes a message and returns the reply text (in-process, no HTTP).
// 若 llmPlugin 非 nil 则调用其 Chat；否则回显占位（便于未配置 LLM 时仍可运行）。
// defaultModel 可选，非空时作为 ChatRequest.Model 传给插件（如 kimi-k2-turbo-preview）。
func Run(ctx context.Context, msgCtx *inbound.MsgContext, llmPlugin llm.Plugin, defaultModel string) (string, error) {
	msgCtx.Finalize()
	if strings.TrimSpace(msgCtx.BodyForCommands) == "" {
		return "", nil
	}
	if llmPlugin != nil {
		req := &llm.ChatRequest{
			Model: defaultModel,
			Messages: []llm.Message{
				{Role: "system", Content: "你是 Kimi，由 Moonshot AI 提供的人工智能助手，你更擅长中文和英文的对话。你会为用户提供安全、有帮助、准确的回答。"},
				{Role: "user", Content: msgCtx.BodyForCommands},
			},
		}
		resp, err := llmPlugin.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("agent llm chat: %w", err)
		}
		return strings.TrimSpace(resp.Content), nil
	}
	// 无 LLM 插件时回显占位
	body := msgCtx.BodyForCommands
	if len(body) > 80 {
		body = body[:80] + "..."
	}
	return fmt.Sprintf("Received: %s (session: %s)", body, msgCtx.SessionKey), nil
}
