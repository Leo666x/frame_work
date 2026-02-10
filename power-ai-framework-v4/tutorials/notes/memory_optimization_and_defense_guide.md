# 短期记忆优化与防御性编程指南

> **目标**: 提高系统并发安全性、数据一致性和性能
> **文档版本**: v1.0
> **更新时间**: 2026-01-26

## 📋 目录

1. [并发风险分析](#并发风险分析)
2. [数据库操作优化](#数据库操作优化)
3. [防御性编程改进](#防御性编程改进)
4. [性能优化建议](#性能优化建议)
5. [代码重构方案](#代码重构方案)

---

## 并发风险分析

### 🔴 高风险：Redis 并发写入冲突

**问题位置**: `powerai_short_memory.go`

#### 风险场景

1. **WriteTurn 并发写入**
```go
// 当前代码（无锁保护）
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    session, err := a.GetShortMemory(req.ConversationID)  // 读取
    // ... 修改 session ...
    session.FlowContext.TurnCount++  // 竞态条件！
    return a.SetShortMemory(req.ConversationID, session)  // 写入
}
```

**问题**:
- 两个并发请求同时读取同一会话
- 都读取 TurnCount = 5
- 都执行 TurnCount++，都变成 6
- 最终丢失一次计数

2. **Checkpoint 并发写入**
```go
// 当前代码（无锁保护）
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    session, err := a.GetShortMemory(conversationID)  // 读取
    // ... 修改 session ...
    session.MessageContext.CheckpointMessageID = checkpointMessageID  // 竞态条件！
    return a.SetShortMemory(conversationID, session)  // 写入
}
```

**问题**:
- 两个并发请求同时触发 checkpoint
- 都创建不同的 checkpointMessageID
- 最终只有一个生效，另一个丢失

#### 解决方案：添加会话级锁

```go
// powerai_short_memory.go

import "sync"

// 添加会话级锁映射
var sessionLocks sync.Map  // map[conversationID]*sync.Mutex

// 获取会话锁
func getSessionLock(conversationID string) *sync.Mutex {
    lock, _ := sessionLocks.LoadOrStore(conversationID, &sync.Mutex{})
    return lock.(*sync.Mutex)
}

// 清理过期锁（定时任务）
func cleanupExpiredLocks() {
    sessionLocks.Range(func(key, value interface{}) bool {
        conversationID := key.(string)
        lock := value.(*sync.Mutex)
        
        // 检查会话是否过期
        // 这里需要实现检查逻辑
        
        // 如果过期，删除锁
        sessionLocks.Delete(key)
        return true
    })
}

// 优化后的 WriteTurn
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    if req == nil {
        return nil, fmt.Errorf("memory write request is nil")
    }
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    
    // 获取会话锁
    lock := getSessionLock(req.ConversationID)
    lock.Lock()
    defer lock.Unlock()
    
    // 原有逻辑...
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        session = newDefaultSessionValue(req.ConversationID, req.UserID)
    }
    session = normalizeSessionValue(session)
    
    if req.UserID != "" {
        session.Meta.UserID = req.UserID
        if session.UserSnapshot != nil {
            session.UserSnapshot.UserID = req.UserID
        }
    }
    if req.AgentCode != "" {
        session.FlowContext.CurrentAgentKey = req.AgentCode
    }
    if req.AgentResponse != "" {
        session.FlowContext.LastBotMessage = req.AgentResponse
    }
    session.FlowContext.TurnCount++
    
    if err := a.SetShortMemory(req.ConversationID, session); err != nil {
        return nil, err
    }
    
    return &MemoryWriteResult{
        ConversationID: req.ConversationID,
        Mode:           session.MessageContext.Mode,
        UpdatedAt:      session.Meta.UpdatedAt,
    }, nil
}

// 优化后的 CheckpointShortMemory
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    if conversationID == "" {
        return fmt.Errorf("conversation_id is empty")
    }
    if recentTurns <= 0 {
        recentTurns = defaultMemoryRecentTurns
    }
    
    // 获取会话锁
    lock := getSessionLock(conversationID)
    lock.Lock()
    defer lock.Unlock()
    
    // 原有逻辑...
    session, err := a.GetShortMemory(conversationID)
    if err != nil {
        session = newDefaultSessionValue(conversationID, "")
    }
    session = normalizeSessionValue(session)
    
    messages, err := a.QueryMessageByConversationIDASC(conversationID)
    if err != nil {
        return err
    }
    
    summaryAndRecent := composeSummaryAndRecent(session)
    
    checkpointMessageID := xuid.UUID()
    timeNow := xdatetime.GetNowDateTime()
    
    sql := `INSERT INTO ai_message (message_id, conversation_id, query, answer, create_time, create_by, update_time, update_by) 
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err = a.DBExec(sql, checkpointMessageID, conversationID, "[MEMORY_CHECKPOINT]", summaryAndRecent, timeNow, "system", timeNow, "system")
    if err != nil {
        return fmt.Errorf("failed to insert checkpoint message: %w", err)
    }
    
    session.MessageContext.Summary = strings.TrimSpace(summary)
    session.MessageContext.WindowMessages = buildRecentMessages(messages, recentTurns)
    session.MessageContext.Mode = MemoryModeSummaryN
    session.MessageContext.CheckpointMessageID = checkpointMessageID
    
    return a.SetShortMemory(conversationID, session)
}
```

### 🟡 中风险：数据库查询效率问题

**问题位置**: `powerai_db.go`

#### 问题1：Checkpoint 查询效率低

**当前代码**:
```go
func (a *AgentApp) QueryMessageByConversationIDASCFromCheckpoint(conversationID, checkpointMessageID string) ([]*AIMessage, error) {
    // ...
    sql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field 
            from ai_message 
            where conversation_id = $1 and create_time > (select create_time from ai_message where message_id = $2) 
            ORDER BY create_time ASC`
    // ...
}
```

**问题**:
- 使用子查询获取 checkpoint 的 create_time
- 每次查询都需要执行子查询
- 如果没有索引，性能会很差

**优化方案**:
```go
// 方案1：使用 JOIN 代替子查询
func (a *AgentApp) QueryMessageByConversationIDASCFromCheckpoint(conversationID, checkpointMessageID string) ([]*AIMessage, error) {
    if conversationID == "" {
        return nil, fmt.Errorf("conversationID不能为空")
    }
    if checkpointMessageID == "" {
        return a.QueryMessageByConversationIDASC(conversationID)
    }
    
    client, err := a.GetPgSqlClient()
    if err != nil {
        return nil, err
    }
    
    // 使用 JOIN 优化
    sql := `SELECT m.message_id, m.conversation_id, m.query, m.answer, m.rating, 
                    m.inputs, m.errors, m.agent_code, m.file_id, m.create_time, 
                    m.update_time, m.extended_field
            FROM ai_message m
            INNER JOIN ai_message cp ON m.conversation_id = cp.conversation_id
            WHERE m.conversation_id = $1 
              AND cp.message_id = $2
              AND m.create_time > cp.create_time
            ORDER BY m.create_time ASC`
    
    var r []*AIMessage
    if err := client.QueryMultiple(&r, sql, conversationID, checkpointMessageID); err != nil {
        return nil, err
    }
    
    return r, nil
}

// 方案2：添加索引（需要在数据库层面执行）
/*
-- 为 checkpoint 查询添加复合索引
CREATE INDEX idx_ai_message_conversation_create_time 
ON ai_message(conversation_id, create_time);

-- 为 message_id 添加索引（如果还没有）
CREATE INDEX idx_ai_message_message_id 
ON ai_message(message_id);
*/
```

#### 问题2：重复查询会话信息

**当前代码**:
```go
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    // 1. 查询 Redis
    session, err := a.GetShortMemory(req.ConversationID)
    
    // 2. 查询数据库
    if session.MessageContext.CheckpointMessageID != "" {
        messages, err = a.QueryMessageByConversationIDASCFromCheckpoint(req.ConversationID, session.MessageContext.CheckpointMessageID)
    } else {
        messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
    }
    
    // 3. CheckpointShortMemory 中又查询一次全部消息
    // func CheckpointShortMemory:
    messages, err := a.QueryMessageByConversationIDASC(conversationID)  // 重复查询！
}
```

**问题**:
- `QueryMemoryContext` 查询了部分消息
- `CheckpointShortMemory` 又查询全部消息
- 造成重复数据库查询

**优化方案**:
```go
// 优化后的 CheckpointShortMemory，接收消息列表作为参数
func (a *AgentApp) CheckpointShortMemoryWithMessages(conversationID, summary string, recentTurns int, messages []*AIMessage) error {
    if conversationID == "" {
        return fmt.Errorf("conversation_id is empty")
    }
    if recentTurns <= 0 {
        recentTurns = defaultMemoryRecentTurns
    }
    
    // 获取会话锁
    lock := getSessionLock(conversationID)
    lock.Lock()
    defer lock.Unlock()
    
    session, err := a.GetShortMemory(conversationID)
    if err != nil {
        session = newDefaultSessionValue(conversationID, "")
    }
    session = normalizeSessionValue(session)
    
    // 使用传入的消息列表，避免重复查询
    if messages == nil {
        // 如果没有传入，才查询数据库
        messages, err = a.QueryMessageByConversationIDASC(conversationID)
        if err != nil {
            return err
        }
    }
    
    summaryAndRecent := composeSummaryAndRecent(session)
    
    checkpointMessageID := xuid.UUID()
    timeNow := xdatetime.GetNowDateTime()
    
    sql := `INSERT INTO ai_message (message_id, conversation_id, query, answer, create_time, create_by, update_time, update_by) 
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err = a.DBExec(sql, checkpointMessageID, conversationID, "[MEMORY_CHECKPOINT]", summaryAndRecent, timeNow, "system", timeNow, "system")
    if err != nil {
        return fmt.Errorf("failed to insert checkpoint message: %w", err)
    }
    
    session.MessageContext.Summary = strings.TrimSpace(summary)
    session.MessageContext.WindowMessages = buildRecentMessages(messages, recentTurns)
    session.MessageContext.Mode = MemoryModeSummaryN
    session.MessageContext.CheckpointMessageID = checkpointMessageID
    
    return a.SetShortMemory(conversationID, session)
}

// 优化后的 QueryMemoryContext，传递消息列表
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    if req == nil {
        return nil, fmt.Errorf("memory query request is nil")
    }
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    
    threshold, _, contextWindow := applyMemoryQueryDefaults(req)
    
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        session = newDefaultSessionValue(req.ConversationID, req.PatientID)
    }
    session = normalizeSessionValue(session)
    mode := session.MessageContext.Mode
    
    var messages []*AIMessage
    if session.MessageContext.CheckpointMessageID != "" {
        messages, err = a.QueryMessageByConversationIDASCFromCheckpoint(req.ConversationID, session.MessageContext.CheckpointMessageID)
    } else {
        messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
    }
    if err != nil {
        messages = nil
    }
    
    fullHistory := buildHistoryFromAIMessages(messages)
    
    estimatedTokens := estimateTokenCount(fullHistory + "\n" + req.Query)
    tokenRatio := float64(estimatedTokens) / float64(contextWindow)
    
    history := fullHistory
    if mode == MemoryModeSummaryN {
        history = composeSummaryAndRecent(session)
        if strings.TrimSpace(history) == "" {
            history = fullHistory
            mode = MemoryModeFullHistory
        }
    }
    
    estimatedTokens = estimateTokenCount(history + "\n" + req.Query)
    tokenRatio = float64(estimatedTokens) / float64(contextWindow)
    
    shouldCheckpoint := tokenRatio >= threshold
    
    return &MemoryContext{
        ConversationID:          req.ConversationID,
        Mode:                    mode,
        Session:                 session,
        History:                 history,
        FullHistory:             fullHistory,
        EstimatedTokens:         estimatedTokens,
        TokenRatio:              tokenRatio,
        ShouldCheckpointSummary: shouldCheckpoint,
        Messages:                messages,  // 添加消息列表到返回结果
    }, nil
}
```

### 🟡 中风险：数据库插入重复检查

**问题位置**: `powerai_memory.go:CheckpointShortMemory`

**当前代码**:
```go
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    // ...
    checkpointMessageID := xuid.UUID()  // 生成新ID
    // ...
    sql := `INSERT INTO ai_message ... VALUES ($1, $2, ...)`
    _, err = a.DBExec(sql, checkpointMessageID, ...)
    // ...
}
```

**问题**:
- 虽然使用 UUID 生成 ID，但理论上存在极小概率的冲突
- 没有检查 message_id 是否已存在
- 如果插入失败（如主键冲突），没有重试机制

**优化方案**:
```go
import (
    "errors"
    "database/sql"
)

// 优化后的 CheckpointShortMemory，添加唯一性检查和重试
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    if conversationID == "" {
        return fmt.Errorf("conversation_id is empty")
    }
    if recentTurns <= 0 {
        recentTurns = defaultMemoryRecentTurns
    }
    
    lock := getSessionLock(conversationID)
    lock.Lock()
    defer lock.Unlock()
    
    session, err := a.GetShortMemory(conversationID)
    if err != nil {
        session = newDefaultSessionValue(conversationID, "")
    }
    session = normalizeSessionValue(session)
    
    messages, err := a.QueryMessageByConversationIDASC(conversationID)
    if err != nil {
        return err
    }
    
    summaryAndRecent := composeSummaryAndRecent(session)
    
    // 最多重试3次
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        checkpointMessageID := xuid.UUID()
        timeNow := xdatetime.GetNowDateTime()
        
        // 检查 message_id 是否已存在
        exists, err := a.checkMessageIDExists(checkpointMessageID)
        if err != nil {
            return fmt.Errorf("failed to check message_id existence: %w", err)
        }
        if exists {
            continue  // 重新生成 ID
        }
        
        sql := `INSERT INTO ai_message (message_id, conversation_id, query, answer, create_time, create_by, update_time, update_by) 
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
        _, err = a.DBExec(sql, checkpointMessageID, conversationID, "[MEMORY_CHECKPOINT]", summaryAndRecent, timeNow, "system", timeNow, "system")
        
        if err != nil {
            // 检查是否是主键冲突
            if isDuplicateKeyError(err) {
                continue  // 重新生成 ID 重试
            }
            return fmt.Errorf("failed to insert checkpoint message: %w", err)
        }
        
        // 插入成功，更新 session
        session.MessageContext.Summary = strings.TrimSpace(summary)
        session.MessageContext.WindowMessages = buildRecentMessages(messages, recentTurns)
        session.MessageContext.Mode = MemoryModeSummaryN
        session.MessageContext.CheckpointMessageID = checkpointMessageID
        
        return a.SetShortMemory(conversationID, session)
    }
    
    return fmt.Errorf("failed to generate unique message_id after %d retries", maxRetries)
}

// 检查 message_id 是否已存在
func (a *AgentApp) checkMessageIDExists(messageID string) (bool, error) {
    sql := `SELECT COUNT(*) FROM ai_message WHERE message_id = $1`
    var count int
    err := a.DBQuerySingle(&count, sql, messageID)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// 判断是否是主键冲突错误
func isDuplicateKeyError(err error) bool {
    if err == nil {
        return false
    }
    
    // PostgreSQL 主键冲突错误码
    if strings.Contains(err.Error(), "duplicate key") || 
       strings.Contains(err.Error(), "23505") {
        return true
    }
    
    return false
}
```

---

## 数据库操作优化

### 🔴 高风险：SQL 注入风险

**问题位置**: `powerai_db.go:UpdateMessage`

**当前代码**:
```go
func (a *AgentApp) UpdateMessage(messageID, answer, rating, inputs, errors, agentCode, fileID string) error {
    // 构造动态更新语句
    var setClauses []string
    var values []any
    if answer != "" {
        setClauses = append(setClauses, "answer = ?")
        values = append(values, answer)
    }
    // ...
    
    setClause := ""
    for i, clause := range setClauses {
        if i == 0 {
            setClause = strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", i+1))
        } else {
            setClause += ", " + strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", i+1))
        }
    }
    
    sql := fmt.Sprintf("UPDATE ai_message SET %s WHERE message_id = $%d", setClause, len(setClauses)+1)
    _, err = client.Exec(sql, values...)
    return err
}
```

**问题**:
- 虽然 `answer` 等参数通过占位符传递，但 `messageID` 直接拼接到 SQL 中
- 如果 `messageID` 包含恶意 SQL，可能导致注入

**优化方案**:
```go
func (a *AgentApp) UpdateMessage(messageID, answer, rating, inputs, errors, agentCode, fileID string) error {
    if messageID == "" {
        return fmt.Errorf("messageID不能为空")
    }
    
    // 验证 messageID 格式（只允许 UUID 格式）
    if !isValidUUID(messageID) {
        return fmt.Errorf("invalid messageID format")
    }
    
    // 构造动态更新语句（使用参数化查询）
    var setClauses []string
    var values []any
    paramIndex := 1
    
    if answer != "" {
        setClauses = append(setClauses, fmt.Sprintf("answer = $%d", paramIndex))
        values = append(values, answer)
        paramIndex++
    }
    if rating != "" {
        setClauses = append(setClauses, fmt.Sprintf("rating = $%d", paramIndex))
        values = append(values, rating)
        paramIndex++
    }
    if inputs != "" {
        setClauses = append(setClauses, fmt.Sprintf("inputs = $%d", paramIndex))
        values = append(values, inputs)
        paramIndex++
    }
    if errors != "" {
        setClauses = append(setClauses, fmt.Sprintf("errors = $%d", paramIndex))
        values = append(values, errors)
        paramIndex++
    }
    if agentCode != "" {
        setClauses = append(setClauses, fmt.Sprintf("agent_code = $%d", paramIndex))
        values = append(values, agentCode)
        paramIndex++
    }
    if fileID != "" {
        setClauses = append(setClauses, fmt.Sprintf("file_id = $%d", paramIndex))
        values = append(values, fileID)
        paramIndex++
    }
    
    if len(setClauses) == 0 {
        return fmt.Errorf("no fields to update")
    }
    
    setClauses = append(setClauses, fmt.Sprintf("update_time = $%d", paramIndex))
    values = append(values, xdatetime.GetNowDateTime())
    paramIndex++
    
    values = append(values, messageID)
    
    setClause := strings.Join(setClauses, ", ")
    sql := fmt.Sprintf("UPDATE ai_message SET %s WHERE message_id = $%d", setClause, paramIndex)
    
    client, err := a.GetPgSqlClient()
    if err != nil {
        return err
    }
    
    _, err = client.Exec(sql, values...)
    return err
}

// 验证 UUID 格式
func isValidUUID(uuid string) bool {
    // 简单验证：UUID 应该是 36 个字符
    if len(uuid) != 36 {
        return false
    }
    // 可以添加更严格的验证
    return true
}
```

### 🟡 中风险：数据库连接池配置

**问题**:
- 当前代码没有显示数据库连接池配置
- 高并发时可能出现连接池耗尽

**优化方案**:
```go
// powerai_db.go

import "github.com/jmoiron/sqlx"

// 添加数据库连接池配置
const (
    maxOpenConns     = 25  // 最大打开连接数
    maxIdleConns     = 10  // 最大空闲连接数
    connMaxLifetime  = 5 * time.Minute  // 连接最大生命周期
    connMaxIdleTime  = 1 * time.Minute  // 连接最大空闲时间
    connMaxIdleCount = 5   // 最大空闲连接数
)

func initPgSql(etcd *etcd_mw.Etcd) (*pgsql_mw.PgSql, error) {
    // ... 原有初始化逻辑 ...
    
    // 配置连接池
    db.SetMaxOpenConns(maxOpenConns)
    db.SetMaxIdleConns(maxIdleConns)
    db.SetConnMaxLifetime(connMaxLifetime)
    db.SetConnMaxIdleTime(connMaxIdleTime)
    
    return client, nil
}
```

### 🟡 中风险：事务使用不当

**问题位置**: `powerai_db.go:CreateConversationWithFile`

**当前代码**:
```go
func (a *AgentApp) CreateConversationWithFile(...) (string, string, []string, error) {
    // ...
    if err := client.BatchExecTransaction(sqls); err != nil {
        return "", "", []string{}, err
    }
    // ...
}
```

**问题**:
- 事务失败后，没有记录详细的错误信息
- 没有事务超时控制
- 长事务可能导致锁等待

**优化方案**:
```go
func (a *AgentApp) CreateConversationWithFile(conversationName, userID, channel, channelApp, enterpriseID, query,
    inputs,
    fileID string, fileIDs []string) (string, string, []string, error) {
    
    conversationId := xuid.UUID()
    messageId := xuid.UUID()
    timeNow := xdatetime.GetNowDateTime()
    
    var sqls []*pgsql_mw.TransactionSql
    
    // 添加会话记录
    sqls = append(sqls, &pgsql_mw.TransactionSql{
        SqlStatement: `INSERT INTO ai_conversation (conversation_id, conversation_name, user_id, channel, channel_app, enterprise_id, create_time, create_by, update_time, update_by) 
                       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
        Args: []any{
            conversationId, conversationName, userID, channel, channelApp, enterpriseID, timeNow, "admin", timeNow, "admin",
        },
    })
    
    // 添加消息记录
    sqls = append(sqls, &pgsql_mw.TransactionSql{
        SqlStatement: `INSERT INTO ai_message (message_id,conversation_id,query,inputs,file_id,create_time, create_by, update_time,update_by)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
        Args: []any{
            messageId, conversationId, query, inputs, fileID, timeNow, "admin", timeNow, "admin",
        },
    })
    
    var messageFileIds []string
    for _, fileID := range fileIDs {
        messageFileId := xuid.UUID()
        sqls = append(sqls, &pgsql_mw.TransactionSql{
            SqlStatement: `INSERT INTO ai_message_file (message_file_id,conversation_id,message_id,file_id,create_time, create_by, update_time,update_by)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
            Args: []any{
                messageFileId, conversationId, messageId, fileID, timeNow, "admin", timeNow, "admin",
            },
        })
        messageFileIds = append(messageFileIds, messageFileId)
    }
    
    client, err := a.GetPgSqlClient()
    if err != nil {
        return "", "", nil, fmt.Errorf("failed to get database client: %w", err)
    }
    
    // 执行事务，添加超时控制
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    if err := client.BatchExecTransactionWithContext(ctx, sqls); err != nil {
        // 记录详细的错误信息
        log.Printf("Failed to create conversation: %v", err)
        return "", "", []string{}, fmt.Errorf("failed to create conversation: %w", err)
    }
    
    return conversationId, messageId, messageFileIds, nil
}
```

---

## 防御性编程改进

### 🔴 高风险：空指针和空值检查

**问题位置**: 多处

#### 问题1：normalizeSessionValue 不够健壮

**当前代码**:
```go
func normalizeSessionValue(session *SessionValue) *SessionValue {
    if session == nil {
        return newDefaultSessionValue("", "")
    }
    if session.Meta == nil {
        session.Meta = &MetaInfo{}
    }
    // ... 其他字段检查 ...
    return session
}
```

**问题**:
- 没有检查嵌套指针是否为 nil
- 如果 `session.UserSnapshot` 为 nil，访问 `session.UserSnapshot.UserID` 会 panic

**优化方案**:
```go
func normalizeSessionValue(session *SessionValue) *SessionValue {
    if session == nil {
        return newDefaultSessionValue("", "")
    }
    
    // MetaInfo
    if session.Meta == nil {
        session.Meta = &MetaInfo{}
    }
    // 确保 Meta 字段有默认值
    if session.Meta.ConversationID == "" {
        session.Meta.ConversationID = ""
    }
    
    // FlowContext
    if session.FlowContext == nil {
        session.FlowContext = &FlowContext{}
    }
    
    // MessageContext
    if session.MessageContext == nil {
        session.MessageContext = &MessageContext{}
    }
    if session.MessageContext.Mode == "" {
        session.MessageContext.Mode = MemoryModeFullHistory
    }
    if session.MessageContext.WindowMessages == nil {
        session.MessageContext.WindowMessages = []*Message{}
    }
    
    // GlobalState
    if session.GlobalState == nil {
        session.GlobalState = &GlobalState{}
    }
    // 确保 Shared 和 Entities 同步
    if session.GlobalState.Shared == nil && session.GlobalState.Entities != nil {
        session.GlobalState.Shared = session.GlobalState.Entities
    }
    if session.GlobalState.Entities == nil && session.GlobalState.Shared != nil {
        session.GlobalState.Entities = session.GlobalState.Shared
    }
    if session.GlobalState.AgentSlots == nil {
        session.GlobalState.AgentSlots = make(map[string]interface{})
    }
    
    // UserSnapshot
    if session.UserSnapshot == nil {
        session.UserSnapshot = &UserProfile{}
    }
    // 确保 UserID 有默认值
    if session.UserSnapshot.UserID == "" {
        session.UserSnapshot.UserID = ""
    }
    // 确保切片初始化
    if session.UserSnapshot.Allergies == nil {
        session.UserSnapshot.Allergies = []string{}
    }
    if session.UserSnapshot.ChronicDiseases == nil {
        session.UserSnapshot.ChronicDiseases = []string{}
    }
    if session.UserSnapshot.SurgeryHistory == nil {
        session.UserSnapshot.SurgeryHistory = []string{}
    }
    if session.UserSnapshot.Preferences == nil {
        session.UserSnapshot.Preferences = []string{}
    }
    
    return session
}
```

#### 问题2：buildHistoryFromAIMessages 缺少防御

**当前代码**:
```go
func buildHistoryFromAIMessages(messages []*AIMessage) string {
    if len(messages) == 0 {
        return ""
    }
    var builder strings.Builder
    for _, msg := range messages {
        if msg == nil {
            continue
        }
        userMessage := strings.TrimSpace(msg.Query.String)  // 可能 panic
        agentMessage := extractAgentAnswer(msg.Answer.String)  // 可能 panic
        // ...
    }
    return strings.TrimSpace(builder.String())
}
```

**问题**:
- `msg.Query.String` 可能 panic（如果 Query 是无效的 NullString）
- `msg.Answer.String` 可能 panic

**优化方案**:
```go
func buildHistoryFromAIMessages(messages []*AIMessage) string {
    if len(messages) == 0 {
        return ""
    }
    
    var builder strings.Builder
    for _, msg := range messages {
        if msg == nil {
            continue
        }
        
        // 安全获取用户消息
        var userMessage string
        if msg.Query.Valid {
            userMessage = strings.TrimSpace(msg.Query.String)
        }
        
        // 安全获取智能体消息
        var agentMessage string
        if msg.Answer.Valid {
            agentMessage = extractAgentAnswer(msg.Answer.String)
        }
        
        // 添加用户消息
        if userMessage != "" {
            builder.WriteString("用户: ")
            builder.WriteString(userMessage)
            builder.WriteString("\n")
        }
        
        // 添加智能体消息
        if agentMessage != "" {
            builder.WriteString("AI: ")
            builder.WriteString(agentMessage)
            builder.WriteString("\n")
        }
    }
    
    return strings.TrimSpace(builder.String())
}
```

### 🟡 中风险：错误处理不完善

**问题位置**: 多处

**当前代码**:
```go
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    // ...
    messages, err := a.QueryMessageByConversationIDASC(req.ConversationID)
    if err != nil {
        messages = nil  // 吞掉错误！
    }
    // ...
}
```

**问题**:
- 数据库查询失败时，只是将 messages 设为 nil
- 没有记录错误日志
- 调用者无法区分"没有消息"和"查询失败"

**优化方案**:
```go
import "orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"

