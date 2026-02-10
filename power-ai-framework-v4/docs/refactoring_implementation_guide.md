# Power AI Framework V4 - 主文件重构实施指南

> **版本**: v4.0.0
> **创建时间**: 2026-01-26
> **状态**: 🔄 进行中

---

## 📋 重构目标

简化 `powerai_short_memory.go` 和 `powerai_memory.go`，移除重复代码，使用已创建的工具类。

---

## 🔍 需要修改的文件

### 1. powerai_short_memory.go

#### 需要移除的内容：

1. **会话级并发锁管理**
   ```go
   var sessionLocks sync.Map
   
   func getSessionLock(conversationID string) *sync.Mutex
   ```

2. **normalizeSessionValue 函数**
   ```go
   func normalizeSessionValue(session *SessionValue) *SessionValue
   ```

3. **常量定义**
   ```go
   const (
       ShortMemorySessionKeyPrefix = "short_term_memory:session:%s"
       expiration = 30 * 60
       MemoryModeFullHistory = "FULL_HISTORY"
       MemoryModeSummaryN = "SUMMARY_N"
   )
   ```

#### 需要保留的内容：

- 数据结构定义（SessionValue, MetaInfo, FlowContext, MessageContext, Message, GlobalState, SharedEntities, PendingAction, UserProfile）
- newDefaultSessionValue 函数
- CreateShortMemory, GetShortMemory, SetShortMemory 函数

#### 需要修改的内容：

- 在所有使用 `getSessionLock` 的地方，改为使用 `a.sessionLockMgr.GetLock()`
- 在所有使用 `normalizeSessionValue` 的地方，改为使用 `a.sessionNormalizer.Normalize()`
- 在所有使用常量的地方，改为使用 `a.memoryConfig.*`

---

### 2. powerai_memory.go

#### 需要移除的内容：

1. **常量定义**
   ```go
   const (
       defaultMemoryTokenThresholdRatio = 0.75
       defaultMemoryRecentTurns = 8
       defaultModelContextWindow = 16000
       maxQueryLength = 10000
       maxResponseLength = 50000
       maxUserIDLength = 100
       maxAgentCodeLength = 50
       maxSummaryLength = 2000
   )
   ```

2. **验证函数**
   ```go
   func isValidAgentCode(code string) bool
   func isDuplicateKeyError(err error) bool
   func isValidUUID(uuid string) bool
   ```

3. **消息构建函数**
   ```go
   func buildHistoryFromAIMessages(messages []*AIMessage) string
   func composeSummaryAndRecent(session *SessionValue) string
   func buildRecentMessages(messages []*AIMessage, recentTurns int) []*Message
   func extractAgentAnswer(answer string) string
   func estimateTokenCount(text string) int
   ```

4. **辅助函数**
   ```go
   func applyMemoryQueryDefaults(req *MemoryQueryRequest) (float64, int, int)
   ```

#### 需要保留的内容：

- 数据结构定义（MemoryQueryRequest, MemoryContext, MemoryWriteRequest, MemoryWriteResult, SessionFinalizeRequest, MedicalFact, UserPreferenceMemory, FactUpsertRequest, PreferenceUpsertRequest）
- 核心API函数（QueryMemoryContext, WriteTurn, CheckpointShortMemory, FinalizeSessionMemory, UpsertFacts, UpsertPreferences）
- checkMessageIDExists 方法

#### 需要修改的内容：

- 在所有使用常量的地方，改为使用 `a.memoryConfig.*`
- 在所有使用验证函数的地方，改为使用 `a.sessionNormalizer.*`
- 在所有使用消息构建函数的地方，改为使用 `a.messageBuilder.*`

---

### 3. powerai.go

#### 需要添加的内容：

