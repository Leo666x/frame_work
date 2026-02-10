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

func (a *DecisionAgent) SendMsg(c *gin.Context) {
	req, _, _, ok := powerai.DoValidateAgentRequest(c, a.App.Manifest.Code)
	if !ok {
		return
	}

	var out OutReponse
	layer1Result := a.Layer1_FastPatternMatch(req.Query)
	if layer1Result.Hit {
		out.ClassiFication = DecisionAgentUnknown
		out.Data.Msg = layer1Result.Content
		out.UnKnown = layer1Result.Content
		RespJsonSuccess(c, req.SysTrackCode, out)
		return
	}

	safetyRes, err := a.Layer2_SafetyAudit(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "safety audit", fmt.Sprintf("[%s]-safety audit failed", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-safety audit failed", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}
	if safetyRes != "SAFE" {
		out.ClassiFication = DecisionAgentUnknown
		out.Data.Msg = safetyRes
		out.UnKnown = safetyRes
		RespJsonSuccess(c, req.SysTrackCode, out)
		return
	}

	msgHistory, err := a.GetHistoryDialogue(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "load history", fmt.Sprintf("[%s]-load history failed", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorPsql, fmt.Sprintf("[%s]-load history failed", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}

	session := a.GetRedis(c, req)
	if session == nil {
		session = &powerai.SessionValue{}
	}
	if session.FlowContext == nil {
		session.FlowContext = &powerai.FlowContext{}
	}
	if session.GlobalState == nil {
		session.GlobalState = &powerai.GlobalState{}
	}

	currentAgentKey := session.FlowContext.CurrentAgentKey
	previousDomain := ""
	previousAgentName := ""
	previousAgentDesc := ""
	if currentAgentKey != "" {
		agentConfig, configErr := a.GetAgentConfigByKey(currentAgentKey)
		if configErr != nil {
			xlog.LogInfoF(req.SysTrackCode, "send_msg", "load previous agent config", fmt.Sprintf("[%s]-load previous agent config failed, fallback to global routing: %v", a.App.Manifest.Code, configErr))
		} else {
			previousDomain = agentConfig.DomainID.String
			previousAgentName = agentConfig.AgentName.String
			previousAgentDesc = agentConfig.Description.String
		}
	}

	if currentAgentKey != "" && previousAgentName != "" && msgHistory != "" {
		action, routeErr := a.Layer3_ContextRouter(req, previousAgentName, previousAgentDesc, msgHistory)
		if routeErr != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "context routing", fmt.Sprintf("[%s]-context routing failed", a.App.Manifest.Code), routeErr)
			RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-context routing failed", a.App.Manifest.Code), req.SysTrackCode, nil)
			return
		}
		if action == "CONTINUE" {
			out.ClassiFication = currentAgentKey
			xlog.LogInfoF(req.SysTrackCode, "send_msg", "context routing", fmt.Sprintf("[%s]-continue previous agent: %v", a.App.Manifest.Code, currentAgentKey))
			RespJsonSuccess(c, req.SysTrackCode, out)
			return
		}
	}

	xlog.LogInfoF(req.SysTrackCode, "send_msg", "context routing", fmt.Sprintf("[%s]-first turn or intent transfer, previous agent: %v", a.App.Manifest.Code, currentAgentKey))
	targetAgent, err := a.Layer4_SupervisorDispatch(req, previousDomain, msgHistory, session)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "intent routing", fmt.Sprintf("[%s]-intent routing failed", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorCallLlm, fmt.Sprintf("[%s]-intent routing failed", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}

	out.ClassiFication = targetAgent
	RespJsonSuccess(c, req.SysTrackCode, out)
}

func (a *DecisionAgent) DemoPost(c *gin.Context) {
}

func (a *DecisionAgent) DemoGet(c *gin.Context) {
}
