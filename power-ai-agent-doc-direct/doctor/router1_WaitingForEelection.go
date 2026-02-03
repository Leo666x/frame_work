package doctor

import (
	"encoding/json"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strings"
)

func (a *DoctorAgent) router1_WaitingForEelection(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	slots *DoctorSlots,
) {
	// 调用 LLM 判断用户选了谁
	// Prompt: "用户输入[input]，候选列表[slots.CandidateDocs]，用户选了哪一个？"
	selectedIndex, err := a.analyzeUserSelection(req, slots.CandidateDocs)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "大模型调用-多医生选择", fmt.Sprintf("[%s]-未成功完成多医生选择", a.App.Manifest.Code), err)
		_ = event.WriteAgentResponseError(resp, ErrorCallLlm, fmt.Sprintf("[%s]-未成功完成多医生选择", a.App.Manifest.Code))
		return
	}

	if selectedIndex != -1 {
		// 用户选选中了第 i 个
		selectedDoc := slots.CandidateDocs[selectedIndex]
		docName := selectedDoc["doc_name"].(string)
		deptname := selectedDoc["dept_name"].(string)
		// 1. 更新公共记忆 (Global Shared) -> 这样 Router 知道任务完成了
		session.GlobalState.Shared.TargetDoctor = docName
		session.GlobalState.Shared.TargetDept = deptname

		// 2. 清理私有状态
		slots.Status = "idle"
		slots.CandidateDocs = nil

		reply := fmt.Sprintf("已为您锁定 %s (%s)，请点击卡片挂号。", docName, deptname)

		// 查号源

		// 更新记忆
		err = a.Updatememory(req, session, reply, slots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
		}

		_ = event.WriteAgentResponseMessage(resp, reply)
		event.Done(resp)
		return
	} else {
		// 用户好像没选，或者选了“都不是”
		// 继续保持 waiting 状态，或者重置
		reply := "请明确告诉我您想找哪一位张伟医生？例如：'眼科的那个'。"

		// 更新记忆
		err = a.Updatememory(req, session, reply, slots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
		}

		_ = event.WriteAgentResponseMessage(resp, reply)
		event.Done(resp)
		return
	}

	// 数据模拟
	//candidates := []map[string]interface{}{
	//	// 第一条数据：产科 - 丁文成
	//	{
	//		"hospital_id":   "同济医院",
	//		"hospital_name": "tjyyggyq",
	//		"dept_name":     "妊娠期并发症门诊",
	//		"dept_level":    "4",
	//		"dept_id":       "0203020301",
	//		"doc_name":      "丁文成",
	//		"doc_id":        "4443281430369",
	//		"doc_typeName":  "副主任医师", // 职称：关键排序依据
	//		"doc_sex":       "男",
	//		// 擅长领域：LLM 匹配核心
	//		"doc_specialty": "1、各种妊娠合并症如妊娠期高血压，妊娠期糖尿病，妊娠合并肝炎、风湿免疫性疾病、子宫肌瘤、卵巢囊肿等的诊治；2、妊娠期阴道出血的诊治；3、孕期安全用药咨询、营养与体重管理；4、倡导自然分娩、鼓励分娩镇痛，擅长各种剖宫产手术。",
	//		// 简介：学术地位来源
	//		"doc_introduction": "医学博士，副主任医师，副教授，同济医院产科副主任。学术兼职包括中华医学会围产医学分会青年学组成员...主要研究方向：子痫前期发病机制研究...发表SCI论文20余篇...申请国家发明专利7项...",
	//	},
	//
	//	// 第二条数据：心内科 - 丁虎
	//	{
	//		"hospital_id":   "同济医院",
	//		"hospital_name": "tjyyhkyq",
	//		"dept_name":     "心血管内科",
	//		"dept_level":    "2",
	//		"dept_id":       "010102",
	//		"doc_name":      "丁虎",
	//		"doc_id":        "5535501052360",
	//		"doc_typeName":  "主任医师", // 职称：关键排序依据
	//		"doc_sex":       "男",
	//		// 擅长领域
	//		"doc_specialty": "擅长复杂冠心病微创介入治疗，心血管急危重症多学科救治。对家族性高脂血症，冠心病、心力衰竭、心肌病、心肌炎、心律失常、心包疾病、心脏瓣膜病、高血压、下肢静脉血栓、肺栓塞、肺动脉高压和晕厥的诊治有丰富的临床经验。",
	//		// 简介
	//		"doc_introduction": "教授，主任医师，冠脉介入导师。主要从事复杂冠脉介入和急危重症机械支持治疗。近年承担冠心病多项国家级课题...是中华老年心脑血管病杂志编委...",
	//	},
	//}

}

func (a *DoctorAgent) analyzeUserSelection(
	req *server.AgentRequest,
	candidates []map[string]interface{},
) (int, error) {
	// 1. 拼装候选列表文本
	var listBuilder strings.Builder
	for i, doc := range candidates {
		docName := doc["doc_name"]
		DeptName := doc["dept_name"]
		docTypeName := doc["doc_typeName"]
		docSpecialty := doc["doc_specialty"]
		// 格式: [0] 姓名: 张伟 | 科室: 眼科 | 职称: 主任医师
		line := fmt.Sprintf("[%d] 姓名: %s | 科室: %s | 职称: %s | 擅长: %s\n",
			i, docName, DeptName, docTypeName, docSpecialty)
		listBuilder.WriteString(line)
	}
	prompt := strings.NewReplacer(
		"CandidateList", listBuilder.String(),
		"UserQuery", req.Query,
	).Replace(WaitingForEelectionPrompt)

	// 4. 调用大模型
	resLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return -1, fmt.Errorf("未成功调用大模型")
	}
	resContentRep := LlmRespDeal(resLlm)
	var res SelectionResult
	if err := json.Unmarshal([]byte(resContentRep), &res); err != nil {
		return -1, fmt.Errorf("Selection JSON parse error: %w", err)
	}

	if res.SelectedIndex < 0 || res.SelectedIndex >= len(candidates) {
		return -1, fmt.Errorf("未成功调用大模型")
	}

	return res.SelectedIndex, nil

}
