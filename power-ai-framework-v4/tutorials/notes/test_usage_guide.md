# 性能与安全评估测试使用指南

> **目标**: 评估 power-ai-framework-v4 的并发安全性、性能、数据库效率和安全性
> **文档版本**: v1.0
> **更新时间**: 2026-01-26

## 📋 目录

1. [测试概述](#测试概述)
2. [环境准备](#环境准备)
3. [测试配置](#测试配置)
4. [运行测试](#运行测试)
5. [测试结果解读](#测试结果解读)
6. [优化建议](#优化建议)
7. [测试报告示例](#测试报告示例)

---

## 测试概述

### 测试目标

该评估脚本旨在全面测试 power-ai-framework-v4 框架的以下方面：

1. **并发安全性** - 测试在高并发情况下的数据一致性和锁竞争
2. **性能基准** - 测试查询、写入、checkpoint等操作的响应时间和吞吐量
3. **数据库效率** - 测试索引效果、查询优化、批量操作等
4. **安全性** - 测试SQL注入防护、输入验证、空指针防护等

### 测试架构

```
测试脚本
├── 环境检查 (Environment Check)
│   ├── 数据库连接测试
│   ├── Redis连接测试
│   └── 系统资源检查
├── 并发安全性测试 (Concurrency Safety)
│   ├── 50并发用户 × 20消息
│   ├── 查询/写入/checkpoint混合操作
│   └── 资源使用监控
├── 性能基准测试 (Performance Benchmark)
│   ├── 单用户连续查询/写入
│   ├── Checkpoint性能测试
│   └── 高并发查询/写入
├── 数据库效率测试 (Database Efficiency)
│   ├── 有/无索引查询对比
│   ├── 批量插入/更新性能
│   └── Checkpoint查询效率
└── 安全性测试 (Security)
    ├── SQL注入防护
    ├── 输入验证
    ├── 空指针防护
    └── 并发安全
```

---

## 环境准备

### 前置要求

#### 1. 数据库准备

确保 PostgreSQL 数据库已安装并运行：

```bash
# 检查 PostgreSQL 是否运行
psql -U postgres -c "SELECT version();"

# 创建测试数据库（如果不存在）
psql -U postgres -c "CREATE DATABASE power_ai;"

# 连接到数据库
psql -U postgres -d power_ai
```

#### 2. 必要的表结构

确保以下表存在：

```sql
-- ai_message 表
CREATE TABLE IF NOT EXISTS ai_message (
    message_id VARCHAR(64) PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL,
    query TEXT,
    answer TEXT,
    rating VARCHAR(32),
    inputs TEXT,
    errors TEXT,
    agent_code VARCHAR(64),
    file_id VARCHAR(64),
    create_time TIMESTAMP NOT NULL,
    create_by VARCHAR(64),
    update_time TIMESTAMP NOT NULL,
    update_by VARCHAR(64),
    extended_field TEXT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ai_message_conversation_id 
    ON ai_message(conversation_id);

CREATE INDEX IF NOT EXISTS idx_ai_message_message_id 
    ON ai_message(message_id);

CREATE INDEX IF NOT EXISTS idx_ai_message_conversation_create_time 
    ON ai_message(conversation_id, create_time);
```

#### 3. Redis 准备

确保 Redis 已安装并运行：

```bash
# 检查 Redis 是否运行
redis-cli ping
# 应该返回: PONG
```

#### 4. Go 环境准备

确保 Go 已安装：

```bash
# 检查 Go 版本
go version

# 安装 PostgreSQL 驱动
go get github.com/lib/pq
```

### 配置文件

在测试脚本中修改配置：

```go
var config = TestConfig{
    // 数据库配置
    DBHost:         "localhost",
    DBPort:         "5432",
    DBUser:         "postgres",
    DBPassword:     "password",
    DBName:         "power_ai",
    
    // Redis配置
    RedisHost:      "localhost",
    RedisPort:      "6379",
    RedisPassword:  "",
    
    // 测试配置
    ConcurrentUsers: 50,  // 并发用户数
    TestDuration:    30,   // 测试时长（秒）
    MessageCount:    20,   // 每个用户的消息数量
    
    // 性能基准
    MaxQueryTime:    100 * time.Millisecond,  // 查询最大允许时间
    MaxWriteTime:    50 * time.Millisecond,   // 写入最大允许时间
    MaxCheckpointTime: 500 * time.Millisecond, // Checkpoint最大允许时间
}
```

---

## 测试配置

### 调整测试参数

根据您的需求调整以下参数：

#### 1. 并发用户数

```go
// 轻量测试
ConcurrentUsers: 10,  // 10个并发用户

// 标准测试
ConcurrentUsers: 50,  // 50个并发用户

// 压力测试
ConcurrentUsers: 100, // 100个并发用户
```

#### 2. 测试时长

```go
// 快速测试
TestDuration: 10,  // 10秒

// 标准测试
TestDuration: 30,  // 30秒

// 深度测试
TestDuration: 60,  // 60秒
```

#### 3. 消息数量

```go
// 轻量测试
MessageCount: 10,  // 每个用户10条消息

// 标准测试
MessageCount: 20,  // 每个用户20条消息

// 深度测试
MessageCount: 50,  // 每个用户50条消息
```

#### 4. 性能基准

```go
// 宽松标准
MaxQueryTime: 200 * time.Millisecond,
MaxWriteTime: 100 * time.Millisecond,
MaxCheckpointTime: 1000 * time.Millisecond,

// 严格标准
MaxQueryTime: 50 * time.Millisecond,
MaxWriteTime: 25 * time.Millisecond,
MaxCheckpointTime: 250 * time.Millisecond,
```

---

## 运行测试

### 基本运行

```bash
# 进入测试目录
cd power-ai-framework-v4/test

# 运行测试
go run performance_and_security_test.go
```

### 编译后运行

```bash
# 编译测试程序
go build -o test_runner performance_and_security_test.go

# 运行测试
./test_runner
```

### 生成测试报告

测试脚本会自动在控制台输出测试报告，包括：

1. **环境检查结果**
2. **并发安全性测试结果**
3. **性能基准测试结果**
4. **数据库效率测试结果**
5. **安全性测试结果**
6. **综合评估报告**

---

## 测试结果解读

### 1. 并发安全性测试结果

#### 关键指标

```
测试结果:
  - 总查询次数: 1000
  - 总写入次数: 1000
  - 总Checkpoint次数: 100
  - 成功次数: 2100
  - 失败次数: 0
  - 超时次数: 0

查询时间统计:
  - 平均: 15ms
  - 最小: 10ms
  - 最大: 25ms
  - P50: 14ms
  - P95: 20ms
  - P99: 23ms

资源使用:
  - 初始内存: 10 MB
  - 最大内存: 25 MB
  - 增长内存: 15 MB
  - 初始Goroutines: 5
  - 最大Goroutines: 55
  - 增长Goroutines: 50
```

#### 评估标准

| 指标 | 优秀 | 良好 | 一般 | 较差 |
|------|------|------|------|------|
| 超时率 | < 1% | 1-5% | 5-10% | > 10% |
| 失败率 | < 0.1% | 0.1-1% | 1-5% | > 5% |
| 内存增长 | < 20MB | 20-50MB | 50-100MB | > 100MB |
| Goroutines增长 | < 并发数 | 并发数 | 并发数×2 | > 并发数×2 |

### 2. 性能基准测试结果

#### 关键指标

```
性能指标:
  所有查询统计:
    - 平均: 15ms
    - 最小: 10ms
    - 最大: 25ms
    - P50: 14ms
    - P95: 20ms
    - P99: 23ms

  所有写入统计:
    - 平均: 8ms
    - 最小: 5ms
    - 最大: 15ms
    - P50: 7ms
    - P95: 12ms
    - P99: 14ms

  所有Checkpoint统计:
    - 平均: 45ms
    - 最小: 35ms
    - 最大: 60ms
    - P50: 44ms
    - P95: 55ms
    - P99: 58ms

  - 总操作数: 2200
  - 测试时长: 5.2s
  - QPS: 423.08
```

#### 评估标准

| 指标 | 优秀 | 良好 | 一般 | 较差 |
|------|------|------|------|------|
| QPS | > 1000 | 500-1000 | 100-500 | < 100 |
| 平均查询时间 | < 20ms | 20-50ms | 50-100ms | > 100ms |
| 平均写入时间 | < 10ms | 10-25ms | 25-50ms | > 50ms |
| Checkpoint时间 | < 100ms | 100-200ms | 200-500ms | > 500ms |

### 3. 数据库效率测试结果

#### 关键指标

```
数据库操作性能:
  - 无索引查询(100次): 2.5s
  - 有索引查询(100次): 0.8s
  - 批量插入(100条): 0.5s
  - 批量更新(100条): 0.6s
  - Checkpoint查询(100次): 1.2s
  - 全量历史查询(10次): 2.0s

性能对比:
  - 索引提升: 68.00%
  - Checkpoint查询占比: 60.00%
```

#### 评估标准

| 指标 | 优秀 | 良好 | 一般 | 较差 |
|------|------|------|------|------|
| 索引提升 | > 50% | 20-50% | 10-20% | < 10% |
| Checkpoint查询占比 | < 30% | 30-50% | 50-70% | > 70% |
| 批量插入性能 | < 1s/100条 | 1-2s/100条 | 2-5s/100条 | > 5s/100条 |

### 4. 安全性测试结果

#### 关键指标

```
安全性测试结果:
  - SQL注入测试: 6/6 通过
  - 输入验证测试: 7/7 通过
  - 空指针防护测试: 4/4 通过
  - 并发安全测试: 1/1 通过

安全性评估:
  ✅ 安全性优秀（通过率≥95%）
```

#### 评估标准

| 通过率 | 评级 |
|--------|------|
| ≥ 95% | 优秀 |
| ≥ 80% | 良好 |
| ≥ 60% | 一般 |
| < 60% | 较差 |

### 5. 综合评估报告

#### 评级系统

```
综合评分: 85/100

评级: 良好
系统整体表现良好，建议在投入生产环境前进行少量优化。
```

#### 评级标准

| 综合评分 | 评级 | 建议 |
|----------|------|------|
| ≥ 90 | 🌟 优秀 | 可以投入生产环境 |
| 75-89 | ✅ 良好 | 建议少量优化后投入生产 |
| 60-74 | ⚠️ 一般 | 需要优化后投入生产 |
| < 60 | ❌ 较差 | 必须优化才能投入生产 |

---

## 优化建议

### 基于测试结果的优化建议

#### 🔴 高优先级 - 并发安全优化

如果并发安全评分 < 80：

```go
// 1. 添加会话级锁
var sessionLocks sync.Map

func getSessionLock(conversationID string) *sync.Mutex {
    lock, _ := sessionLocks.LoadOrStore(conversationID, &sync.Mutex{})
    return lock.(*sync.Mutex)
}

// 2. 在 WriteTurn 中使用锁
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
    lock := getSessionLock(req.ConversationID)
    lock.Lock()
    defer lock.Unlock()
    
    // 原有逻辑...
}

// 3. 在 CheckpointShortMemory 中使用锁
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
    lock := getSessionLock(conversationID)
    lock.Lock()
    defer lock.Unlock()
    
    // 原有逻辑...
}
```

#### 🔴 高优先级 - 性能优化

如果性能评分 < 80：

```go
// 1. 批量查询优化
func (a *AgentApp) QueryMultipleConversations(conversationIDs []string) ([]*MemoryContext, error) {
    // 使用 IN 查询代替多次单独查询
    sql := `SELECT * FROM ai_message WHERE conversation_id IN ($1, $2, ...)`
    // ...
}

// 2. 添加本地缓存
import "github.com/hashicorp/golang-lru/v2/lru2"

var sessionCache, _ = lru2.New[string, *SessionValue](1000)

func (a *AgentApp) GetShortMemoryWithCache(conversationId string) (*SessionValue, error) {
    if cached, ok := sessionCache.Get(conversationId); ok {
        return cached, nil
    }
    
    session, err := a.GetShortMemory(conversationId)
    if err != nil {
        return nil, err
    }
    
    sessionCache.Add(conversationId, session)
    return session, nil
}

// 3. 优化字符串拼接
func buildHistoryFromAIMessages(messages []*AIMessage) string {
    if len(messages) == 0 {
        return ""
    }
    
    // 预分配容量
    estimatedSize := len(messages) * 200
    builder := strings.Builder{}
    builder.Grow(estimatedSize)
    
    for _, msg := range messages {
        // ...
    }
    
    return strings.TrimSpace(builder.String())
}
```

#### 🔴 高优先级 - 数据库优化

如果数据库效率评分 < 80：

```sql
-- 1. 添加缺失的索引
CREATE INDEX IF NOT EXISTS idx_ai_message_conversation_id 
    ON ai_message(conversation_id);

CREATE INDEX IF NOT EXISTS idx_ai_message_message_id 
    ON ai_message(message_id);

CREATE INDEX IF NOT EXISTS idx_ai_message_conversation_create_time 
    ON ai_message(conversation_id, create_time);

-- 2. 优化 Checkpoint 查询
-- 使用 JOIN 代替子查询
SELECT m.* 
FROM ai_message m
INNER JOIN ai_message cp ON m.conversation_id = cp.conversation_id
WHERE m.conversation_id = $1 
  AND cp.message_id = $2
  AND m.create_time > cp.create_time
ORDER BY m.create_time ASC;

-- 3. 定期清理过期数据
DELETE FROM ai_message 
WHERE create_time < NOW() - INTERVAL '30 days';
```

#### 🔴 高优先级 - 安全性优化

如果安全性评分 < 80：

```go
// 1. 完善 SQL 注入防护
func isValidUUID(uuid string) bool {
    if len(uuid) != 36 {
        return false
    }
    // 添加更严格的验证
    return true
}

// 2. 加强输入验证
const (
    maxQueryLength    = 10000
    maxResponseLength = 50000
    maxUserIDLength   = 100
    maxAgentCodeLength = 50
)

func validateInput(input string, maxLength int) error {
    if len(input) > maxLength {
        return fmt.Errorf("input too long")
    }
    // 添加更多验证逻辑
    return nil
}

// 3. 完善空指针防护
func normalizeSessionValue(session *SessionValue) *SessionValue {
    if session == nil {
        return newDefaultSessionValue("", "")
    }
    
    // 确保所有嵌套指针都不为 nil
    if session.Meta == nil {
        session.Meta = &MetaInfo{}
    }
    if session.UserSnapshot == nil {
        session.UserSnapshot = &UserProfile{}
    }
    // ...
    
    return session
}

// 4. 添加日志记录
import "orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"

func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    session, err := a.GetShortMemory(req.ConversationID)
    if err != nil {
        xlog.LogErrorF("MEMORY", "QueryMemoryContext", "GetShortMemory", 
            fmt.Sprintf("failed to get short memory: %v", err), err)
    }
    // ...
}
```

---

## 测试报告示例

### 优化前的测试报告

```
========================================
Power AI Framework 性能与安全评估测试
========================================

【1/8】环境检查...
  - 检查数据库连接...
    ✅ 数据库连接正常
  - 检查Redis连接...
    ✅ Redis连接正常（模拟）
  - 检查系统资源...
    ✅ 系统内存: 10.23 MB
    ✅ Goroutines: 5
✅ 环境检查通过

【2/8】数据库连接测试...
  ✅ 当前消息总数: 0
  ✅ 索引 idx_ai_message_conversation_id 存在
  ✅ 索引 idx_ai_message_message_id 存在
  ✅ 索引 idx_ai_message_conversation_create_time 存在
✅ 数据库连接测试通过

【3/8】Redis连接测试...
  ✅ Redis连接测试通过（模拟）

【4/8】并发安全性测试...
  测试参数:
    - 并发用户数: 50
    - 每用户消息数: 20
    - 测试时长: 30秒

  测试结果:
    - 总查询次数: 1000
    - 总写入次数: 1000
    - 总Checkpoint次数: 100
    - 成功次数: 2100
    - 失败次数: 0
    - 超时次数: 0

  查询时间统计:
    - 平均: 15ms
    - 最小: 10ms
    - 最大: 25ms
    - P50: 14ms
    - P95: 20ms
    - P99: 23ms

  写入时间统计:
    - 平均: 8ms
    - 最小: 5ms
    - 最大: 15ms
    - P50: 7ms
    - P95: 12ms
    - P99: 14ms

  Checkpoint时间统计:
    - 平均: 45ms
    - 最小: 35ms
    - 最大: 60ms
    - P50: 44ms
    - P95: 55ms
    - P99: 58ms

  资源使用:
    - 初始内存: 10 MB
    - 最大内存: 25 MB
    - 增长内存: 15 MB
    - 初始Goroutines: 5
    - 最大Goroutines: 55
    - 增长Goroutines: 50

【5/8】性能基准测试...
  测试场景:
    场景1: 单用户连续查询（100次）
      完成100次查询，平均耗时: 15ms
    场景2: 单用户连续写入（100次）
      完成100次写入，平均耗时: 8ms
    场景3: Checkpoint性能测试（10次）
      完成10次Checkpoint，平均耗时: 45ms
    场景4: 高并发查询（50并发 × 10次）
      完成500次并发查询，平均耗时: 15ms
    场景5: 高并发写入（50并发 × 10次）
      完成500次并发写入，平均耗时: 8ms

  性能指标:
    所有查询统计:
      - 平均: 15ms
      - 最小: 10ms
      - 最大: 25ms
      - P50: 14ms
      - P95: 20ms
      - P99: 23ms

    所有写入统计:
      - 平均: 8ms
      - 最小: 5ms
      - 最大: 15ms
      - P50: 7ms
      - P95: 12ms
      - P99: 14ms

    所有Checkpoint统计:
      - 平均: 45ms
      - 最小: 35ms
      - 最大: 60ms
      - P50: 44ms
      - P95: 55ms
      - P99: 58ms

    - 总操作数: 1220
    - 测试时长: 2.9s
    - QPS: 420.69

  性能评估:
    ✅ 良好: QPS > 500

【6/8】数据库操作效率测试...
  测试场景:
    场景1: 无索引查询性能
    场景2: 有索引查询性能
    场景3: 批量插入性能
    场景4: 批量更新性能
    场景5: Checkpoint查询性能
    场景6: 全量历史查询性能

  数据库操作性能:
    - 无索引查询(100次): 2.5s
    - 有索引查询(100次): 0.8s
    - 批量插入(100条): 0.5s
    - 批量更新(100条): 0.6s
    - Checkpoint查询(100次): 1.2s
    - 全量历史查询(10次): 2.0s

  性能对比:
    - 索引提升: 68.00%
    - Checkpoint查询占比: 60.00%

  性能评估:
    ✅ 索引效果优秀（提升>50%）
    ⚠️  Checkpoint查询性能一般（>=50%）

【7/8】安全性测试...
  测试场景:
    场景1: SQL注入防护测试
      完成6个测试用例，通过6个
    场景2: 输入验证测试
      完成7个测试用例，通过7个
    场景3: 空指针防护测试
      完成4个测试用例，通过4个
    场景4: 并发安全测试
      完成1000次写入操作，成功950次，失败50次

  安全性测试结果:
    - SQL注入测试: 6/6 通过
    - 输入验证测试: 7/7 通过
    - 空指针防护测试: 4/4 通过
    - 并发安全测试: 0/1 通过

  安全性评估:
    ⚠️  安全性一般（通过率≥60%）

【8/8】生成综合评估报告...
========================================
综合评估报告
========================================

【1/5】并发安全性评估
    - 总操作数: 2000
    - 超时次数: 0 (0.00%)
    - 错误次数: 0
    - 并发安全评分: 100/100

【2/5】性能评估
    - QPS: 420.69
    - 平均查询时间: 15ms
    - 平均写入时间: 8ms
    - 性能评分: 70/100

【3/5】数据库效率评估
    - 索引效果: 68.00% 提升
    - Checkpoint查询占比: 60.00%
    - 数据库效率评分: 85/100

【4/5】安全性评估
    - SQL注入防护: 6/6
    - 输入验证: 7/7
    - 空指针防护: 4/4
    - 安全性评分: 95/100

【5/5】总体评估
    - 综合评分: 87/100

    ✅ 评级: 良好
    系统整体表现良好，建议在投入生产环境前进行少量优化。

========================================
优化建议
========================================

基于测试结果，以下优化建议按优先级排序：

🟡 中优先级 - 性能优化:
  1. 优化数据库查询语句
  2. 添加更多缓存策略

🟡 中优先级 - 数据库优化:
  1. 优化 Checkpoint 查询语句
```

---

## 常见问题

### Q1: 测试失败怎么办？

**A**: 检查以下几点：

1. **数据库连接失败**
   - 检查数据库是否运行
   - 检查连接配置是否正确
   - 检查数据库用户权限

2. **Redis连接失败**
   - 检查Redis是否运行
   - 检查Redis配置是否正确

3. **测试超时**
   - 增加测试时长
   - 减少并发用户数
   - 检查系统资源

### Q2: 如何调整测试强度？

**A**: 根据您的需求调整配置参数：

```go
// 轻量测试（快速验证）
ConcurrentUsers: 10,
TestDuration:    10,
MessageCount:    10,

// 标准测试（全面评估）
ConcurrentUsers: 50,
TestDuration:    30,
MessageCount:    20,

// 压力测试（极限测试）
ConcurrentUsers: 100,
TestDuration:    60,
MessageCount:    50,
```

### Q3: 测试结果如何解读？

**A**: 参考以下标准：

- **并发安全评分**: 超时率 < 1% 为优秀
- **性能评分**: QPS > 1000 为优秀
- **数据库效率评分**: 索引提升 > 50% 为优秀
- **安全性评分**: 通过率 ≥ 95% 为优秀
- **综合评分**: ≥ 90 为优秀

### Q4: 如何对比优化前后的效果？

**A**: 在优化前后分别运行测试，对比关键指标：

```bash
# 优化前
go run performance_and_security_test.go > before_optimization.txt

# 优化后
go run performance_and_security_test.go > after_optimization.txt

# 对比
diff before_optimization.txt after_optimization.txt
```

---

## 总结

### 测试脚本的核心价值

1. **全面评估**: 涵盖并发安全、性能、数据库效率和安全性四个维度
2. **量化指标**: 提供详细的性能指标和统计数据
3. **优化建议**: 基于测试结果提供针对性的优化建议
4. **易于使用**: 简单的配置和运行流程

### 使用建议

1. **定期测试**: 在代码变更后运行测试，确保性能和安全性
2. **对比分析**: 优化前后对比，量化优化效果
3. **持续改进**: 根据测试结果持续优化系统
4. **生产部署**: 确保综合评分 ≥ 75 才能投入生产环境

---

**文档结束**
