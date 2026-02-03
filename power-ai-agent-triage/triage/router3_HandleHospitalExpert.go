package triage

import (
	"encoding/json"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"strconv"
	"strings"
)

func (a *TriageAgent) router3_HandleHospitalExpert(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	mySlots *TriageSlots,
	intentRes InternalIntent,
	msgHistory string,
) {

	//你们医院治帕金森最厉害的专家是谁

	// 疾病匹配医生
	out, err := a.Illness2DocKnowledge(req, resp, intentRes.KeyEntity)
	if err != nil {
		_ = event.WriteAgentResponseError(resp, ErrorRAG, fmt.Sprintf("[%s]-未成功完成医生推荐：%s", a.App.Manifest.Code, err))
		return
	}

	// 模拟数据
	//out.DeptList = []map[string]interface{}{
	//	// 第一条数据：神经内科 - 张三医生
	//	{
	//		"hospital_id":   "HOS001",
	//		"hospital_name": "同济医院",
	//		"dept_name":     "神经内科",
	//		"dept_id":       "DEPT_NEURO_01",
	//		"dept_his_id":   "HIS_1001",
	//		"doc_name":      "刘启功",
	//		"doc_id":        "DOC_ZS_888",
	//		"doc_his_id":    "HIS_DOC_001",
	//		"doc_sex":       "男",
	//		"doc_typeName":  "主任医师",
	//		// 假设原始模板被替换后的结果
	//		"doc_specialty":    "高血压、冠心病、心力衰竭等心血管疾病和老年代谢性疾病的诊治以及老年多器官功能障碍的救治。",
	//		"doc_introduction": "医学博士、二级教授、主任医师、博士生导师。武汉老年医学会副会长兼秘书长，湖北省老年医师学会副主任委员，中华医学会老年病分会心血管学组副组长。在高血压、冠心病、心力衰竭等心血管疾病和老年代谢性疾病的诊治以及老年多器官功能障碍的救治方面具有丰富的临床经验。研究方向：衰老和抗衰老机制研究；老年心血管疾病和代谢性疾病的预防和治疗；老年急危重症的临床诊治。主持多项国家自然科学基金和省部级课题，国家重点研发计划子项目一项，发表SCI论文50多篇。",
	//	},
	//
	//	// 第二条数据：消化内科 - 李四医生
	//	{
	//		"hospital_id":      "HOS001",
	//		"hospital_name":    "同济医院",
	//		"dept_name":        "消化内科",
	//		"dept_id":          "DEPT_GASTRO_02",
	//		"dept_his_id":      "HIS_1002",
	//		"doc_name":         "凃玲",
	//		"doc_id":           "DOC_LS_999",
	//		"doc_his_id":       "HIS_DOC_002",
	//		"doc_sex":          "女",
	//		"doc_typeName":     "副主任医师",
	//		"doc_introduction": "擅长各种快速心律失常包括室上速、房扑、房颤、室早、室速的导管消融和心脏起搏器包括ICD、CRT/CRT-D的植入及心血管疾病的分子生物学研究。   1991年毕业于华中科技大学同济医学院医疗系德语班并分配到同济医院内科任住院医师，1994年到1999年在华中科技大学同济医学院硕博连读并于1999年获同济医院心内科专业博士学位。1999年起在同济医院心内科从事心内科专业，并从主治医师逐步提升到副教授、副主任医师和主任医师，其中2011年9月到2012年8月在同济咸宁医院任心肾内科主任并取得突出成绩。2006年至2007年先后在德国杜伊斯堡-埃森大学和图宾根大学附属医院心内科作访问学者，研究临床心脏电生理和复杂心律失常的导管消融。2001年和2011年先后被聘任为华中科技大学硕士和博士研究生导师，共招收硕士研究生30余人，已毕业近30人，转博1人。先后主持和参与各级科研课题10余项，参编心血管专著近20部，发表医学论文80余篇，其中SCI论文10余篇。论文“腺病毒载体介导hVEGF165基因预防血管成形术后再狭窄的实验研究”获2000年度湖北省优秀博士学位论文和2004年武汉市科技进步奖3等奖。现担任内科急危重症杂志编委（2012），湖北省咸宁市心血管病学会名誉主任委员（2012），湖北省心脏电生理和起搏学会常委（2014），武汉市中西医结合学会心血管分会常委（2014），武汉市心血管病学会心律失常学组常委（2012）。擅长各种快速心律失常包括室上速、房扑、房颤、室早、室速的导管消融和心脏起搏器包括ICD、CRT/CRT-D的植入及心血管疾病的分子生物学研究。",
	//		// 假设原始模板被替换后的结果
	//		"doc_specialty": "擅长室上速、室早、室速、房扑、房颤的射频导管消融和心脏起搏器、ICD、CRT-P/CRT-D的植入，对高血压、冠心病、心衰、心肌病、心肌炎、心律失常、肺栓塞、主动脉夹层等特别是心律失常的诊断和治疗有着丰富的临床经验",
	//	},
	//}

	jsonBytes, _ := json.Marshal(out.DocList)
	docListStr := string(jsonBytes)

	// 提示词工程
	prompt := strings.NewReplacer(
		"UserQuery", req.Query,
		"DoctorListData", docListStr,
	).Replace(ExpertPrompt)

	requestLlm := buildPromptRequest(prompt)
	var aiMsg string
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
			msg := "根据您的要求，匹配到的医生数据，如下:"
			out.Msg = msg
			out.Type = ReturnCardTypeDoc
			out.EndFlag = "true"
			_ = event.WriteAgentResponseMessage(resp, msg)
			_ = event.WriteAgentResponseStruct(resp, out)
			event.Done(resp)
			xlog.LogInfoF(req.SysTrackCode, "send_msg", "调用大模型", "返回结束[Done]")

			// 更新记忆
			err := a.Updatememory(req, resp, session, aiMsg, msgHistory, mySlots)
			if err != nil {
				xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
			}

			return true
		}
		aiMsg += msg
		content := xjson.Get(msg, "choices.0.delta.content").String()
		content = strings.Replace(content, "~", "-", -1)
		_ = event.WriteAgentResponseMessage(resp, content)
		return true
	})

	return
}

func (a *TriageAgent) Illness2DocKnowledge(
	req *server.AgentRequest,
	r *server.AgentResponse,
	confirmedDisease string,
) (DocResponse, error) {

	//  知识库检索医生
	// 开始检索
	// 查 召回
	className := "Illness_match_recommend_doc_"
	returnFields := []string{"hospital_id", "hospital_name", "dept_his_id", "dept_id", "dept_level", "master_code", "dept_name",
		"doc_his_id", "doc_id", "doc_typeName", "doc_sex", "doc_name", "doc_introduction", "doc_specialty"}
	topK := "20"
	topKi, _ := strconv.Atoi(topK)

	res, err := a.ReadKnowledge(req, confirmedDisease, className, "doc_specialty", returnFields, topKi)
	if err != nil {
		xlog.LogErrorF(r.SysTrackCode, "send_msg", "医生推荐", fmt.Sprintf("[%s]-未成功推荐医生", a.App.Manifest.Code), err)
		return DocResponse{}, fmt.Errorf("未成功推荐医生: %w", err)
	}
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "医生推荐", fmt.Sprintf("[%s]-医生推荐初始匹配: %v", a.App.Manifest.Code, res[0].Data))
	return DocResponse{DocList: res}, nil
}
