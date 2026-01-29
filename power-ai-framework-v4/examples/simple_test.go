package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// 简化版测试示例 - 不依赖 etcd 等外部服务
func main() {
	fmt.Println("🚀 启动简化版 Agent 测试...")

	// 设置必要的环境变量（避免依赖外部服务）
	os.Setenv("IP_ADDR", "127.0.0.1")
	os.Setenv("PORT", "8080")

	// 创建 Gin 路由器
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Content-Length, Accept-Encoding, X-CSRF-Token, token, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 模拟框架的路由结构
	baseUrl := "/demo/agent"

	// 基础路由
	r.GET(baseUrl+"/health", healthHandler)
	r.GET(baseUrl+"/version", versionHandler)

	// 自定义路由
	r.GET(baseUrl+"/status", statusHandler)
	r.POST(baseUrl+"/send_msg", sendMsgHandler)
	r.POST(baseUrl+"/echo", echoHandler)

	fmt.Println("📍 测试地址:")
	fmt.Println("   健康检查: http://localhost:8080/demo/agent/health")
	fmt.Println("   版本信息: http://localhost:8080/demo/agent/version")
	fmt.Println("   状态查询: http://localhost:8080/demo/agent/status")
	fmt.Println("   消息发送: POST http://localhost:8080/demo/agent/send_msg")
	fmt.Println("   回声测试: POST http://localhost:8080/demo/agent/echo")
	fmt.Println()
	fmt.Println("🔧 测试命令:")
	fmt.Println("   curl http://localhost:8080/demo/agent/health")
	fmt.Println("   curl -X POST http://localhost:8080/demo/agent/send_msg -H \"Content-Type: application/json\" -d '{\"message\":\"Hello AI!\"}'")

	// 启动服务器
	log.Fatal(r.Run(":8080"))
}

// 健康检查处理器
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    "healthy",
	})
}

// 版本信息处理器
func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"agent_code":        "demo-agent",
			"agent_name":        "演示代理",
			"agent_version":     "v1.0.0",
			"framework_version": "v1.0.27",
		},
	})
}

// 状态查询处理器
func statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"uptime":    "测试中",
		"version":   "v1.0.0",
		"timestamp": "2026-01-26T10:00:00Z",
	})
}

// send_msg 处理器 - AI 服务的核心接口
func sendMsgHandler(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求格式",
			"error":   err.Error(),
		})
		return
	}

	// 模拟 AI 处理逻辑
	message := "未知消息"
	if msg, ok := request["message"]; ok {
		message = fmt.Sprintf("%v", msg)
	}

	response := gin.H{
		"code":    200,
		"message": "处理成功",
		"data": gin.H{
			"reply":     fmt.Sprintf("AI 回复: 收到您的消息「%s」", message),
			"timestamp": "2026-01-26T10:00:00Z",
			"agent":     "demo-agent",
			"request":   request,
		},
	}

	c.JSON(http.StatusOK, response)
}

// 回声测试处理器
func echoHandler(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 JSON 格式",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "回声测试成功",
		"data": gin.H{
			"echo":        body,
			"received_at": "2026-01-26T10:00:00Z",
			"server":      "demo-agent",
		},
	})
}
