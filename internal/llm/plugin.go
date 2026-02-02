package llm

import "context"

// ProviderID 标识一个 LLM 插件（如 "kimi"、"openai"）。
type ProviderID string

// Message 表示单条对话消息。
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatRequest 请求 LLM 完成一轮对话。
type ChatRequest struct {
	Model    string    `json:"model,omitempty"`    // 可选，不填则用插件默认
	Messages []Message `json:"messages"`
}

// ChatResponse LLM 回复。
type ChatResponse struct {
	Content string `json:"content"`
}

// Plugin 是 LLM 插件接口。与 channel 插件并列，互不耦合；后续切换大模型只需换用不同插件。
type Plugin interface {
	ID() ProviderID
	// Chat 根据消息列表生成回复。由 agent 调用，不依赖具体 HTTP/SDK 实现。
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
