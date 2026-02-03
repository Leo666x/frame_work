package decision

import (
	"context"
	"encoding/json"
	"fmt"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	milvus_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"strings"
)

func (a *DecisionAgent) Layer4_SupervisorDispatch(req *server.AgentRequest, previousDomain, msgHistory string, session *powerai.SessionValue) (string, error) {

	// 目的: 将搜索范围从 全局 缩小到 特定领域
	targetDomain, err := a.l4_IdentifyDomain(req, previousDomain, msgHistory)
	if err != nil {
		return "", err
	}

	// 1. 获取该领域下所有活跃的 Agent (从内存缓存读取)
	allAgentsInDomain, err := a.GetAgentByDomain(targetDomain)

	var candidates []milvus_mw.SearchResult
	// 2. 动态策略选择
	if len(allAgentsInDomain) > RAG_TopK {
		milvusCli, err := a.App.GetMilvusClient()
		if err != nil {
			return "", err
		}
		// 在向量库中搜索描述最匹配的 n 个 Agent
		ctx := context.Background()

		queryVectors, err := a.App.EmbedTexts(req.EnterpriseId, []string{req.Query}) // 查询向量  [][]float32{{...}}
		if err != nil {
			return "", err
		}
		className := "Sys_agent_registry_"                                                     // 知识库名称  智能体注册表 (Sys_agent_registry)
		vectorFieldName := "emb_field"                                                         // 向量字段名 (如 "doc_embedding")
		outputFields := []string{"agent_code", "agent_name", "domain_category", "description"} // 需要返回的标量字段列表 (如 ["doc_name", "doc_content"])
		filterExpr := fmt.Sprintf("domain_category=='%s'", targetDomain)
		searchResult, err := milvusCli.MilvusVectorSearch(ctx, className, vectorFieldName, queryVectors, RAG_TopK, filterExpr, outputFields)
		if err != nil {
			return "", fmt.Errorf("RAG search failed: %w", err)
		}

		// 重排序
		candidates, err = a.RerankDeal(searchResult, req.EnterpriseId, req.Query)
		if err != nil {
			return "", err
		}

	} else {
		// 直接把这 <RAG_TopK 个 Agent 的描述都塞进 Prompt
		for _, each := range allAgentsInDomain { //统一格式
			data := map[string]string{
				"agent_code":      each.AgentKey.String,
				"agent_name":      each.AgentName.String,
				"domain_category": each.DomainID.String,
				"description":     each.Description.String,
			}
			candidates = append(candidates, milvus_mw.SearchResult{
				Data: data,
			})
		}
	}

	// 如果没有候选者 (异常情况)，兜底给客服或导诊
	if len(candidates) == 0 {
		return "power-ai-agent-triage", nil
	}

	// 若top1 分数大于0.85 且与top2相差很大  则直接跳top1
	isSafeToSkipLLM := false
	top1 := candidates[0]
	if top1.Score >= ThresholdScore {
		if len(candidates) == 1 {
			isSafeToSkipLLM = true
		} else {
			// 如果第一名远超第二名，直接信第一名
			top2 := candidates[1]
			if (top1.Score - top2.Score) >= ThresholdScoreDiff {
				isSafeToSkipLLM = true
			}
		}

	}
	if isSafeToSkipLLM {
		// 【极速模式】不调大模型，直接返回
		return top1.Data["agent_code"], nil
	}

	// ============================================================
	// Step 3: 第二层 - LLM 精准决策 (Agent Selection)
	// ============================================================

	agentCode, err := a.l4_SelectBestAgent_LLM(req, candidates, msgHistory, session)
	if err != nil {
		return "", fmt.Errorf("LLM 精准决策未成功: %w", err)
	}
	return agentCode, nil
}

func (a *DecisionAgent) l4_IdentifyDomain(req *server.AgentRequest, previousDomain, msgHistory string) (string, error) {

	// 组装 Prompt
	prompt := strings.NewReplacer(
		"PreviousDomain", previousDomain,
		"RecentHistory", msgHistory,
		"UserQuery", req.Query,
	).Replace(L4DominPrompt)
	respLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return "", fmt.Errorf("领域意图分类未成功：%w", err)
	}

	res := LlmRespDeal(respLlm)
	return res, nil
}

func (a *DecisionAgent) l4_SelectBestAgent_LLM(req *server.AgentRequest, candidates []milvus_mw.SearchResult, msgHistory string, session *powerai.SessionValue) (string, error) {

	// 1. 拼装候选列表 (Requirement 1)
	var candidatesBuilder strings.Builder
	for _, agent := range candidates {
		// 格式: [agent_key]: 描述文本
		candidatesBuilder.WriteString(fmt.Sprintf("- [%s]: %s\n", agent.Data["agent_code"], agent.Data["description"]))
	}

	// 2. 获取上一轮 Agent (Last Agent)
	lastAgent := session.FlowContext.CurrentAgentKey
	if lastAgent == "" {
		lastAgent = "None (New Session)"
	}

	// 3. 拼装全局槽位 (Global Slots)
	slotsBytes, err := json.Marshal(session.GlobalState.Entities)
	if err != nil {
		// 如果序列化失败，给一个空对象，不阻断流程
		slotsBytes = []byte("{}")
	}
	slotsJsonStr := string(slotsBytes)

	// 4. Prompt
	prompt := strings.NewReplacer(
		"LastAgent", lastAgent,
		"GlobalSlots", slotsJsonStr,
		"History", msgHistory,
		"CandidateAgents", candidatesBuilder.String(),
		"UserInput", req.Query,
	).Replace(L4SelectBestAgentPrompt)

	respLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return "", err
	}
	res := LlmRespDeal(respLlm)

	var resTargetAgent ResTargetAgent
	if err := json.Unmarshal([]byte(res), &resTargetAgent); err != nil {
		return "", err
	}

	return resTargetAgent.TargetAgent, nil
}
