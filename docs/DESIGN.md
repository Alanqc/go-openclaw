# OpenClaw Go 设计：多通道、多端与 Mac 控制

本文档合并多通道/多端调用设计，以及“控制 Mac 应用”的两种实现方式（含 OpenClaw 做法调研与对比）。

---

## 一、设计原则与当前实现

### 1.1 设计原则

- **安全优先**：不新增对公网暴露的入口；任何 HTTP 仅绑定 localhost，且必须鉴权。
- **与现有架构一致**：以 Gateway Runtime + Channel 插件 + DispatchInbound 为核心，多通道共用同一 Agent/LLM。
- **可演进**：先做进程内多通道（零新端口），再按需增加“仅本机 + 鉴权”的 HTTP 网关或本地 IPC 的 Mac 控制。

### 1.2 当前实现（单通道、无端口）

- **单进程**：main → 注册 LLM + Discord 插件 → `plugin.StartAccount` 在进程内跑 Discord bot。
- **调用链**：Discord 消息 → Preflight → ProcessMessage → `Runtime.DispatchInbound` → `agent.Run` → Dispatcher 回写 Discord。
- **无本地监听**：只有出站连接（Discord、Kimi API），无 18789 等端口。

---

## 二、OpenClaw 控制 Mac 应用的方式（调研结论）

OpenClaw 通过 **Agent + Node 工具** 控制 Mac 应用，与 **HTTP/WebSocket Gateway（默认 18789）** 设计直接相关。

### 2.1 角色分工

| 角色 | 作用 |
|------|------|
| **Gateway**（默认端口 **18789**） | 跑 Agent、会话、多通道入口（Discord/Telegram/Web）；管理所有 **Nodes**；提供 HTTP/WebSocket。 |
| **macOS Companion App**（菜单栏应用） | 负责 TCC 权限（通知、无障碍、录屏、麦克风、语音识别、AppleScript）；以 **Node** 身份连到 Gateway；暴露 Mac 专属能力给 Agent。 |

### 2.2 Agent 如何“控制 Mac”

Agent 在 Gateway 上跑，不直接调 Mac 应用 API，而是调用 **Node 工具**，由 Mac 上的 Node 服务/App 执行：

| 能力 | 工具示例 | 说明 |
|------|----------|------|
| 执行系统命令 | `system.run` | 在 Mac 上执行 shell；审批与执行在 App 内（Exec approvals）。 |
| 通知 | `system.notify` | 系统通知。 |
| 画布/UI | `canvas.present`, `canvas.navigate`, `canvas.eval`, `canvas.snapshot` | 展示/操作画布。 |
| 摄像头 | `camera.snap`, `camera.clip` | 拍照/录像。 |
| 录屏 | `screen.record` | 屏幕录制。 |
| 浏览器等 | Browser 工具、Chrome Extension | 浏览器自动化。 |

即：**“控制很多 Mac 应用” = Agent 通过上述 Node 工具在你这台 Mac（Node）上执行操作**。

### 2.3 通信路径（与 18789 的关系）

- **Mac Node 服务**（无头）↔ **Gateway**：通过 **WebSocket** 连接 Gateway 的 **18789**（或配置端口）。
- **Node 服务** ↔ **macOS App**（有 UI、TCC）：通过 **本地 Unix Socket**；例如 `system.run` 请求从 Gateway 经 WebSocket 到 Node 服务，再经 UDS 转给 App 执行、弹审批、返回结果。

整体链路：

```
用户消息 (Discord/Telegram/Web) → Gateway (HTTP/WS :18789)
  → Agent 执行
  → 需要 Mac 能力时 node.invoke(system.run / canvas.* / …)
  → Gateway 经 WebSocket 发给 Mac Node
  → Mac Node 经 Unix Socket 转给 App → 执行 → 结果回传
```

**结论**：OpenClaw 能控制 Mac 应用，依赖 **Gateway（18789 上的 HTTP/WebSocket）** + **Mac 作为 Node 连上 Gateway** + **Node 与 App 间本地 IPC（UDS）**。没有 Gateway 端口，Mac 就无法作为 Node 被 Agent 控制。

---

## 三、多通道、多端调用（不涉及 Mac 控制时）

### 3.1 目标

- **多通道**：Discord、Telegram、Web 等都能把用户消息交给同一 Agent。
- **多端**：不同端（不同进程/服务）在需要时调用同一套 Agent，而不是每个端各自连 LLM。

### 3.2 方案：进程内多通道（推荐优先，零新端口）

- 沿用“Channel 插件 + Runtime.DispatchInbound”：一个进程、一个 Runtime，注册多个 Channel 插件（discord、telegram、webhook 等）。
- 每个插件收到消息后构造 `MsgContext`，调用同一 `DispatchInbound`；各插件实现自己的 Dispatcher 回写对应通道。
- **安全**：无本地监听端口（除非某 channel 内部必须起 HTTP 且仅 127.0.0.1），所有入口在同一进程内。

### 3.3 可选：仅本机 + 鉴权的 HTTP 网关

