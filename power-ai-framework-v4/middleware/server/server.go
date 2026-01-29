package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"time"
)

type HttpServer struct {
	*gin.Engine
}

func New() *HttpServer {
	s := &HttpServer{
		gin.New(),
	}

	//日志打印中间件,跨域中间件
	s.Use(loggerMiddleware, corsMiddleware)

	return s
}

func (s *HttpServer) RunServer(ip, port string) error {
	return s.Engine.Run(fmt.Sprintf("%s:%s", ip, port))
}

func (s *HttpServer) StopServer() {

}

func loggerMiddleware(c *gin.Context) {
	// Start timer
	start := time.Now()
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery
	method := c.Request.Method
	c.Next()
	// Stop timer
	end := time.Now()
	latency := end.Sub(start)
	statusCode := c.Writer.Status()
	clientIP := c.ClientIP()
	if raw != "" {
		path = path + "?" + raw
	}
	xlog.LogInfoF("10000", "httpserver", "access", fmt.Sprintf("METHOD:%s | PATH:%s | CODE:%d | IP:%s | TIME:%d ", method, path, statusCode, clientIP, latency/time.Millisecond))
}

// CorsMiddleware 跨域中间件
func corsMiddleware(c *gin.Context) {

	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Content-Length, Accept-Encoding, X-CSRF-Token, token, accept, origin, Cache-Control, X-Requested-With, appid, noncestr, sign, timestamp")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT,DELETE,PATCH")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
	}
	c.Next()
}
