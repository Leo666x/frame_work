package doctor

import (
	"encoding/json"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	milvus_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strconv"
	"strings"
)

func (a *DoctorAgent) router3_SearchDoctorKB(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	sess *powerai.SessionValue,
	slots *DoctorSlots,
	doctorName string,
) {
	out, err := a.DocName2docKnowledge(req, resp, doctorName)
	if err != nil {
		_ = event.WriteAgentResponseError(resp, ErrorParse, fmt.Sprintf("[%s]-未成功完成医生推荐：%s", a.App.Manifest.Code, err))
		return
	}
	candidates := make([]map[string]interface{}, 0)
	if originalList, ok := out.DocList.([]milvus_mw.SearchResult); ok {
		for _, item := range originalList {
			candidate := make(map[string]interface{})
			for k, v := range item.Data {
				candidate[k] = v
			}
			candidates = append(candidates, candidate)
		}
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

	// 精准匹配
	doctors := StrictMatchDoctor(doctorName, candidates)
	if len(doctors) == 0 {
		// A. 查无此人
		reply := fmt.Sprintf("抱歉，我们在系统中没有找到叫“%s”的医生，请确认名字是否正确。", doctorName)

		out.Msg = reply
		out.EndFlag = "false"

		// 更新记忆
		err := a.Updatememory(req, sess, reply, slots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
		}

		_ = event.WriteAgentResponseMessage(resp, reply)
		event.Done(resp)
		return

	} else if len(doctors) == 1 {
		out.Type = ReturnCardTypeDoc
		out.EndFlag = "true"
		out.DocList = doctors

		jsonBytes, _ := json.Marshal(out.DocList)
		docListStr := string(jsonBytes)

		// 提示词工程
		prompt := strings.NewReplacer(
			"UserQuery", req.Query,
			"DoctorListData", docListStr,
		).Replace(PROMPT_DOCTOR_INTRO)

		requestLlm := buildPromptRequest(prompt)
		var reply string
		// 大模型结合回答
		a.App.SyncStreamCallSystemLLM(req.EnterpriseId, requestLlm, func(bytes []byte, err error) bool {
			if err != nil {
				_ = event.WriteAgentResponseError(resp, ErrorCallLlm, "调用大模型错误")
				xlog.LogErrorF(resp.SysTrackCode, "send_msg", "调用大模型", fmt.Sprintf("调用错误,err: %v", err), nil)
				return false
			}
			if bytes == nil || len(bytes) <= 0 {
				return true
			}
			msg := string(bytes)
			if strings.Contains(strings.ToUpper(msg), "[DONE]") {
				// 更新记忆
				err := a.Updatememory(req, sess, reply, slots)
				if err != nil {
					xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
				}
				_ = event.WriteAgentResponseStruct(resp, out)
				event.Done(resp)
				xlog.LogInfoF(req.SysTrackCode, "send_msg", "调用大模型", "返回结束[Done]")
				return true
			}
			content := xjson.Get(msg, "choices.0.delta.content").String()
			content = strings.Replace(content, "~", "-", -1)
			reply += content
			_ = event.WriteAgentResponseMessage(resp, content)
			return true
		})
	} else if len(doctors) > 1 {
		// === 命中多条：进入消歧模式 ===

		// 1. 填充私有记忆
		slots.TargetName = doctorName
		slots.Status = "waiting_for_selection"
		slots.CandidateDocs = doctors // // 存下来，下一轮用

		// 2. 生成回复
		reply := fmt.Sprintf("找到了 %d 位叫 %s 的医生，请问您找的是哪位？\n", len(doctors), doctorName)

		for _, d := range slots.CandidateDocs {
			deptName := d["dept_name"]
			docTypeName := d["doc_typeName"]
			reply += fmt.Sprintf("- %s (%s)\n", deptName, docTypeName)
		}

		// 更新记忆
		err := a.Updatememory(req, sess, reply, slots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
		}

		_ = event.WriteAgentResponseMessage(resp, reply)
		event.Done(resp)
		return

	}
}

func (a *DoctorAgent) DocName2docKnowledge(
	req *server.AgentRequest,
	r *server.AgentResponse,
	confirmedDisease string,
) (DocResponse, error) {

	//  知识库检索科室
	// 开始检索
	// 查 召回
	className := "Doc_info_"
	returnFields := []string{"hospital_id", "hospital_name", "dept_his_id", "dept_id", "dept_level", "master_code", "dept_name",
		"doc_his_id", "doc_id", "doc_typeName", "doc_sex", "doc_name", "doc_introduction", "doc_specialty"}
	topK := "20"
	topKi, _ := strconv.Atoi(topK)

	res, err := a.ReadKnowledge(req, confirmedDisease, className, "doc_name", returnFields, topKi)
	if err != nil {
		xlog.LogErrorF(r.SysTrackCode, "send_msg", "医生推荐", fmt.Sprintf("[%s]-未成功推荐医生", a.App.Manifest.Code), err)
		return DocResponse{}, fmt.Errorf("未成功推荐医生: %w", err)
	}
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "医生推荐", fmt.Sprintf("[%s]-医生推荐初始匹配: %v", a.App.Manifest.Code, res[0].Data))
	return DocResponse{DocList: res}, nil
}

// getString 安全地从 map[string]interface{} 获取字符串
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
