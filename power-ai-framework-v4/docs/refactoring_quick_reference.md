# Power AI Framework V4 - 重构快速参考

> **版本**: v4.0.0
> **完成时间**: 2026-01-26
> **状态**: ✅ 基础设施已完成

---

## 📦 已创建的文件清单

### 工具类（5个）

| 文件 | 路径 | 功能 |
|------|------|------|
| 会话锁管理器 | `pkg/xlock/session_lock.go` | 管理会话级别的并发锁 |
| 会话状态规范化器 | `pkg/xdefense/session_normalizer.go` | 规范化会话状态、验证输入 |
| 配置加载器 | `pkg/xconfig/memory_config.go` | 加载和解析 YAML 配置 |
| 消息历史构建器 | `pkg/xmemory/message_builder.go` | 构建对话历史、估算Token |
| 初始化工具类 | `pkg/xinit/memory_init.go` | 初始化所有工具类 |

### 配置文件（1个）

| 文件 | 路径 | 功能 |
|------|------|------|
| 记忆管理配置 | `config/memory_config.yaml` | 集中管理所有参数 |

### 文档（4个）

| 文件 | 路径 | 内容 |
|------|------|------|
| 重构指南 | `docs/code_refactoring_guide.md` | 详细的重构方案和使用指南 |
| 重构总结 | `docs/refactoring_summary.md` | 工作总结和后续建议 |
| 完成报告 | `docs/refactoring_completion_report.md` | 完整的工作清单和使用示例 |
| 快速参考 | `docs/refactoring_quick_reference.md` | 本文档 |

---

## 🚀 快速开始

### 1. 初始化配置和工具类

```go
import (
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xinit"
)

// 初始化所有工具类
memoryInitResult := xinit.InitMemoryManager()
if memoryInitResult.Error != nil {
    log.Warn("Failed to init memory manager, using defaults")
}

// 访问配置
config := memoryInitResult.Config
lockManager := memoryInitResult.LockManager
messageBuilder := memoryInitResult.MessageBuilder
```

### 2. 使用配置

```go
// 获取配置参数
threshold := config.TokenThresholdRatio
maxRetries := config.CheckpointMaxRetries
maxQueryLength := config.MaxQueryLength
redisExpiration := config.RedisExpiration
```

### 3. 使用锁管理器

```go
// 方式1：直接获取锁
lock := lockManager.GetLock(conversationID)
lock.Lock()
defer lock.Unlock()

// 方式2：使用便捷方法
err := lockManager.LockWith(conversationID, func() error {
    // 在锁保护下执行操作
    return nil
})

// 方式3：使用带返回值的便捷方法
result, err := lockManager.LockWithVal(conversationID, func() (string, error) {
    // 在锁保护下执行操作
    return "result", nil
})
```

### 4. 使用规范化器

```go
import (
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xdefense"
)

// 创建规范化器
normalizer := xdefense.NewSessionNormalizer(config.MemoryModeFullHistory)

// 验证智能体代码
if !normalizer.ValidateAgentCode(req.AgentCode) {
    return nil, fmt.Errorf("invalid agent_code format")
}

// 验证UUID
if !normalizer.ValidateUUID(messageID) {
    return fmt.Errorf("invalid uuid format")
}

// 判断是否是主键冲突错误
if normalizer.IsDuplicateKeyError(err) {
    // 处理重复键错误
}

// 规范化字符串
normalizedStr := normalizer.NormalizeString(input, defaultValue)

// 规范化字符串切片
normalizedSlice := normalizer.NormalizeStringSlice(inputSlice)
```

### 5. 使用消息构建器

```go
import (
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xmemory"
)

// 构建对话历史
fullHistory := messageBuilder.BuildHistoryFromMessages(messages)

// 组合摘要和最近消息
history := messageBuilder.ComposeSummaryAndRecent(summary, messages)

// 提取最近N轮消息
recent := messageBuilder.BuildRecentMessages(messages, recentTurns)

// 估算Token数量（静态方法）
tokenCount := xmemory.EstimateTokenCount(text)

// 提取智能体答案（静态方法）
answer := xmemory.ExtractAgentAnswer(response)
```

