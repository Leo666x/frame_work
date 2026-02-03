package triage

import (
	"context"
	"encoding/json"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	milvus_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strconv"
	"strings"
)

func (a *TriageAgent) router1_SymptomDiagnosis(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	mySlots *TriageSlots,
	msgHistory string,
) {

	// 1. 更新回合数
	mySlots.DiagnosisRound++

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
	).Replace(PROMPT_TEMPLATE_GP_DOCTOR)

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

	// 6. 判断是否确诊 / 结束
	if HasValidContent(resLlmCallGuide.DiseaseList) || HasValidContent(resLlmCallGuide.Msg) || mySlots.DiagnosisRound >= 5 {
		// === 确诊逻辑 ===

		// A. 解锁状态
		mySlots.Status = "idle"
		mySlots.DiagnosisRound = 0

		// B. 填入公共黑板 (Shared)
		confirmedDisease := strings.Join(resLlmCallGuide.DiseaseList, ",")
		session.GlobalState.Shared.Disease = confirmedDisease
		out := DeptResponse{}

		// 疾病匹配科室
		out, err := a.Illness2DeptKnowledge(req, resp, confirmedDisease)
		if err != nil {
			_ = event.WriteAgentResponseError(resp, ErrorParse, fmt.Sprintf("[%s]-未成功完成疾病推科室", a.App.Manifest.Code))
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

		_ = event.WriteAgentResponseMessage(resp, aiMsg)
		_ = event.WriteAgentResponseMessage(resp, msg)
		_ = event.WriteAgentResponseStruct(resp, out)
		event.Done(resp)

		return

	} else {
		// === 继续提问 ===
		// 直接返回医生的问题
		// 7 当模型 未总结 出疾病名称，引导患者进一步描述症状
		aiMsg := resLlmCallGuide.Question[0]
		out := DeptResponse{
			Msg:     aiMsg,
			EndFlag: "false",
			Type:    ReturnCardTypeGuide,
			Answers: resLlmCallGuide.Answers,
		}
		// 更新记忆
		err := a.Updatememory(req, resp, session, aiMsg, msgHistory, mySlots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "引导提示词大模型记忆管理", fmt.Sprintf("[%s]-未成功完成引导提示词大模型记忆管理", a.App.Manifest.Code), err)
		}
		xlog.LogInfoF(req.SysTrackCode, "send_msg", "引导患者进一步描述症状", fmt.Sprintf("[%s]-未总结出疾病名称，引导患者进一步描述症状: %v", a.App.Manifest.Code, resLlmCallGuide))
		_ = event.WriteAgentResponseMessage(resp, aiMsg)
		_ = event.WriteAgentResponseStruct(resp, out)
		event.Done(resp)
		return
	}
}

