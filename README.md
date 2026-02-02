# OpenClaw Go

Go 语言版本的 OpenClaw，与 TS 版保持相同架构：**Gateway 为主进程，Discord 作为 channel 插件**，channel、agent、routing 等均在同一进程内通过函数调用完成。

## 项目结构

```
go-openclaw/
├── cmd/openclaw-go/main.go   # 入口：注册 LLM/Channel 插件，启动 Gateway
├── internal/
│   ├── gateway/              # Gateway 主进程运行时 (对应 src/gateway)
│   ├── channels/             # Channel 插件接口与注册 (对应 src/channels/plugins)
│   │   └── discord/          # Discord channel 插件
│   ├── llm/                  # LLM 插件接口与注册（与 channel 解耦，可切换大模型）
│   │   ├── plugin.go        # Plugin 接口、ChatRequest/ChatResponse
│   │   ├── registry.go      # Register/Get/List
│   │   └── kimi/            # Kimi 大模型插件 (月之暗面 API)
│   │       └── plugin.go
│   ├── config/               # 配置加载 (对应 src/config)
│   ├── routing/              # 路由解析 (对应 src/routing)
│   ├── inbound/              # 入站上下文 (对应 src/auto-reply/reply/inbound-context)
│   ├── discord/              # Discord 监听、预检、处理 (对应 src/discord/monitor)
│   ├── dispatch/             # Agent 分发 (对应 src/auto-reply/dispatch)
│   └── agent/                # Agent 执行：调用 LLM 插件或回显占位 (对应 src/commands/agent)
├── go.mod
└── README.md
```

## 架构（与 TypeScript 版一致，并增加 LLM 插件）

- **Gateway**：主进程，创建 Runtime（Config + LLM + DispatchInbound），管理 channel 插件
- **Channel 插件**：实现 `ChannelPlugin` 接口，`StartAccount` 在进程内运行（如 Discord bot）
- **LLM 插件**：实现 `llm.Plugin` 接口（`ID()` + `Chat(ctx, req)`），与 channel 解耦；配置中通过 `llm_provider` 指定（如 `kimi`），后续换其他大模型只需新增插件并改配置
- **耦合方式**：同一进程，函数调用，无 HTTP 依赖（仅 LLM 插件内部调用外部 API）

## 调用栈

```
main → 注册 LLM 插件 (如 kimi) → 按配置选择 llm_provider → Gateway 启动
  → 注册 Discord 插件
  → plugin.StartAccount (进程内运行 Discord bot)

Discord MessageCreate 事件
  → MessageHandler.Handle
    → Preflight (过滤 bot、校验 DM/Guild、resolveAgentRoute)
    → ProcessMessage
      → Runtime.DispatchInbound (函数调用)
        → dispatch.DispatchInbound(..., rt.LLM, defaultModel)
          → agent.Run(..., llmPlugin, defaultModel)
            → llmPlugin.Chat(ctx, req)  # 若配置了 llm_provider 则调用 Kimi 等
          → Dispatcher.SendFinal → Discord 回复
```

## 运行

```bash
cd go-openclaw
go mod tidy
go build -o openclaw-go ./cmd/openclaw-go
./openclaw-go --token "YOUR_DISCORD_BOT_TOKEN"
```

环境变量：
- `DISCORD_TOKEN`：Discord Bot Token
- `OPENCLAW_CONFIG`：配置文件路径（默认 `~/.openclaw/openclaw.yaml`）
- `MOONSHOT_API_KEY`：使用 Kimi 插件时必填，月之暗面 API Key（[平台](https://platform.moonshot.cn) 创建）

## 配置示例

```yaml
# openclaw.yaml
agents:
  defaults:
    default_model: kimi-k2-turbo-preview   # 可选：kimi-k2-thinking 等
    llm_provider: kimi                     # LLM 插件 id，空则不调用大模型（仅 echo）
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

## 切换大模型（插件模式）

- 当前内置 **Kimi** 插件（`internal/llm/kimi`），配置 `llm_provider: kimi` 并设置 `MOONSHOT_API_KEY` 即可使用。
- 后续接入其他大模型（如 OpenAI、Claude 等）：在 `internal/llm/` 下新增目录实现 `llm.Plugin`（`ID()` + `Chat(ctx, req)`），在 `main` 中 `llm.Register(新插件)`，配置里将 `llm_provider` 改为新插件 id 即可，无需改 agent/dispatch 逻辑。

## 与 TypeScript 版差异

- **Agent 执行**：已通过 LLM 插件接入 Kimi；未配置 `llm_provider` 时为 echo 占位
- **Debounce**：未实现入站防抖
- **Ack 表情**：未实现
- **Typing 指示**：未实现
- **媒体处理**：未实现附件/图片解析
- **Thread 支持**：未实现 Forum/Thread 逻辑

## 依赖

- [discordgo](https://github.com/bwmarrin/discordgo) - Discord Go 库
- [yaml.v3](https://github.com/go-yaml/yaml) - 配置解析
