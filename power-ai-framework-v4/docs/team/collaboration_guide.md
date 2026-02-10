# Power AI Framework V4 - 团队协作指南

> **版本**: v4.0.0
> **更新时间**: 2026-01-26
> **维护团队**: AI Team

## 📋 目录

1. [团队角色与职责](#团队角色与职责)
2. [开发流程](#开发流程)
3. [代码规范](#代码规范)
4. [测试规范](#测试规范)
5. [文档规范](#文档规范)
6. [协作工具](#协作工具)
7. [常见问题](#常见问题)

---

## 团队角色与职责

### 核心团队

| 角色 | 职责 | 联系方式 |
|------|------|----------|
| **架构师** | 架构设计、技术选型、代码审查 | architect@example.com |
| **后端开发** | API开发、业务逻辑实现 | backend@example.com |
| **智能体开发** | 智能体逻辑、提示词工程 | agent@example.com |
| **测试工程师** | 测试用例编写、性能测试 | qa@example.com |
| **运维工程师** | 部署、监控、维护 | ops@example.com |

### 跨团队协作

| 团队 | 协作内容 | 对接人 |
|------|----------|--------|
| **产品团队** | 需求沟通、功能验收 | product@example.com |
| **前端团队** | API对接、数据格式 | frontend@example.com |
| **数据团队** | 数据分析、报表对接 | data@example.com |

---

## 开发流程

### 1. 需求分析

```
产品团队 → 需求文档 → 技术评审 → 开发计划
```

**检查清单**:
- [ ] 需求文档是否完整
- [ ] 技术方案是否可行
- [ ] 开发计划是否合理
- [ ] 资源是否充足

### 2. 设计阶段

```
架构设计 → 数据库设计 → API设计 → 接口文档
```

**输出文档**:
- 架构设计文档
- 数据库设计文档
- API接口文档
- 时序图/流程图

### 3. 开发阶段

```
环境搭建 → 功能开发 → 单元测试 → 代码审查
```

**开发规范**:
- 遵循代码规范（见下方）
- 编写单元测试
- 提交代码前自测
- 代码覆盖率 > 80%

### 4. 测试阶段

```
单元测试 → 集成测试 → 性能测试 → 安全测试
```

**测试标准**:
- 单元测试通过率 100%
- 集成测试通过率 > 95%
- 性能测试达标
- 安全测试无高危漏洞

### 5. 部署阶段

```
预发布环境 → 灰度发布 → 全量发布 → 监控
```

**部署检查**:
- [ ] 配置文件是否正确
- [ ] 数据库迁移是否完成
- [ ] 依赖服务是否就绪
- [ ] 监控告警是否配置

---

## 代码规范

### 1. 命名规范

#### 文件命名

```go
// 使用小写字母和下划线
powerai_memory.go
powerai_short_memory.go
powerai_db.go
```

#### 变量命名

```go
// 驼峰命名法
var conversationID string
var sessionValue *SessionValue
var maxRetryCount int

// 常量使用大写字母和下划线
const (
    MaxQueryLength = 10000
    DefaultTimeout = 30 * time.Second
)
```

#### 函数命名

```go
// 驼峰命名法，首字母大写（导出函数）
func QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error)
func WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error)

// 首字母小写（私有函数）
func normalizeSessionValue(session *SessionValue) *SessionValue
func buildHistoryFromAIMessages(messages []*AIMessage) string
```

#### 结构体命名

```go
// 驼峰命名法，首字母大写（导出结构体）
type MemoryQueryRequest struct {}
type MemoryContext struct {}

// 首字母小写（私有结构体）
type internalState struct {}
```

### 2. 注释规范

#### 包注释

```go
// Package powerai 提供了 Power AI Framework V4 的核心功能
//
// 主要功能:
//   - 记忆管理（短期记忆、长期记忆）
//   - 智能体执行
//   - 意图识别
//   - 工具调用
//
// 使用示例:
//   app := powerai.NewAgentApp()
//   ctx, err := app.QueryMemoryContext(req)
package powerai
```

#### 函数注释

```go
// QueryMemoryContext 查询记忆上下文
// 根据会话ID查询并构建适合当前对话的记忆上下文
//
// 参数:
//   - req: 记忆查询请求
// 返回:
//   - *MemoryContext: 记忆上下文
//   - error: 错误信息
//
// 使用场景:
//   - 每次处理用户消息前
//   - 需要获取对话历史时
//
// 工作流程:
//   1. 参数验证
//   2. 获取短期记忆（Redis）
//   3. 根据Checkpoint查询消息（PostgreSQL）
//   4. 构建对话历史
//   5. 计算Token占用率
//   6. 判断是否需要触发摘要
//
// 注意事项:
//   - 如果Redis读取失败，会创建默认会话状态并继续执行
//   - 如果数据库查询失败，会将messages设为nil并继续执行
//   - 这确保了系统的健壮性，不会因为单点故障导致整个流程中断
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    // ...
}
```

#### 结构体注释

```go
// SessionValue 对应 Redis Value 的顶层结构
// 存储会话的完整状态信息，包括元数据、流程上下文、消息上下文、全局状态和用户快照
//
// 序列化格式: JSON
// 存储位置: Redis
// Key格式: short_term_memory:session:{conversation_id}
// 过期时间: 30分钟（1800秒）
type SessionValue struct {
    Meta           *MetaInfo       `json:"meta"`           // 元信息
    FlowContext    *FlowContext    `json:"flow_context"`   // 流程上下文
    MessageContext *MessageContext `json:"message_context"` // 消息上下文（核心）
    GlobalState    *GlobalState    `json:"global_state"`   // 全局共享状态
    UserSnapshot   *UserProfile    `json:"user_snapshot"`   // 用户快照
}
```

#### 行内注释

```go
// 获取会话锁（防止并发冲突）
lock := getSessionLock(conversationID)
lock.Lock()
defer lock.Unlock()

// 防御性编程：确保 UserSnapshot 不为 nil
if session.UserSnapshot != nil {
    session.UserSnapshot.UserID = req.UserID
}

// 性能优化：预分配容量（假设每条消息平均200字符）
estimatedSize := len(messages) * 200
builder := strings.Builder{}
builder.Grow(estimatedSize)
```

### 3. 错误处理规范

#### 错误返回

```go
// 总是返回错误信息
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
    if req == nil {
        return nil, fmt.Errorf("memory query request is nil")
    }
    // ...
}

// 使用 fmt.Errorf 包装错误
if err := a.SetShortMemory(conversationID, session); err != nil {
    return nil, fmt.Errorf("failed to set short memory: %w", err)
}
```

#### 降级处理

```go
// Redis读取失败时，使用默认值
session, err := a.GetShortMemory(req.ConversationID)
if err != nil {
    xlog.LogWarnF("MEMORY", "QueryMemoryContext", "GetShortMemory",
        fmt.Sprintf("failed to get short memory: %v, using default session", err))
    session = newDefaultSessionValue(req.ConversationID, req.PatientID)
}
```

#### 日志记录

```go
// 使用统一的日志格式
xlog.LogErrorF("MEMORY", "QueryMemoryContext", "GetShortMemory",
    fmt.Sprintf("failed to get short memory: %v", err))

xlog.LogWarnF("MEMORY", "QueryMemoryContext", "GetShortMemory",
    fmt.Sprintf("failed to get short memory: %v, using default session", err))

xlog.LogInfoF("MEMORY", "QueryMemoryContext", "GetShortMemory",
    fmt.Sprintf("successfully retrieved session: %s", conversationID))
```

### 4. 并发安全规范

#### 使用锁

```go
// 获取会话锁（防止并发冲突）
lock := getSessionLock(conversationID)
lock.Lock()
defer lock.Unlock()

// 修改会话状态...
```

#### 避免数据竞争

```go
// ✅ 正确：使用锁保护共享数据
lock := getSessionLock(conversationID)
lock.Lock()
defer lock.Unlock()
session.FlowContext.TurnCount++

// ❌ 错误：没有保护共享数据
session.FlowContext.TurnCount++
```

---

## 测试规范

### 1. 单元测试

#### 测试文件命名

```go
// 测试文件名：{源文件名}_test.go
powerai_memory_test.go
powerai_short_memory_test.go
```

#### 测试函数命名

```go
// 格式：Test{函数名}
func TestQueryMemoryContext(t *testing.T) {}
func TestWriteTurn(t *testing.T) {}
func TestCheckpointShortMemory(t *testing.T) {}
```

#### 测试示例

```go
func TestQueryMemoryContext(t *testing.T) {
    app := setupTestApp(t)
    defer app.Close()

    req := &MemoryQueryRequest{
        ConversationID: "test_conv_001",
        Query:          "测试查询",
    }

    ctx, err := app.QueryMemoryContext(req)
    if err != nil {
        t.Fatalf("QueryMemoryContext failed: %v", err)
    }

    if ctx.ConversationID != "test_conv_001" {
        t.Errorf("Expected conversation ID %s, got %s", "test_conv_001", ctx.ConversationID)
    }
}
```

### 2. 集成测试

```go
func TestMemoryIntegration(t *testing.T) {
    // 启动测试环境
    app := setupTestApp(t)
    defer app.Close()

    // 测试完整流程
    conversationID := "test_conv_integration"

    // 1. 创建会话
    err := app.CreateShortMemory(&server.AgentRequest{
        ConversationId: conversationID,
        UserId:         "test_user",
    })
    if err != nil {
        t.Fatalf("CreateShortMemory failed: %v", err)
    }

    // 2. 写入对话轮次
    _, err = app.WriteTurn(&MemoryWriteRequest{
        ConversationID: conversationID,
        UserID:         "test_user",
        AgentCode:      "test_agent",
        UserQuery:      "测试问题",
        AgentResponse:  "测试回答",
    })
    if err != nil {
        t.Fatalf("WriteTurn failed: %v", err)
    }

    // 3. 查询记忆上下文
    ctx, err := app.QueryMemoryContext(&MemoryQueryRequest{
        ConversationID: conversationID,
        Query:          "新问题",
    })
    if err != nil {
        t.Fatalf("QueryMemoryContext failed: %v", err)
    }

    // 4. 验证结果
    if ctx.Session.FlowContext.TurnCount != 1 {
        t.Errorf("Expected TurnCount 1, got %d", ctx.Session.FlowContext.TurnCount)
    }
}
```

### 3. 性能测试

```go
func BenchmarkQueryMemoryContext(b *testing.B) {
    app := setupTestApp(b)
    defer app.Close()

    req := &MemoryQueryRequest{
        ConversationID: "test_conv_bench",
        Query:          "测试查询",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := app.QueryMemoryContext(req)
        if err != nil {
            b.Fatalf("QueryMemoryContext failed: %v", err)
        }
    }
}
```

### 4. 测试覆盖率

```bash
# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看覆盖率
go tool cover -func=coverage.out

# 生成HTML报告
go tool cover -html=coverage.out -o coverage.html
```

**覆盖率要求**:
- 核心业务逻辑: > 90%
- 工具函数: > 80%
- 整体覆盖率: > 80%

---

## 文档规范

### 1. API文档

#### 文档位置

```
docs/api/
├── memory_management_api.md
├── agent_execution_api.md
└── intent_recognition_api.md
```

#### 文档格式

```markdown
# API名称

> **版本**: v1.0.0
> **更新时间**: 2026-01-26

## 概述
简要描述API的功能

## 请求
### 请求参数
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 用户ID |

### 请求示例
```json
{
  "id": "123"
}
```

## 响应
### 响应参数
| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | 用户名称 |

### 响应示例
```json
{
  "name": "张三"
}
```

## 错误码
| 错误码 | 说明 |
|--------|------|
| 400 | 参数错误 |

## 使用示例
```go
// 示例代码
```
```

### 2. 架构文档

```
docs/architecture/
├── overview.md
├── memory_architecture.md
└── agent_architecture.md
```

### 3. 开发指南

```
docs/guides/
├── getting_started.md
├── development_guide.md
└── deployment_guide.md
```

---

## 协作工具

### 1. 版本控制

#### Git工作流

```
main (生产环境)
  ↑
develop (开发环境)
  ↑
feature/* (功能分支)
```

#### 分支命名规范

```
feature/{功能名}
bugfix/{问题描述}
hotfix/{紧急修复}
release/{版本号}
```

#### 提交信息规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type类型**:
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**:
```
feat(memory): add checkpoint retry mechanism

- Add retry logic for UUID collision
- Add checkMessageIDExists function
- Improve error handling

Closes #123
```

### 2. 代码审查

#### 审查清单

- [ ] 代码是否符合规范
- [ ] 是否有足够的注释
- [ ] 是否有单元测试
- [ ] 是否有性能问题
- [ ] 是否有安全问题
- [ ] 是否有错误处理

#### 审查工具

- **GitLab Merge Request**
- **GitHub Pull Request**
- **Gerrit Code Review**

### 3. 持续集成

#### CI流程

```
代码提交 → 自动化测试 → 代码审查 → 合并 → 自动部署
```

#### CI工具

- **Jenkins**
- **GitLab CI**
- **GitHub Actions**

### 4. 项目管理

#### 任务管理

- **JIRA**: 任务跟踪
- **Trello**: 看板管理
- **飞书**: 任务协作

#### 文档协作

- **Confluence**: 文档管理
- **飞书文档**: 实时协作
- **GitBook**: API文档

### 5. 通讯工具

- **飞书**: 即时通讯
- **邮件**: 正式通知
- **Slack**: 国际团队协作

---

## 常见问题

### 1. 开发环境问题

#### Q: 如何启动开发环境？

A: 
```bash
# 1. 克隆代码
git clone https://github.com/example/power-ai-framework-v4.git
cd power-ai-framework-v4

# 2. 安装依赖
go mod download

# 3. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，配置数据库和Redis连接信息

# 4. 启动服务
go run main.go
```

#### Q: 如何运行测试？

A:
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./powerai/

# 运行特定测试函数
go test -run TestQueryMemoryContext ./powerai/

# 查看覆盖率
go test -cover ./...
```

### 2. 代码问题

#### Q: 如何添加新的智能体？

A:
1. 在 `agents/` 目录下创建新的智能体文件
2. 实现智能体接口
3. 在 `powerai_agent.go` 中注册智能体
4. 编写测试用例
5. 更新文档

#### Q: 如何修改记忆管理逻辑？

A:
1. 修改 `powerai_memory.go` 或 `powerai_short_memory.go`
2. 确保符合代码规范
3. 添加单元测试
4. 运行测试确保通过
5. 更新API文档

### 3. 部署问题

#### Q: 如何部署到生产环境？

A:
1. 确保所有测试通过
2. 创建发布分支
3. 更新版本号
4. 打包编译
5. 部署到预发布环境
6. 验证功能
7. 灰度发布
8. 全量发布
9. 监控告警

#### Q: 如何回滚部署？

A:
```bash
# 1. 停止服务
systemctl stop power-ai-framework

# 2. 回滚到上一个版本
cd /opt/power-ai-framework
git checkout <previous-version>

# 3. 重新编译
go build -o power-ai-framework main.go

# 4. 启动服务
systemctl start power-ai-framework

# 5. 验证服务
curl http://localhost:8080/health
```

### 4. 协作问题

#### Q: 如何申请代码审查？

A:
1. 推送代码到远程仓库
2. 创建 Merge Request / Pull Request
3. 填写审查模板
4. @ 相关审查人员
5. 等待审查反馈
6. 根据反馈修改代码
7. 合并代码

#### Q: 如何报告bug？

A:
1. 在项目管理工具中创建issue
2. 填写bug模板
3. 提供复现步骤
4. @ 相关开发人员
5. 跟踪bug修复进度

---

## 最佳实践

### 1. 代码质量

- ✅ 遵循代码规范
- ✅ 编写清晰的注释
- ✅ 编写单元测试
- ✅ 进行代码审查
- ✅ 持续重构

### 2. 性能优化

- ✅ 避免不必要的数据库查询
- ✅ 使用缓存减少重复计算
- ✅ 优化字符串拼接
- ✅ 使用并发处理
- ✅ 定期进行性能测试

### 3. 安全防护

- ✅ 验证所有输入
- ✅ 使用参数化查询
- ✅ 防止SQL注入
- ✅ 防止XSS攻击
- ✅ 定期进行安全审计

### 4. 文档维护

- ✅ 及时更新API文档
- ✅ 编写清晰的注释
- ✅ 维护架构文档
- ✅ 记录重要决策
- ✅ 分享技术经验

---

## 联系方式

- **技术支持**: tech-support@example.com
- **项目管理**: project@example.com
- **团队负责人**: lead@example.com

---

**文档结束**