func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    if req == nil {
        return nil, fmt.Errorf("memory query request is nil")
    }
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    
    threshold, _, contextWindow := applyMemoryQueryDefaults(req)
    
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        xlog.LogErrorF("MEMORY", "QueryMemoryContext", "GetShortMemory", 
            fmt.Sprintf("failed to get short memory for conversation %s: %v", req.ConversationID, err), err)
        session = newDefaultSessionValue(req.ConversationID, req.PatientID)
    }
    
    session = normalizeSessionValue(session)
    mode := session.MessageContext.Mode
    
    var messages []*AIMessage
    if session.MessageContext.CheckpointMessageID != "" {
        messages, err = a.QueryMessageByConversationIDASCFromCheckpoint(req.ConversationID, session.MessageContext.CheckpointMessageID)
        if err != nil {
            xlog.LogErrorF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASCFromCheckpoint", 
                fmt.Sprintf("failed to query messages from checkpoint %s: %v", session.MessageContext.CheckpointMessageID, err), err)
            // 查询失败，尝试查询全部消息
            messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
            if err != nil {
                xlog.LogErrorF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASC", 
                    fmt.Sprintf("failed to query all messages for conversation %s: %v", req.ConversationID, err), err)
                messages = nil
            }
        }
    } else {
        messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
        if err != nil {
            xlog.LogErrorF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASC", 
                fmt.Sprintf("failed to query messages for conversation %s: %v", req.ConversationID, err), err)
            messages = nil
        }
    }
    
    fullHistory := buildHistoryFromAIMessages(messages)
    
    estimatedTokens := estimateTokenCount(fullHistory + "\n" + req.Query)
    tokenRatio := float64(estimatedTokens) / float64(contextWindow)
    
    history := fullHistory
    if mode == MemoryModeSummaryN {
        history = composeSummaryAndRecent(session)
        if strings.TrimSpace(history) == "" {
            history = fullHistory
            mode = MemoryModeFullHistory
        }
    }
    
    estimatedTokens = estimateTokenCount(history + "\n" + req.Query)
    tokenRatio = float64(estimatedTokens) / float64(contextWindow)
    
    shouldCheckpoint := tokenRatio >= threshold
    
    return &MemoryContext{
        ConversationID:          req.ConversationID,
        Mode:                    mode,
        Session:                 session,
        History:                 history,
        FullHistory:             fullHistory,
        EstimatedTokens:         estimatedTokens,
        TokenRatio:              tokenRatio,
        ShouldCheckpointSummary: shouldCheckpoint,
    }, nil
}
```

### 🟡 中风险：输入验证不足

**当前代码**:
```go
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    if req == nil {
        return nil, fmt.Errorf("memory write request is nil")
    }
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    // 没有验证其他字段！
    // ...
}
```

**问题**:
- 只验证了 ConversationID
- 没有验证 UserID、AgentCode 等字段的格式
- 没有验证 UserQuery 和 AgentResponse 的长度

**优化方案**:
```go
const (
    maxQueryLength   = 10000  // 最大查询长度
    maxResponseLength = 50000 // 最大响应长度
    maxUserIDLength  = 100   // 最大用户ID长度
    maxAgentCodeLength = 50   // 最大智能体代码长度
)

