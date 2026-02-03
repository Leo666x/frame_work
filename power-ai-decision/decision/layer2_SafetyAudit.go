package decision

import (
	"encoding/json"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"strings"
)

type ResClass struct {
	Category string `json:"category"`
}

// 安全审计
func (a *DecisionAgent) Layer2_SafetyAudit(req *server.AgentRequest) (string, error) {

	// 提示词工程
	prompt := strings.Replace(Layer2SafetyAuditPrompt, "CONTEXT", req.Query, 1)
	resp, err := a.LlmCall(req, prompt)
	if err != nil {
		return "", fmt.Errorf("安全审计分类未成功：%w", err)
	}

	res := LlmRespDeal(resp)
	var resClass ResClass
	if err := json.Unmarshal([]byte(res), &resClass); err != nil {
		return "", err
	}

	switch resClass.Category {
	case "SAFE":
		// 安全，放行，进入下一步 Router
		return "SAFE", nil

	case "PROHIBITED":
		// 红色高危：冷处理
		return "抱歉，我是一个专注于医疗领域的智能助手。您输入的内容涉及敏感话题，我无法回答。", nil

	case "EMERGENCY":
		// 橙色急救：强干预，带前端 Metadata
		msg := "⚠️【紧急警报】检测到您可能处于危急医疗状况！AI无法替代急救，请立即拨打 120！"
		return msg, nil

	case "ILLEGAL_MEDICAL":
		// 黄色合规：软拒绝
		return "抱歉，根据医疗法规，我无法提供违禁药品或违规医疗证明的相关服务。请通过正规渠道就医。", nil
	}
	return "SAFE", nil
}
