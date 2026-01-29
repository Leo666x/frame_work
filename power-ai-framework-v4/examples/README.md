# 最小 Agent 示例

## 🎯 学习目标

通过这个示例，你将学会：
1. 如何创建一个最小的 Agent 应用
2. 如何注册自定义路由
3. 如何处理 HTTP 请求和响应
4. 理解框架的基本启动流程

## 🚀 快速开始

### 1. 环境准备

确保你已经安装了：
- Go 1.19+
- etcd（框架依赖）

### 2. 运行示例

```bash
# 进入示例目录
cd power-ai-framework-v4/examples

# 运行最小 Agent
go run minimal_agent.go
```

### 3. 测试接口

启动成功后，你可以测试以下接口：

#### 健康检查
```bash
curl http://localhost:8080/demo/agent/health
```

#### 版本信息
```bash
curl http://localhost:8080/demo/agent/version
```

#### 状态查询
```bash
curl http://localhost:8080/demo/agent/status
```

#### 发送消息（AI 核心接口）
```bash
curl -X POST http://localhost:8080/demo/agent/send_msg \
  -H "Content-Type: application/json" \
  -d '{"message": "你好，AI助手！", "user_id": "test_user"}'
```

#### 回声测试
```bash
curl -X POST http://localhost:8080/demo/agent/echo \
  -H "Content-Type: application/json" \
  -d '{"test": "hello world", "number": 123}'
```

## 📋 关键知识点

### 1. Manifest 配置

```go
manifest := map[string]string{
    "code":        "demo-agent",      // Agent 代码（用于路由生成）
    "name":        "演示代理",         // Agent 名称
    "version":     "v1.0.0",         // 版本号
    "description": "这是一个最小的 Agent 示例", // 描述
}
```

- `code` 会被转换为路由前缀：`demo-agent` → `/demo/agent`
- 所有字段都是必填的，缺少任何一个都会导致启动失败

### 2. 路由注册方式

```go
// 专用的 send_msg 路由（AI 服务标准接口）
powerai.WithSendMsgRouter(sendMsgHandler)

// 自定义 GET 路由
powerai.WithCustomGetRouter("status", statusHandler)

// 自定义 POST 路由
powerai.WithCustomPostRouter("echo", echoHandler)
```

### 3. 自动生成的基础路由

框架会自动为每个 Agent 生成：
- `/{agent_code}/health` - 健康检查
- `/{agent_code}/version` - 版本信息

### 4. 请求处理模式

```go
func handlerFunc(c *gin.Context) {
    // 1. 解析请求
    var request map[string]interface{}
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
        return
    }
    
    // 2. 业务逻辑处理
    // ... 你的处理逻辑
    
    // 3. 返回响应
    c.JSON(http.StatusOK, response)
}
```

## 🔍 下一步学习

完成这个示例后，你应该能够：
- ✅ 理解 Agent 的创建和启动流程
- ✅ 知道如何注册自定义路由
- ✅ 掌握基本的请求处理模式

准备好进入第2阶段：**请求协议与流式响应** 了吗？