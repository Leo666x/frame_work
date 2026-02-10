# 短期记忆开发指南

> **目标读者**: 智能体开发者
> **文档版本**: v1.0
> **更新时间**: 2026-01-26

## 📋 目录

1. [核心概念](#核心概念)
2. [数据结构详解](#数据结构详解)
3. [核心API接口](#核心api接口)
4. [完整使用流程](#完整使用流程)
5. [智能体对接示例](#智能体对接示例)
6. [最佳实践](#最佳实践)
7. [常见问题](#常见问题)

---

## 核心概念

### 1.1 记忆管理职责

短期记忆系统负责在单次对话会话中管理和提供对话上下文，主要职责包括：

- **上下文查询**: 根据会话ID获取适合当前对话的历史上下文
- **Token管理**: 智能控制对话历史长度，避免超过模型上下文窗口
- **模式切换**: 自动在"全历史"和"摘要+最近N轮"模式间切换
- **状态管理**: 维护会话状态、用户画像和智能体间共享状态

### 1.2 双模式机制

#### FULL_HISTORY 模式（全历史模式）
- **适用场景**: 对话初期（token占用 < 75%）
- **返回内容**: 从checkpoint或会话开始的所有完整消息
- **优势**: 保持对话完整性，无信息丢失
- **劣势**: 长对话时token占用高

#### SUMMARY_N 模式（摘要+最近N轮模式）
- **适用场景**: 长对话（token占用 ≥ 75%）
- **返回内容**: 历史摘要 + 最近N轮对话（默认8轮）
- **优势**: 大幅降低token占用
- **劣势**: 历史细节被摘要压缩

### 1.3 Checkpoint 机制

Checkpoint是对话历史的分界点，实现增量式记忆管理：

**核心特性**:
- 每个 checkpoint 包含"历史摘要 + 最近N轮对话"
- Checkpoint 作为特殊消息存储在数据库（query = "[MEMORY_CHECKPOINT]"）
- Checkpoint 之后的消息范围更小，查询性能更高
- 支持多次 checkpoint，实现累积式摘要

**工作原理**:
```
对话历史: [msg1, msg2, ..., msg10]
               ↑
         checkpoint_001 (摘要+最近8轮)
         
查询时会返回:
- 摘要: "用户咨询头痛问题..."
- 最近8轮: msg3-msg10
```

### 1.4 Token 阈值管理

**默认配置**:
```go
defaultMemoryTokenThresholdRatio = 0.75  // 75% 阈值
defaultRecentTurns = 8                    // 保留8轮
defaultModelContextWindow = 16000          // 16000 tokens
```

**触发逻辑**:
```go
// 1. 计算当前历史 + 新查询的 token 占用率
tokenRatio = estimatedTokens / contextWindow

// 2. 无论什么模式，只要超过阈值就触发摘要
shouldCheckpoint = tokenRatio >= threshold
```

---

## 数据结构详解

### 2.1 SessionValue (会话状态)

**存储位置**: Redis
**Key格式**: `short_term_memory:session:{conversation_id}`
**过期时间**: 30分钟（1800秒）

```go
type SessionValue struct {
    Meta           *MetaInfo       // 元信息
    FlowContext    *FlowContext    // 流程上下文
    MessageContext *MessageContext // 消息上下文（核心）
    GlobalState    *GlobalState    // 全局共享状态
    UserSnapshot   *UserProfile    // 用户快照
}
```

#### 2.1.1 MetaInfo (元信息)

```go
type MetaInfo struct {
    ConversationID string  // 会话唯一标识
    UserID         string  // 用户ID
    UpdatedAt      int64   // 最后更新时间戳（Unix时间戳）
}
```

**作用**:
- 唯一标识会话
- 记录会话活跃度
- 用于Redis Key的过期判断

#### 2.1.2 FlowContext (流程上下文)

```go
type FlowContext struct {
    CurrentAgentKey string  // 当前执行的智能体代码
    LastBotMessage  string  // 最后一条AI回复
    TurnCount       int     // 对话轮次计数
}
```

**作用**:
- 跟踪当前执行的智能体
- 记录对话进度
- 用于流程控制和调试

**使用场景**:
```go
// 在智能体执行前更新
session.FlowContext.CurrentAgentKey = "triage_agent"

// 在智能体执行后更新
session.FlowContext.LastBotMessage = "请问您头痛多久了？"
session.FlowContext.TurnCount++
```

#### 2.1.3 MessageContext (消息上下文 - 核心)

```go
type MessageContext struct {
    Summary            string     // 历史摘要文本
    WindowMessages     []*Message // 最近N轮消息窗口
    Mode               string     // 当前模式: FULL_HISTORY / SUMMARY_N
    CheckpointMessageID string     // 当前checkpoint的消息ID
}
```

**作用**: 这是短期记忆的核心，控制对话历史的返回方式

**字段详解**:

| 字段 | 类型 | 说明 | 更新时机 |
|------|------|------|----------|
| `Summary` | string | 历史摘要文本 | Checkpoint时更新 |
| `WindowMessages` | []*Message | 最近N轮消息数组 | Checkpoint时更新 |
| `Mode` | string | 当前记忆模式 | Checkpoint时切换为SUMMARY_N |
| `CheckpointMessageID` | string | Checkpoint消息ID | Checkpoint时更新 |

**Message 结构**:
```go
type Message struct {
    Role    string // "user" 或 "assistant"
    Content string // 消息内容
}
```

#### 2.1.4 GlobalState (全局共享状态)

```go
type GlobalState struct {
    // 1. 公共协议区（Router和Supervisor决策依据）
    Shared   *SharedEntities            // 共享实体（兼容旧版本）
    Entities *SharedEntities            // 共享实体（新版本）
    
    // 2. 智能体私有槽位（Agent独享记忆）
    AgentSlots map[string]interface{}   // Key: agent_code, Value: agent专属状态
    
    // 3. 流程控制
    CurrentIntent string                 // 当前意图
    PendingAction *PendingAction        // 挂起操作
}
```

**SharedEntities (公共实体)**:
```go
type SharedEntities struct {
    SymptomSummary string  // 症状摘要
    Disease        string  // 疾病
    TargetDept     string  // 目标科室
    TargetDoctor   string  // 目标医生
    IntentTag      string  // 意图标签（如"book_ticket"）
}
```

**使用场景**:
```go
// Router 更新公共实体
session.GlobalState.Shared = &SharedEntities{
    SymptomSummary: "头痛、发热3天",
    Disease: "",
    TargetDept: "内科",
    IntentTag: "consult",
}

// 智能体更新私有槽位
if session.GlobalState.AgentSlots == nil {
    session.GlobalState.AgentSlots = make(map[string]interface{})
}
session.GlobalState.AgentSlots["triage_agent"] = map[string]interface{}{
    "symptoms_collected": true,
    "triage_level": "moderate",
}
```

**PendingAction (挂起操作)**:
```go
type PendingAction struct {
    ToolName   string                 // 工具名称
    ToolParams map[string]interface{} // 工具参数
    Reason     string                 // 挂起原因
}
```

**使用场景**:
```go
// 智能体需要用户确认时挂起
session.GlobalState.PendingAction = &PendingAction{
    ToolName: "create_payment_order",
    ToolParams: map[string]interface{}{
        "amount": 100,
        "bill_id": "123",
    },
    Reason: "waiting_for_user_confirmation",
}
```

#### 2.1.5 UserProfile (用户快照)

```go
type UserProfile struct {
    UserID          string   // 用户唯一标识
    Name            string   // 用户称呼
    
    // 安全红线数据（必须注入System Prompt）
    Allergies       []string // 过敏史
    
    // 医疗背景数据
    ChronicDiseases []string // 慢病史
    SurgeryHistory  []string // 手术史
    
    // 偏好数据
    Preferences     []string // 就医偏好
}
```

**作用**: 存储用户画像，用于个性化服务和安全检查

**使用场景**:
```go
// 开药Agent检查过敏史
for _, allergy := range session.UserSnapshot.Allergies {
    if contains(medicine, allergy) {
        return "该药物可能引起过敏反应"
    }
}

// 保存用户偏好
session.UserSnapshot.Preferences = append(
    session.UserSnapshot.Preferences,
    "prefer_weekend",
)
```

---

## 核心API接口

### 3.1 QueryMemoryContext - 查询记忆上下文

**函数签名**:
```go
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error)
```

**功能**: 根据会话ID查询并构建适合当前对话的记忆上下文

**请求参数**:
```go
type MemoryQueryRequest struct {
    ConversationID      string  // 会话ID（必填）
    EnterpriseID        string  // 企业ID
    PatientID           string  // 患者ID
    Query               string  // 当前用户查询（用于计算Token）
    TokenThresholdRatio float64 // Token阈值比例（默认0.75）
    RecentTurns         int     // 保留轮数（默认8）
    ModelContextWindow  int     // 模型上下文窗口（默认16000）
}
```

**返回结果**:
```go
type MemoryContext struct {
    ConversationID          string       // 会话ID
    Mode                    string       // 当前使用的模式
    Session                 *SessionValue // 完整的会话状态
    History                 string       // 最终返回的对话历史
    FullHistory             string       // 完整对话历史
    EstimatedTokens         int          // 预估Token数量
    TokenRatio              float64      // Token占用比例
    ShouldCheckpointSummary bool         // 是否需要触发摘要
}
```

**核心流程**:
```go
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    // 1. 获取短期记忆（Redis）
    session, _ := a.GetShortMemory(req.ConversationID)
    mode := session.MessageContext.Mode
    
    // 2. 根据checkpoint查询消息
    var messages []*AIMessage
    if session.MessageContext.CheckpointMessageID != "" {
        // 从checkpoint之后查询
        messages = a.QueryMessageByConversationIDASCFromCheckpoint(
            req.ConversationID, 
            session.MessageContext.CheckpointMessageID,
        )
    } else {
        // 查询全部消息
        messages = a.QueryMessageByConversationIDASC(req.ConversationID)
    }
    
    // 3. 构建完整历史
    fullHistory := buildHistoryFromAIMessages(messages)
    
    // 4. 计算token占用率
    estimatedTokens := estimateTokenCount(fullHistory + "\n" + req.Query)
    tokenRatio := float64(estimatedTokens) / float64(contextWindow)
    
    // 5. 根据模式构建最终历史
    history := fullHistory
    if mode == MemoryModeSummaryN {
        history = composeSummaryAndRecent(session)
    }
    
    // 6. 判断是否需要触发摘要
    shouldCheckpoint := tokenRatio >= threshold
    
    return &MemoryContext{
        History:                 history,
        ShouldCheckpointSummary: shouldCheckpoint,
        // ...
    }, nil
}
```

**使用示例**:
```go
// 基础用法
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv_123",
    Query: "我最近感觉头痛",
})

// 高级用法：自定义阈值
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv_123",
    Query: "我最近感觉头痛",
    TokenThresholdRatio: 0.8,  // 80%阈值
    RecentTurns: 10,           // 保留10轮
    ModelContextWindow: 32000,   // 32k上下文
})

// 使用返回的历史上下文
systemPrompt := fmt.Sprintf("你是一个医疗助手，以下是对话历史：\n%s\n请根据历史回答用户问题。", ctx.History)
```

### 3.2 WriteTurn - 写入对话轮次

**函数签名**:
```go
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error)
```

**功能**: 记录一次对话轮次，更新短期记忆

**请求参数**:
```go
type MemoryWriteRequest struct {
    ConversationID string  // 会话ID（必填）
    UserID         string  // 用户ID
    AgentCode      string  // 智能体代码
    UserQuery      string  // 用户查询
    AgentResponse  string  // 智能体响应
}
```

**返回结果**:
```go
type MemoryWriteResult struct {
    ConversationID string // 会话ID
    Mode           string // 当前记忆模式
    UpdatedAt      int64  // 更新时间戳
}
```

**核心流程**:
```go
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    // 1. 获取会话状态
    session, _ := a.GetShortMemory(req.ConversationID)
    
    // 2. 更新用户信息
    if req.UserID != "" {
        session.Meta.UserID = req.UserID
        session.UserSnapshot.UserID = req.UserID
    }
    
    // 3. 更新流程上下文
    if req.AgentCode != "" {
        session.FlowContext.CurrentAgentKey = req.AgentCode
    }
    if req.AgentResponse != "" {
        session.FlowContext.LastBotMessage = req.AgentResponse
    }
    session.FlowContext.TurnCount++
    
    // 4. 保存到Redis
    a.SetShortMemory(req.ConversationID, session)
    
    return &MemoryWriteResult{
        ConversationID: req.ConversationID,
        Mode: session.MessageContext.Mode,
        UpdatedAt: session.Meta.UpdatedAt,
    }, nil
}
```

**使用示例**:
```go
// 基础用法
result, err := app.WriteTurn(&MemoryWriteRequest{
    ConversationID: "conv_123",
    UserID: "user_456",
    AgentCode: "triage_agent",
    UserQuery: "我头痛",
    AgentResponse: "请问您头痛持续多久了？",
})

// 完整流程：查询 -> 处理 -> 写入
func handleUserMessage(conversationID, userQuery string) {
    // 1. 查询记忆上下文
    ctx, _ := app.QueryMemoryContext(&MemoryQueryRequest{
        ConversationID: conversationID,
        Query: userQuery,
    })
    
    // 2. 使用历史上下文调用LLM
    response := callLLM(ctx.History, userQuery)
    
    // 3. 写入对话轮次
    app.WriteTurn(&MemoryWriteRequest{
        ConversationID: conversationID,
        UserQuery: userQuery,
        AgentResponse: response,
    })
    
    // 4. 检查是否需要checkpoint
    if ctx.ShouldCheckpointSummary {
        // 生成摘要并创建checkpoint
        summary := generateSummary(ctx.FullHistory)
        app.CheckpointShortMemory(conversationID, summary, 8)
    }
}
```

### 3.3 CheckpointShortMemory - 创建Checkpoint

**函数签名**:
```go
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error
```

**功能**: 将当前对话历史压缩为Checkpoint，包含摘要和最近N轮对话

**参数**:
- `conversationID`: 会话ID
- `summary`: 历史摘要文本
- `recentTurns`: 保留轮数（默认8）

**核心流程**:
```go
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    // 1. 获取会话状态
    session, _ := a.GetShortMemory(conversationID)
    
    // 2. 查询所有消息
    messages, _ := a.QueryMessageByConversationIDASC(conversationID)
    
    // 3. 构建"摘要+最近N轮"内容
    summaryAndRecent := composeSummaryAndRecent(session)
    
    // 4. 插入checkpoint消息到数据库
    checkpointMessageID := xuid.UUID()
    sql := `INSERT INTO ai_message (message_id, conversation_id, query, answer, ...) 
            VALUES ($1, $2, $3, $4, ...)`
    a.DBExec(sql, checkpointMessageID, conversationID, 
              "[MEMORY_CHECKPOINT]", summaryAndRecent, ...)
    
    // 5. 更新session状态
    session.MessageContext.Summary = strings.TrimSpace(summary)
    session.MessageContext.WindowMessages = buildRecentMessages(messages, recentTurns)
    session.MessageContext.Mode = MemoryModeSummaryN
    session.MessageContext.CheckpointMessageID = checkpointMessageID
    
    // 6. 保存到Redis
    return a.SetShortMemory(conversationID, session)
}
```

**使用示例**:
```go
// 基础用法
err := app.CheckpointShortMemory("conv_123", "用户咨询头痛问题，持续3天", 8)

// 完整流程：检测并创建checkpoint
func handleMemoryCheckpoint(conversationID string) error {
    // 1. 查询当前状态
    ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
        ConversationID: conversationID,
        Query: "",
    })
    
    // 2. 检查是否需要checkpoint
    if ctx.ShouldCheckpointSummary {
        // 3. 生成摘要（调用LLM）
        summary := callLLMForSummary(ctx.FullHistory)
        
        // 4. 创建checkpoint
        err := app.CheckpointShortMemory(conversationID, summary, 8)
        if err != nil {
            return err
        }
        
        // 5. 日志记录
        log.Printf("Checkpoint created for conversation %s", conversationID)
    }
    
    return nil
}
```

### 3.4 FinalizeSessionMemory - 结束会话

**函数签名**:
```go
func (a *AgentApp) FinalizeSessionMemory(req *SessionFinalizeRequest) error
```

**功能**: 会话结束时创建最终Checkpoint

**参数**:
```go
type SessionFinalizeRequest struct {
    ConversationID string // 会话ID
    Summary        string // 会话摘要
    RecentTurns    int    // 保留轮数
}
```

**使用场景**:
```go
// 用户主动结束对话
func handleUserEndConversation(conversationID string) {
    // 1. 查询完整历史
    ctx, _ := app.QueryMemoryContext(&MemoryQueryRequest{
        ConversationID: conversationID,
        Query: "",
    })
    
    // 2. 生成最终摘要
    finalSummary := callLLMForSummary(ctx.FullHistory)
    
    // 3. 结束会话
    app.FinalizeSessionMemory(&SessionFinalizeRequest{
        ConversationID: conversationID,
        Summary: finalSummary,
        RecentTurns: 8,
    })
}
```

### 3.5 辅助函数

#### 3.5.1 CreateShortMemory - 创建会话

**函数签名**:
```go
func (a *AgentApp) CreateShortMemory(req *server.AgentRequest) error
```

**功能**: 为新对话创建短期记忆

**使用场景**: 对话开始时调用

```go
// 在第一个消息到达时创建
app.CreateShortMemory(&server.AgentRequest{
    ConversationId: "conv_123",
    UserId: "user_456",
})
```

#### 3.5.2 GetShortMemory - 获取会话状态

**函数签名**:
```go
func (a *AgentApp) GetShortMemory(conversationId string) (*SessionValue, error)
```

**功能**: 从Redis获取会话状态

#### 3.5.3 SetShortMemory - 保存会话状态

**函数签名**:
```go
func (a *AgentApp) SetShortMemory(conversationId string, session *SessionValue) error
```

**功能**: 保存会话状态到Redis

---

## 完整使用流程

### 4.1 智能体处理消息的完整流程

```go
func handleMessage(c *gin.Context) {
    // 1. 验证请求
    req, resp, event, ok := powerai.DoValidateAgentRequest(c, "my-agent")
    if !ok {
        return
    }
    
    // 2. 创建会话（如果是新对话）
    app.CreateShortMemory(req)
    
    // 3. 查询记忆上下文
    memoryCtx, err := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: req.ConversationId,
        Query: req.Query,
    })
    
    // 4. 构建System Prompt
    systemPrompt := buildSystemPrompt(memoryCtx)
    
    // 5. 调用LLM处理
    llmResponse := callLLM(systemPrompt, memoryCtx.History, req.Query)
    
    // 6. 保存消息到数据库
    app.UpdateMessage(req.MessageId, req.Query, llmResponse, "my-agent")
    
    // 7. 写入对话轮次
    app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: req.ConversationId,
        UserID: req.UserId,
        AgentCode: "my-agent",
        UserQuery: req.Query,
        AgentResponse: llmResponse,
    })
    
    // 8. 检查是否需要checkpoint
    if memoryCtx.ShouldCheckpointSummary {
        go func() {
            // 异步生成摘要
            summary := callLLMForSummary(memoryCtx.FullHistory)
            // 创建checkpoint
            app.CheckpointShortMemory(req.ConversationId, summary, 8)
        }()
    }
    
    // 9. 返回响应
    event.WriteAgentResponse(resp, llmResponse)
    event.Done(resp)
}
```

### 4.2 智能体间共享状态的流程

```go
// 智能体A处理
func agentAHandler(conversationID string, userQuery string) string {
    // 1. 查询记忆上下文
    ctx, _ := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: conversationID,
        Query: userQuery,
    })
    
    // 2. 更新公共实体
    ctx.Session.GlobalState.Shared = &powerai.SharedEntities{
        SymptomSummary: "头痛、发热",
        Disease: "感冒",
        IntentTag: "consult",
    }
    
    // 3. 保存状态
    app.SetShortMemory(conversationID, ctx.Session)
    
    // 4. 处理并返回
    return "您可能感冒了，建议多休息"
}

// 智能体B处理（可以访问智能体A更新的状态）
func agentBHandler(conversationID string, userQuery string) string {
    // 1. 查询记忆上下文
    ctx, _ := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: conversationID,
        Query: userQuery,
    })
    
    // 2. 读取智能体A更新的公共实体
    disease := ctx.Session.GlobalState.Shared.Disease
    intent := ctx.Session.GlobalState.Shared.IntentTag
    
    // 3. 根据上下文处理
    if intent == "consult" {
        return fmt.Sprintf("针对您的%s症状，建议挂号内科", disease)
    }
    
    return "请问还有什么可以帮助您的？"
}
```

### 4.3 多轮对话的完整生命周期

```
第1轮（新对话）
┌─────────────────────────────────┐
│ 1. CreateShortMemory          │ 创建会话
│ 2. QueryMemoryContext          │ 查询（返回空）
│ 3. 处理消息                     │
│ 4. WriteTurn                    │ 记录轮次
│ 5. ShouldCheckpoint = false      │ 不触发摘要
└─────────────────────────────────┘

第2-8轮
┌─────────────────────────────────┐
│ 1. QueryMemoryContext          │ 查询（返回完整历史）
│ 2. 处理消息                     │
│ 3. WriteTurn                    │ 记录轮次
│ 4. ShouldCheckpoint = false      │ 不触发摘要
└─────────────────────────────────┘

第9轮（触发checkpoint）
┌─────────────────────────────────┐
│ 1. QueryMemoryContext          │ 查询
│    - TokenRatio = 0.8 > 0.75    │ 超过阈值
│    - ShouldCheckpoint = true  │ 需要摘要
│ 2. 生成摘要                      │ LLM生成
│ 3. CheckpointShortMemory        │ 创建checkpoint
│    - Mode: FULL_HISTORY → SUMMARY_N
│    - CheckpointMessageID: msg_cp_001
│ 4. 处理消息                     │
│ 5. WriteTurn                    │ 记录轮次
└─────────────────────────────────┘

第10-17轮
┌─────────────────────────────────┐
│ 1. QueryMemoryContext          │ 查询
│    - Mode: SUMMARY_N            │ 使用摘要模式
│    - History: 摘要 + 最近8轮   │
│ 2. 处理消息                     │
│ 3. WriteTurn                    │ 记录轮次
│ 4. ShouldCheckpoint = false      │ 不触发摘要
└─────────────────────────────────┘

第18轮（再次触发checkpoint）
┌─────────────────────────────────┐
│ 1. QueryMemoryContext          │ 查询
│    - TokenRatio = 0.8 > 0.75    │ 再次超过阈值
│    - ShouldCheckpoint = true  │ 需要摘要
│ 2. 生成累积摘要                  │ LLM生成
│ 3. CheckpointShortMemory        │ 创建checkpoint
│    - CheckpointMessageID: msg_cp_002
│    - Summary: 包含之前的摘要
│ 4. 处理消息                     │
│ 5. WriteTurn                    │ 记录轮次
└─────────────────────────────────┘

会话结束
┌─────────────────────────────────┐
│ 1. QueryMemoryContext          │ 查询完整历史
│ 2. 生成最终摘要                  │ LLM生成
│ 3. FinalizeSessionMemory         │ 结束会话
└─────────────────────────────────┘
```

---

## 智能体对接示例

### 5.1 基础智能体模板

```go
package main

import (
    "github.com/gin-gonic/gin"
    "orgine.com/ai-team/power-ai-framework-v4"
)

func main() {
    manifest := `{
        "code": "my-medical-agent",
        "name": "医疗助手Agent",
        "version": "1.0.0",
        "description": "医疗咨询助手"
    }`

    app, err := powerai.NewAgent(
        manifest,
        powerai.WithSendMsgRouter(handleMessage),
    )
    if err != nil {
        panic(err)
    }

    app.Run()
}

func handleMessage(c *gin.Context) {
    // 1. 验证请求
    req, resp, event, ok := powerai.DoValidateAgentRequest(c, "my-medical-agent")
    if !ok {
        return
    }
    
    // 2. 获取AgentApp实例
    app := c.MustGet("app").(*powerai.AgentApp)
    
    // 3. 创建会话
    app.CreateShortMemory(req)
    
    // 4. 查询记忆上下文
    memoryCtx, err := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: req.ConversationId,
        Query: req.Query,
    })
    if err != nil {
        event.WriteAgentResponseError(resp, "500", "查询记忆失败")
        event.Done(resp)
        return
    }
    
    // 5. 构建提示词
    prompt := buildPrompt(memoryCtx, req.Query)
    
    // 6. 调用LLM
    response := callLLM(prompt)
    
    // 7. 保存消息
    app.UpdateMessage(req.MessageId, req.Query, response, "my-medical-agent")
    
    // 8. 写入对话轮次
    app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: req.ConversationId,
        UserID: req.UserId,
        AgentCode: "my-medical-agent",
        UserQuery: req.Query,
        AgentResponse: response,
    })
    
    // 9. 检查是否需要checkpoint
    if memoryCtx.ShouldCheckpointSummary {
        go func() {
            summary := generateSummary(memoryCtx.FullHistory)
            app.CheckpointShortMemory(req.ConversationId, summary, 8)
        }()
    }
    
    // 10. 返回响应
    event.WriteAgentResponse(resp, response)
    event.Done(resp)
}

func buildPrompt(ctx *powerai.MemoryContext, userQuery string) string {
    // 构建包含历史上下文的提示词
    return fmt.Sprintf(`你是一个专业的医疗助手。

对话历史：
%s

当前用户问题：%s

请根据对话历史回答用户的问题。`, ctx.History, userQuery)
}
```

### 5.2 使用GlobalState的智能体

```go
func triageAgentHandler(c *gin.Context) {
    req, resp, event, ok := powerai.DoValidateAgentRequest(c, "triage-agent")
    if !ok {
        return
    }
    
    app := c.MustGet("app").(*powerai.AgentApp)
    
    // 1. 查询记忆上下文
    memoryCtx, _ := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: req.ConversationId,
        Query: req.Query,
    })
    
    // 2. 提取症状信息
    symptoms := extractSymptoms(req.Query, memoryCtx.History)
    
    // 3. 更新公共实体
    if memoryCtx.Session.GlobalState.Shared == nil {
        memoryCtx.Session.GlobalState.Shared = &powerai.SharedEntities{}
    }
    memoryCtx.Session.GlobalState.Shared.SymptomSummary = symptoms
    memoryCtx.Session.GlobalState.Shared.IntentTag = "triage"
    
    // 4. 保存状态
    app.SetShortMemory(req.ConversationId, memoryCtx.Session)
    
    // 5. 处理并返回
    response := processTriage(symptoms)
    
    // 6. 记录轮次
    app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: req.ConversationId,
        UserID: req.UserId,
        AgentCode: "triage-agent",
        UserQuery: req.Query,
        AgentResponse: response,
    })
    
    event.WriteAgentResponse(resp, response)
    event.Done(resp)
}

func reportAgentHandler(c *gin.Context) {
    req, resp, event, ok := powerai.DoValidateAgentRequest(c, "report-agent")
    if !ok {
        return
    }
    
    app := c.MustGet("app").(*powerai.AgentApp)
    
    // 1. 查询记忆上下文
    memoryCtx, _ := app.QueryMemoryContext(&powerai.QueryMemoryContext{
        ConversationID: req.ConversationId,
        Query: req.Query,
    })
    
    // 2. 读取智能体A更新的症状信息
    symptoms := memoryCtx.Session.GlobalState.Shared.SymptomSummary
    
    // 3. 根据症状生成报告解读
    reportInterpretation := generateReportInterpretation(req.Files, symptoms)
    
    // 4. 记录轮次
    app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: req.ConversationId,
        UserID: req.UserId,
        AgentCode: "report-agent",
        UserQuery: req.Query,
        AgentResponse: reportInterpretation,
    })
    
    event.WriteAgentResponse(resp, reportInterpretation)
    event.Done(resp)
}
```

### 5.3 使用UserProfile的智能体

```go
func drugAgentHandler(c *gin.Context) {
    req, resp, event, ok := powerai.DoValidateAgentRequest(c, "drug-agent")
    if !ok {
        return
    }
    
    app := c.MustGet("app").(*powerai.AgentApp)
    
    // 1. 查询记忆上下文
    memoryCtx, _ := app.QueryMemoryContext(&powerai.QueryMemoryRequest{
        ConversationID: req.ConversationId,
        Query: req.Query,
    })
    
    // 2. 检查过敏史（安全红线）
    for _, allergy := range memoryCtx.Session.UserSnapshot.Allergies {
        if contains(req.Query, allergy) {
            response := "警告：该药物可能引起过敏反应，请咨询医生后再使用"
            
            app.WriteTurn(&powerai.MemoryWriteRequest{
                ConversationID: req.ConversationId,
                UserID: req.UserId,
                AgentCode: "drug-agent",
                UserQuery: req.Query,
                AgentResponse: response,
            })
            
            event.WriteAgentResponse(resp, response)
            event.Done(resp)
            return
        }
    }
    
    // 3. 正常处理
    response := processDrugRequest(req.Query)
    
    app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: req.ConversationId,
        UserID: req.UserId,
        AgentCode: "drug-agent",
        UserQuery: req.Query,
        AgentResponse: response,
    })
    
    event.WriteAgentResponse(resp, response)
    event.Done(resp)
}
```

---

## 最佳实践

### 6.1 Token管理最佳实践

#### 6.1.1 根据模型调整阈值

```go
// 对于小模型（4k上下文）
ctx, _ := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
    ConversationID: conversationID,
    Query: userQuery,
    TokenThresholdRatio: 0.6,  // 降低阈值
    RecentTurns: 5,           // 减少保留轮数
    ModelContextWindow: 4096,  // 4k上下文
})

// 对于大模型（32k上下文）
ctx, _ := app.QueryMemoryContext(&powerai.QueryMemoryRequest{
    ConversationID: conversationID,
    Query: userQuery,
    TokenThresholdRatio: 0.85, // 提高阈值
    RecentTurns: 12,          // 增加保留轮数
    ModelContextWindow: 32768, // 32k上下文
})
```

#### 6.1.2 动态调整策略

```go
// 根据对话复杂度动态调整
func getDynamicThreshold(conversationID string) float64 {
    // 查询对话轮次
    session, _ := app.GetShortMemory(conversationID)
    turnCount := session.FlowContext.TurnCount
    
    // 对话初期使用较低阈值
    if turnCount < 5 {
        return 0.6
    }
    
    // 对话中期使用默认阈值
    if turnCount < 20 {
        return 0.75
    }
    
    // 对话后期使用较高阈值
    return 0.85
}
```

### 6.2 摘要生成最佳实践

#### 6.2.1 摘要内容要求

```go
// 好的摘要示例
goodSummary := "患者因头痛、发热3天就诊，主诉症状为持续性头痛伴低热。患者有青霉素过敏史和高血压病史。经初步问诊，建议内科就诊，需监测体温变化。"

// 不好的摘要示例
badSummary := "用户说了很多话，关于头痛和发热的事情"
```

**摘要应包含**:
- 主要症状和持续时间
- 重要病史（过敏史、慢病史）
- 已给出的建议或诊断
- 关键的决策点

#### 6.2.2 摘要生成示例

```go
func generateSummary(fullHistory string) string {
    prompt := fmt.Sprintf(`请为以下对话生成一个简洁的医疗摘要：

对话历史：
%s

要求：
1. 包含主要症状和持续时间
2. 提及重要的过敏史或病史
3. 记录已给出的建议
4. 控制在100字以内

摘要：`, fullHistory)
    
    return callLLM(prompt)
}
```

### 6.3 状态管理最佳实践

#### 6.3.1 GlobalState使用规范

```go
// ✅ 正确：使用公共协议区
session.GlobalState.Shared = &powerai.SharedEntities{
    SymptomSummary: "头痛、发热",
    IntentTag: "consult",
}

// ✅ 正确：使用智能体私有槽位
if session.GlobalState.AgentSlots == nil {
    session.GlobalState.AgentSlots = make(map[string]interface{})
}
session.GlobalState.AgentSlots["my-agent"] = map[string]interface{}{
    "custom_state": "value",
}

// ❌ 错误：直接覆盖Shared
session.GlobalState.Shared = nil  // 不要这样做！

// ❌ 错误：混用Shared和Entities
session.GlobalState.Entities = &powerai.SharedEntities{...}
session.GlobalState.Shared = &powerai.SharedEntities{...}  // 混乱！
```

#### 6.3.2 并发控制

```go
// 使用sync.Mutex保护并发写入
var memoryMutex sync.Mutex

func safeWriteTurn(app *powerai.AgentApp, req *powerai.MemoryWriteRequest) error {
    memoryMutex.Lock()
    defer memoryMutex.Unlock()
    
    _, err := app.WriteTurn(req)
    return err
}
```

### 6.4 错误处理最佳实践

```go
// 完整的错误处理示例
func handleUserMessage(conversationID, userQuery string) error {
    // 1. 查询记忆上下文
    memoryCtx, err := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: conversationID,
        Query: userQuery,
    })
    if err != nil {
        log.Printf("查询记忆上下文失败: %v", err)
        // 返回默认响应，而不是中断流程
        return fmt.Errorf("系统繁忙，请稍后再试")
    }
    
    // 2. 调用LLM
    response, err := callLLM(memoryCtx.History, userQuery)
    if err != nil {
        log.Printf("调用LLM失败: %v", err)
        // 保存错误信息
        app.UpdateMessage(req.MessageId, req.Query, 
            "抱歉，我遇到了一些问题，请稍后再试。", "my-agent")
        return nil
    }
    
    // 3. 写入对话轮次
    _, err = app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: conversationID,
        UserQuery: userQuery,
        AgentResponse: response,
    })
    if err != nil {
        log.Printf("写入对话轮次失败: %v", err)
        // 继续返回响应，不中断用户体验
    }
    
    // 4. 异步checkpoint（失败不影响主流程）
    if memoryCtx.ShouldCheckpointSummary {
        go func() {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("Checkpoint panic: %v", r)
                }
            }()
            
            summary := generateSummary(memoryCtx.FullHistory)
            if err := app.CheckpointShortMemory(conversationID, summary, 8); err != nil {
                log.Printf("Checkpoint失败: %v", err)
            }
        }()
    }
    
    return nil
}
```

---

## 常见问题

### Q1: Redis过期后会发生什么？

**A**: Redis中的SessionValue会在30分钟后过期，但PostgreSQL中的消息记录会永久保存。下次用户再次对话时，会创建新的会话，历史消息需要从数据库重新查询。

**解决方案**:
```go
// 在Redis过期前刷新过期时间
func refreshSessionExpiration(conversationID string) {
    session, _ := app.GetShortMemory(conversationID)
    if session != nil {
        app.SetShortMemory(conversationID, session)  // 刷新30分钟
    }
}
```

### Q2: 如何处理并发写入冲突？

**A**: 使用互斥锁保护并发写入：

```go
var memoryWriteMutex sync.Mutex

func WriteTurn(app *powerai.AgentApp, req *powerai.MemoryWriteRequest) (*powerai.MemoryWriteResult, error) {
    memoryWriteMutex.Lock()
    defer memoryWriteMutex.Unlock()
    
    return app.WriteTurn(req)
}
```

### Q3: Checkpoint消息会占用多少数据库空间？

**A**: 每个Checkpoint消息包含摘要+最近N轮对话，假设：
- 摘要: 100字
- 最近8轮: 每轮20字 × 16条 = 320字
- 总计约400字，数据库存储成本很低

### Q4: 如何查看当前会话的状态？

**A**: 直接查询Redis：

```go
session, err := app.GetShortMemory("conv_123")
if err == nil {
    fmt.Printf("模式: %s\n", session.MessageContext.Mode)
    fmt.Printf("轮次: %d\n", session.FlowContext.TurnCount)
    fmt.Printf("摘要: %s\n", session.MessageContext.Summary)
    fmt.Printf("当前智能体: %s\n", session.FlowContext.CurrentAgentKey)
}
```

### Q5: 如何实现跨会话的记忆？

**A**: 当前框架不支持，但可以扩展：

```go
// 1. 查询用户历史会话
conversations, _ := app.QueryConversationsByUserID("user_123")

// 2. 提取关键信息
for _, conv := range conversations {
    // 分析对话历史，提取用户偏好
    analyzeConversation(conv)
}

// 3. 保存到UserProfile
session.UserSnapshot.Preferences = append(
    session.UserSnapshot.Preferences,
    "prefer_morning",
)
```

### Q6: 如何调试记忆管理问题？

**A**: 添加详细日志：

```go
func QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    log.Printf("[MEMORY] QueryMemoryContext: conversationID=%s", req.ConversationID)
    
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        log.Printf("[MEMORY] GetShortMemory failed: %v", err)
        session = newDefaultSessionValue(req.ConversationID, req.PatientID)
    }
    
    log.Printf("[MEMORY] Mode: %s", session.MessageContext.Mode)
    log.Printf("[MEMORY] CheckpointID: %s", session.MessageContext.CheckpointMessageID)
    
    // ... 业务逻辑
    
    log.Printf("[MEMORY] History length: %d", len(history))
    log.Printf("[MEMORY] TokenRatio: %.2f", tokenRatio)
    log.Printf("[MEMORY] ShouldCheckpoint: %v", shouldCheckpoint)
    
    return &MemoryContext{...}, nil
}
```

---

## 总结

### 核心要点

1. **双模式机制**: FULL_HISTORY（短期对话）→ SUMMARY_N（长对话）
2. **Checkpoint机制**: 分段管理，增量摘要
3. **Token管理**: 自动检测，智能切换
4. **状态共享**: GlobalState支持智能体间协作
5. **用户画像**: UserProfile支持个性化服务

### 关键API

| API | 作用 | 调用时机 |
|-----|------|----------|
| `CreateShortMemory` | 创建会话 | 对话开始时 |
| `QueryMemoryContext` | 查询上下文 | 每次处理消息前 |
| `WriteTurn` | 记录轮次 | 每次处理消息后 |
| `CheckpointShortMemory` | 创建摘要 | Token超过阈值时 |
| `FinalizeSessionMemory` | 结束会话 | 对话结束时 |

### 数据流

```
用户消息 → QueryMemoryContext(获取上下文) → LLM处理
    ↓
    ↓ UpdateMessage(保存到数据库) → WriteTurn(更新Redis)
    ↓
    ↓ ShouldCheckpoint? → 是: CheckpointShortMemory
    ↓
返回响应
```

---

**文档结束**
