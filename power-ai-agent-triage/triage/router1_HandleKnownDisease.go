package triage

import (
	"encoding/json"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strings"
)

func (a *TriageAgent) router1_HandleKnownDisease(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	mySlots *TriageSlots,
	intentRes InternalIntent,
	msgHistory string,
) {

	// 2. 准备 Prompt 变量
	userSex := req.Inputs["sex"].(string)
	userAge := req.Inputs["age"].(string)
	chronicDiseasesStr := formatMedicalHistory(session.UserSnapshot.ChronicDiseases)
	surgeryHistoryStr := formatMedicalHistory(session.UserSnapshot.SurgeryHistory)
	allergiesStr := formatMedicalHistory(session.UserSnapshot.Allergies)

	// 3. 渲染全科医生 Prompt (略去长文本，使用你提供的模板)
	guidePrompt := strings.NewReplacer(
		"SEX_REPLACE", userSex,
		"AGE_REPLACE", userAge,
		"USER_QUERY", req.Query,
		"DIALOGUE", msgHistory,
		"CHRONIC_DISEASES", chronicDiseasesStr,
		"SURGERY_HISTORY", surgeryHistoryStr,
		"ALLERGIES", allergiesStr,
	).Replace(illnessPrompt)

	// 4. 调用大模型
	resLlm, err := a.LlmCall(req, guidePrompt)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "引导提示词大模型调用", fmt.Sprintf("[%s]-未成功调用引导提示词大模型", a.App.Manifest.Code), err)
		_ = event.WriteAgentResponseError(resp, ErrorCallLlm, fmt.Sprintf("[%s]-未成功调用引导提示词大模型", a.App.Manifest.Code))
		return
	}
	resContentRep := LlmRespDealGuide(resLlm)
	// 5. 解析 JSON 响应
	var resLlmCallGuide ResLlmCallGuide
	if err := json.Unmarshal([]byte(resContentRep), &resLlmCallGuide); err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "引导提示词大模型调用", fmt.Sprintf("未成功解析大模型返回字符串: %s", resContentRep), err)
		_ = event.WriteAgentResponseError(resp, ErrorCallLlm, fmt.Sprintf("[%s]-未成功解析大模型返回字符串", a.App.Manifest.Code))
	}
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "引导提示词大模型调用", fmt.Sprintf("[%s]-引导提示词: %s", a.App.Manifest.Code, guidePrompt))

	// 填入公共黑板 (Shared)
	confirmedDisease := strings.Join(resLlmCallGuide.DiseaseList, ",")
	session.GlobalState.Shared.Disease = confirmedDisease

	out := DeptResponse{}

	// 疾病匹配科室
	out, err = a.Illness2DeptKnowledge(req, resp, confirmedDisease)
	if err != nil {
		_ = event.WriteAgentResponseError(resp, ErrorParse, fmt.Sprintf("[%s]-未成功完成科室推荐：%s", a.App.Manifest.Code, err))
		return
	}

	// 模拟数据
	//out.DeptList = []map[string]interface{}{
	//	{
	//		"hospital_id":    "H001",
	//		"hospital_name":  "市第一人民医院",
	//		"dept_name":      "神经内科",
	//		"dept_id":        "D101",
	//		"dept_his_id":    "HIS_N_01",
	//		"dept_level":     "2",
	//		"go_url":         "https://yiyla.com/dept?code=D101&name=神经内科&hid=H001&lvl=2&p=D100",
	//		"parent_dept_id": "D100",
	//	},
	//	{
	//		"hospital_id":    "H001",
	//		"hospital_name":  "市第一人民医院",
	//		"dept_name":      "心血管内科",
	//		"dept_id":        "D102",
	//		"dept_his_id":    "HIS_C_02",
	//		"dept_level":     "2",
	//		"go_url":         "https://yiyla.com/dept?code=D102&name=心血管内科&hid=H001&lvl=2&p=D100",
	//		"parent_dept_id": "D100",
	//	},
	//}

	msg := "根据您的要求，匹配到的科室数据，如下:"
	out.Msg = msg
	out.Type = ReturnCardTypeDept
	out.EndFlag = "true"
	aiMsg := resLlmCallGuide.Msg[0]

	// 更新记忆
	err = a.Updatememory(req, resp, session, aiMsg, msgHistory, mySlots)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "引导提示词大模型记忆管理", fmt.Sprintf("[%s]-未成功完成引导提示词大模型记忆管理", a.App.Manifest.Code), err)
	}

	_ = event.WriteAgentResponseMessage(resp, resLlmCallGuide.Msg[0])
	_ = event.WriteAgentResponseMessage(resp, msg)
	_ = event.WriteAgentResponseStruct(resp, out)
	event.Done(resp)

	return
}
