# OpenClaw Go

Go 语言版本的 OpenClaw，与 TS 版保持相同架构：**Gateway 为主进程，Discord 作为 channel 插件**，channel、agent、routing 等均在同一进程内通过函数调用完成。

## 项目结构

```
go-openclaw/
├── cmd/openclaw-go/main.go   # 入口：启动 Gateway，注册并加载 Discord 插件
├── internal/
│   ├── gateway/              # Gateway 主进程运行时 (对应 src/gateway)
│   ├── channels/             # Channel 插件接口与注册 (对应 src/channels/plugins)
│   │   └── discord/          # Discord channel 插件
│   ├── config/               # 配置加载 (对应 src/config)
│   ├── routing/              # 路由解析 (对应 src/routing)
│   ├── inbound/              # 入站上下文 (对应 src/auto-reply/reply/inbound-context)
│   ├── discord/              # Discord 监听、预检、处理 (对应 src/discord/monitor)
│   ├── dispatch/             # Agent 分发 (对应 src/auto-reply/dispatch)
│   └── agent/                # Agent 执行 (对应 src/commands/agent)
├── go.mod
└── README.md
```

## 架构（与 TypeScript 版一致）

- **Gateway**：主进程，创建 Runtime（Config + DispatchInbound），管理 channel 插件
- **Channel 插件**：实现 `ChannelPlugin` 接口，`StartAccount` 在进程内运行（如 Discord bot）
- **耦合方式**：同一进程，函数调用，无 HTTP 依赖

## 调用栈

```
main → Gateway 启动
  → 注册 Discord 插件
  → plugin.StartAccount (进程内运行 Discord bot)

Discord MessageCreate 事件
  → MessageHandler.Handle
    → Preflight (过滤 bot、校验 DM/Guild、resolveAgentRoute)
    → ProcessMessage
      → Runtime.DispatchInbound (函数调用)
        → dispatch.DispatchInbound
          → agent.Run (进程内，占位 echo)
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

- **Agent 执行**：当前为占位实现（echo），后续可接入 LLM
- **Debounce**：未实现入站防抖
- **Ack 表情**：未实现
- **Typing 指示**：未实现
- **媒体处理**：未实现附件/图片解析
- **Thread 支持**：未实现 Forum/Thread 逻辑

## 依赖

- [discordgo](https://github.com/bwmarrin/discordgo) - Discord Go 库
- [yaml.v3](https://github.com/go-yaml/yaml) - 配置解析
