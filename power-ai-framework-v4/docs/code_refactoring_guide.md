# Power AI Framework V4 - 记忆管理重构方案

> **版本**: v4.0.0
> **重构时间**: 2026-01-26
> **重构目标**: 简化代码结构，提升可维护性

---

## 📋 重构概述

当前 `powerai_memory.go` 和 `powerai_short_memory.go` 文件代码量过大，包含了太多功能。本重构方案旨在：

1. ✅ 将锁管理提取到工具类（`pkg/xlock`）
2. ✅ 将防御性编程提取到工具类（`pkg/xdefense`）
3. ✅ 将配置参数提取到 YAML 配置文件（`config/memory_config.yaml`）
4. ✅ 创建配置加载器（`pkg/xconfig`）
5. 🔄 简化主文件，只保留核心业务逻辑

---

## 📁 重构后文件结构

```
power-ai-framework-v4/
├── config/
│   └── memory_config.yaml          # 记忆管理配置文件（新建）
├── pkg/
│   ├── xlock/
│   │   └── session_lock.go          # 会话锁管理器（新建）
│   ├── xdefense/
│   │   └── session_normalizer.go    # 会话状态规范化器（新建）
│   └── xconfig/
│       └── memory_config.go          # 配置加载器（新建）
├── powerai_memory.go                 # 记忆管理主文件（简化）
├── powerai_short_memory.go           # 短期记忆主文件（简化）
└── docs/
    └── code_refactoring_guide.md    # 代码重构指南（本文档）
```

---

## 🔧 重构内容详解

### 1. 会话锁管理器

**文件**: `pkg/xlock/session_lock.go`

**功能**:
- 管理会话级别的并发锁
- 防止同一会话的并发写入冲突
- 提供便捷的锁使用方法

**核心功能**:
```go
type SessionLockManager struct {
    locks sync.Map // map[conversationID]*sync.Mutex
}

// 获取锁
func (m *SessionLockManager) GetLock(conversationID string) *sync.Mutex

// 在锁保护下执行函数
func (m *SessionLockManager) LockWith(conversationID string, fn func()) error

// 在锁保护下执行函数并返回值
func (m *SessionLockManager) LockWithVal[T any](conversationID string, fn func() (T, error)) (T, error)
```

**使用示例**:
```go
// 在 powerai.go 中初始化
var sessionLockManager = xlock.NewSessionLockManager()

// 在 WriteTurn 中使用
lock := sessionLockManager.GetLock(conversationID)
lock.Lock()
defer lock.Unlock()
```

---

### 2. 会话状态规范化器

**文件**: `pkg/xdefense/session_normalizer.go`

**功能**:
- 规范化会话状态，防止空指针异常
- 验证输入格式和长度
- 提供安全的字符串、切片、整数访问方法

**核心功能**:
```go
type SessionNormalizer struct {
    defaultMode string
}

// 规范化字符串
func (n *SessionNormalizer) NormalizeString(value, defaultValue string) string

// 规范化字符串切片
func (n *SessionNormalizer) NormalizeStringSlice(slice []string) []string

// 验证长度
func (n *SessionNormalizer) ValidateLength(value string, maxLength int) bool

// 验证智能体代码格式
func (n *SessionNormalizer) ValidateAgentCode(code string) bool

// 验证UUID格式
func (n *SessionNormalizer) ValidateUUID(uuid string) bool

// 判断是否是主键冲突错误
func (n *SessionNormalizer) IsDuplicateKeyError(err error) bool
```

**使用示例**:
```go
// 在 powerai.go 中初始化
var sessionNormalizer = xdefense.NewSessionNormalizer("FULL_HISTORY")

// 在 WriteTurn 中使用
if !sessionNormalizer.ValidateAgentCode(req.AgentCode) {
    return nil, fmt.Errorf("invalid agent_code format")
}
```

---

### 3. 配置文件

**文件**: `config/memory_config.yaml`

**功能**:
- 集中管理所有记忆管理相关参数
- 支持热更新（重新加载配置）
- 便于团队协作修改参数

**配置项**:
```yaml
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

### 4. 配置加载器

**文件**: `pkg/xconfig/memory_config.go`

**功能**:
- 加载和解析 YAML 配置文件
- 提供配置访问接口
- 支持配置热更新

**核心功能**:
```go
type MemoryConfig struct {
    TokenThresholdRatio float64
    DefaultRecentTurns  int
    // ... 其他配置项
}

// 加载配置
func LoadConfig(configPath string) (*MemoryConfig, error)

// 获取配置实例
func GetConfig() *MemoryConfig

// 重新加载配置
func ReloadConfig(configPath string) error
```

**使用示例**:
```go
// 在 powerai.go 初始化时加载配置
config, err := xconfig.LoadConfig("config/memory_config.yaml")
if err != nil {
    log.Error("Failed to load config, using defaults")
    config = xconfig.GetConfig()
}

