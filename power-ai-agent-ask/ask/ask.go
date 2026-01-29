package ask

import (
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework"
	"orgine.com/ai-team/power-ai-framework/pkg/xdatetime"
	"orgine.com/ai-team/power-ai-framework/pkg/xjson"
	"orgine.com/ai-team/power-ai-framework/pkg/xlog"
	"orgine.com/ai-team/power-ai-framework/pkg/xuid"
	"strconv"
	"strings"
)

type AgentAsk struct {
	App *powerai.AgentApp
}

// 返回前端的响应结构体
type OutResponse struct {
	GoUrl   string      `json:"go_url"`
	EndFlag string      `json:"endflag"`
	Type    string      `json:"type"`
	List    interface{} `json:"list"`
}

func createDefaultResponse(content interface{}) OutResponse {
	return OutResponse{
		EndFlag: "true",
		Type:    "text",
		List:    content,
	}
}

func (a *AgentAsk) SendMsg(req *powerai.AgentRequest, event *powerai.HttpStreamEvent) {

	r := &powerai.AgentResponse{
		ConversationId: xuid.UUID(),
		MessageId:      xuid.UUID(),
		CreatedAt:      xdatetime.GetNowDateTimeNano(),
		User:           "zzz",
		Channel:        "源启",
		ChannelApp:     "源启小程序",
		EnterpriseId:   "yuanqi",
		SysTrackCode:   xuid.UUID(),
		AgentCode:      "this-test-code",
	}
	sysTrackCode := req.SysTrackCode

	// 2. 参数验证与判断 req.Inputs { "patient_id":"8000005074" }
	_, ok := req.Inputs["patient_id"]
	if !ok {
		xlog.LogErrorF(sysTrackCode, "send_msg", "请求参数传入处理", fmt.Sprintf("[%s]-%s:{input.patient_id}为空", a.App.AgentAppInfo.Code, powerai.InvalidParam.Message), nil)
		_ = event.WriteAgentResponseError(r, powerai.InvalidParam.Code, fmt.Sprintf("[%s]-%s:{input.patient_id}为空", a.App.AgentAppInfo.Code, powerai.InvalidParam.Message))
		return
	}
	patientId := req.Inputs["patient_id"].(string)
	xlog.LogInfoF(sysTrackCode, "send_msg", "请求参数传入处理", fmt.Sprintf("传入患者ID为: %s", patientId))

	//2.1 智能客服回答
	answer, err := a.GextractQAXHYYAnswer(req.Query)

	//3. 调用大模型进行二元分类（16 VS 其他）
	var doubleClssPrompt string
	doubleClssPromptConf := a.App.GetAgentConfig("double_class_prompt")
	if doubleClssPromptConf == nil {
		xlog.LogErrorF(sysTrackCode, "send_msg", "提示词配置获取", "未找到大模型提示词配置", nil)
	} else {
		doubleClssPrompt = doubleClssPromptConf.Value
	}

	// 调用大模型--组装参数
	doublemattedPrompt := fmt.Sprintf("用户问题:%s\n你的输出:", req.Query) // 组装提示词
	matchingPrompt := doubleClssPrompt + "\n" + doublemattedPrompt

	requestLlm1 := map[string]interface{}{
		"messages": []interface{}{
			map[string]string{
				"role":    "user",
				"content": matchingPrompt,
			},
		},
	}
	classCard, err := a.App.SyncCallSystemLLM(requestLlm1)
	var contentJson1 string
	//defaultQueryConf := a.App.GetAgentConfig("qa_default_reply")
	//defaultOutResp2 := defaultQueryConf.Value
	if err == nil {
		contentJson1 = xjson.Get(classCard, "choices.0.message.content").String()
		xlog.LogInfoF(sysTrackCode, "send_msg", "大模型返回处理", fmt.Sprintf("[%s]-大模型返回非即问即办类型(%s)，使用默认配置值并结束", a.App.AgentAppInfo.Code, classCard))
		if contentJson1 != "即问即办" {
			// 如果大模型返回不是"即问即办"，使用默认配置并结束
			_ = event.WriteAgentResponseMessageWithSpeed(r, answer, 10)
			event.Done(r)
			return
		}

	} else {
		// 调用大模型出错，使用默认配置并结束
		xlog.LogErrorF(sysTrackCode, "send_msg", "大模型调用", fmt.Sprintf("[%s]-大模型调用出错，使用默认配置值并结束", a.App.AgentAppInfo.Code), err)
		//defaultOutResp := createDefaultResponse(defaultQueryConf.Value)
		_ = event.WriteAgentResponseMessageWithSpeed(r, answer, 10)
		event.Done(r)
		return
	}

	// 4.调用大模型进行16即问即办分类
	var extractCardtypeSystemPrompt string
	extractNamePromptConf := a.App.GetAgentConfig("question_classfication_prompt")
	if extractNamePromptConf == nil {
		xlog.LogErrorF(sysTrackCode, "send_msg", "提示词配置获取", "未找到大模型提示词配置", nil)
	} else {
		extractCardtypeSystemPrompt = extractNamePromptConf.Value
	}

	// 调用大模型--组装参数
	formattedPrompt := fmt.Sprintf("用户query:%s", req.Query) // 组装提示词
	matchingNamePrompt := extractCardtypeSystemPrompt + "\n" + formattedPrompt

	requestLlm := map[string]interface{}{
		"messages": []interface{}{
			map[string]string{
				"role":    "user",
				"content": matchingNamePrompt,
			},
		},
	}
	extractedCard, err := a.App.SyncCallSystemLLM(requestLlm) //extractedName,调用大模型，拿到返回值

	var contentJson string
	var message string

	// 获取默认查询关键词配置
	//defaultQueryConf := a.App.GetAgentConfig("hospital_default_reply")
	message = answer // 直接使用默认配置值

	if err == nil {
		contentJson = xjson.Get(extractedCard, "choices.0.message.content").String()
		xlog.LogInfoF(sysTrackCode, "send_msg", "大模型返回处理", fmt.Sprintf("[%s]-大模型返回为空，使用默认配置值，大模型返回 ：%v", a.App.AgentAppInfo.Code, extractedCard))
		if contentJson != "" {
			message = contentJson // 如果大模型返回不为空，则使用大模型提取的医院名称
		} else {
			// 写入日志，大模型返回为空，使用默认配置
			xlog.LogInfoF(sysTrackCode, "send_msg", "大模型返回处理", fmt.Sprintf("[%s]-大模型返回为空，使用默认配置值", a.App.AgentAppInfo.Code))
			// 返回默认配置，结束对话
			//defaultOutResp := createDefaultResponse(message)
			_ = event.WriteAgentResponseMessageWithSpeed(r, answer, 10)
			event.Done(r)
			return
		}
	} else {
		//defaultOutResp := createDefaultResponse(message)
		_ = event.WriteAgentResponseMessageWithSpeed(r, answer, 10)
		event.Done(r)
		return
	}

	// 4. 查询向量数据库
	a.App.InitWeaviateClient()
	className := "Qa_data_get"
	alpha := float32(0.1)
	returnFields := []string{"function_name", "card_type"}
	var topK int = 1

	res, err := a.App.ReadKnowledge(message, className, alpha, returnFields, topK)
	if err != nil {
		xlog.LogErrorF(r.SysTrackCode, "send_msg", "知识库医院推荐", fmt.Sprintf("[%s]-向量库查询错误", a.App.AgentAppInfo.Code), err)
		_ = event.WriteAgentResponseError(r, EmbErrCode, fmt.Sprintf("[%s]-未成功推荐服务卡片", a.App.AgentAppInfo.Code))
		return // 添加return避免后续执行
	}
	xlog.LogInfoF(sysTrackCode, "send_msg", "成功查出结果", fmt.Sprintf("成功查出结果 %v", res))

	// 5. 组装参数，使用WriteAgentResponseStruct将结构体数据，返回前端。

	returnText := a.App.GetAgentConfig("return_solid_text").Value
	// 从res中提取card_type字段
	if len(res) == 0 {
		// 查询结果为空，直接报错并返回
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "查询结果处理", "查询结果为空，无法获取card_type", nil)
		_ = event.WriteAgentResponseError(r, EmbErrCode, fmt.Sprintf("[%s]-未成功推荐服务卡片", a.App.AgentAppInfo.Code))
		return
	}

	cardType := res[0]["card_type"].(string)
	cardName := res[0]["function_name"].(string)

	returnMessages := returnText + cardName
	_ = event.WriteAgentResponseMessageWithSpeed(r, returnMessages, 10)
	_ = event.WriteAgentResponseStruct(r, &OutResponse{
		GoUrl:   "",
		EndFlag: "true",
		Type:    ReturnCardType,
		List:    []string{cardType},
	})
	xlog.LogInfoF(req.SysTrackCode, "send_msg", "即问即办智能体执行成功", fmt.Sprintf("[%s]-即问即办智能体执行成功", a.App.AgentAppInfo.Code))
	event.Done(r)
	return
}