1. **导入语句**
   ```go
   import (
       "orgine.com/ai-team/power-ai-framework-v4/pkg/xconfig"
       "orgine.com/ai-team/power-ai-framework-v4/pkg/xdefense"
       "orgine.com/ai-team/power-ai-framework-v4/pkg/xlock"
       "orgine.com/ai-team/power-ai-framework-v4/pkg/xmemory"
       "orgine.com/ai-team/power-ai-framework-v4/pkg/xinit"
   )
   ```

2. **AgentApp 结构体字段**
   ```go
   type AgentApp struct {
       Manifest    *Manifest
       HttpServer  *server.HttpServer
       OnShutdown  func(ctx context.Context)
       etcd        *etcd_mw.Etcd
       pgsql       *pgsql_mw.PgSql
       redis       *redis_mw.Redis
       minio       *minio_mw.Minio
       weaviate    *weaviate_mw.Weaviate
       milvus      *milvus_mw.Milvus
       agentConfig *AgentConfig
       agentClient *AgentClient
       mu          sync.Mutex
       
       // 新增：记忆管理相关字段
       memoryConfig     *xconfig.MemoryConfig
       sessionLockMgr   *xlock.SessionLockManager
       sessionNormalizer *xdefense.SessionNormalizer
       messageBuilder   *xmemory.MessageBuilder
   }
   ```

3. **NewAgent 函数中的初始化逻辑**
   ```go
   func NewAgent(manifest string, opts ...Option) (*AgentApp, error) {
       // ... 现有初始化代码 ...
       
       // 初始化记忆管理工具类
       memoryInitResult := xinit.InitMemoryManager()
       if memoryInitResult.Error != nil {
           xlog.LogWarnF("INIT", "NewAgent", "InitMemoryManager",
               fmt.Sprintf("failed to init memory manager: %v", memoryInitResult.Error))
           // 使用默认配置
           memoryInitResult.Config = xconfig.GetConfig()
           memoryInitResult.LockManager = xlock.NewSessionLockManager()
           memoryInitResult.MessageBuilder = xmemory.NewMessageBuilder(200, 100)
       }
       
       a := &AgentApp{
           // ... 现有字段 ...
           memoryConfig:     memoryInitResult.Config,
           sessionLockMgr:   memoryInitResult.LockManager,
           sessionNormalizer: xdefense.NewSessionNormalizer(memoryInitResult.Config.MemoryModeFullHistory),
           messageBuilder:   memoryInitResult.MessageBuilder,
       }
       
       // ... 其他初始化代码 ...
       
       return a, nil
   }
   ```

---

## 📝 重构步骤

### 步骤 1：修改 powerai.go

1. 添加导入语句
2. 在 AgentApp 结构体中添加记忆管理相关字段
3. 在 NewAgent 函数中添加初始化逻辑

### 步骤 2：修改 powerai_short_memory.go

1. 移除 `sessionLocks sync.Map`
2. 移除 `getSessionLock` 函数
3. 移除 `normalizeSessionValue` 函数
4. 移除常量定义
5. 修改所有使用这些函数和常量的地方

### 步骤 3：修改 powerai_memory.go

1. 移除所有常量定义
2. 移除所有验证函数
3. 移除所有消息构建函数
4. 移除辅助函数
5. 修改所有使用这些函数和常量的地方

---

## 🎯 预期效果

| 指标 | 重构前 | 重构后 | 改善 |
|------|--------|--------|------|
| powerai_memory.go | 600+ 行 | ~400 行 | -33% |
| powerai_short_memory.go | 500+ 行 | ~300 行 | -40% |
| 代码耦合度 | 高 | 低 | 显著改善 |
| 可维护性 | 中 | 高 | 显著改善 |

---

## ⚠️ 注意事项

1. **向后兼容**
   - 确保重构后 API 接口保持不变
   - 不影响现有调用代码

2. **充分测试**
   - 重构后需要进行充分的单元测试
   - 进行集成测试确保功能正常

3. **逐步推进**
   - 建议先完成工具类的创建和测试
   - 再逐步简化主文件

---

**文档结束**
