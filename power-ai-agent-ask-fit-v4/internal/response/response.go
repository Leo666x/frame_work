package response

import (
	"orgine.com/power-ai-agent-ask-fit-v4/internal/intent"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/retrieval"
)

type Card struct {
	CardType     string `json:"card_type"`
	FunctionName string `json:"function_name"`
}

type NewPayload struct {
	Text         string `json:"text"`
	Intent       string `json:"intent"`
	Confidence   float64 `json:"confidence"`
	Reason       string `json:"reason"`
	TargetAgent  string `json:"target_agent"`
	TargetRouter string `json:"target_router"`
	Cards        []Card `json:"cards"`
	EndFlag      string `json:"endflag"`
	Type         string `json:"type"`
}

type LegacyPayload struct {
	GoUrl   string      `json:"go_url"`
	EndFlag string      `json:"endflag"`
	Type    string      `json:"type"`
	List    interface{} `json:"list"`
}

func BuildNew(res intent.Result, cards []retrieval.Card, msg string) (string, NewPayload) {
	if msg == "" {
		msg = "已识别您的需求"
		if res.Intent == "general" {
			msg = "未命中具体意图，已走兜底处理"
		}
	}
	payloadCards := make([]Card, 0, len(cards))
	for _, c := range cards {
		payloadCards = append(payloadCards, Card{CardType: c.CardType, FunctionName: c.FunctionName})
	}
	return msg, NewPayload{
		Text:         msg,
		Intent:       res.Intent,
		Confidence:   res.Confidence,
		Reason:       res.Reason,
		TargetAgent:  res.AgentCode,
		TargetRouter: res.Router,
		Cards:        payloadCards,
		EndFlag:      "true",
		Type:         "card",
	}
}

func BuildLegacy(res intent.Result, cards []retrieval.Card, msg string) (string, LegacyPayload) {
	if msg == "" {
		msg, _ = BuildNew(res, cards, "")
	}
	list := []string{}
	if len(cards) > 0 {
		list = append(list, cards[0].CardType)
	}
	return msg, LegacyPayload{
		GoUrl:   "",
		EndFlag: "true",
		Type:    "card",
		List:    list,
	}
}
