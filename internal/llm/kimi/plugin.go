package kimi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/openclaw/openclaw-go/internal/llm"
)

const (
	// ProviderID 是 Kimi 插件的唯一标识。
	ProviderID llm.ProviderID = "kimi"
	// DefaultBaseURL 月之暗面 Kimi API 基础地址（兼容 OpenAI SDK 格式）。
	DefaultBaseURL = "https://api.moonshot.cn/v1"
	// DefaultModel 默认模型，可使用 kimi-k2-turbo-preview / kimi-k2-thinking 等。
	DefaultModel = "kimi-k2-turbo-preview"
	// EnvAPIKey 环境变量名，用于读取 Kimi API Key。
	EnvAPIKey = "MOONSHOT_API_KEY"
)

// Plugin 实现 llm.Plugin，接入月之暗面 Kimi 大模型。
type Plugin struct {
	BaseURL string // 为空则用 DefaultBaseURL
	Model   string // 为空则用 DefaultModel
	APIKey  string // 为空则从 EnvAPIKey 环境变量读取
	Client  *http.Client
}

// ID 返回 "kimi"。
func (p *Plugin) ID() llm.ProviderID {
	return ProviderID
}

// Chat 调用 Kimi Chat Completions API，返回助手回复文本。
func (p *Plugin) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = os.Getenv(EnvAPIKey)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("llm/kimi: API key required (set %s or Plugin.APIKey)", EnvAPIKey)
	}

	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	model := req.Model
	if model == "" {
		model = p.Model
	}
	if model == "" {
		model = DefaultModel
	}

	payload := kimiRequest{
		Model:       model,
		Messages:    req.Messages,
		Temperature: 0.6,
		MaxTokens:   2048,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("llm/kimi: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm/kimi: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm/kimi: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm/kimi: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm/kimi: api error status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var kimiResp kimiResponse
	if err := json.Unmarshal(respBody, &kimiResp); err != nil {
		return nil, fmt.Errorf("llm/kimi: unmarshal response: %w", err)
	}
	if len(kimiResp.Choices) == 0 {
		return &llm.ChatResponse{Content: ""}, nil
	}
	content := kimiResp.Choices[0].Message.Content
	return &llm.ChatResponse{Content: content}, nil
}

type kimiRequest struct {
	Model       string       `json:"model"`
	Messages    []llm.Message `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens"`
}

type kimiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
