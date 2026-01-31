# OpenClaw Go

Go 语言版本的 OpenClaw Discord 消息处理流程，从收到 Discord 消息到调用 Agent 的完整调用栈改写。

## 项目结构

```
go-openclaw/
├── cmd/openclaw-go/main.go   # 入口
├── internal/
│   ├── config/               # 配置加载 (对应 src/config)
│   ├── routing/              # 路由解析 (对应 src/routing)
│   ├── inbound/              # 入站上下文 (对应 src/auto-reply/reply/inbound-context)
│   ├── discord/              # Discord 监听、预检、处理 (对应 src/discord/monitor)
│   └── dispatch/             # Agent 分发 (对应 src/auto-reply/dispatch)
├── go.mod
└── README.md
```

## 调用栈（与 TypeScript 版对应）

```
Carbon/discordgo MessageCreate 事件
  → MessageHandler.Handle
    → Preflight (preflightDiscordMessage)
      - 过滤 bot、校验 DM/Guild 开关
      - resolveAgentRoute 解析 agentId、sessionKey
    → ProcessMessage (processDiscordMessage)
      - 构建 FinalizedMsgContext
      - createReplyDispatcher
    → DispatchInbound (dispatchInboundMessage)
      → dispatchReplyFromConfig
        → getReplyFromConfig (占位实现)
          - 可选：HTTP 调用 OpenClaw Gateway /agent
          - 或：本地占位回复
```

## 运行

```bash
cd go-openclaw
go mod tidy
go run ./cmd/openclaw-go -token "YOUR_DISCORD_BOT_TOKEN"
```

环境变量：
- `DISCORD_TOKEN`：Discord Bot Token
- `OPENCLAW_CONFIG`：配置文件路径（默认 `~/.openclaw/openclaw.yaml`）

## 配置示例

```yaml
# openclaw.yaml
agents:
  defaults:
    default_model: claude-sonnet
  list:
    - id: main

bindings:
  - agent_id: main
    match:
      channel: discord
      account_id: "*"

session:
  dm_scope: main
```

## 与 TypeScript 版差异

- **Agent 执行**：当前为占位实现，可配置 `GatewayClient` 指向 OpenClaw Gateway 的 HTTP API 做真实 LLM 调用
- **Debounce**：未实现入站防抖
- **Ack 表情**：未实现
- **Typing 指示**：未实现
- **媒体处理**：未实现附件/图片解析
- **Thread 支持**：未实现 Forum/Thread 逻辑

## 依赖

- [discordgo](https://github.com/bwmarrin/discordgo) - Discord Go 库
- [yaml.v3](https://github.com/go-yaml/yaml) - 配置解析
