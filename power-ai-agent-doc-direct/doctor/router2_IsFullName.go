package doctor

import (
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
)

func (a *DoctorAgent) router2_IsFullName(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	slots *DoctorSlots,
	sess *powerai.SessionValue,
) {

	// 记录意图到 Session，方便下一轮 Router 知道我们在等名字
	// 实际上 Layer 3 看到 CurrentAgent 是 DoctorDirect，且用户回了名字，会透传进来

	// 此时不调用 RAG，直接反问
	reply := "请问您要找的医生具体叫什么名字？因为同姓的医生可能较多，提供全名能帮我更准确地为您查找。"

	// 更新记忆
	err := a.Updatememory(req, sess, reply, slots)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
	}

	_ = event.WriteAgentResponseMessage(resp, reply)
	event.Done(resp)
	return

}