func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    if req == nil {
        return nil, fmt.Errorf("memory write request is nil")
    }
    
    // 验证 ConversationID
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    if !isValidUUID(req.ConversationID) {
        return nil, fmt.Errorf("invalid conversation_id format")
    }
    
    // 验证 UserID
    if req.UserID != "" {
        if len(req.UserID) > maxUserIDLength {
            return nil, fmt.Errorf("user_id too long (max %d characters)", maxUserIDLength)
        }
    }
    
    // 验证 AgentCode
    if req.AgentCode != "" {
        if len(req.AgentCode) > maxAgentCodeLength {
            return nil, fmt.Errorf("agent_code too long (max %d characters)", maxAgentCodeLength)
        }
        // 只允许字母、数字、下划线、连字符
        if !isValidAgentCode(req.AgentCode) {
            return nil, fmt.Errorf("invalid agent_code format")
        }
    }
    
    // 验证 UserQuery
    if len(req.UserQuery) > maxQueryLength {
        return nil, fmt.Errorf("user_query too long (max %d characters)", maxQueryLength)
    }
    
    // 验证 AgentResponse
    if len(req.AgentResponse) > maxResponseLength {
        return nil, fmt.Errorf("agent_response too long (max %d characters)", maxResponseLength)
    }
    
    // 获取会话锁
    lock := getSessionLock(req.ConversationID)
    lock.Lock()
    defer lock.Unlock()
    
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        session = newDefaultSessionValue(req.ConversationID, req.UserID)
    }
    session = normalizeSessionValue(session)
    
    if req.UserID != "" {
        session.Meta.UserID = req.UserID
        if session.UserSnapshot != nil {
            session.UserSnapshot.UserID = req.UserID
        }
    }
    if req.AgentCode != "" {
        session.FlowContext.CurrentAgentKey = req.AgentCode
    }
    if req.AgentResponse != "" {
        session.FlowContext.LastBotMessage = req.AgentResponse
    }
    session.FlowContext.TurnCount++
    
    if err := a.SetShortMemory(req.ConversationID, session); err != nil {
        return nil, err
    }
    
    return &MemoryWriteResult{
        ConversationID: req.ConversationID,
        Mode:           session.MessageContext.Mode,
        UpdatedAt:      session.Meta.UpdatedAt,
    }, nil
}

