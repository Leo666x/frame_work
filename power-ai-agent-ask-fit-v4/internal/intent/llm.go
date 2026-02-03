package intent

import (
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/rules"
	"strings"
)

type LLMClassifier struct {
	DoublePromptKey string
	MultiPromptKey  string
}

func NewLLMClassifier() *LLMClassifier {
	return &LLMClassifier{
		DoublePromptKey: rules.DoubleClassPromptKey,
		MultiPromptKey:  rules.MultiClassPromptKey,
	}
}

func (c *LLMClassifier) Classify(app *powerai.AgentApp, req *server.AgentRequest) (Result, error) {
	if app == nil || req == nil {
		return Result{}, fmt.Errorf("app or req is nil")
	}

	doublePrompt := getAgentConfigValue(app, c.DoublePromptKey, req.EnterpriseId)
	if doublePrompt == "" {
		doublePrompt = rules.DefaultDoubleClassPrompt
	}

	doublematted := fmt.Sprintf("用户问题:%s\n你的输出:", req.Query)
	matchingPrompt := doublePrompt + "\n" + doublematted
	requestLlm := map[string]interface{}{
		"messages": []interface{}{
			map[string]string{"role": "user", "content": matchingPrompt},
		},
	}

	raw, err := app.SyncCallSystemLLM(req.EnterpriseId, requestLlm)
	if err != nil {
		return Result{}, err
	}
	content := strings.TrimSpace(xjson.Get(raw, "choices.0.message.content").String())
	if content != "即问即办" {
		return Result{
			Intent:     "other",
			AgentCode:  "power-ai-agent-ask-fit",
			Router:     "send_msg",
			Confidence: 0,
			Reason:     "double_class=other",
			IsAsk:      false,
		}, nil
	}

	multiPrompt := getAgentConfigValue(app, c.MultiPromptKey, req.EnterpriseId)
	if multiPrompt == "" {
		multiPrompt = rules.DefaultMultiClassPrompt
	}
	formatted := fmt.Sprintf("用户query:%s", req.Query)
	matchingMulti := multiPrompt + "\n" + formatted
	requestMulti := map[string]interface{}{
		"messages": []interface{}{
			map[string]string{"role": "user", "content": matchingMulti},
		},
	}

	rawMulti, err := app.SyncCallSystemLLM(req.EnterpriseId, requestMulti)
	if err != nil {
		return Result{}, err
	}
	intent := strings.TrimSpace(xjson.Get(rawMulti, "choices.0.message.content").String())

	return Result{
		Intent:     intent,
		AgentCode:  "power-ai-agent-ask-fit",
		Router:     "send_msg",
		Confidence: 1,
		Reason:     "multi_class",
		IsAsk:      true,
	}, nil
}

func getAgentConfigValue(app *powerai.AgentApp, key, enterpriseId string) string {
	conf := app.GetAgentConfig(key, enterpriseId)
	if conf == nil {
		return ""
	}
	return conf.Value
}
