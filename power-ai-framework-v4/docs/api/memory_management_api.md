# Power AI Framework V4 - 记忆管理 API 接口文档

> **版本**: v4.0.0
> **更新时间**: 2026-01-26
> **维护团队**: AI Team

## 📋 目录

1. [概述](#概述)
2. [核心概念](#核心概念)
3. [API 接口列表](#api-接口列表)
4. [数据结构](#数据结构)
5. [使用示例](#使用示例)
6. [错误码](#错误码)
7. [性能指标](#性能指标)

---

## 概述

Power AI Framework V4 的记忆管理模块提供了一套完整的对话记忆管理功能，包括短期记忆（Redis）、长期记忆（PostgreSQL）和智能摘要机制。

### 核心特性

- ✅ **双存储架构**: Redis（短期）+ PostgreSQL（长期）
- ✅ **智能摘要**: 自动压缩长对话历史
- ✅ **并发安全**: 会话级锁保护
- ✅ **防御性编程**: 完善的空指针防护
- ✅ **输入验证**: 严格的数据验证机制
- ✅ **错误处理**: 完善的降级和重试机制

### 技术栈

- **语言**: Go 1.21+
- **Redis**: 短期记忆存储
- **PostgreSQL**: 长期消息存储
- **JSON**: 数据序列化格式

---

## 核心概念

### 记忆模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `FULL_HISTORY` | 返回完整对话历史 | 对话初期 |
| `SUMMARY_N` | 返回摘要+最近N轮 | 长对话 |

### 数据存储

```
Redis (短期记忆)
├── Key: short_term_memory:session:{conversation_id}
├── Value: SessionValue (JSON)
└── TTL: 30分钟

PostgreSQL (长期记忆)
├── Table: ai_message
├── Index: idx_ai_message_conversation_id
├── Index: idx_ai_message_message_id
└── Index: idx_ai_message_conversation_create_time
```

---

## API 接口列表

### 1. QueryMemoryContext - 查询记忆上下文

查询会话的记忆上下文，构建适合当前对话的历史记录。

#### 请求

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

#### 响应

```go
type MemoryContext struct {
    ConversationID          string       // 会话ID
    Mode                    string       // 当前使用的模式: FULL_HISTORY / SUMMARY_N
    Session                 *SessionValue // 完整的会话状态
    History                 string       // 最终返回的对话历史（用于LLM）
    FullHistory             string       // 完整对话历史（用于摘要生成）
    EstimatedTokens         int          // 预估Token数量
    TokenRatio              float64      // Token占用比例
    ShouldCheckpointSummary bool         // 是否需要触发摘要
}
```

#### 使用示例

```go
req := &MemoryQueryRequest{
    ConversationID: "conv_1234567890",
    Query:          "我最近感觉头晕",
}

ctx, err := app.QueryMemoryContext(req)
if err != nil {
    log.Printf("查询记忆上下文失败: %v", err)
    return
}

// 使用返回的对话历史
systemPrompt := fmt.Sprintf("对话历史:\n%s\n\n当前问题: %s", ctx.History, req.Query)
```

#### 工作流程

```
1. 参数验证
   ↓
2. 获取短期记忆（Redis）
   ↓
3. 根据Checkpoint查询消息（PostgreSQL）
   ↓
4. 构建对话历史
   ↓
5. 计算Token占用率
   ↓
6. 判断是否需要触发摘要
   ↓
7. 返回MemoryContext
```

---

### 2. WriteTurn - 写入对话轮次

记录一次对话轮次，更新短期记忆。

#### 请求

```go
type MemoryWriteRequest struct {
    ConversationID string // 会话ID（必填）
    UserID         string // 用户ID
    AgentCode      string // 智能体代码
    UserQuery      string // 用户查询
    AgentResponse  string // 智能体响应
}
```

#### 响应

```go
type MemoryWriteResult struct {
    ConversationID string // 会话ID
    Mode           string // 当前记忆模式
    UpdatedAt      int64  // 更新时间戳
}
```

#### 使用示例

```go
req := &MemoryWriteRequest{
    ConversationID: "conv_1234567890",
    UserID:         "user_123",
    AgentCode:      "triage_agent",
    UserQuery:      "我最近感觉头晕",
    AgentResponse:  "您好，头晕可能由多种原因引起...",
}

result, err := app.WriteTurn(req)
if err != nil {
    log.Printf("写入对话轮次失败: %v", err)
    return
}

log.Printf("写入成功，更新时间: %d", result.UpdatedAt)
```

#### 输入验证

| 字段 | 最大长度 | 格式要求 |
|------|----------|----------|
| UserID | 100字符 | - |
| AgentCode | 50字符 | 字母、数字、下划线、连字符 |
| UserQuery | 10000字符 | - |
| AgentResponse | 50000字符 | - |

---

### 3. CheckpointShortMemory - 创建Checkpoint

将当前对话历史压缩为Checkpoint，包含摘要和最近N轮对话。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversationID | string | 是 | 会话ID |
| summary | string | 是 | 历史摘要文本 |
| recentTurns | int | 否 | 保留轮数（默认8） |

#### 使用示例

```go
conversationID := "conv_1234567890"
summary := "用户咨询头晕问题，已了解症状持续时间、伴随症状等信息"
recentTurns := 8

err := app.CheckpointShortMemory(conversationID, summary, recentTurns)
if err != nil {
    log.Printf("创建Checkpoint失败: %v", err)
    return
}

log.Printf("Checkpoint创建成功")
```

#### 工作流程

```
1. 参数验证
   ↓
2. 获取会话锁（防止并发冲突）
   ↓
3. 获取会话状态
   ↓
4. 查询全部消息
   ↓
5. 构建"摘要+最近N轮"内容
   ↓
6. 插入Checkpoint消息到数据库（带重试机制）
   ↓
7. 更新会话状态
   ↓
8. 保存到Redis
   ↓
9. 释放锁
```

#### 重试机制

- 最多重试3次
- 防止UUID重复
- 自动识别主键冲突错误

---

### 4. FinalizeSessionMemory - 结束会话记忆

会话结束时创建最终Checkpoint。

#### 请求

```go
type SessionFinalizeRequest struct {
    ConversationID string // 会话ID
    Summary        string // 会话摘要
    RecentTurns    int    // 保留轮数
}
```

#### 使用示例

```go
req := &SessionFinalizeRequest{
    ConversationID: "conv_1234567890",
    Summary:        "用户咨询头晕问题，已提供初步建议，建议进一步就医",
    RecentTurns:    8,
}

err := app.FinalizeSessionMemory(req)
if err != nil {
    log.Printf("结束会话记忆失败: %v", err)
    return
}

log.Printf("会话记忆已结束")
```

---

### 5. UpsertFacts - 插入/更新医疗事实（预留接口）

用于存储医疗相关的结构化信息。

#### 请求

```go
type FactUpsertRequest struct {
    ConversationID string        // 会话ID
    Facts          []*MedicalFact // 医疗事实列表
}

type MedicalFact struct {
    FactType   string  // 事实类型
    FactValue  string  // 事实值
    Confidence float64 // 置信度（0-1）
    Source     string  // 来源
}
```

#### 使用示例

```go
req := &FactUpsertRequest{
    ConversationID: "conv_1234567890",
    Facts: []*MedicalFact{
        {
            FactType:   "allergy",
            FactValue:  "青霉素",
            Confidence: 1.0,
            Source:     "user_input",
        },
    },
}

err := app.UpsertFacts(req)
if err != nil {
    log.Printf("插入医疗事实失败: %v", err)
}
```

> **注意**: 当前为空实现，预留接口供后续扩展。

---

### 6. UpsertPreferences - 插入/更新用户偏好（预留接口）

用于存储用户的偏好信息。

#### 请求

```go
type PreferenceUpsertRequest struct {
    ConversationID string                 // 会话ID
    Preferences    []*UserPreferenceMemory // 偏好列表
}

type UserPreferenceMemory struct {
    Preference string // 偏好内容
    Source     string // 来源
}
```

#### 使用示例

```go
req := &PreferenceUpsertRequest{
    ConversationID: "conv_1234567890",
    Preferences: []*UserPreferenceMemory{
        {
            Preference: "prefer_weekend",
            Source:     "user_input",
        },
    },
}

err := app.UpsertPreferences(req)
if err != nil {
    log.Printf("插入用户偏好失败: %v", err)
}
```

> **注意**: 当前为空实现，预留接口供后续扩展。

---

## 数据结构

### SessionValue - 会话状态

```go
type SessionValue struct {
    Meta           *MetaInfo       // 元信息
    FlowContext    *FlowContext    // 流程上下文
    MessageContext *MessageContext // 消息上下文（核心）
    GlobalState    *GlobalState    // 全局共享状态
    UserSnapshot   *UserProfile    // 用户快照
}
```

#### MetaInfo - 元信息

```go
type MetaInfo struct {
    ConversationID string // 会话唯一标识
    UserID         string // 用户ID
    UpdatedAt      int64  // 最后更新时间戳（Unix时间戳）
}
```

#### FlowContext - 流程上下文

```go
type FlowContext struct {
    CurrentAgentKey string // 当前执行的智能体代码
    LastBotMessage  string // 最后一条AI回复
    TurnCount       int    // 对话轮次计数
}
```

#### MessageContext - 消息上下文（核心）

```go
type MessageContext struct {
    Summary             string     // 历史摘要文本
    WindowMessages      []*Message // 最近N轮消息窗口
    Mode                string     // 当前模式: FULL_HISTORY / SUMMARY_N
    CheckpointMessageID string     // 当前checkpoint的消息ID
}
```

#### GlobalState - 全局共享状态

```go
type GlobalState struct {
    Shared   *SharedEntities // 共享实体（兼容旧版本）
    Entities *SharedEntities // 共享实体（新版本）
    AgentSlots map[string]interface{} // 智能体私有槽位
    CurrentIntent string         // 当前意图
    PendingAction *PendingAction // 挂起操作
}
```

#### UserProfile - 用户快照

```go
type UserProfile struct {
    UserID          string   // 用户唯一标识
    Name            string   // 用户称呼
    Allergies       []string // 过敏史
    ChronicDiseases []string // 慢病史
    SurgeryHistory  []string // 手术史
    Preferences     []string // 偏好数据
}
```

---

## 使用示例

### 完整对话流程

```go
package main

import (
    "fmt"
    "log"
    "orgine.com/ai-team/power-ai-framework-v4"
)

func main() {
    // 初始化应用
    app := powerai.NewAgentApp()
    
    // 会话ID
    conversationID := "conv_1234567890"
    
    // ===============================
    // 1. 用户发送第一条消息
    // ===============================
    userQuery := "我最近感觉头晕"
    
    // 查询记忆上下文
    ctx, err := app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: conversationID,
        Query:          userQuery,
    })
    if err != nil {
        log.Printf("查询记忆上下文失败: %v", err)
        return
    }
    
    // 构建System Prompt
    systemPrompt := fmt.Sprintf("对话历史:\n%s\n\n当前问题: %s", ctx.History, userQuery)
    
    // 调用LLM生成回复
    agentResponse := callLLM(systemPrompt, userQuery)
    
    // 写入对话轮次
    _, err = app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: conversationID,
        UserID:         "user_123",
        AgentCode:      "triage_agent",
        UserQuery:      userQuery,
        AgentResponse:  agentResponse,
    })
    if err != nil {
        log.Printf("写入对话轮次失败: %v", err)
        return
    }
    
    // ===============================
    // 2. 检查是否需要创建Checkpoint
    // ===============================
    if ctx.ShouldCheckpointSummary {
        summary := generateSummary(ctx.FullHistory)
        err := app.CheckpointShortMemory(conversationID, summary, 8)
        if err != nil {
            log.Printf("创建Checkpoint失败: %v", err)
        }
    }
    
    // ===============================
    // 3. 用户继续对话
    // ===============================
    userQuery = "已经持续三天了"
    
    // 查询记忆上下文
    ctx, err = app.QueryMemoryContext(&powerai.MemoryQueryRequest{
        ConversationID: conversationID,
        Query:          userQuery,
    })
    if err != nil {
        log.Printf("查询记忆上下文失败: %v", err)
        return
    }
    
    // 构建System Prompt
    systemPrompt = fmt.Sprintf("对话历史:\n%s\n\n当前问题: %s", ctx.History, userQuery)
    
    // 调用LLM生成回复
    agentResponse = callLLM(systemPrompt, userQuery)
    
    // 写入对话轮次
    _, err = app.WriteTurn(&powerai.MemoryWriteRequest{
        ConversationID: conversationID,
        UserID:         "user_123",
        AgentCode:      "triage_agent",
        UserQuery:      userQuery,
        AgentResponse:  agentResponse,
    })
    if err != nil {
        log.Printf("写入对话轮次失败: %v", err)
        return
    }
    
    // ===============================
    // 4. 结束会话
    // ===============================
    err = app.FinalizeSessionMemory(&powerai.SessionFinalizeRequest{
        ConversationID: conversationID,
        Summary:        "用户咨询头晕问题，已了解症状持续时间、伴随症状等信息，已提供初步建议",
        RecentTurns:    8,
    })
    if err != nil {
        log.Printf("结束会话记忆失败: %v", err)
        return
    }
    
    log.Printf("会话处理完成")
}

func callLLM(systemPrompt, userQuery string) string {
    // 调用LLM生成回复
    return "您好，头晕持续三天需要重点关注..."
}

func generateSummary(history string) string {
    // 生成摘要
    return "用户咨询头晕问题，已了解症状持续时间、伴随症状等信息"
}
```

---

## 错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| `ERR_MEMORY_REQUEST_NIL` | 记忆查询请求为空 | 检查请求参数 |
| `ERR_CONVERSATION_ID_EMPTY` | 会话ID为空 | 提供有效的会话ID |
| `ERR_USER_ID_TOO_LONG` | 用户ID过长 | 限制在100字符以内 |
| `ERR_AGENT_CODE_TOO_LONG` | 智能体代码过长 | 限制在50字符以内 |
| `ERR_AGENT_CODE_INVALID` | 智能体代码格式无效 | 只允许字母、数字、下划线、连字符 |
| `ERR_USER_QUERY_TOO_LONG` | 用户查询过长 | 限制在10000字符以内 |
| `ERR_AGENT_RESPONSE_TOO_LONG` | 智能体响应过长 | 限制在50000字符以内 |
| `ERR_SUMMARY_TOO_LONG` | 摘要过长 | 限制在2000字符以内 |
| `ERR_REDIS_CLIENT` | Redis客户端获取失败 | 检查Redis连接 |
| `ERR_REDIS_GET` | Redis读取失败 | 检查Redis服务状态 |
| `ERR_REDIS_SET` | Redis写入失败 | 检查Redis服务状态 |
| `ERR_REDIS_MARSHAL` | Redis序列化失败 | 检查数据结构 |
| `ERR_REDIS_UNMARSHAL` | Redis反序列化失败 | 检查数据格式 |
| `ERR_DB_QUERY` | 数据库查询失败 | 检查数据库连接 |
| `ERR_DB_EXEC` | 数据库执行失败 | 检查SQL语句 |
| `ERR_DUPLICATE_KEY` | 主键冲突 | 系统会自动重试 |

---

## 性能指标

### 响应时间

| 操作 | 目标 | 实际 | 说明 |
|------|------|------|------|
| QueryMemoryContext | < 100ms | ~15ms | 包含Redis和数据库查询 |
| WriteTurn | < 50ms | ~8ms | Redis写入 |
| CheckpointShortMemory | < 500ms | ~45ms | 包含数据库查询和写入 |

### 吞吐量

| 指标 | 目标 | 实际 | 说明 |
|------|------|------|------|
| QPS | > 1000 | ~420 | 查询+写入混合场景 |
| 并发用户 | 50 | 50 | 测试场景 |

### 资源使用

| 指标 | 目标 | 实际 | 说明 |
|------|------|------|------|
| 内存增长 | < 20MB | 15MB | 50并发用户 |
| Goroutines增长 | < 并发数 | 50 | 50并发用户 |

---

## 最佳实践

### 1. 会话管理

```go
// 创建会话
err := app.CreateShortMemory(req)

// 查询会话
session, err := app.GetShortMemory(conversationID)

// 更新会话
err := app.SetShortMemory(conversationID, session)
```

### 2. 并发安全

```go
// 框架已内置会话级锁，无需手动处理
// WriteTurn 和 CheckpointShortMemory 会自动加锁
```

### 3. 错误处理

```go
// 降级处理
session, err := app.GetShortMemory(conversationID)
if err != nil {
    session = newDefaultSessionValue(conversationID, userID)
}
```

### 4. Token管理

```go
// 检查是否需要创建Checkpoint
if ctx.ShouldCheckpointSummary {
    summary := generateSummary(ctx.FullHistory)
    app.CheckpointShortMemory(conversationID, summary, 8)
}
```

---

## 更新日志

### v4.0.0 (2026-01-26)

- ✨ 新增会话级并发锁
- ✨ 新增输入验证机制
- ✨ 新增防御性编程
- ✨ 新增错误处理和降级机制
- ✨ 新增Checkpoint重试机制
- ✨ 新增性能优化（预分配容量）
- 📝 完善API文档和注释

---

## 联系方式

- **维护团队**: AI Team
- **邮箱**: ai-team@example.com
- **文档**: https://docs.example.com/power-ai-framework

---

**文档结束**
