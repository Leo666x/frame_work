package main

import (
	_ "embed"
	"fmt"
	"github.com/gin-gonic/gin"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/intent"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/retrieval"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/response"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/rules"
)

//go:embed manifest.json
var manifest string

var app *powerai.AgentApp

func main() {
	var err error
	app, err = powerai.NewAgent(
		manifest,
		powerai.WithSendMsgRouter(sendMsg),
		powerai.WithDefaultConfigs(defaultConfigs()),
	)
	if err != nil {
		panic(err)
	}
	app.Run()
}

func sendMsg(c *gin.Context) {
	req, resp, event, ok := powerai.DoValidateAgentRequest(c, "power-ai-agent-ask-fit")
	if !ok {
		return
	}

	// 1) required input validation (keep legacy behavior)
	if req.Inputs == nil || req.Inputs["patient_id"] == nil {
		msg := fmt.Sprintf("[%s]-%s:{input.patient_id}为空", "power-ai-agent-ask-fit", server.InvalidParam.Message)
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "validate", msg, nil)
		_ = event.WriteAgentResponseError(resp, server.InvalidParam.Code, msg)
		return
	}

	// 2) QA answer from knowledge (legacy flow)
	prompt := getAgentConfigValue(rules.QAXHYYPromptKey, req.EnterpriseId, rules.DefaultQAXHYYPrompt)
	topK := retrieval.ParseTopK(getAgentConfigValue(rules.QAXHYYTopKKey, req.EnterpriseId, "10"), 10)
	answer, _ := retrieval.BuildQAXHYYAnswer(app, req.EnterpriseId, req.Query, prompt, topK, retrieval.DefaultQAConfig())

	// 3) Intent classification (legacy LLM flow)
	cls := intent.NewLLMClassifier()
	intentResult, err := cls.Classify(app, req)
	if err != nil || !intentResult.IsAsk {
		_ = event.WriteAgentResponseMessage(resp, answer)
		event.Done(resp)
		return
	}
	if intentResult.Intent == "" {
		_ = event.WriteAgentResponseMessage(resp, answer)
		event.Done(resp)
		return
	}

	// 4) Retrieve card by intent (Milvus)
	cards, err := retrieval.LookupCards(app, req, intentResult, intentResult.Intent, retrieval.DefaultConfig())
	if err != nil {
		_ = event.WriteAgentResponseError(resp, server.InvokeServiceError.Code, fmt.Sprintf("retrieval failed: %v", err))
		return
	}

	// 5) Build response (legacy compatible)
	returnText := getAgentConfigValue(rules.ReturnSolidTextKey, req.EnterpriseId, rules.DefaultReturnSolidText)
	msg := returnText
	if len(cards) > 0 {
		msg = returnText + cards[0].FunctionName
	}
	_, legacyPayload := response.BuildLegacy(intentResult, cards, msg)
	_ = event.WriteAgentResponseMessage(resp, msg)
	_ = event.WriteAgentResponseStruct(resp, legacyPayload)
	event.Done(resp)
}

func defaultConfigs() map[string]*powerai.Config {
	return map[string]*powerai.Config{
		rules.DoubleClassPromptKey: {
			Key:       rules.DoubleClassPromptKey,
			Value:     rules.DefaultDoubleClassPrompt,
			Name:      "二分类意图识别提示词",
			AgentCode: "power-ai-agent-ask-fit",
			Classify:  powerai.GeneralConfigClassify,
			Remark:    "用于是否进入即问即办",
		},
		rules.MultiClassPromptKey: {
			Key:       rules.MultiClassPromptKey,
			Value:     rules.DefaultMultiClassPrompt,
			Name:      "多分类意图识别提示词",
			AgentCode: "power-ai-agent-ask-fit",
			Classify:  powerai.GeneralConfigClassify,
			Remark:    "细分意图分类",
		},
		rules.DecisionConfigKey: {
			Key:       rules.DecisionConfigKey,
			Value:     rules.DefaultDecisionConfig(),
			Name:      "意图识别配置",
			AgentCode: "power-ai-agent-ask-fit",
			Classify:  powerai.DecisionConfigClassify,
			Remark:    "意图识别规则",
		},
		rules.ReturnSolidTextKey: {
			Key:       rules.ReturnSolidTextKey,
			Value:     rules.DefaultReturnSolidText,
			Name:      "默认响应文本",
			AgentCode: "power-ai-agent-ask-fit",
			Classify:  powerai.GeneralConfigClassify,
			Remark:    "卡片提示文本",
		},
		rules.QAXHYYPromptKey: {
			Key:       rules.QAXHYYPromptKey,
			Value:     rules.DefaultQAXHYYPrompt,
			Name:      "智能客服提示词",
			AgentCode: "power-ai-agent-ask-fit",
			Classify:  powerai.GeneralConfigClassify,
			Remark:    "用于知识库问答",
		},
		rules.QAXHYYTopKKey: {
			Key:       rules.QAXHYYTopKKey,
			Value:     "10",
			Name:      "最多显示条数",
			AgentCode: "power-ai-agent-ask-fit",
			Classify:  powerai.GeneralConfigClassify,
			Remark:    "知识库topK",
		},
	}
}

func getAgentConfigValue(key, enterpriseId, fallback string) string {
	conf := app.GetAgentConfig(key, enterpriseId)
	if conf == nil || conf.Value == "" {
		return fallback
	}
	return conf.Value
}
