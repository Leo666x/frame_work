package triage

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
func (a *TriageAgent) SendMsg(c *gin.Context) {
	req, resp, event, ok := powerai.DoValidateAgentRequest(c, a.App.Manifest.Code)
	if !ok {
		return
	}

	//初始化配置
	a.InitReportConfig(resp, event, map[string]interface{}{
		"register_agent_dept_history":                  "疾病推科室是否按历史挂号量排序返回",
		"power_ai_agent_register_dept_illness_KB_name": "挂号智能体疾病推科室知识库名称",
		"register_agent_return_dept_topK":              "挂号智能体返回推荐科室数量topK",
		"register_agent_return_doc_topK":               "挂号智能体返回推荐医生数量topK",
	})

	// 加载redis   私有状态
	// 数据模拟
	session := a.GetRedis(c, req)
	// 1. 加载私有状态
	var mySlots TriageSlots
	if rawSlot, ok := session.GlobalState.AgentSlots["power-ai-agent-triage"]; ok {
		bytes, _ := json.Marshal(rawSlot)
		json.Unmarshal(bytes, &mySlots)
	} else {
		// 初始化新的
		mySlots = TriageSlots{
			SymptomAttributes: make(map[string]string),
		}
	}

	// ==============================================================
	// Phase 1: 内部守卫 (Internal Guard) & 意图识别
	// ==============================================================

	// 调用轻量级模型进行意图分类 (这一步极快，建议每次都做)
	intentRes, err := a.classifyInternalIntent(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "轻量意图识别", fmt.Sprintf("[%s]-未成功完成轻量意图识别", a.App.Manifest.Code), err)
		_ = event.WriteAgentResponseError(resp, ErrorCallLlm, fmt.Sprintf("[%s]-未成功完成轻量意图识别", a.App.Manifest.Code))
		return
	}

	// 获取历史会话
	msgHistory, _, err := a.GetHistoryDialogue(req)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "获取历史会话", fmt.Sprintf("[%s]-未成功获取历史会话", a.App.Manifest.Code), err)
		RespJsonError(c, ErrorPsql, fmt.Sprintf("[%s]-未成功获取历史会话", a.App.Manifest.Code), req.SysTrackCode, nil)
		return
	}

	// 检查：是否处于多轮问诊锁定中？
	if mySlots.Status == "in_diagnosis_loop" {
		// 守卫逻辑：如果是强烈的意图转移 (如找专家、查流程)，则打破循环
		if intentRes.Intent == "EXPERT" || intentRes.Intent == "SERVICE" || intentRes.Intent == "INTRO" {
			xlog.LogInfoF(req.SysTrackCode, "send_msg", "多轮问诊", fmt.Sprintf(">> [Triage] 内部意图跳跃: DIAGNOSIS -> %s, 解锁状态\n", intentRes.Intent))
			mySlots.Status = "idle" // 解锁
			mySlots.DiagnosisRound = 0
		} else {
			// 否则，默认为对病情的描述或回答，继续问诊
			// 即使 classify 结果不准，只要不是强意图，优先保活问诊流程
			a.router1_SymptomDiagnosis(event, req, resp, session, &mySlots, msgHistory)
			return
		}
	}

	// ==============================================================
	// Phase 2: 意图分发
	// ==============================================================

	switch intentRes.Intent {
	case "DIAGNOSIS":
		if intentRes.EntityType == "DISEASE" {
			// 场景 2: 已知疾病 -> 查规则 (One-shot)
			// 解锁状态
			mySlots.Status = "idle"
			mySlots.DiagnosisRound = 0

			a.router1_HandleKnownDisease(event, req, resp, session, &mySlots, intentRes, msgHistory)
			return
		} else {
			// 症状 -> 进入多轮问诊 (Loop)
			// 初始化状态
			mySlots.Status = "in_diagnosis_loop"
			mySlots.DiagnosisRound = 0
			a.router1_SymptomDiagnosis(event, req, resp, session, &mySlots, msgHistory)
			return
		}

	case "SERVICE":
		// 我需要给家里的老人开长期慢病药，有没有简易门诊或者方便门诊可以挂号？

		// 服务流程
		a.router2_HandleHospitalService(event, req, resp, session, &mySlots, msgHistory)

	case "EXPERT":
		// 查专家实力表
		//你们医院治帕金森最厉害的专家是谁
		a.router3_HandleHospitalExpert(event, req, resp, session, &mySlots, intentRes, msgHistory)

	//case "INTRO":
	//	// 查科室介绍表
	//	//result, err = a.router4_HandleDepartmentIntro(ctx, "search_dept_intro", intentRes.KeyEntity, input, sess)

	default:
		// 兜底
		a.router5_HandleUnknownIntent(event, req, resp, session, &mySlots, msgHistory)
	}

}

func (a *TriageAgent) classifyInternalIntent(req *server.AgentRequest) (InternalIntent, error) {
	// 使用之前定义的 Prompt
	prompt := strings.Replace(PROMPT_INTERNAL_ROUTER, "UserQuery", req.Query, 1)

	respLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return InternalIntent{}, err
	}
	resStr := LlmRespDeal(respLlm)

	var res InternalIntent
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		return InternalIntent{}, err
	}

	// 默认兜底
	if res.Intent == "" {
		res.Intent = "DIAGNOSIS"
		res.EntityType = "SYMPTOM"
	}
	return res, nil
}
