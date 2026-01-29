# power-ai-framework-v4 高效学习指南（全项目覆盖）

生成时间：2026-01-26

## 1. 目标与学习方式确认

### 你的目标（我理解）
- 系统理解 power-ai-framework-v4 框架的整体结构与关键能力。
- 能快速掌握入口、服务、组件与数据流，便于后续基于此框架开发 AI 应用。
- 学习路径要高效、可执行、覆盖完整项目。

### 学习方式（推荐）
- **先“全景地图”再“关键链路”**：先掌握整体模块与调用链，再逐个深入组件实现。
- **以“可复用模板”驱动理解**：每学一块都要产出可复用的骨架示例（如 Agent 模板）。
- **用“验证任务”做闭环**：每一阶段都有小目标（能跑、能调、能解释）。

> 如果你的目标或方式有偏差，请告诉我，我会调整本指南。

---

## 2. 项目总览（全局结构图）

```
D:\leo\frame_work\power-ai-framework-v4
├─ powerai.go                # 入口：NewAgent / Run
├─ powerai_options.go        # Options + 路由注入
├─ powerai_consts.go         # etcd key / 路由规则
├─ powerai_public.go         # 请求校验 / SSE 构造
├─ powerai_agent.go          # 调用其他 Agent
├─ powerai_db.go             # Postgres 数据层
├─ powerai_model.go          # 模型调用封装
├─ powerai_short_memory.go   # Redis 短期记忆
├─ powerai_minio.go          # MinIO 存储
├─ powerai_weaviate.go       # Weaviate 检索
├─ powerai_milvus.go         # Milvus 检索
├─ powerai_internal.go       # 中间件初始化逻辑
├─ agent_config.go           # 配置管理 + watch
├─ agent_client.go           # 服务注册 + 发现
├─ env/                       # 环境配置
├─ middleware/                # 基础中间件
├─ tools/                     # HTTP 客户端 / LLM 请求
├─ pkg/                       # 工具库
└─ tutorials/                 # 文档与教程
```

---

## 3. 你需要掌握的“关键链路”

### 3.1 启动与服务链路（入口级关键链路）
**最关键路径**：
```
NewAgent -> env.Init -> tools.Init -> initEtcd -> AgentConfig -> AgentClient -> 路由注册
Run -> HTTP Server 启动 -> etcd 注册 + KeepAlive
```

学习重点：
- `powerai.go` 中的启动顺序和依赖初始化
- `agent_client.go` 里的服务注册与发现
- `middleware/server` 里的 Gin 初始化与中间件

---

### 3.2 请求处理链路（业务入口）
**最关键路径**：
```
POST /{agent_code}/send_msg -> handler -> DoValidateAgentRequest -> SSEEvent
```

学习重点：
- `powerai_public.go` 的请求校验与响应构建
- `middleware/server/context.go` 的 SSE 结构和事件协议

---

### 3.3 AI 能力链路（LLM / Embedding / Rerank）
**最关键路径**：
```
GetSystemConfig -> GetSystemLlmConfig -> tools.SyncCallSystemLLM
```

学习重点：
- `powerai_model.go` 的模型配置获取
- `tools/http_llm.go` 对 LLM API 的封装细节

---

### 3.4 记忆与数据链路（Redis / Postgres）
**最关键路径**：
```
GetRedisClient -> CreateShortMemory / GetShortMemory
GetPgSqlClient -> QueryMessage / CreateConversation
```

学习重点：
- `powerai_short_memory.go` 短期记忆数据结构
- `powerai_db.go` 对话与消息模型

---

### 3.5 检索链路（向量库）
**最关键路径**：
```
EmbedTexts -> WeaviateInsertObjects -> WeaviateHybridSearch
```

学习重点：
- `powerai_weaviate.go`、`powerai_milvus.go` 的能力边界

---

## 4. 系统化学习路线（阶段式、可验证）

### 阶段 1：框架启动与路由（1-2 小时）
目标：能解释 NewAgent/Run 的完整流程，能自己注册一个 send_msg 路由。
- 阅读：`powerai.go`, `powerai_options.go`, `middleware/server/server.go`
- 输出：写一个最小 Agent 骨架并能运行

### 阶段 2：请求协议与流式响应（1-2 小时）
目标：能手写一个 send_msg handler，返回 SSE。
- 阅读：`powerai_public.go`, `middleware/server/context.go`
- 输出：实现一个流式回应的 handler

### 阶段 3：配置与服务发现（1-2 小时）
目标：理解 etcd 存储结构和服务注册机制。
- 阅读：`agent_config.go`, `agent_client.go`, `powerai_consts.go`
- 输出：能解释 etcd key 结构

### 阶段 4：数据层（2-3 小时）
目标：理解对话/消息存储结构与调用。
- 阅读：`powerai_db.go`, `middleware/pgsql`
- 输出：演示一个 conversation 查询

### 阶段 5：模型调用（2-3 小时）
目标：理解系统模型配置+调用，能调用 LLM 或 embedding。
- 阅读：`powerai_model.go`, `tools/http_llm.go`
- 输出：演示一个 LLM 调用

### 阶段 6：检索与存储（2-3 小时）
目标：理解向量库与对象存储。
- 阅读：`powerai_weaviate.go`, `powerai_milvus.go`, `powerai_minio.go`

---

## 5. 项目中你可能忽略的关键点（潜在问题）

1. **入口没有 Recovery 中间件**
   - `middleware/server/server.go` 中未启用 `gin.Recovery()`，异常可能导致服务崩溃。

2. **Manifest 是硬性入口参数**
   - `initManifest` 对 code/name/version/description 的校验很严格，部署中常见错误是 manifest 缺字段。

3. **etcd 是核心依赖**
   - 许多功能（配置、服务发现、组件初始化）都依赖 etcd，etcd 不可用会导致系统部分能力失效。

4. **服务注册逻辑不会自动退出**
   - `agent_client.register` 是无限循环注册，如果 etcd 有问题可能导致频繁重试。

5. **配置 watch 可能触发大量数据更新**
   - decision agent 会监听意图配置前缀，数据量大时可能造成内存压力。

6. **默认配置与环境变量覆盖关系**
   - `env.Init()` 默认值和 etcd 获取的值会覆盖环境变量，需要明确配置顺序。

7. **短期记忆结构偏业务化**
   - `powerai_short_memory.go` 中的结构是医疗场景偏重（如病症、科室），不一定适合所有 AI 应用。

8. **模型调用会强行处理 <think> 标签**
   - `tools/http_llm.go` 会过滤 `<think>`，如果你需要完整原始输出要注意。

---

## 6. 建议你先做的两个验证任务

1) **写一个最小 Agent 并跑通 /send_msg**
- 目标：确保你理解入口、路由注册、handler 结构。

2) **调用系统 LLM 配置并返回结果**
- 目标：你能从 config -> model -> output 打通模型调用链。

---

## 7. 下一步（我可以帮你做的事）
- 为你生成一个“最小可运行 Demo 项目”（含启动脚本）
- 为你制作“入口与路由注册逐行注释版”
- 为你整理“etcd key 与配置规范速查表”
- 帮你把当前框架改造成适用于你业务场景的骨架

---

> 如果你确认本学习指南符合预期，我可以按它一步一步带你走；或者你指定先从某一阶段开始。