- 当存在**无法纳入同一进程的调用端**（如独立 Web UI、另一台机器上的服务）时，可增加**可选** HTTP 网关：仅绑定 **127.0.0.1**（如 18789），**禁止** 0.0.0.0，强制鉴权（如 X-API-Key / Bearer）。
- 接口形态：如 `POST /inbound`，JSON 对应 `MsgContext`，网关构造上下文后调用现有 `DispatchInbound`，回复写回响应。
- 配置建议：`gateway.http_enabled`（默认 false）、`gateway.http_listen`（仅允许 127.0.0.1）、`gateway.http_api_key` 或环境变量。

---

## 四、控制 Mac 应用的两种实现方式

在“需要 Agent 控制 Mac 应用”的前提下，有两种实现方式：**方式 A 使用本地 Gateway（类似 OpenClaw）**，**方式 B 不使用本地 Gateway，仅用本地 IPC**。

### 4.1 方式 A：本地 HTTP/WebSocket Gateway + Mac 作为 Node

- **架构**：本机起 Gateway（如 127.0.0.1:18789，HTTP/WebSocket），Mac 上运行 **Node 服务** 通过 WebSocket 连上该 Gateway；Node 与 **macOS 小助手/App** 之间用 **Unix Socket（或 XPC）** 通信，小助手具备 TCC、执行 `system.run`、通知、Canvas 等。
- **流程**：用户消息 → Gateway → Agent → 需要 Mac 能力时 `node.invoke` → Gateway 经 WebSocket 下发给 Mac Node → Node 经 UDS 转给小助手执行 → 结果回传。
- **特点**：
  - 与 OpenClaw 架构一致，便于对齐文档与生态。
  - 多端/多机可共用一个 Gateway（例如 Web UI、另一台机器上的服务也连 18789）。
  - **有本地端口**：必须严格 127.0.0.1 + 鉴权，避免误绑 0.0.0.0。
- **安全要点**：Gateway 只绑 127.0.0.1；强制鉴权；Node 与 App 间 UDS + 校验对端 UID/Token。

### 4.2 方式 B：仅本地 IPC（无 Gateway 端口）

- **架构**：**不开放**任何 TCP 端口。Go 进程（openclaw-go）与一台 **本机小助手**（Mac 原生进程）通过 **Unix Domain Socket（UDS）** 或 **XPC** 通信；需要“控制 Mac”时，Go 经 UDS 发请求给小助手，小助手执行 `system.run`、通知、AppleScript 等后经 UDS 回传。
- **流程**：用户消息 → Go 进程（出站连接）→ Agent 执行 → 若需在 Mac 上执行 → Go 通过 UDS 发请求给本机小助手 → 小助手执行并返回 → 继续 Agent → 回复用户。
- **特点**：
  - **零网络端口**，无 18789、无 HTTP，攻击面更小。
  - 仅本机进程间通信，小助手可校验对端 UID、Token/HMAC。
  - 适合**单机**：Go 与小助手同在这台 Mac 上。
- **安全要点**：小助手只监听 UDS/XPC；Go 与小助手间鉴权（token/HMAC）；`system.run` 做审批/白名单（如 `~/.openclaw/exec-approvals.json`）。

---

## 五、两种方式对比

| 对比项 | 方式 A：本地 Gateway + Node | 方式 B：仅本地 IPC |
|--------|-----------------------------|---------------------|
| **网络端口** | 有（127.0.0.1:18789 等），需严格绑定 + 鉴权 | **无** |
| **暴露面** | 本机端口，存在误绑/误暴露风险 | 仅本机进程，无网络入口 |
| **控制 Mac 的路径** | Gateway → WebSocket → Mac Node → UDS → 小助手/App | Go → UDS → 本机小助手 |
| **与 OpenClaw 一致性** | 高（Gateway + Node 模型一致） | 低（无 Gateway，无 Node 概念） |
| **多端/多机** | 支持（Web UI、他机服务可连同一 Gateway） | 仅单机（Go 与助手同机） |
| **实现复杂度** | 较高（Gateway 服务 + Node 协议 + UDS） | 较低（Go ↔ 小助手 UDS 协议即可） |
| **适用场景** | 需要多端共用一个 Gateway、或希望与 OpenClaw 架构对齐 | 单机使用、优先零端口与最小攻击面 |

---

## 六、实施建议

1. **多通道、多端（不涉及 Mac 控制）**  
   - 优先做进程内多 Channel 插件（零新端口）；按需再增加可选“仅本机 + 鉴权”的 HTTP 网关。

2. **需要控制 Mac 时**  
   - **若希望与 OpenClaw 一致、或多端/多机共用**：采用 **方式 A**（本地 Gateway + Mac Node），严格 127.0.0.1 + 鉴权。  
   - **若仅单机、且不希望开放任何端口**：采用 **方式 B**（Go ↔ 本机小助手 UDS），无 Gateway、无 18789。

3. **配置与文档**  
   - 在配置与 README 中明确：禁止 0.0.0.0、必须鉴权；并说明两种 Mac 控制方式的区别与适用场景。

这样既覆盖多通道/多端调用，又明确 OpenClaw 控制 Mac 的方式，以及两种实现方式及其对比，便于选型与落地。
