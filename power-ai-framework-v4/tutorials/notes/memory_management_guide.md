# 记忆管理功能指南

## 概述

Power AI Framework V4 提供了一套完整的对话记忆管理系统，支持短期记忆、长期记忆和智能摘要功能，确保对话上下文的高效管理和合理利用。

## 核心概念

### 1. 记忆模式

系统支持两种记忆模式：

- **FULL_HISTORY（全历史模式）**：存储完整的对话历史，适用于对话较短的场景
- **SUMMARY_N（摘要+最近N轮模式）**：存储历史摘要和最近N轮对话，适用于长对话场景

### 2. Token 管理

- **默认阈值比例**：0.75（75%）
- **默认保留轮数**：8轮
- **默认模型上下文窗口**：16000 tokens

当对话内容超过模型上下文窗口的75%时，系统会自动触发摘要机制。

### 3. Checkpoint 机制

系统通过 Checkpoint 机制实现对话历史的分段管理：
- 每个 Checkpoint 包含历史摘要和最近N轮对话
- Checkpoint 作为特殊消息存储在数据库中
- 支持从指定 Checkpoint 开始查询后续对话

## 数据结构

### SessionValue（会话状态）

```go
type SessionValue struct {
    Meta           *MetaInfo       // 元信息
    FlowContext    *FlowContext    // 流程上下文
    MessageContext *MessageContext // 消息上下文
    GlobalState    *GlobalState    // 全局状态
    UserSnapshot   *UserProfile    // 用户快照
}
```

#### MetaInfo
- `ConversationID`：会话唯一标识
- `UserID`：用户ID
- `UpdatedAt`：更新时间戳

#### FlowContext
- `CurrentAgentKey`：当前智能体标识
- `LastBotMessage`：最后一条机器人消息
- `TurnCount`：对话轮数

#### MessageContext
- `Summary`：历史摘要
- `WindowMessages`：最近N轮消息窗口
- `Mode`：记忆模式（FULL_HISTORY/SUMMARY_N）
- `CheckpointMessageID`：当前 Checkpoint 消息ID

#### GlobalState
- `Shared`/`Entities`：共享实体（症状、疾病、科室、医生、意图等）
- `AgentSlots`：各智能体私有状态
- `CurrentIntent`：当前意图
- `PendingAction`：待处理操作

#### UserProfile
- `UserID`：用户唯一标识
- `Name`：用户称呼
- `Allergies`：过敏史（安全红线数据）
- `ChronicDiseases`：慢病史（医疗背景数据）
- `SurgeryHistory`：手术史
- `Preferences`：就医偏好

## 核心功能

### 1. 查询记忆上下文

**函数**：`QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error)`

**功能**：根据会话ID查询并构建适合当前对话的记忆上下文

**参数**：
- `ConversationID`：会话ID（必填）
- `PatientID`：患者ID
- `Query`：当前用户查询（用于计算Token）
- `TokenThresholdRatio`：Token阈值比例（默认0.75）
- `RecentTurns`：保留轮数（默认8）
- `ModelContextWindow`：模型上下文窗口（默认16000）

**返回**：
- `History`：最终返回的对话历史
- `FullHistory`：完整对话历史
- `Mode`：当前使用的记忆模式
- `EstimatedTokens`：预估Token数量
- `TokenRatio`：Token占用比例
- `ShouldCheckpointSummary`：是否需要触发摘要

**示例**：
```go
req := &MemoryQueryRequest{
    ConversationID: "conv_123",
    Query: "我最近感觉头痛",
    TokenThresholdRatio: 0.75,
    RecentTurns: 8,
    ModelContextWindow: 16000,
}
ctx, err := app.QueryMemoryContext(req)
```

### 2. 写入对话轮次

**函数**：`WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error)`

**功能**：记录一次对话轮次，更新短期记忆

**参数**：
- `ConversationID`：会话ID（必填）
- `UserID`：用户ID
- `AgentCode`：智能体代码
- `UserQuery`：用户查询
- `AgentResponse`：智能体响应

**返回**：
- `ConversationID`：会话ID
- `Mode`：当前记忆模式
- `UpdatedAt`：更新时间戳

**示例**：
```go
req := &MemoryWriteRequest{
    ConversationID: "conv_123",
    UserID: "user_456",
    AgentCode: "triage_agent",
    UserQuery: "我头痛",
    AgentResponse: "请问您头痛持续多久了？",
}
result, err := app.WriteTurn(req)
```

### 3. 创建 Checkpoint（摘要）

**函数**：`CheckpointShortMemory(conversationID, summary string, recentTurns int) error`

**功能**：将当前对话历史压缩为 Checkpoint，包含摘要和最近N轮对话