// 在代码中使用配置
threshold := config.TokenThresholdRatio
maxRetries := config.CheckpointMaxRetries
```

---

## 🔄 主文件简化方案

### powerai_short_memory.go 简化后

**移除的内容**:
- ❌ `sessionLocks sync.Map` （移到 `pkg/xlock`）
- ❌ `getSessionLock` 函数 （使用 `xlock.SessionLockManager`）
- ❌ `normalizeSessionValue` 函数（使用 `xdefense.SessionNormalizer`）
- ❌ 所有常量定义（移到配置文件）

**保留的内容**:
- ✅ 数据结构定义（`SessionValue`, `MetaInfo`, `FlowContext` 等）
- ✅ Redis 操作函数（`CreateShortMemory`, `GetShortMemory`, `SetShortMemory`）
- ✅ 核心业务逻辑

**简化后的代码量**: 约 300 行（原来 500+ 行）

---

### powerai_memory.go 简化后

**移除的内容**:
- ❌ 所有常量定义（移到配置文件）
- ❌ `isValidAgentCode` 函数（使用 `xdefense.SessionNormalizer`）
- ❌ `isDuplicateKeyError` 函数（使用 `xdefense.SessionNormalizer`）
- ❌ `isValidUUID` 函数（使用 `xdefense.SessionNormalizer`）

**保留的内容**:
- ✅ 数据结构定义（`MemoryQueryRequest`, `MemoryContext` 等）
- ✅ 核心API函数（`QueryMemoryContext`, `WriteTurn`, `CheckpointShortMemory` 等）
- ✅ 辅助函数（`buildHistoryFromAIMessages`, `composeSummaryAndRecent` 等）

**简化后的代码量**: 约 400 行（原来 600+ 行）

---

## 📊 重构效果

| 指标 | 重构前 | 重构后 | 改善 |
|------|--------|--------|------|
| powerai_memory.go | 600+ 行 | 400 行 | -33% |
| powerai_short_memory.go | 500+ 行 | 300 行 | -40% |
| 代码耦合度 | 高 | 低 | 显著改善 |
| 可维护性 | 中 | 高 | 显著改善 |
| 参数配置 | 硬编码 | YAML配置 | 显著改善 |

---

## 🚀 使用指南

### 1. 初始化配置

```go
// 在 powerai.go 初始化时
import (
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xconfig"
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xlock"
    "orgine.com/ai-team/power-ai-framework-v4/pkg/xdefense"
)

// 初始化配置
config, err := xconfig.LoadConfig(xconfig.GetConfigPath())
if err != nil {
    log.Warn("Failed to load config, using defaults")
}

// 初始化工具类
sessionLockManager = xlock.NewSessionLockManager()
sessionNormalizer = xdefense.NewSessionNormalizer(config.MemoryModeFullHistory)
```

### 2. 修改参数

**方式一**: 修改 YAML 配置文件
```yaml
# 编辑 config/memory_config.yaml
token_threshold_ratio: 0.8  # 修改阈值
max_query_length: 20000     # 修改最大查询长度
```

**方式二**: 重新加载配置
```go
err := xconfig.ReloadConfig("config/memory_config.yaml")
if err != nil {
    log.Error("Failed to reload config")
}
```

### 3. 使用工具类

**使用锁管理器**:
```go
lock := sessionLockManager.GetLock(conversationID)
lock.Lock()
defer lock.Unlock()
```

**使用规范化器**:
```go
if !sessionNormalizer.ValidateAgentCode(req.AgentCode) {
    return nil, fmt.Errorf("invalid agent_code format")
}
```

**使用配置**:
```go
threshold := config.TokenThresholdRatio
maxRetries := config.CheckpointMaxRetries
```

---

## 📝 后续工作

### 待完成任务

1. ✅ 创建会话锁管理器（`pkg/xlock/session_lock.go`）
2. ✅ 创建会话状态规范化器（`pkg/xdefense/session_normalizer.go`）
3. ✅ 创建配置文件（`config/memory_config.yaml`）
4. ✅ 创建配置加载器（`pkg/xconfig/memory_config.go`）
5. ⏳ 简化 `powerai_short_memory.go`
6. ⏳ 简化 `powerai_memory.go`
7. ⏳ 更新相关文档
8. ⏳ 编写单元测试

### 建议

1. **逐步重构**: 建议先完成工具类的创建和测试，再逐步简化主文件
2. **保持兼容**: 确保重构后 API 接口保持不变，不影响现有调用
3. **充分测试**: 重构后需要进行充分的单元测试和集成测试
4. **团队协作**: YAML 配置文件便于团队协作修改参数

---

## 🤝 联系方式

- **技术支持**: tech-support@example.com
- **项目管理**: project@example.com

---

**文档结束**