// 验证 AgentCode 格式
func isValidAgentCode(code string) bool {
    if code == "" {
        return false
    }
    // 只允许字母、数字、下划线、连字符
    for _, c := range code {
        if !((c >= 'a' && c <= 'z') || 
             (c >= 'A' && c <= 'Z') || 
             (c >= '0' && c <= '9') || 
             c == '_' || c == '-') {
            return false
        }
    }
    return true
}
```

---

## 性能优化建议

### 🟡 中风险：字符串拼接性能

**问题位置**: `powerai_memory.go:buildHistoryFromAIMessages`

**当前代码**:
```go
func buildHistoryFromAIMessages(messages []*AIMessage) string {
    var builder strings.Builder
    for _, msg := range messages {
        // ...
        builder.WriteString("用户: ")
        builder.WriteString(userMessage)
        builder.WriteString("\n")
        // ...
    }
    return strings.TrimSpace(builder.String())
}
```

**优化方案**:
```go
func buildHistoryFromAIMessages(messages []*AIMessage) string {
    if len(messages) == 0 {
        return ""
    }
    
    // 预分配容量，减少扩容
    estimatedSize := len(messages) * 200  // 假设每条消息平均200字符
    builder := strings.Builder{}
    builder.Grow(estimatedSize)
    
    for _, msg := range messages {
        if msg == nil {
            continue
        }
        
        var userMessage string
        if msg.Query.Valid {
            userMessage = strings.TrimSpace(msg.Query.String)
        }
        
        var agentMessage string
        if msg.Answer.Valid {
            agentMessage = extractAgentAnswer(msg.Answer.String)
        }
        
        if userMessage != "" {
            builder.WriteString("用户: ")
            builder.WriteString(userMessage)
            builder.WriteString("\n")
        }
        
        if agentMessage != "" {
            builder.WriteString("AI: ")
            builder.WriteString(agentMessage)
            builder.WriteString("\n")
        }
    }
    
    return strings.TrimSpace(builder.String())
}
```

### 🟡 中风险：重复的数据库查询

**问题**: `CheckpointShortMemory` 中查询全部消息，而 `QueryMemoryContext` 已经查询过

**优化方案**: 见前面的"数据库操作优化"部分

### 🟢 低风险：缓存优化

**当前代码**:
```go
func (a *AgentApp) GetShortMemory(conversationId string) (*SessionValue, error) {
    client, err := a.GetRedisClient()
    if err != nil {
        return nil, err
    }
    key := fmt.Sprintf(ShortMemorySessionKeyPrefix, conversationId)
    s, err := client.Get(key)
    // ...
}
```

**优化方案**: 添加本地缓存
```go
import (
    "lru"
    "time"
)

