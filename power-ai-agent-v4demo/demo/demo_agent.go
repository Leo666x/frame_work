package v4demo

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
)

type DemoAgent struct {
	App *powerai.AgentApp
}

// OnShutdown 智能体退出的时候回调，可有可无
func (d *DemoAgent) OnShutdown(c context.Context) {

}

// SendMsg post send_msg 路由
func (d *DemoAgent) SendMsg(c *gin.Context) {
	req, resp, event, ok := powerai.DoValidateAgentRequest(c, d.App.Manifest.Code)
	if !ok {
		return
	}
	fmt.Println(resp)
	fmt.Println(req)
	fmt.Println(event)
}

// DemoPost post DemoTest路由
func (d *DemoAgent) DemoPost(c *gin.Context) {

}

// DemoGet Get DemoTest路由
func (d *DemoAgent) DemoGet(c *gin.Context) {

}
