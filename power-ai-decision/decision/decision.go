package decision

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
)

type DecisionAgent struct {
	App *powerai.AgentApp
}

// OnShutdown 智能体退出的时候回调，可有可无
func (a *DecisionAgent) OnShutdown(c context.Context) {

}

type OutReponse struct {
	ClassiFication string     `json:"classification"`
	UnKnown        string     `json:"unknown"`
	Data           dataStruct `json:"data"`
}

type dataStruct struct {
	Msg     interface{} `json:"msg"`
	EndFlag string      `json:"endflag"`
}

// SendMsg post send_msg 路由
func (a *DecisionAgent) SendMsg(c *gin.Context) {
	req, _, _, ok := powerai.DoValidateAgentRequest(c, a.App.Manifest.Code)
	if !ok {
		return
	}

	var out OutReponse
	// 1. Layer 1: 正则/关键词 (极速)
	layer1Result := a.Layer1_FastPatternMatch(req.Query)
	if layer1Result.Hit { // 命中  返回固定消息
		out.ClassiFication = DecisionAgentUnknown
		out.Data.Msg = layer1Result.Content
		out.UnKnown = layer1Result.Content
		RespJsonSuccess(c, req.SysTrackCode, out)
		return
	}

	// 2. Layer 2: 安全审计 (模型)
	safetyRes, err := a.Layer2_SafetyAudit(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "安全审计", fmt.Sprintf("[%s]-未成功完成安全审计", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-未成功完成安全审计", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}
	if safetyRes != "SAFE" {
		out.ClassiFication = DecisionAgentUnknown
		out.Data.Msg = safetyRes
		out.UnKnown = safetyRes
		RespJsonSuccess(c, req.SysTrackCode, out)
		return
	}

	// 获取历史会话
	msgHistory, err := a.GetHistoryDialogue(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "获取历史会话", fmt.Sprintf("[%s]-未成功获取历史会话", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorPsql, fmt.Sprintf("[%s]-未成功获取历史会话", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}

	// 数据模拟
	session := a.GetRedis(c, req)
	currentAgentKey := session.FlowContext.CurrentAgentKey

	// 从数据库获取当前 Agent 的描述 (Description)
	// 对应表: ai_agent_registry
	agentConfig, err := a.GetAgentConfigByKey(currentAgentKey)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "数据库", fmt.Sprintf("[%s]-未成功完成数据库查询", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-未成功完成数据库查询", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}

	// Layer 3: 上下文路由
	// 前提: 当前必须有活跃的 Agent 且非第一轮会话
	if session.FlowContext.CurrentAgentKey != "" && msgHistory != "" {
		action, err := a.Layer3_ContextRouter(req, agentConfig.AgentName.String, agentConfig.Description.String, msgHistory)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "意图转移识别", fmt.Sprintf("[%s]-未成功完成意图转移识别", a.App.Manifest.Code), err)
			RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-未成功完成意图转移识别", a.App.Manifest.Code), req.SysTrackCode, nil)
			return
		}

		// 3. 判定结果
		// 未发生意图转移，透传agent
		if action == "CONTINUE" {
			out.ClassiFication = currentAgentKey
			xlog.LogInfoF(req.SysTrackCode, "send_msg", "意图转移识别", fmt.Sprintf("[%s]-未发生意图转移:%v", a.App.Manifest.Code, currentAgentKey))
			RespJsonSuccess(c, req.SysTrackCode, out)
		}
	}
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "意图转移识别", fmt.Sprintf("[%s]-首轮对话或发生意图转移，上轮意图:%v", a.App.Manifest.Code, currentAgentKey))
	// 4. Layer 4: 总控调度 (RAG + 精准分发)
	targetAgent, err := a.Layer4_SupervisorDispatch(req, agentConfig.DomainID.String, msgHistory, session)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "意图识别", fmt.Sprintf("[%s]-未成功完成意图识别", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-未成功完成意图识别", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}
	out.ClassiFication = targetAgent
	RespJsonSuccess(c, req.SysTrackCode, out)

	return
}

// DemoPost post DemoTest路由
func (a *DecisionAgent) DemoPost(c *gin.Context) {

}

// DemoGet Get DemoTest路由
func (a *DecisionAgent) DemoGet(c *gin.Context) {

}