// 添加本地缓存（LRU）
var (
    sessionCache *lru.Cache
    cacheLock    sync.RWMutex
)

func init() {
    // 初始化本地缓存，最多缓存1000个会话
    sessionCache = lru.New(1000)
}

// 带缓存的 GetShortMemory
func (a *AgentApp) GetShortMemoryWithCache(conversationId string) (*SessionValue, error) {
    // 先查本地缓存
    cacheLock.RLock()
    if cached, ok := sessionCache.Get(conversationId); ok {
        cacheLock.RUnlock()
        return cached.(*SessionValue), nil
    }
    cacheLock.RUnlock()
    
    // 缓存未命中，查询 Redis
    session, err := a.GetShortMemory(conversationId)
    if err != nil {
        return nil, err
    }
    
    // 写入本地缓存
    cacheLock.Lock()
    sessionCache.Add(conversationId, session)
    cacheLock.Unlock()
    
    return session, nil
}

// 更新会话时清除缓存
func (a *AgentApp) SetShortMemory(conversationId string, session *SessionValue) error {
    client, err := a.GetRedisClient()
    if err != nil {
        return err
    }
    key := fmt.Sprintf(ShortMemorySessionKeyPrefix, conversationId)
    session = normalizeSessionValue(session)
    session.Meta.ConversationID = conversationId
    session.Meta.UpdatedAt = time.Now().Unix()
    b, _ := json.Marshal(session)
    
    // 清除本地缓存
    cacheLock.Lock()
    sessionCache.Remove(conversationId)
    cacheLock.Unlock()
    
    return client.Set(key, string(b), expiration)
}
```

---

## 代码重构方案

### 完整的重构代码

#### 1. powerai_memory.go 优化版本

```go
package powerai