func (a *AgentAsk) GextractQAXHYYAnswer(query string) (string, error) {
	// 1. 取配置并做类型转换
	a.App.InitWeaviateClient()
	topkstring := a.App.GetAgentConfig("qa_xhyy_agent_topk_conf").Value
	topKs, err := strconv.Atoi(topkstring)
	if err != nil {
		return "", err
	}

	alpha := float32(0.9)
	returnFields := []string{"q", "a"}

	// 2. 读取知识库
	knowledgeList, err := a.App.ReadKnowledge(query, "QAXHYY", alpha, returnFields, topKs)
	if err != nil {
		return "", err
	}

	// 3. 拼接知识字符串
	var knowledgestr string
	for _, item := range knowledgeList {
		knowledgestr += fmt.Sprintf("问题: %v, 答案: %v\n", item["q"], item["a"])
	}

	// 4. 组装 prompt
	QAXHYYExtractionPrompt := strings.NewReplacer(
		"CONTEXT1", knowledgestr,
		"CONTEXT2", query,
	).Replace(a.App.GetAgentConfig("qa_xhyy_agent_QAXHYYExtraction_prompt").Value)

	// 5. 构造 LLM 请求
	requestLlm2 := map[string]interface{}{
		"temperature": 0,
		"messages": []interface{}{
			map[string]string{
				"role":    "system",
				"content": "You are a helpful assistant.",
			},
			map[string]interface{}{
				"role":    "user",
				"content": QAXHYYExtractionPrompt,
			},
		},
	}

	// 6. 调用大模型
	resp, err := a.App.SyncCallSystemLLM(requestLlm2)
	if err != nil {
		return "", err
	}

	contenJson2 := xjson.Get(resp, "choices.0.message.content").String()
	return contenJson2, nil
}