---

## 📊 重构效果对比

| 指标 | 重构前 | 重构后（预期） | 改善 |
|------|--------|---------------|------|
| powerai_memory.go | 600+ 行 | ~400 行 | -33% |
| powerai_short_memory.go | 500+ 行 | ~300 行 | -40% |
| 代码耦合度 | 高 | 低 | 显著改善 |
| 可维护性 | 中 | 高 | 显著改善 |
| 参数配置 | 硬编码 | YAML配置 | 显著改善 |

---

## 🔧 配置文件示例

```yaml
# config/memory_config.yaml

# Token 相关配置
token_threshold_ratio: 0.75
default_recent_turns: 8
default_model_context_window: 16000

# 输入验证配置
max_query_length: 10000
max_response_length: 50000
max_user_id_length: 100
max_agent_code_length: 50
max_summary_length: 2000

# Redis 配置
redis_key_prefix: "short_term_memory:session:%s"
redis_expiration: 1800

# Checkpoint 配置
checkpoint_max_retries: 3

# 性能优化配置
estimated_message_chars: 200
estimated_window_message_chars: 100

# 记忆模式配置
memory_mode_full_history: "FULL_HISTORY"
memory_mode_summary_n: "SUMMARY_N"

# 日志配置
enable_verbose_logging: false
log_level: "info"
```

---

## 📝 后续工作

### 待完成任务

1. ⏳ 简化 `powerai_short_memory.go`
   - 移除 `sessionLocks sync.Map`
   - 移除 `getSessionLock` 函数
   - 移除 `normalizeSessionValue` 函数
   - 移除常量定义

2. ⏳ 简化 `powerai_memory.go`
   - 移除所有常量定义
   - 移除验证函数
   - 移除消息构建函数

3. ⏳ 更新 `powerai.go`
   - 在 `AgentApp` 结构体中添加记忆管理相关字段
   - 在 `NewAgent` 函数中添加初始化逻辑

4. ⏳ 编写单元测试
   - 测试所有工具类
   - 测试重构后的主文件

5. ⏳ 更新文档
   - 更新 API 文档
   - 创建使用示例

---

## 💡 工具类使用示例

### 示例1：完整的记忆管理流程

```go
// 初始化
memoryInitResult := xinit.InitMemoryManager()
config := memoryInitResult.Config
lockManager := memoryInitResult.LockManager
messageBuilder := memoryInitResult.MessageBuilder
normalizer := xdefense.NewSessionNormalizer(config.MemoryModeFullHistory)

// 验证输入
if !normalizer.ValidateAgentCode(agentCode) {
    return nil, fmt.Errorf("invalid agent_code")
}

// 使用锁保护
err := lockManager.LockWith(conversationID, func() error {
    // 获取会话状态
    session, err := GetShortMemory(conversationID)
    if err != nil {
        return err
    }
    
    // 构建对话历史
    history := messageBuilder.BuildHistoryFromMessages(messages)
    
    // 估算Token
    tokenCount := xmemory.EstimateTokenCount(history)
    
    // 更新会话状态
    session.FlowContext.TurnCount++
    
    return SetShortMemory(conversationID, session)
})
```

### 示例2：配置热更新

```go
// 重新加载配置
err := xconfig.ReloadConfig("config/memory_config.yaml")
if err != nil {
    log.Error("Failed to reload config")
    return err
}

// 获取更新后的配置
config := xconfig.GetConfig()
```

---

## 🎯 重构优势

1. **代码更简洁** - 主文件只保留核心业务逻辑
2. **参数可配置** - YAML 配置文件便于团队协作
3. **工具类复用** - 可在其他模块中使用
4. **易于维护** - 职责清晰，修改影响范围小
5. **性能优化** - 预分配容量，减少内存分配

---

## 📞 联系方式

- **技术支持**: tech-support@example.com
- **项目管理**: project@example.com

---

**文档结束**
