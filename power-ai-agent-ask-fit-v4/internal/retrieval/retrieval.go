package retrieval

import (
	"context"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/power-ai-agent-ask-fit-v4/internal/intent"
	"time"
)

type Card struct {
	CardType     string `json:"card_type"`
	FunctionName string `json:"function_name"`
}

type Config struct {
	Collection    string
	VectorField   string
	OutputFields  []string
	TopK          int
	FilterExpr    string
	SearchTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		Collection:    "qa_data_get",
		VectorField:   "embedding",
		OutputFields:  []string{"card_type", "function_name"},
		TopK:          1,
		FilterExpr:    "",
		SearchTimeout: 5 * time.Second,
	}
}

func LookupCards(app *powerai.AgentApp, req *server.AgentRequest, res intent.Result, queryText string, cfg Config) ([]Card, error) {
	if app == nil {
		return nil, fmt.Errorf("agent app is nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if queryText == "" {
		queryText = req.Query
	}
	// 1) Embed query (bge-m3 -> 1024 dim)
	vecs, err := app.EmbedTexts(req.EnterpriseId, []string{queryText})
	if err != nil || len(vecs) == 0 {
		return nil, fmt.Errorf("embed query failed: %v", err)
	}

	// 2) Milvus search
	client, err := app.GetMilvusClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.SearchTimeout)
	defer cancel()
	results, err := client.MilvusVectorSearch(
		ctx,
		cfg.Collection,
		cfg.VectorField,
		[][]float32{vecs[0]},
		cfg.TopK,
		cfg.FilterExpr,
		cfg.OutputFields,
	)
	if err != nil {
		return nil, err
	}

	// 3) Flatten results
	var cards []Card
	if len(results) == 0 || len(results[0]) == 0 {
		return cards, nil
	}
	for _, r := range results[0] {
		card := Card{
			CardType:     r.Data["card_type"],
			FunctionName: r.Data["function_name"],
		}
		cards = append(cards, card)
	}
	return cards, nil
}
