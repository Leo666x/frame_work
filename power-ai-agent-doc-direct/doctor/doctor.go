package doctor

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strings"
)

// SendMsg post send_msg 路由
func (a *DoctorAgent) SendMsg(c *gin.Context) {
	req, resp, event, ok := powerai.DoValidateAgentRequest(c, a.App.Manifest.Code)
	if !ok {
		return
	}

	//初始化配置
	a.InitReportConfig(resp, event, map[string]interface{}{})

	// 加载redis   私有状态
	// 数据模拟
	session := a.GetRedis(c, req)
	// 1. 加载私有状态
	var mySlots DoctorSlots
	// 如果 Redis 里有数据，mySlots 会被填充；如果没有，mySlots 保持默认空值
	err := LoadAgentSlots(session.GlobalState.AgentSlots, "power-ai-agent-doc-direct", &mySlots)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功加载记忆", a.App.Manifest.Code), err)
		mySlots = DoctorSlots{} // 重置
		//_ = event.WriteAgentResponseError(resp, ErrorMemory, fmt.Sprintf("[%s]-未成功加载记忆", a.App.Manifest.Code))
		//return   // 业务评估 直接返回错误 或 next
	}

	// ====================================================
	// 分支 1: 处于等待用户选择状态 (多轮消歧)
	// ====================================================
	if mySlots.Status == "waiting_for_selection" {
		a.router1_WaitingForEelection(event, req, resp, session, &mySlots)
		return
	}

	extractResp, err := a.extractIntentAndName(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "大模型调用-关键词提取", fmt.Sprintf("[%s]-未成功提取医生姓名", a.App.Manifest.Code), err)
		_ = event.WriteAgentResponseError(resp, ErrorCallLlm, fmt.Sprintf("[%s]-未成功提取医生姓名", a.App.Manifest.Code))
		return
	}

	// ====================================================
	// 分支 2: 名字不完整 (e.g. "找赵医生")
	// ====================================================
	if !extractResp.IsFullName {
		a.router2_IsFullName(event, req, resp, &mySlots, session)
		return
	}

	// ====================================================
	// 分支 3: 名字完整 (e.g. "赵三") -> 执行 RAG
	// ====================================================
	a.router3_SearchDoctorKB(event, req, resp, session, &mySlots, extractResp.DoctorName)

}

func (a *DoctorAgent) extractIntentAndName(req *server.AgentRequest) (ExtractionResult, error) {
	// 使用之前定义的 Prompt
	prompt := strings.Replace(PROMPT_INTERNAL_ROUTER, "UserQuery", req.Query, 1)

	respLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return ExtractionResult{}, err
	}
	resStr := LlmRespDeal(respLlm)

	var res ExtractionResult
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		return ExtractionResult{}, err
	}
	return res, nil
}
