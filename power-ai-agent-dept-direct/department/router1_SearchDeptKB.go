package department

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

func (a *DeptAgent) router1_SearchDeptKB(
	event *server.SSEEvent,
	req *server.AgentRequest,
	resp *server.AgentResponse,
	session *powerai.SessionValue,
	mySlots *DeptSlots,
	extractResp DeptExtractionResult,
) {
	out, err := a.DeptName2deptKnowledge(req, resp, extractResp.DeptName)
	if err != nil {
		_ = event.WriteAgentResponseError(resp, ErrorParse, fmt.Sprintf("[%s]-未成功完成科室推荐：%s", a.App.Manifest.Code, err))
		return
	}
	candidates := make([]map[string]interface{}, 0)
	if originalList, ok := out.DeptList.([]milvus_mw.SearchResult); ok {
		for _, item := range originalList {
			candidate := make(map[string]interface{})
			for k, v := range item.Data {
				candidate[k] = v
			}
			candidates = append(candidates, candidate)
		}
	}

	// 精准匹配
	deptList := StrictMatchMapList(extractResp.DeptName, "dept_name", candidates)
	if len(deptList) == 0 {
		// A. 查无此
		reply := fmt.Sprintf("抱歉，没找到名为“%s”的科室。请确认科室名称是否正确。", extractResp.DeptName)

		out.Msg = reply
		out.EndFlag = "false"

		// 记忆更新
		err = a.Updatememory(req, session, reply, mySlots)
		if err != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
		}

		_ = event.WriteAgentResponseMessage(resp, reply)
		event.Done(resp)
		return

	}

	// 数据模拟
	// 模拟 RAG 检索到的科室列表
	//deptList := []map[string]interface{}{
	//	// 数据 1：完全匹配或高相关度
	//	{
	//		"hospital_name": "同济医院",
	//		"hospital_id":   "tjyyggyq",
	//		"dept_name":     "产科", // 科室名称
	//		"dept_id":       "020302",
	//		"dept_his_id":   "020302",
	//		"dept_level":    "1", // 1级科室/2级科室
	//		"master_code":   "2", // 父级编码或分类码
	//		// 简介通常用于确认该科室是否符合用户需求
	//		"dept_introduction": "1、各种妊娠合并症如妊娠期高血压，妊娠期糖尿病，妊娠合并肝炎、风湿免疫性疾病、子宫肌瘤、卵巢囊肿等的诊治；2、妊娠期阴道出血的诊治；3、孕期安全用药咨询、营养与体重管理；4、倡导自然分娩、鼓励分娩镇痛，擅长各种剖宫产手术。",
	//	},
	//	// 数据 2：相似科室（用于消歧，防止用户想挂的是泌尿外科）
	//	{
	//		"hospital_name":     "同济医院",
	//		"hospital_id":       "tjyyggyq",
	//		"dept_name":         "妊娠期并发症门诊",
	//		"dept_id":           "0203020301",
	//		"dept_his_id":       "0203020301",
	//		"dept_level":        "1",
	//		"master_code":       "2",
	//		"dept_introduction": "1、各种妊娠合并症如妊娠期高血压，妊娠期糖尿病，妊娠合并肝炎、风湿免疫性疾病、子宫肌瘤、卵巢囊肿等的诊治；2、妊娠期阴道出血的诊治；3、孕期安全用药咨询、营养与体重管理；4、倡导自然分娩、鼓励分娩镇痛，擅长各种剖宫产手术。",
	//	},
	//}

	//msg := "根据您的要求，匹配到的科室数据，如下:"
	//out.Msg = msg
	//out.Type = ReturnCardTypeDept
	//out.EndFlag = "true"
	//out.DeptList = deptList
	//
	//// 更新记忆
	////err = a.Updatememory(req, resp, session, aiMsg, msgHistory, mySlots)
	////if err != nil {
	////	xlog.LogErrorF(req.SysTrackCode, "send_msg", "引导提示词大模型记忆管理", fmt.Sprintf("[%s]-未成功完成引导提示词大模型记忆管理", a.App.Manifest.Code), err)
	////}
	//
	//_ = event.WriteAgentResponseMessage(resp, msg)
	//_ = event.WriteAgentResponseStruct(resp, out)
	//event.Done(resp)

	if extractResp.Intent == "INTRO" {
		out.Type = ReturnCardTypeDept

		jsonBytes, _ := json.Marshal(out.DeptList)
		deptListStr := string(jsonBytes)

		// 提示词工程
		prompt := strings.NewReplacer(
			"UserQuery", req.Query,
			"DeptJsonData", deptListStr,
		).Replace(PROMPT_DEPT_INTRO_MULTI)

		var reply string
		requestLlm := buildPromptRequest(prompt)
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
			//fmt.Println(msg)
			if strings.Contains(strings.ToUpper(msg), "[DONE]") {
				// 记忆更新
				err = a.Updatememory(req, session, reply, mySlots)
				if err != nil {
					xlog.LogErrorF(req.SysTrackCode, "send_msg", "记忆管理", fmt.Sprintf("[%s]-未成功完成记忆管理", a.App.Manifest.Code), err)
				}
				
				out.DeptList = candidates
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

	} else {
		// --- 场景 1: 找/挂科室 ---
		// 直接返回卡片
		reply := fmt.Sprintf("为您找到以下相关科室，您可以直接点击下方卡片挂号。")
		out.Msg = reply
		out.Type = ReturnCardTypeDept
		out.EndFlag = "true"
		_ = event.WriteAgentResponseMessage(resp, reply)
		_ = event.WriteAgentResponseStruct(resp, out)
		event.Done(resp)

	}

	return

}

func (a *DeptAgent) DeptName2deptKnowledge(
	req *server.AgentRequest,
	r *server.AgentResponse,
	confirmedDisease string,
) (DeptResponse, error) {

	//  知识库检索科室
	// 开始检索
	// 查 召回
	className := "Dept_info_"
	returnFields := []string{"hospital_id", "hospital_name", "dept_his_id", "dept_id", "dept_level", "master_code", "dept_name",
		"dept_introduction"}
	topK := "10"
	topKi, _ := strconv.Atoi(topK)

	res, err := a.ReadKnowledge(req, confirmedDisease, className, "dept_name", returnFields, topKi)
	if err != nil {
		xlog.LogErrorF(r.SysTrackCode, "send_msg", "科室推荐", fmt.Sprintf("[%s]-未成功推荐科室", a.App.Manifest.Code), err)
		return DeptResponse{}, fmt.Errorf("未成功推荐科室: %w", err)
	}
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "科室推荐", fmt.Sprintf("[%s]-科室推荐初始匹配: %v", a.App.Manifest.Code, res[0].Data))
	return DeptResponse{DeptList: res}, nil
}