**参数**：
- `conversationID`：会话ID
- `summary`：历史摘要
- `recentTurns`：保留轮数

**执行步骤**：
1. 构建"摘要+最近N轮"内容
2. 插入 Checkpoint 消息到数据库
3. 更新 Session 状态

**示例**：
```go
err := app.CheckpointShortMemory("conv_123", "用户咨询头痛问题，持续3天", 8)
```

### 4. 结束会话

**函数**：`FinalizeSessionMemory(req *SessionFinalizeRequest) error`

**功能**：会话结束时创建最终 Checkpoint

**参数**：
- `ConversationID`：会话ID
- `Summary`：会话摘要
- `RecentTurns`：保留轮数

### 5. 实体和偏好管理

**函数**：
- `UpsertFacts(req *FactUpsertRequest) error`：更新医疗事实
- `UpsertPreferences(req *PreferenceUpsertRequest) error`：更新用户偏好

## 辅助功能

### 1. Token 估算

**函数**：`estimateTokenCount(text string) int`

**策略**：按字符数/4估算Token数量（适用于中文场景）

### 2. 历史构建

**函数**：
- `buildHistoryFromAIMessages(messages []*AIMessage) string`：从数据库消息构建历史
- `composeSummaryAndRecent(session *SessionValue) string`：组合摘要和最近消息
- `buildRecentMessages(messages []*AIMessage, recentTurns int) []*Message`：提取最近N轮消息

### 3. 答案提取

**函数**：`extractAgentAnswer(answer string) string`

**功能**：从智能体响应中提取纯文本答案（去除JSON包装）

## 使用场景

### 场景1：新对话开始

```go
// 1. 创建短期记忆
err := app.CreateShortMemory(&server.AgentRequest{
    ConversationId: "conv_123",
    UserId: "user_456",
})

// 2. 查询记忆上下文（首次查询为空）
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv_123",
    Query: "我头痛",
})
```

### 场景2：对话进行中

```go
// 1. 查询记忆上下文
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv_123",
    Query: "持续3天了",
})

// 2. 使用 ctx.History 作为对话历史
// 3. 检查是否需要创建 Checkpoint
if ctx.ShouldCheckpointSummary {
    // 生成摘要并创建 Checkpoint
    app.CheckpointShortMemory("conv_123", "用户咨询头痛问题", 8)
}

// 4. 记录对话轮次
app.WriteTurn(&MemoryWriteRequest{
    ConversationID: "conv_123",
    UserQuery: "持续3天了",
    AgentResponse: "请问还有其他症状吗？",
})
```

### 场景3：长对话管理

```go
// 当 Token 占用超过阈值时
if ctx.TokenRatio >= 0.75 {
    // 生成摘要（可调用 LLM）
    summary := generateSummary(ctx.FullHistory)
    
    // 创建 Checkpoint
    app.CheckpointShortMemory("conv_123", summary, 8)
    
    // 重新查询（现在会使用 SUMMARY_N 模式）
    ctx, _ = app.QueryMemoryContext(&MemoryQueryRequest{
        ConversationID: "conv_123",
        Query: "还有其他症状吗？",
    })
}
```

### 场景4：会话结束

```go
// 生成最终摘要
finalSummary := generateSummary(ctx.FullHistory)

// 结束会话
app.FinalizeSessionMemory(&SessionFinalizeRequest{
    ConversationID: "conv_123",
    Summary: finalSummary,
    RecentTurns: 8,
})
```

## 最佳实践

### 1. Token 管理

- 根据模型实际能力调整 `ModelContextWindow`
- 对于长文档场景，适当降低 `TokenThresholdRatio`
- 定期清理过期的 Checkpoint 消息

### 2. 摘要生成

- 摘要应包含关键信息（症状、诊断、建议等）
- 保留必要的上下文信息以便后续对话
- 避免摘要过于冗长

### 3. 用户画像管理

- 及时更新过敏史、慢病史等安全红线数据
- 定期同步最新的用户信息
- 谨慎处理敏感数据

### 4. 状态管理

- 合理使用 `GlobalState` 的共享区和私有区
- 明确各智能体的数据访问边界
- 避免状态冲突和覆盖

## 注意事项

1. **Redis 过期**：短期记忆默认30分钟过期，需要定期刷新
2. **数据一致性**：确保 Session 和数据库消息的一致性
3. **并发控制**：同一会话的并发操作需要加锁保护
4. **错误处理**：所有 Redis 和数据库操作都需要错误处理
5. **性能优化**：避免频繁的 Redis 读写，合理使用缓存

## 扩展方向

- 支持向量存储的长期记忆
- 支持跨会话的记忆共享
- 支持记忆的增量更新和合并
- 支持多模态记忆（图片、语音等）
- 支持记忆的隐私保护和加密存储
