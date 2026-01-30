package main

import (
	"encoding/json"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
)

const (
	category = "[Demo:demo]"

	content = `
这里是意图识别提示词
    `
)

func getDecision() string {
	d := &powerai.Decision{
		Category:       category,
		Content:        content,
		Identification: "power-ai-agent-v4demo",
	}

	b, _ := json.Marshal(d)

	return string(b)
}
