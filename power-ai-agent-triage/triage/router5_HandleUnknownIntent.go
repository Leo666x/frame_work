package triage

import (
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
)

func (a *TriageAgent) router5_HandleUnknownIntent(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	mySlots *TriageSlots,
	msgHistory string,
) {

	aiMsg := `
抱歉，我没有完全理解您的医疗需求。您可以尝试这样问：\n1. 头疼挂哪科？(症状咨询)\n2. 帕金森看哪个专家？(找专家)\n3. 无痛胃镜怎么约？(查流程)
`
	// 更新记忆
	err := a.Updatememory(req, resp, session, aiMsg, msgHistory, mySlots)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
	}
	_ = event.WriteAgentResponseMessage(resp, aiMsg)
	event.Done(resp)

	return
}
