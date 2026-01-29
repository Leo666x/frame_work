package powerai

import (
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/weaviate"
	"sort"
)

func (a *AgentApp) GetWeaviateClient() (*weaviate_mw.Weaviate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.weaviate == nil {
		client, err := initWeaviate(a.etcd)
		if err != nil {
			return nil, err
		}
		a.weaviate = client
	}
	return a.weaviate, nil
}

// WeaviateInsertObjects 批量将 records 和 vectors 插入 Weaviate
func (a *AgentApp) WeaviateInsertObjects(enterpriseId, className string, records []map[string]string, vectors [][]float32) ([]string, error) {
	if enterpriseId != "" {
		className = fmt.Sprintf("%s_%s", className, enterpriseId)
	}
	client, err := a.GetWeaviateClient()
	if err != nil {
		return nil, err
	}
	return client.Insert(className, records, vectors)
}

// WeaviateHybridSearch 在 Weaviate 上执行混合检索，返回原始结果
func (a *AgentApp) WeaviateHybridSearch(enterpriseId, className, query string, vector []float32, returnFields []string, topK int, alpha float32) ([]map[string]interface{}, error) {
	if enterpriseId != "" {
		className = fmt.Sprintf("%s_%s", className, enterpriseId)
	}
	client, err := a.GetWeaviateClient()
	if err != nil {
		return nil, err
	}
	return client.HybridSearch(className, query, vector, returnFields, topK, alpha)
}

// WeaviateMergeAndSort 合并 _score 与 _rerank_score，并按 rerank 分数降序返回
func (a *AgentApp) WeaviateMergeAndSort(raw []map[string]interface{}, rerankScores []float64) []map[string]interface{} {
	type tmp struct {
		obj   map[string]interface{}
		score float64
	}
	list := make([]tmp, len(raw))
	for i, obj := range raw {
		if add, ok := obj["_additional"].(map[string]interface{}); ok {
			obj["_score"] = add["score"]
		}
		obj["_rerank_score"] = rerankScores[i]
		list[i] = tmp{obj: obj, score: rerankScores[i]}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].score > list[j].score
	})
	final := make([]map[string]interface{}, len(list))
	for i, t := range list {
		final[i] = t.obj
	}
	return final
}

// WeaviateDeleteClass 删除整个 class 及其所有数据
func (a *AgentApp) WeaviateDeleteClass(enterpriseId, className string) error {
	if enterpriseId != "" {
		className = fmt.Sprintf("%s_%s", className, enterpriseId)
	}
	client, err := a.GetWeaviateClient()
	if err != nil {
		return err
	}
	return client.DeleteClass(className)
}

// WeaviateEnsureClassExists 校验 Weaviate 中指定 class 是否已创建
func (a *AgentApp) WeaviateEnsureClassExists(enterpriseId, className string) error {
	if enterpriseId != "" {
		className = fmt.Sprintf("%s_%s", className, enterpriseId)
	}
	client, err := a.GetWeaviateClient()
	if err != nil {
		return err
	}
	return client.EnsureClassExists(className)
}

// ***************************************************************************************************************
//
// ReadKnowledge (查)：混合检索 + 重排序
//
//	query:        查询文本
//	className:    class 名称
//	alpha:        向量/关键词 权重
//	returnFields: 返回的属性列表
//	topK:         返回数量
//
// 返回 排序后的对象数组，每个 map 包含 returnFields + "_score" + "_rerank_score"
//
// ***************************************************************************************************************

func (a *AgentApp) ReadKnowledge(enterpriseId, query, className string, alpha float32, returnFields []string, topK int) ([]map[string]interface{}, error) {
	if enterpriseId != "" {
		className = fmt.Sprintf("%s_%s", className, enterpriseId)
	}
	client, err := a.GetWeaviateClient()
	if err != nil {
		return nil, err
	}
	// 1. 检查 class
	if err := client.EnsureClassExists(className); err != nil {
		return nil, err
	}
	// 2. 向量化 query
	vec, err := a.EmbedTexts(enterpriseId, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vec[0]
	// 3. 混合检索
	raw, err := client.HybridSearch(className, query, queryVec, returnFields, topK, alpha)
	if err != nil {
		return nil, err
	}
	// 4. 重排序
	docs := extractColumnInterface(enterpriseId, raw, returnFields[0])
	scores, err := a.RerankResults(enterpriseId, query, docs)
	if err != nil {
		return nil, err
	}
	// 5. 合并打分并排序
	final := mergeAndSort(enterpriseId, raw, scores)
	return final, nil
}

func (a *AgentApp) ReadKnowledgeInclude(enterpriseId, query, className string, alpha float32, returnFields []string, topK int, include map[string][]string) ([]map[string]interface{}, error) {
	if enterpriseId != "" {
		className = fmt.Sprintf("%s_%s", className, enterpriseId)
	}
	client, err := a.GetWeaviateClient()
	if err != nil {
		return nil, err
	}
	// 1. 检查 class
	if err := client.EnsureClassExists(className); err != nil {
		return nil, err
	}
	// 2. 向量化 query
	vec, err := a.EmbedTexts(enterpriseId, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vec[0]
	// 3. 混合检索
	raw, err := client.HybridSearchAllInclude(className, query, queryVec, returnFields, topK, alpha, include)
	if err != nil {
		return nil, err
	}
	// 4. 重排序
	docs := extractColumnInterface(enterpriseId, raw, returnFields[0])
	scores, err := a.RerankResults(enterpriseId, query, docs)
	if err != nil {
		return nil, err
	}
	// 5. 合并打分并排序
	final := mergeAndSort(enterpriseId, raw, scores)
	return final, nil
}

// mergeAndSort 合并 _score 与 _rerank_score，并按 rerank 分数降序返回
func mergeAndSort(enterpriseId string, raw []map[string]interface{}, rerankScores []float64) []map[string]interface{} {
	type tmp struct {
		obj   map[string]interface{}
		score float64
	}
	list := make([]tmp, len(raw))
	for i, obj := range raw {
		if add, ok := obj["_additional"].(map[string]interface{}); ok {
			obj["_score"] = add["score"]
		}
		obj["_rerank_score"] = rerankScores[i]
		list[i] = tmp{obj: obj, score: rerankScores[i]}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].score > list[j].score
	})
	final := make([]map[string]interface{}, len(list))
	for i, t := range list {
		final[i] = t.obj
	}
	return final
}

// extractColumnInterface 从原始结果中提取某字段值列表（string）
func extractColumnInterface(enterpriseId string, raw []map[string]interface{}, field string) []string {
	out := make([]string, len(raw))
	for i, obj := range raw {
		out[i] = fmt.Sprintf("%v", obj[field])
	}
	return out
}