import (
    "fmt"
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xdatetime"
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xuid"
    "strings"
    "sync"
    "time"
)

const (
    defaultMemoryTokenThresholdRatio = 0.75
    defaultMemoryRecentTurns         = 8
    defaultModelContextWindow        = 16000
    
    maxQueryLength   = 10000
    maxResponseLength = 50000
    maxUserIDLength  = 100
    maxAgentCodeLength = 50
)

// 添加会话锁映射
var sessionLocks sync.Map  // map[conversationID]*sync.Mutex

// 获取会话锁
func getSessionLock(conversationID string) *sync.Mutex {
    lock, _ := sessionLocks.LoadOrStore(conversationID, &sync.Mutex{})
    return lock.(*sync.Mutex)
}

type MemoryQueryRequest struct {
    ConversationID      string
    EnterpriseID        string
    PatientID           string
    Query               string
    TokenThresholdRatio float64
    RecentTurns         int
    ModelContextWindow  int
}

type MemoryContext struct {
    ConversationID          string
    Mode                    string
    Session                 *SessionValue
    History                 string
    FullHistory             string
    EstimatedTokens         int
    TokenRatio              float64
    ShouldCheckpointSummary bool
    Messages                []*AIMessage  // 添加消息列表
}

type MemoryWriteRequest struct {
    ConversationID string
    UserID         string
    AgentCode      string
    UserQuery      string
    AgentResponse  string
}

type MemoryWriteResult struct {
    ConversationID string
    Mode           string
    UpdatedAt      int64
}

