# 短期记忆模块对接文档

## 版本信息
- 版本：v2.0
- 更新日期：2026-02-09
- 作者：AI团队

---

## 目录
1. [设计概述](#设计概述)
2. [核心概念](#核心概念)
3. [架构设计](#架构设计)
4. [数据结构](#数据结构)
5. [核心接口](#核心接口)
6. [使用流程](#使用流程)
7. [配置参数](#配置参数)
8. [性能优化](#性能优化)
9. [注意事项](#注意事项)
10. [常见问题](#常见问题)

---

## 设计概述

### 背景
在长时间对话场景中，随着对话轮次的增加，上下文长度会不断增长，导致：
1. **Token超限**：超过模型上下文窗口限制
2. **性能下降**：处理超长上下文耗时增加
3. **成本上升**：API调用费用增加
4. **效果下降**：模型注意力分散，影响回答质量

### 解决方案
通过**分段记忆管理**机制，将长对话切分为多个段，每段包含：
- **历史摘要**：该段之前所有对话的摘要
- **最近N轮**：该段内的最近N轮对话
- **Checkpoint标记**：段与段之间的分割点

### 核心特性
- ✅ 支持多次摘要，每次都是"重新开始"新的对话周期
- ✅ 自动检测token占用率，超过阈值自动触发摘要
- ✅ 分段查询，避免读取过长历史记录
- ✅ 数据库持久化，支持历史追溯
- ✅ Redis缓存，提升读取性能

---

## 核心概念

### 1. 记忆模式

#### FULL_HISTORY（全量历史模式）
- 适用场景：对话初期，token未超过阈值
- 行为：返回所有历史对话
- 触发条件：默认模式

#### SUMMARY_N（摘要模式）
- 适用场景：对话中后期，token超过阈值
- 行为：返回"历史摘要 + 最近N轮"
- 触发条件：token超过阈值后自动切换

### 2. Checkpoint（检查点）
- 定义：对话历史的分段标记点
- 存储：数据库中的特殊消息（query = `[MEMORY_CHECKPOINT]`）
- 作用：标识当前段的起始位置

### 3. Token阈值
- 默认值：75%
- 含义：当历史对话的token占用率超过75%时，触发摘要
- 可配置：通过 `TokenThresholdRatio` 参数调整

---

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                     短期记忆架构                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐        ┌──────────────┐            │
│  │   Redis      │        │  PostgreSQL  │            │
│  │              │        │              │            │
│  │ SessionValue │◄──────►│  AIMessage   │            │
│  │              │        │              │            │
│  │ - Meta       │        │  - 普通消息   │            │
│  │ - FlowCtx    │        │  - Checkpoint │            │
│  │ - MsgCtx     │        │    消息      │            │
│  │ - GlobalState │        │              │            │
│  │ - UserSnap   │        └──────────────┘            │
│  └──────────────┘                                       │
│         ▲                                               │
│         │                                               │
│  ┌──────────────┐                                       │
│  │  业务代码     │                                       │
│  │              │                                       │
│  │ QueryMemory  │                                       │
│  │ WriteTurn    │                                       │
│  │ Checkpoint   │                                       │
│  └──────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

### 数据流转

```
用户发起对话
    ↓
CreateShortMemory (创建Redis会话)
    ↓
QueryMemoryContext (查询记忆上下文)
    ↓
    ├─ FULL_HISTORY模式：返回所有历史
    └─ SUMMARY_N模式：返回摘要+最近N轮
    ↓
Token占用率检查
    ↓
    ├─ < 75%：继续对话
    └─ ≥ 75%：触发摘要
        ↓
        CheckpointShortMemory (创建检查点)
            ↓
            1. 生成摘要
            2. 插入checkpoint消息到数据库
            3. 更新Redis中的SessionValue
            4. 切换到SUMMARY_N模式
    ↓
WriteTurn (写入对话记录)
    ↓
循环往复
```

---

## 数据结构

### 1. SessionValue（Redis存储）

```go
type SessionValue struct {
    Meta           *MetaInfo       `json:"meta"`           // 元信息
    FlowContext    *FlowContext    `json:"flow_context"`    // 流程上下文
    MessageContext *MessageContext `json:"message_context"` // 消息上下文
    GlobalState    *GlobalState    `json:"global_state"`    // 全局状态
    UserSnapshot   *UserProfile    `json:"user_snapshot"`   // 用户快照
}
```

#### MetaInfo
```go
type MetaInfo struct {
    ConversationID string `json:"conversation_id"` // 对话ID
    UserID         string `json:"user_id"`         // 用户ID
    UpdatedAt      int64  `json:"updated_at"`      // 更新时间
}
```

#### FlowContext
```go
type FlowContext struct {
    CurrentAgentKey string `json:"current_agent_key"` // 当前智能体
    LastBotMessage  string `json:"last_bot_message"`  // 最后AI回复
    TurnCount       int    `json:"turn_count"`       // 轮次计数
}
```

#### MessageContext
```go
type MessageContext struct {
    Summary            string     `json:"summary"`              // 历史摘要
    WindowMessages     []*Message `json:"window_messages"`      // 最近N轮消息
    Mode               string     `json:"mode"`                 // 记忆模式
    CheckpointMessageID string     `json:"checkpoint_message_id"` // 检查点ID
}
```

#### Message
```go
type Message struct {
    Role    string `json:"role"`    // 角色：user/assistant
    Content string `json:"content"` // 内容
}
```

#### GlobalState
```go
type GlobalState struct {
    Shared        *SharedEntities              `json:"shared"`        // 公共实体
    Entities      *SharedEntities              `json:"entities"`      // 实体信息
    AgentSlots    map[string]interface{}       `json:"agent_slots"`   // 智能体槽位
    CurrentIntent string                       `json:"current_intent"` // 当前意图
    PendingAction *PendingAction               `json:"pending_action"` // 挂起操作
}
```

#### UserProfile
```go
type UserProfile struct {
    UserID          string   `json:"user_id"`           // 用户ID
    Name            string   `json:"name"`              // 用户名称
    Allergies       []string `json:"allergies"`         // 过敏史
    ChronicDiseases []string `json:"history"`           // 慢病史
    SurgeryHistory  []string `json:"surgery_history"`   // 手术史
    Preferences     []string `json:"preferences"`       // 偏好
}
```

### 2. MemoryContext（查询结果）

```go
type MemoryContext struct {
    ConversationID          string  // 对话ID
    Mode                    string  // 记忆模式
    Session                 *SessionValue  // 会话状态
    History                 string  // 最终返回的历史
    FullHistory             string  // 当前段的完整历史
    EstimatedTokens         int     // 估算的token数
    TokenRatio              float64 // Token占用率
    ShouldCheckpointSummary bool    // 是否需要生成摘要
}
```

### 3. 数据库表结构

#### ai_message表
| 字段 | 类型 | 说明 |
|------|------|------|
| message_id | string | 消息ID |
| conversation_id | string | 对话ID |
| query | string | 用户消息 |
| answer | string | AI回复 |
| create_time | timestamp | 创建时间 |
| create_by | string | 创建者 |
| update_time | timestamp | 更新时间 |
| update_by | string | 更新者 |

**Checkpoint消息特征：**
- `query = '[MEMORY_CHECKPOINT]'`
- `answer = '历史摘要: ...'`

---

## 核心接口

### 1. QueryMemoryContext

查询记忆上下文，获取适合LLM的历史对话。

#### 函数签名
```go
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error)
```

#### 请求参数
```go
type MemoryQueryRequest struct {
    ConversationID      string  // 必须：对话ID
    EnterpriseID        string  // 可选：企业ID
    PatientID           string  // 可选：患者ID
    Query               string  // 可选：当前用户查询
    TokenThresholdRatio float64 // 可选：token阈值（默认0.75）
    RecentTurns         int     // 可选：最近轮次（默认8）
    ModelContextWindow  int     // 可选：模型上下文窗口（默认16000）
}
```

#### 返回结果
```go
type MemoryContext struct {
    ConversationID          string  // 对话ID
    Mode                    string  // 记忆模式：FULL_HISTORY 或 SUMMARY_N
    Session                 *SessionValue  // 会话状态
    History                 string  // 最终返回的历史（用于构建prompt）
    FullHistory             string  // 当前段的完整历史
    EstimatedTokens         int     // 估算的token数量
    TokenRatio              float64 // Token占用率
    ShouldCheckpointSummary bool    // 是否需要生成摘要
}
```

#### 使用示例
```go
// 基础用法
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv-123",
    Query:          "用户的新问题",
})
if err != nil {
    // 处理错误
}

// 构建prompt
prompt := fmt.Sprintf("历史对话:\n%s\n\n用户问题: %s", ctx.History, ctx.Query)

// 检查是否需要生成摘要
if ctx.ShouldCheckpointSummary {
    // 调用LLM生成摘要
    summary := generateSummary(ctx.FullHistory)
    
    // 创建检查点
    err := app.CheckpointShortMemory(ctx.ConversationID, summary, 8)
}

// 调用LLM
llmResponse := callLLM(prompt)

// 写入对话记录
app.WriteTurn(&MemoryWriteRequest{
    ConversationID: ctx.ConversationID,
    AgentResponse: llmResponse,
})
```

#### 高级用法
```go
// 自定义阈值和窗口
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv-123",
    Query:          "用户的新问题",
    TokenThresholdRatio: 0.8,  // 80%阈值
    RecentTurns:         10, // 保留最近10轮
    ModelContextWindow:  32000, // 32k上下文窗口
})
```

---

### 2. WriteTurn

写入对话轮次，更新会话状态。

#### 函数签名
```go
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error)
```

#### 请求参数
```go
type MemoryWriteRequest struct {
    ConversationID string  // 必须：对话ID
    UserID         string  // 可选：用户ID
    AgentCode      string  // 可选：智能体代码
    UserQuery      string  // 可选：用户查询
    AgentResponse  string  // 可选：AI回复
}
```

#### 返回结果
```go
type MemoryWriteResult struct {
    ConversationID string  // 对话ID
    Mode           string  // 记忆模式
    UpdatedAt      int64   // 更新时间
}
```

#### 使用示例
```go
result, err := app.WriteTurn(&MemoryWriteRequest{
    ConversationID: "conv-123",
    UserID:         "user-456",
    AgentCode:      "triage_agent",
    AgentResponse:  "这是AI的回复",
})
if err != nil {
    // 处理错误
}
```

---

### 3. CheckpointShortMemory

创建记忆检查点，将对话历史分段。

#### 函数签名
```go
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error
```

#### 参数说明
- `conversationID`：对话ID
- `summary`：LLM生成的摘要内容
- `recentTurns`：保留的最近轮次（默认8）

#### 执行流程
1. 从Redis读取SessionValue
2. 从数据库读取所有消息
3. 构建"摘要+最近N轮"的内容
4. 生成checkpoint消息ID
5. 将checkpoint消息插入数据库
6. 更新Redis中的SessionValue
7. 切换到SUMMARY_N模式

#### 使用示例
```go
// 生成摘要
summary := "用户咨询了关于高血压的症状和治疗方案..."

// 创建检查点
err := app.CheckpointShortMemory("conv-123", summary, 8)
if err != nil {
    // 处理错误
    log.Printf("Failed to create checkpoint: %v", err)
}
```

---

### 4. CreateShortMemory

创建短期记忆会话（新对话开始时调用）。

#### 函数签名
```go
func (a *AgentApp) CreateShortMemory(req *server.AgentRequest) error
```

#### 使用示例
```go
err := app.CreateShortMemory(&server.AgentRequest{
    ConversationId: "conv-123",
    UserId:         "user-456",
})
if err != nil {
    // 处理错误
}
```

---

### 5. GetShortMemory

获取短期记忆会话状态。

#### 函数签名
```go
func (a *AgentApp) GetShortMemory(conversationId string) (*SessionValue, error)
```

#### 使用示例
```go
session, err := app.GetShortMemory("conv-123")
if err != nil {
    // 处理错误
}

// 访问会话状态
fmt.Printf("当前模式: %s\n", session.MessageContext.Mode)
fmt.Printf("轮次计数: %d\n", session.FlowContext.TurnCount)
```

---

## 使用流程

### 完整对话流程

```go
package main

import (
    "fmt"
    "orgine.com/ai-team/power-ai-framework-v4"
)

func main() {
    // 初始化
    app := powerai.NewAgentApp()
    
    // 1. 用户发起对话
    conversationID := "conv-123"
    userID := "user-456"
    
    // 2. 创建短期记忆会话
    err := app.CreateShortMemory(&powerai.AgentRequest{
        ConversationId: conversationID,
        UserId:         userID,
    })
    if err != nil {
        panic(err)
    }
    
    // 3. 用户第一轮对话
    handleUserMessage(app, conversationID, "我最近感觉头晕，应该怎么办？")
    
    // 4. 用户第二轮对话
    handleUserMessage(app, conversationID, "我还有高血压，需要注意什么？")
    
    // ... 更多对话轮次
}

func handleUserMessage(app *powerai.AgentApp, conversationID, userMessage string) {
    // 1. 查询记忆上下文
    ctx, err := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: conversationID,
        Query:          userMessage,
    })
    if err != nil {
        fmt.Printf("QueryMemoryContext error: %v\n", err)
        return
    }
    
    // 2. 构建prompt
    prompt := fmt.Sprintf("历史对话:\n%s\n\n用户问题: %s", ctx.History, userMessage)
    
    // 3. 调用LLM
    llmResponse := callLLM(prompt)
    
    // 4. 检查是否需要生成摘要
    if ctx.ShouldCheckpointSummary {
        // 异步生成摘要
        go func() {
            summary := generateSummary(ctx.FullHistory)
            err := app.CheckpointShortMemory(ctx.ConversationID, summary, 8)
            if err != nil {
                fmt.Printf("CheckpointShortMemory error: %v\n", err)
            }
        }()
    }
    
    // 5. 写入对话记录
    _, err = app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: conversationID,
        AgentResponse:  llmResponse,
    })
    if err != nil {
        fmt.Printf("WriteTurn error: %v\n", err)
    }
    
    // 6. 返回AI回复
    fmt.Printf("AI回复: %s\n", llmResponse)
}

func callLLM(prompt string) string {
    // 调用LLM API
    return "这是AI的回复"
}

func generateSummary(history string) string {
    // 调用LLM生成摘要
    return "这是对话摘要"
}
```

### 异步摘要流程

```go
// 异步生成摘要（推荐）
if ctx.ShouldCheckpointSummary {
    go func() {
        // 1. 生成摘要
        summary := generateSummary(ctx.FullHistory)
        
        // 2. 创建检查点
        err := app.CheckpointShortMemory(ctx.ConversationID, summary, 8)
        if err != nil {
            // 记录错误，但不阻塞主流程
            log.Printf("CheckpointShortMemory error: %v", err)
        }
    }()
}
```

### 同步摘要流程

```go
// 同步生成摘要（不推荐，会阻塞）
if ctx.ShouldCheckpointSummary {
    // 1. 生成摘要
    summary := generateSummary(ctx.FullHistory)
    
    // 2. 创建检查点
    err := app.CheckpointShortMemory(ctx.ConversationID, summary, 8)
    if err != nil {
        return err
    }
}
```

---

## 配置参数

### 默认参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| defaultMemoryTokenThresholdRatio | 0.75 | Token阈值（75%） |
| defaultMemoryRecentTurns | 8 | 最近轮次（8轮） |
| defaultModelContextWindow | 16000 | 模型上下文窗口（16k） |

### 自定义参数

```go
// 方式1：通过请求参数自定义
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv-123",
    TokenThresholdRatio: 0.8,  // 自定义阈值
    RecentTurns:         10, // 自定义轮次
    ModelContextWindow:  32000, // 自定义窗口
})

// 方式2：修改默认值（需要修改源码）
const (
    defaultMemoryTokenThresholdRatio = 0.8
    defaultMemoryRecentTurns         = 10
    defaultModelContextWindow        = 32000
)
```

---

## 性能优化

### 1. Redis缓存

**当前实现：**
- SessionValue存储在Redis中
- 过期时间：30分钟
- 读取速度：~1-2ms

**优化建议：**
```go
// 增加checkpoint缓存
type MessageContext struct {
    Summary            string     `json:"summary"`
    WindowMessages     []*Message `json:"window_messages"`
    Mode               string     `json:"mode,omitempty"`
    CheckpointMessageID string     `json:"checkpoint_message_id,omitempty"`
    CachedCheckpoint   string     `json:"cached_checkpoint,omitempty"` // 新增
}
```

### 2. 批量读取

**当前实现：**
- 每次只读取当前段的消息
- 可能产生多次数据库查询

**优化建议：**
```go
// 一次性读取一批消息
messages, err := a.QueryMessageByConversationIDASCWithLimit(conversationID, 100)
```

### 3. 异步摘要

**当前实现：**
- 可以通过goroutine实现异步摘要

**最佳实践：**
```go
// 异步生成摘要，不阻塞主流程
if ctx.ShouldCheckpointSummary {
    go func() {
        summary := generateSummaryAsync(ctx.FullHistory)
        app.CheckpointShortMemory(ctx.ConversationID, summary, 8)
    }()
}
```

---

## 注意事项

### 1. 错误处理

```go
// 检查checkpoint是否成功
if err := app.CheckpointShortMemory(conversationID, summary, 8); err != nil {
    // 失败处理：
    // 1. 记录日志
    log.Printf("CheckpointShortMemory error: %v", err)
    
    // 2. 重试机制（可选）
    retryCount := 0
    for retryCount < 3 {
        err = app.CheckpointShortMemory(conversationID, summary, 8)
        if err == nil {
            break
        }
        retryCount++
        time.Sleep(time.Second * time.Duration(retryCount))
    }
    
    // 3. 降级处理
    if err != nil {
        // 降级到全量历史模式
        session.MessageContext.Mode = MemoryModeFullHistory
        app.SetShortMemory(conversationID, session)
    }
}
```

### 2. 兼容性处理

```go
// 检查旧数据是否包含CheckpointMessageID
session, err := app.GetShortMemory(conversationID)
if err != nil {
    // 处理错误
}

if session.MessageContext.CheckpointMessageID == "" {
    // 旧数据，使用全量历史模式
    session.MessageContext.Mode = MemoryModeFullHistory
    app.SetShortMemory(conversationID, session)
}
```

### 3. 监控指标

建议监控以下指标：
- checkpoint创建频率
- 摘要长度分布
- token占用率分布
- DB查询耗时
- Redis读取耗时

### 4. 并发控制

```go
// 使用sync.Mutex保护共享资源
var memoryMutex sync.Mutex

func handleUserMessage(app *AgentApp, conversationID, userMessage string) {
    memoryMutex.Lock()
    defer memoryMutex.Unlock()
    
    // 查询记忆上下文
    ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
        ConversationID: conversationID,
        Query:          userMessage,
    })
    
    // ... 处理逻辑
}
```

---

## 常见问题

### Q1: 每次都要读数据库吗？

**A:** 
- **Redis读取**：每次都要读Redis获取SessionValue（包含CheckpointMessageID）
- **DB读取**：每次都要读DB查询当前段的消息

**性能分析：**
- Redis读：~1-2ms（内存操作）
- DB读：~10-50ms（取决于数据量和索引）

**优化建议：**
- 增加Redis缓存checkpoint内容
- 实现批量读取
- 异步摘要生成

### Q2: checkpoint支持Redis吗？

**A:** 当前checkpoint消息只存储在数据库中，但SessionValue存储在Redis中。

**优化方案：**
可以将checkpoint内容也存储在Redis中，提升读取速度。

### Q3: 如何查看checkpoint历史？

**A:** 可以通过SQL查询：

```sql
-- 查询所有checkpoint历史
SELECT message_id, conversation_id, answer, create_time 
FROM ai_message 
WHERE conversation_id = 'conv-123' 
  AND query = '[MEMORY_CHECKPOINT]'
ORDER BY create_time ASC;

-- 查询当前段的消息
SELECT message_id, conversation_id, query, answer, create_time 
FROM ai_message 
WHERE conversation_id = 'conv-123' 
  AND create_time > (
    SELECT create_time FROM ai_message 
    WHERE message_id = 'checkpoint-message-id'
  )
ORDER BY create_time ASC;
```

### Q4: 如何调整token阈值？

**A:** 可以通过请求参数调整：

```go
ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
    ConversationID: "conv-123",
    Query:          userMessage,
    TokenThresholdRatio: 0.8,  // 调整为80%
})
```

### Q5: SUMMARY_N模式下还会触发摘要吗？

**A:** 会的！无论什么模式，只要token超过阈值就会触发摘要。

这是本次重构的重要改进，解决了之前SUMMARY_N模式下不触发摘要的问题。

### Q6: 如何手动触发摘要？

**A:** 可以直接调用 `CheckpointShortMemory` 方法：

```go
summary := "这是手动生成的摘要"
err := app.CheckpointShortMemory("conv-123", summary, 8)
```

### Q7: 如何重置对话历史？

**A:** 可以清空Redis中的SessionValue：

```go
// 方式1：删除Redis key
client, _ := app.GetRedisClient()
key := fmt.Sprintf("short_term_memory:session:%s", conversationID)
client.Del(key)

// 方式2：重新创建
app.CreateShortMemory(&AgentRequest{
    ConversationId: conversationID,
    UserId:         userID,
})
```

---

## 更新日志

### v2.0 (2026-02-09)
- ✅ 新增 CheckpointMessageID 字段实现分段记忆管理
- ✅ 新增 QueryMessageByConversationIDASCFromCheckpoint 方法
- ✅ 修复 SUMMARY_N 模式下不触发摘要的漏洞
- ✅ 修复 token 计算不准确的问题
- ✅ 支持无论什么模式，只要token超过阈值就触发摘要

---

## 联系方式
- 技术支持：AI团队
- 问题反馈：GitHub Issues
