# NewAgent 函数详细分析

## 🔍 执行流程分解

```go
func NewAgent(manifest string, opts ...Option) (*AgentApp, error) {
```

### 步骤 1: 解析 Manifest
```go
mf, err := initManifest(manifest)
if err != nil {
    return nil, err
}
```
**作用**: 解析 Agent 的基本信息（代码、名称、版本、描述）
**重要性**: Manifest 是 Agent 的身份证，缺一不可

### 步骤 2: 处理选项参数
```go
newOpts := newOptions(opts)
```
**作用**: 合并用户传入的配置选项（路由、回调函数等）

### 步骤 3: 环境初始化
```go
env.Init()    // 加载环境变量和默认配置
tools.Init()  // 初始化 HTTP 客户端工具
```
**作用**: 设置运行环境，准备基础工具

### 步骤 4: 连接 etcd
```go
etcd, err := initEtcd()
if err != nil {
    return nil, fmt.Errorf("init etcd middleware err:%s", err.Error())
}
```
**作用**: 连接分布式配置中心，这是框架的核心依赖

### 步骤 5: 创建 AgentApp 实例
```go
a := &AgentApp{
    Manifest:    mf,
    HttpServer:  server.New(),  // 创建 HTTP 服务器
    OnShutdown:  newOpts.OnShutDown,
    etcd:        etcd,
    agentConfig: newAgentConfig(...),  // 配置管理
    agentClient: newAgentClient(...),  // 服务发现
}
```

### 步骤 6: 注册路由
```go
// 生成 base URL（将 agent-code 转换为 /agent/code）
baseUrl := strings.ReplaceAll(mf.Code, "-", "/")

// 注册基础路由
a.HttpServer.GET(fmt.Sprintf("/%s/health", baseUrl), a.health)
a.HttpServer.GET(fmt.Sprintf("/%s/version", baseUrl), a.version)

// 注册自定义路由
for k, v := range newOpts.PostRouters {
    a.HttpServer.POST(fmt.Sprintf("/%s/%s", baseUrl, k), v)
}
for k, v := range newOpts.GetRouters {
    a.HttpServer.GET(fmt.Sprintf("/%s/%s", baseUrl, k), v)
}
```

## 🎯 关键设计理念

### 1. 约定优于配置
- Agent 代码自动转换为 URL 路径
- 自动注册健康检查和版本接口
- 默认配置覆盖大部分使用场景

### 2. 依赖注入模式
- 通过 Options 模式注入自定义配置
- 支持多种中间件的可选初始化
- 灵活的扩展机制

### 3. 微服务架构
- etcd 作为服务注册中心
- 支持服务发现和配置管理
- 每个 Agent 都是独立的微服务

## 🔧 实际应用示例

假设你要创建一个名为 "chat-bot" 的 AI 聊天机器人：

```go
manifest := `{
    "code": "chat-bot",
    "name": "智能聊天机器人", 
    "version": "v1.0.0",
    "description": "基于大语言模型的智能对话系统"
}`

app, err := powerai.NewAgent(
    manifest,
    powerai.WithSendMsgRouter(chatHandler),  // 聊天接口
    powerai.WithCustomGetRouter("models", listModelsHandler), // 模型列表
)
```

这会自动创建以下路由：
- GET  `/chat/bot/health`     - 健康检查
- GET  `/chat/bot/version`    - 版本信息  
- POST `/chat/bot/send_msg`   - 聊天接口
- GET  `/chat/bot/models`     - 模型列表

## 💡 学习要点

1. **理解 Manifest 的重要性** - 它定义了 Agent 的身份
2. **掌握 Options 模式** - 这是 Go 中常用的配置模式
3. **熟悉路由生成规则** - code 字段直接影响 API 路径
4. **认识依赖关系** - etcd 是必需的，其他组件按需初始化