type SessionFinalizeRequest struct {
    ConversationID string
    Summary        string
    RecentTurns    int
}

func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    if req == nil {
        return nil, fmt.Errorf("memory query request is nil")
    }
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    
    threshold, _, contextWindow := applyMemoryQueryDefaults(req)
    
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        xlog.LogErrorF("MEMORY", "QueryMemoryContext", "GetShortMemory", 
            fmt.Sprintf("failed to get short memory for conversation %s: %v", req.ConversationID, err), err)
        session = newDefaultSessionValue(req.ConversationID, req.PatientID)
    }
    
    session = normalizeSessionValue(session)
    mode := session.MessageContext.Mode
    
    var messages []*AIMessage
    if session.MessageContext.CheckpointMessageID != "" {
        messages, err = a.QueryMessageByConversationIDASCFromCheckpoint(req.ConversationID, session.MessageContext.CheckpointMessageID)
        if err != nil {
            xlog.LogErrorF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASCFromCheckpoint", 
                fmt.Sprintf("failed to query messages from checkpoint %s: %v", session.MessageContext.CheckpointMessageID, err), err)
            messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
            if err != nil {
                xlog.LogErrorF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASC", 
                    fmt.Sprintf("failed to query all messages for conversation %s: %v", req.ConversationID, err), err)
                messages = nil
            }
        }
    } else {
        messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
        if err != nil {
            xlog.LogErrorF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASC", 
                fmt.Sprintf("failed to query messages for conversation %s: %v", req.ConversationID, err), err)
            messages = nil
        }
    }
    
    fullHistory := buildHistoryFromAIMessages(messages)
    
    estimatedTokens := estimateTokenCount(fullHistory + "\n" + req.Query)
    tokenRatio := float64(estimatedTokens) / float64(contextWindow)
    
    history := fullHistory
    if mode == MemoryModeSummaryN {
        history = composeSummaryAndRecent(session)
        if strings.TrimSpace(history) == "" {
            history = fullHistory
            mode = MemoryModeFullHistory
        }
    }
    
    estimatedTokens = estimateTokenCount(history + "\n" + req.Query)
    tokenRatio = float64(estimatedTokens) / float64(contextWindow)
    
    shouldCheckpoint := tokenRatio >= threshold
    
    return &MemoryContext{
        ConversationID:          req.ConversationID,
        Mode:                    mode,
        Session:                 session,
        History:                 history,
        FullHistory:             fullHistory,
        EstimatedTokens:         estimatedTokens,
        TokenRatio:              tokenRatio,
        ShouldCheckpointSummary: shouldCheckpoint,
        Messages:                messages,
    }, nil
}

func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    if req == nil {
        return nil, fmt.Errorf("memory write request is nil")
    }
    
    if req.ConversationID == "" {
        return nil, fmt.Errorf("conversation_id is empty")
    }
    if !isValidUUID(req.ConversationID) {
        return nil, fmt.Errorf("invalid conversation_id format")
    }
    
    if req.UserID != "" {
        if len(req.UserID) > maxUserIDLength {
            return nil, fmt.Errorf("user_id too long (max %d characters)", maxUserIDLength)
        }
    }
    
    if req.AgentCode != "" {
        if len(req.AgentCode) > maxAgentCodeLength {
            return nil, fmt.Errorf("agent_code too long (max %d characters)", maxAgentCodeLength)
        }
        if !isValidAgentCode(req.AgentCode) {
            return nil, fmt.Errorf("invalid agent_code format")
        }
    }
    
    if len(req.UserQuery) > maxQueryLength {
        return nil, fmt.Errorf("user_query too long (max %d characters)", maxQueryLength)
    }
    
    if len(req.AgentResponse) > maxResponseLength {
        return nil, fmt.Errorf("agent_response too long (max %d characters)", maxResponseLength)
    }
    
    lock := getSessionLock(req.ConversationID)
    lock.Lock()
    defer lock.Unlock()
    
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        session = newDefaultSessionValue(req.ConversationID, req.UserID)
    }
    session = normalizeSessionValue(session)
    
    if req.UserID != "" {
        session.Meta.UserID = req.UserID
        if session.UserSnapshot != nil {
            session.UserSnapshot.UserID = req.UserID
        }
    }
    if req.AgentCode != "" {
        session.FlowContext.CurrentAgentKey = req.AgentCode
    }
    if req.AgentResponse != "" {
        session.FlowContext.LastBotMessage = req.AgentResponse
    }
    session.FlowContext.TurnCount++
    
    if err := a.SetShortMemory(req.ConversationID, session); err != nil {
        return nil, err
    }
    
    return &MemoryWriteResult{
        ConversationID: req.ConversationID,
        Mode:           session.MessageContext.Mode,
        UpdatedAt:      session.Meta.UpdatedAt,
    }, nil
}

