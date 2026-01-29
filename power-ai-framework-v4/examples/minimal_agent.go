package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
)

// 最小 Agent 示例
func main() {
	// 1. 定义 Agent 清单
	manifest := map[string]string{
		"code":        "demo-agent",
		"name":        "演示代理",
		"version":     "v1.0.0",
		"description": "这是一个最小的 Agent 示例",
	}

	manifestJson, _ := json.Marshal(manifest)

	// 2. 创建 Agent 应用
	app, err := powerai.NewAgent(
		string(manifestJson),
		// 注册自定义路由
		powerai.WithSendMsgRouter(sendMsgHandler),
		powerai.WithCustomGetRouter("status", statusHandler),
		powerai.WithCustomPostRouter("echo", echoHandler),
	)

	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}

	fmt.Println("🚀 启动 Demo Agent...")
	fmt.Println("📍 访问地址:")
	fmt.Println("   健康检查: http://localhost:8080/demo/agent/health")
	fmt.Println("   版本信息: http://localhost:8080/demo/agent/version")
	fmt.Println("   状态查询: http://localhost:8080/demo/agent/status")
	fmt.Println("   消息发送: POST http://localhost:8080/demo/agent/send_msg")
	fmt.Println("   回声测试: POST http://localhost:8080/demo/agent/echo")

	// 3. 启动服务
	app.Run()
}

// send_msg 处理器 - AI 服务的核心接口
func sendMsgHandler(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	// 模拟 AI 处理逻辑
	response := map[string]interface{}{
		"code":    200,
		"message": "处理成功",
		"data": map[string]interface{}{
			"reply":     fmt.Sprintf("收到消息: %v", request["message"]),
			"timestamp": "2026-01-26T10:00:00Z",
			"agent":     "demo-agent",
		},
	}

	c.JSON(http.StatusOK, response)
}

// 状态查询处理器
func statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "running",
		"uptime":  "1h30m",
		"version": "v1.0.0",
	})
}

// 回声测试处理器
func echoHandler(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 JSON"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"echo":        body,
		"received_at": "2026-01-26T10:00:00Z",
	})
}
