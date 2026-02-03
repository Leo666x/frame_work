package triage

import (
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
)

func (a *TriageAgent) router2_HandleHospitalService(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	mySlots *TriageSlots,
	msgHistory string,
) {

	aiMsg := `
非常抱歉，关于医院具体的检查流程、体检业务及特殊门诊通道的相关数据，目前正在接入和更新中。我目前主要擅长症状导诊、医生查找及科室查找，您可以尝试问我这些方面的问题。
`

	// 更新记忆
	err := a.Updatememory(req, resp, session, aiMsg, msgHistory, mySlots)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "引导提示词大模型记忆管理", fmt.Sprintf("[%s]-未成功完成引导提示词大模型记忆管理", a.App.Manifest.Code), err)
	}
	_ = event.WriteAgentResponseMessage(resp, aiMsg)
	event.Done(resp)

	return
}