func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    if conversationID == "" {
        return fmt.Errorf("conversation_id is empty")
    }
    if recentTurns <= 0 {
        recentTurns = defaultMemoryRecentTurns
    }
    
    lock := getSessionLock(conversationID)
    lock.Lock()
    defer lock.Unlock()
    
    session, err := a.GetShortMemory(conversationID)
    if err != nil {
        session = newDefaultSessionValue(conversationID, "")
    }
    session = normalizeSessionValue(session)
    
    messages, err := a.QueryMessageByConversationIDASC(conversationID)
    if err != nil {
        return err
    }
    
    summaryAndRecent := composeSummaryAndRecent(session)
    
    // 重试机制
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        checkpointMessageID := xuid.UUID()
        timeNow := xdatetime.GetNowDateTime()
        
        exists, err := a.checkMessageIDExists(checkpointMessageID)
        if err != nil {
            return fmt.Errorf("failed to check message_id existence: %w", err)
        }
        if exists {
            continue
        }
        
        sql := `INSERT INTO ai_message (message_id, conversation_id, query, answer, create_time, create_by, update_time, update_by) 
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
        _, err = a.DBExec(sql, checkpointMessageID, conversationID, "[MEMORY_CHECKPOINT]", summaryAndRecent, timeNow, "system", timeNow, "system")
        
        if err != nil {
            if isDuplicateKeyError(err) {
                continue
            }
            return fmt.Errorf("failed to insert checkpoint message: %w", err)
        }
        
        session.MessageContext.Summary = strings.TrimSpace(summary)
        session.MessageContext.WindowMessages = buildRecentMessages(messages, recentTurns)
        session.MessageContext.Mode = MemoryModeSummaryN
        session.MessageContext.CheckpointMessageID = checkpointMessageID
        
        return a.SetShortMemory(conversationID, session)
    }
    
    return fmt.Errorf("failed to generate unique message_id after %d retries", maxRetries)
}

func (a *AgentApp) checkMessageIDExists(messageID string) (bool, error) {
    sql := `SELECT COUNT(*) FROM ai_message WHERE message_id = $1`
    var count int
    err := a.DBQuerySingle(&count, sql, messageID)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

func isDuplicateKeyError(err error) bool {
    if err == nil {
        return false
    }
    if strings.Contains(err.Error(), "duplicate key") || 
       strings.Contains(err.Error(), "23505") {
        return true
    }
    return false
}

func (a *AgentApp) FinalizeSessionMemory(req *SessionFinalizeRequest) error {
    if req == nil {
        return fmt.Errorf("session finalize request is nil")
    }
    return a.CheckpointShortMemory(req.ConversationID, req.Summary, req.RecentTurns)
}

func applyMemoryQueryDefaults(req *MemoryQueryRequest) (float64, int, int) {
    threshold := req.TokenThresholdRatio
    if threshold <= 0 {
        threshold = defaultMemoryTokenThresholdRatio
    }
    recentTurns := req.RecentTurns
    if recentTurns <= 0 {
        recentTurns = defaultMemoryRecentTurns
    }
    contextWindow := req.ModelContextWindow
    if contextWindow <= 0 {
        contextWindow = defaultModelContextWindow
    }
    return threshold, recentTurns, contextWindow
}

func buildHistoryFromAIMessages(messages []*AIMessage) string {
    if len(messages) == 0 {
        return ""
    }
    
    estimatedSize := len(messages) * 200
    builder := strings.Builder{}
    builder.Grow(estimatedSize)
    
    for _, msg := range messages {
        if msg == nil {
            continue
        }
        
        var userMessage string
        if msg.Query.Valid {
            userMessage = strings.TrimSpace(msg.Query.String)
        }
        
        var agentMessage string
        if msg.Answer.Valid {
            agentMessage = extractAgentAnswer(msg.Answer.String)
        }
        
        if userMessage != "" {
            builder.WriteString("用户: ")
            builder.WriteString(userMessage)
            builder.WriteString("\n")
        }
        
        if agentMessage != "" {
            builder.WriteString("AI: ")
            builder.WriteString(agentMessage)
            builder.WriteString("\n")
        }
    }
    
    return strings.TrimSpace(builder.String())
}

func composeSummaryAndRecent(session *SessionValue) string {
    if session == nil || session.MessageContext == nil {
        return ""
    }
    
    estimatedSize := len(session.MessageContext.Summary) + len(session.MessageContext.WindowMessages)*100
    builder := strings.Builder{}
    builder.Grow(estimatedSize)
    
    summary := strings.TrimSpace(session.MessageContext.Summary)
    if summary != "" {
        builder.WriteString("历史摘要: ")
        builder.WriteString(summary)
        builder.WriteString("\n")
    }
    
    for _, msg := range session.MessageContext.WindowMessages {
        if msg == nil || strings.TrimSpace(msg.Content) == "" {
            continue
        }
        role := strings.ToLower(strings.TrimSpace(msg.Role))
        if role == "user" {
            builder.WriteString("用户: ")
        } else {
            builder.WriteString("AI: ")
        }
        builder.WriteString(strings.TrimSpace(msg.Content))
        builder.WriteString("\n")
    }
    
    return strings.TrimSpace(builder.String())
}

func buildRecentMessages(messages []*AIMessage, recentTurns int) []*Message {
    if len(messages) == 0 {
        return nil
    }
    
    start := len(messages) - recentTurns
    if start < 0 {
        start = 0
    }
    
    recent := make([]*Message, 0, recentTurns*2)
    for _, msg := range messages[start:] {
        if msg == nil {
            continue
        }
        
        var userMessage string
        if msg.Query.Valid {
            userMessage = strings.TrimSpace(msg.Query.String)
        }
        
        var agentMessage string
        if msg.Answer.Valid {
            agentMessage = extractAgentAnswer(msg.Answer.String)
        }
        
        if userMessage != "" {
            recent = append(recent, &Message{
                Role:    "user",
                Content: userMessage,
            })
        }
        
        if agentMessage != "" {
            recent = append(recent, &Message{
                Role:    "assistant",
                Content: agentMessage,
            })
        }
    }
    
    return recent
}

func extractAgentAnswer(answer string) string {
    answer = strings.TrimSpace(answer)
    if answer == "" {
        return ""
    }
    data := xjson.Get(answer, "data")
    if data.Exists() {
        msg := xjson.Get(data.String(), "msg")
        if msg.Exists() {
            return strings.TrimSpace(msg.String())
        }
    }
    return answer
}

func estimateTokenCount(text string) int {
    text = strings.TrimSpace(text)
    if text == "" {
        return 0
    }
    runeCount := len([]rune(text))
    tokens := runeCount / 4
    if tokens <= 0 {
        return 1
    }
    return tokens
}

func isValidUUID(uuid string) bool {
    if len(uuid) != 36 {
        return false
    }
    return true
}

func isValidAgentCode(code string) bool {
    if code == "" {
        return false
    }
    for _, c := range code {
        if !((c >= 'a' && c <= 'z') || 
             (c >= 'A' && c <= 'Z') || 
             (c >= '0' && c <= '9') || 
             c == '_' || c == '-') {
            return false
        }
    }
    return true
}
```

---

## 总结

### 优先级排序

#### 🔴 高优先级（立即修复）
1. **添加会话级锁** - 防止并发写入冲突
2. **SQL注入防护** - UpdateMessage 函数
3. **空指针防护** - normalizeSessionValue 和 buildHistoryFromAIMessages

#### 🟡 中优先级（近期修复）
4. **数据库查询优化** - Checkpoint 查询效率
5. **错误处理完善** - 添加日志记录
6. **输入验证** - WriteTurn 函数

#### 🟢 低优先级（长期优化）
7. **缓存优化** - 添加本地缓存
8. **性能优化** - 字符串拼接优化

### 关键改进点

1. **并发安全**: 使用 sync.Map 实现会话级锁
2. **数据一致性**: 添加重试机制和唯一性检查
3. **防御性编程**: 完善的空指针检查和输入验证
4. **性能优化**: 减少重复查询，优化字符串操作
5. **可观测性**: 添加详细的错误日志

### 建议实施顺序

1. 第一阶段（1-2天）：添加并发锁和空指针防护
2. 第二阶段（2-3天）：优化数据库查询和错误处理
3. 第三阶段（1-2天）：添加输入验证和日志
4. 第四阶段（长期）：性能优化和缓存
