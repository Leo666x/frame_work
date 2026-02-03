package retrieval

import (
	"context"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/rules"
	"strconv"
	"strings"
	"time"
)

type QAConfig struct {
	Collection   string
	VectorField  string
	OutputFields []string
	TopK         int
	Timeout      time.Duration
}

func DefaultQAConfig() QAConfig {
	return QAConfig{
		Collection:   "QAXHYY",
		VectorField:  "embedding",
		OutputFields: []string{"q", "a"},
		TopK:         10,
		Timeout:      5 * time.Second,
	}
}

// BuildQAXHYYAnswer replicates old GextractQAXHYYAnswer flow
func BuildQAXHYYAnswer(app *powerai.AgentApp, enterpriseId, query, prompt string, topK int, cfg QAConfig) (string, error) {
	if topK > 0 {
		cfg.TopK = topK
	}
	// 1) Retrieve knowledge from Milvus
	vecs, err := app.EmbedTexts(enterpriseId, []string{query})
	if err != nil || len(vecs) == 0 {
		return "", err
	}
	client, err := app.GetMilvusClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	results, err := client.MilvusVectorSearch(
		ctx,
		cfg.Collection,
		cfg.VectorField,
		[][]float32{vecs[0]},
		cfg.TopK,
		"",
		cfg.OutputFields,
	)
	if err != nil {
		return "", err
	}

	// 2) Build knowledge string
	var knowledgestr strings.Builder
	if len(results) > 0 {
		for _, item := range results[0] {
			q := item.Data["q"]
			a := item.Data["a"]
			if q == "" && a == "" {
				continue
			}
			knowledgestr.WriteString(fmt.Sprintf("问题: %v, 答案: %v\n", q, a))
		}
	}

	// 3) Prompt
	if prompt == "" {
		prompt = rules.DefaultQAXHYYPrompt
	}
	finalPrompt := strings.NewReplacer(
		"CONTEXT1", knowledgestr.String(),
		"CONTEXT2", query,
	).Replace(prompt)

	// 4) LLM
	request := map[string]interface{}{
		"temperature": 0,
		"messages": []interface{}{
			map[string]string{"role": "system", "content": "You are a helpful assistant."},
			map[string]interface{}{"role": "user", "content": finalPrompt},
		},
	}
	resp, err := app.SyncCallSystemLLM(enterpriseId, request)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(xjson.Get(resp, "choices.0.message.content").String()), nil
}

func ParseTopK(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