func (a *TriageAgent) Illness2DeptKnowledge(
	req *server.AgentRequest,
	r *server.AgentResponse,
	confirmedDisease string,
) (DeptResponse, error) {

	//  当模型总结出疾病名称
	var deptHistory DeptHistory
	deptHistory.Sort = "false"
	// etcd配置获取
	deptHistoryConf := a.resultConfig["register_agent_dept_history"].(string)
	if deptHistoryConf == "" {
		xlog.LogInfoF(r.SysTrackCode, "send_msg", "etcd配置获取", fmt.Sprintf("[%s]-疾病推科室是否按历史挂号量排序配置获取为空", a.App.Manifest.Code))
	} else {
		if err := json.Unmarshal([]byte(deptHistoryConf), &deptHistory); err != nil {
			xlog.LogErrorF(r.SysTrackCode, "send_msg", "etcd配置获取", fmt.Sprintf("[%s]-未成功解析疾病推科室是否按历史挂号量排序配置", a.App.Manifest.Code), err)
			return DeptResponse{}, fmt.Errorf("未成功解析疾病推科室是否按历史挂号量排序配置: %w", err)
		}
	}

	//  知识库检索科室
	// 开始检索
	// 查 召回
	className := a.resultConfig["power_ai_agent_register_dept_illness_KB_name"].(string)
	returnFields := []string{"hospital_id", "hospital_name", "dept_his_id", "dept_id", "dept_level", "master_code", "dept_name", "dept_introduction", "illness"}
	topK := a.resultConfig["register_agent_return_dept_topK"].(string)
	topKi, _ := strconv.Atoi(topK)

	res, err := a.ReadKnowledge(req, confirmedDisease, className, "illness", returnFields, topKi)
	if err != nil {
		xlog.LogErrorF(r.SysTrackCode, "send_msg", "引导描述症状-知识库科室推荐", fmt.Sprintf("[%s]-未成功推荐科室", a.App.Manifest.Code), err)
		return DeptResponse{}, fmt.Errorf("未成功推荐科室: %w", err)
	}
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "知识库推荐", fmt.Sprintf("[%s]-疾病科室推荐初始匹配: %v", a.App.Manifest.Code, res[0].Data))
	return DeptResponse{DeptList: res}, nil
	//var deptList []map[string]interface{}
	//var deptInfoStr strings.Builder // 提示词
	//// 结果 多个科室 去重
	//tempMap := map[interface{}]byte{}
	//for _, item := range res {
	//	distinctId := item["hospital_id"].(string) + item["dept_id"].(string)
	//	l := len(tempMap)
	//	tempMap[distinctId] = 0
	//	if len(tempMap) != l {
	//		hosNameStr, _ := getString(item["hospital_name"])
	//		hosIdStr, _ := getString(item["hospital_id"])
	//		deptNameStr, _ := getString(item["dept_name"])
	//		deptIdStr, _ := getString(item["dept_id"])
	//		deptHisIdStr, _ := getString(item["dept_his_id"])
	//		deptLevelStr, _ := getString(item["dept_level"])
	//		masterCodeStr, _ := getString(item["master_code"])
	//
	//		// go_url 拼接
	//		goDeptURL := strings.NewReplacer(
	//			"DEPTCODE", deptIdStr,
	//			"DEPTNAME", deptNameStr,
	//			"HOSTCODE", hosIdStr,
	//			"DEPTLEVEL", deptLevelStr,
	//			"MASTERCODE", masterCodeStr,
	//		).Replace(goDeptURL)
	//
	//		dept := map[string]interface{}{
	//			"hospital_id":    hosIdStr,
	//			"hospital_name":  hosNameStr,
	//			"dept_name":      deptNameStr,
	//			"dept_id":        deptIdStr,
	//			"dept_his_id":    deptHisIdStr,
	//			"dept_level":     deptLevelStr,
	//			"go_url":         goDeptURL,
	//			"parent_dept_id": masterCodeStr,
	//		}
	//		deptList = append(deptList, dept)
	//		deptInfoStr.WriteString(
	//			fmt.Sprintf("{\"dept_id\":\"%s\",\"dept_name\":\"%s\"}\n", deptIdStr, deptNameStr))
	//	}
	//
	//}
	// 匹配是否符合年龄 性别
	// 提示词工程

	//deptMatchOfSexAgePrompt := strings.NewReplacer(
	//	"CONTEXT1", sex_,
	//	"CONTEXT2", age_,
	//	"CONTEXT3", deptInfoStr.String(),
	//).Replace(deptMatchOfSexAgePromptConf.Value)
	//xlog.LogInfoF(req.SysTrackCode, "send_msg", "疾病推科室二次精准匹配大模型调用", fmt.Sprintf("[%s]-开始调用疾病推科室二次精准匹配大模型处理结果：%v", a.App.AgentAppInfo.Code, deptInfoStr.String()))
	//resLlmSearchDeptBySexAge, err := LlmCallSearchDeptBySexAge(deptMatchOfSexAgePrompt, func(request map[string]interface{}) (string, error) {
	//	return a.App.SyncCallSystemLLM(req.EnterpriseId, request)
	//})
	//if err != nil {
	//	xlog.LogErrorF(req.SysTrackCode, "send_msg", "疾病推科室二次精准匹配大模型调用", fmt.Sprintf("[%s]-未成功调用疾病推科室二次精准匹配大模型", a.App.AgentAppInfo.Code), err)
	//	_ = event.WriteAgentResponseError(r, ErrorCallLlm, fmt.Sprintf("[%s]-未成功调用疾病推科室二次精准匹配大模型", a.App.AgentAppInfo.Code))
	//	return
	//}
	//if resLlmSearchDeptBySexAge == nil {
	//	xlog.LogErrorF(req.SysTrackCode, "send_msg", "疾病推科室二次精准匹配大模型调用", fmt.Sprintf("[%s]-无符合指定年龄：%v，及性别的科室推荐%v: %s", a.App.AgentAppInfo.Code, age_, sex_, deptList), err)
	//	_ = event.WriteAgentResponseError(r, ErrorCallLlm, fmt.Sprintf("[%s]-未成功调用疾病推科室二次精准匹配大模型", a.App.AgentAppInfo.Code))
	//	return
	//}
	//xlog.LogInfoF(req.SysTrackCode, "send_msg", "疾病推科室二次精准匹配大模型调用", fmt.Sprintf("[%s]-成功调用疾病推科室二次精准匹配大模型: %v", a.App.AgentAppInfo.Code, resLlmSearchDeptBySexAge))
	//
	//deptList = findMatchesOfDeptId(resLlmSearchDeptBySexAge, deptList)
	//if deptList == nil {
	//	msg := fmt.Sprintf("根据您的要求，我们没有找到到相关科室。")
	//	_ = event.WriteAgentResponseMessageWithSpeed(r, msg, StreamSpeed)
	//	event.Done(r)
	//	return
	//}
	//
	//if deptHistory.Sort == "true" && len(deptList) > 1 && len(deptHistory.DiagnoseCount) > 0 {
	//	xlog.LogInfoF(req.SysTrackCode, "send_msg", "知识库推荐", fmt.Sprintf("[%s]-启用排序功能", a.App.AgentAppInfo.Code))
	//	counts := deptHistory.DiagnoseCount // map[string]int64，缺失即 0
	//	sort.SliceStable(deptList, func(i, j int) bool {
	//		idI, _ := getString(deptList[i]["dept_id"])
	//		idJ, _ := getString(deptList[j]["dept_id"])
	//		// 统一去空白，避免 key 不一致
	//		idI = strings.TrimSpace(idI)
	//		idJ = strings.TrimSpace(idJ)
	//		cI := counts[idI]
	//		cJ := counts[idJ]
	//		if cI != cJ {
	//			return cI > cJ // 挂号量大的排前
	//		}
	//		return false // 相等则保持原相对顺序（稳定排序）
	//	})
	//}
	//
	//// 结果
	//msg := "根据您的要求，匹配到的科室数据，如下:"
	//out := DeptResponse{
	//	Msg:      msg,
	//	Type:     ReturnCardTypeDept,
	//	EndFlag:  "true",
	//	DeptList: deptList,
	//}
	//xlog.LogInfoF(req.SysTrackCode, "send_msg", "知识库推荐", fmt.Sprintf("[%s]-疾病科室推荐结束: %v", a.App.AgentAppInfo.Code, deptList))
	//
	//// 预问诊总结功能
	//if preConsultationConf.Value == "true" {
	//	// etcd配置获取
	//	preConsultationPromptConf := a.App.GetAgentConfig("register_agent_pre_consultation_prompt", req.EnterpriseId)
	//	if preConsultationPromptConf == nil || preConsultationPromptConf.Value == "" {
	//		xlog.LogErrorF(r.SysTrackCode, "send_msg", "etcd配置获取", fmt.Sprintf("[%s]-挂号智能体预问诊总结提示词获取为空", a.App.AgentAppInfo.Code), nil)
	//		_ = event.WriteAgentResponseError(r, server.Unavailable.Code, fmt.Sprintf("[%s]-挂号智能体预问诊总结提示词获取为空", a.App.AgentAppInfo.Code))
	//		return
	//	}
	//	// 提示词工程 组装提示词
	//	PreConsultationPrompt := strings.NewReplacer(
	//		"SEX_REPLACE", sex_,
	//		"AGE_REPLACE", age_,
	//		"CONTEXT", preConsultationDiag.String(),
	//	).Replace(preConsultationPromptConf.Value)
	//	PreConsultation, err := LlmCallPreConsultation(PreConsultationPrompt, func(request map[string]interface{}) (string, error) {
	//		return a.App.SyncCallSystemLLM(req.EnterpriseId, request)
	//	})
	//	if err != nil {
	//		xlog.LogErrorF(req.SysTrackCode, "send_msg", "预问诊总结大模型调用", fmt.Sprintf("[%s]-未成功调用预问诊总结大模型", a.App.AgentAppInfo.Code), err)
	//		_ = event.WriteAgentResponseError(r, ErrorCallLlm, fmt.Sprintf("[%s]-未成功调用预问诊总结大模型", a.App.AgentAppInfo.Code))
	//		return
	//	}
	//	record, err := ParseMedicalRecord(PreConsultation)
	//	if err != nil {
	//		xlog.LogErrorF(req.SysTrackCode, "send_msg", "预问诊总结大模型调用", fmt.Sprintf("[%s]-未成功调用预问诊总结大模型", a.App.AgentAppInfo.Code), err)
	//		_ = event.WriteAgentResponseError(r, ErrorCallLlm, fmt.Sprintf("[%s]-未成功调用预问诊总结大模型", a.App.AgentAppInfo.Code))
	//		return
	//	}
	//	record_json, _ := json.Marshal(record)
	//	dept_res_json, _ := json.Marshal(deptList)
	//	_, err = a.App.DBExec(`INSERT INTO ai_business_register (conversation_id,ai_summary_message,ai_recommend_branch,ai_recommend_info,ai_recommend_create_time,status,ai_summary_json,message_id)VALUES($1, $2, $3, $4, $5,$6,$7,$8)`,
	//		req.ConversationId, PreConsultation, "3", string(dept_res_json), xdatetime.GetNowDateTimeNano(), "1", record_json, req.MessageId,
	//	)
	//	out.Type = "card_register_doctor_dept"
	//	out.PreConsult = record
	//	if err != nil {
	//		xlog.LogErrorF(req.SysTrackCode, "send_msg", "预问诊数据存储", fmt.Sprintf("[%s]-未成功存储预问诊数据", a.App.AgentAppInfo.Code), err)
	//		_ = event.WriteAgentResponseError(r, ErrorDataBase, fmt.Sprintf("[%s]-未成功存储预问诊数据", a.App.AgentAppInfo.Code))
	//		return
	//	}
	//}
	//_ = event.WriteAgentResponseMessageWithSpeed(r, ResLlmCallGuide.Msg[0], StreamSpeed)
	//_ = event.WriteAgentResponseMessageWithSpeed(r, msg, StreamSpeed)
	//_ = event.WriteAgentResponseStruct(r, out)
	//event.Done(r)

}
func (a *TriageAgent) ReadKnowledge(req *server.AgentRequest, query, className, matchField string, outputFields []string, topK int) ([]milvus_mw.SearchResult, error) {
	milvusCli, err := a.App.GetMilvusClient()
	if err != nil {
		return nil, err
	}
	// 在向量库中搜索描述最匹配的 n 个 Agent
	ctx := context.Background()

	queryVectors, err := a.App.EmbedTexts(req.EnterpriseId, []string{query}) // 查询向量  [][]float32{{...}}
	if err != nil {
		return nil, err
	}
	vectorFieldName := "emb_field" // 向量字段名 (如 "doc_embedding")
	filterExpr := ""
	searchResult, err := milvusCli.MilvusVectorSearch(ctx, className, vectorFieldName, queryVectors, topK, filterExpr, outputFields)
	if err != nil {
		return nil, fmt.Errorf("RAG search failed: %w", err)
	}

	// 重排序
	candidates, err := a.RerankDeal(searchResult, req.EnterpriseId, req.Query, matchField)
	if err != nil {
		return nil, err
	}
	return candidates, nil

}
