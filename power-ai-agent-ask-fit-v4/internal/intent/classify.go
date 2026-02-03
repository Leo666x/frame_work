package intent

import (
	"strings"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/rules"
)

func Classify(query string, rs rules.RuleSet) Result {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, r := range rs.Rules {
		for _, kw := range r.Keywords {
			if kw == "" {
				continue
			}
			if strings.Contains(q, strings.ToLower(kw)) {
				return Result{
					Intent:     r.Name,
					AgentCode:  r.AgentCode,
					Router:     r.Router,
					Confidence: r.Confidence,
					Reason:     r.Reason,
					IsAsk:      true,
				}
			}
		}
	}
	return Result{
		Intent:     rs.DefaultIntent,
		AgentCode:  rs.DefaultAgentCode,
		Router:     rs.DefaultRouter,
		Confidence: rs.DefaultConfidence,
		Reason:     rs.DefaultReason,
		IsAsk:      false,
	}
}
