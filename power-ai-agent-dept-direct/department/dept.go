package department

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
func (a *DeptAgent) SendMsg(c *gin.Context) {
	req, resp, event, ok := powerai.DoValidateAgentRequest(c, a.App.Manifest.Code)
	if !ok {
		return
	}

	//初始化配置
	a.InitReportConfig(resp, event, map[string]interface{}{})

	// 加载redis   私有状态
	// 数据模拟
	session := a.GetRedis(c, req)

	var mySlots DeptSlots
	err := LoadAgentSlots(session.GlobalState.AgentSlots, "dept_direct_agent", &mySlots)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功加载记忆", a.App.Manifest.Code), err)
		//mySlots = DoctorSlots{} // 重置
		_ = event.WriteAgentResponseError(resp, ErrorMemory, fmt.Sprintf("[%s]-未成功加载记忆", a.App.Manifest.Code))
		return // 业务评估 直接返回错误 或 next
	}

	extractResp, err := a.extractIntentAndName(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "关键词提取", fmt.Sprintf("[%s]-未成功提取科室姓名", a.App.Manifest.Code), err)
		_ = event.WriteAgentResponseError(resp, ErrorCallLlm, fmt.Sprintf("[%s]-未成功提取科室姓名", a.App.Manifest.Code))
		return
	}

	// 异常处理：虽然 Router 保证了意图，但万一用户只发了"挂号"两个字
	// =================================================
	if extractResp.DeptName == "" {
		reply := "请问您具体想挂哪个科室？如果您不确定挂哪个科，可以描述您的症状，我会帮您推荐。"

		// 记忆更新
		err = a.Updatememory(req, session, reply, &mySlots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
		}
		_ = event.WriteAgentResponseMessage(resp, reply)
		event.Done(resp)
		return
	}

	// 逻辑处理：更新状态
	mySlots.TargetName = extractResp.DeptName
	mySlots.Intent = extractResp.Intent
	session.GlobalState.Shared.TargetDept = extractResp.DeptName // 同步到公共区

	a.router1_SearchDeptKB(event, req, resp, session, &mySlots, extractResp)

}
func (a *DeptAgent) extractIntentAndName(req *server.AgentRequest) (DeptExtractionResult, error) {
	// 使用之前定义的 Prompt
	prompt := strings.Replace(PROMPT_INTERNAL_ROUTER, "UserQuery", req.Query, 1)

	respLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return DeptExtractionResult{}, err
	}
	resStr := LlmRespDeal(respLlm)

	var res DeptExtractionResult
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		return DeptExtractionResult{}, err
	}
	return res, nil
}
