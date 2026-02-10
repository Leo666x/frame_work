package decision

import (
	"fmt"
	"github.com/gin-gonic/gin"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	milvus_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"sort"
	"strings"
)

func LlmRespDeal(resContent string) string {
	resContentRep := strings.ReplaceAll(resContent, "```json", "")
	resContentRep = strings.ReplaceAll(resContentRep, "```", "")
	resContentRep = strings.ReplaceAll(resContentRep, "<think>\n\n</think>\n\n", "")
	return resContentRep
}

func (a *DecisionAgent) LlmCall(req *server.AgentRequest, prompt string) (string, error) {
	requestLlm1st := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": "You are a helpful assistant.",
			},
			map[string]interface{}{
				"role":    "user",
				"content": prompt,
			},
		},
	}
	res, err := a.App.SyncCallSystemLLM(req.EnterpriseId, requestLlm1st)
	if err != nil {
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "llm_call", fmt.Sprintf("[%s]-未成功请求调用大模型", a.App.Manifest.Code), err)
		return "", fmt.Errorf("未成功请求调用大模型: %w", err)
	}
	resRaw := xjson.Get(res, "choices.0.message.content")
	return resRaw.String(), nil
}

func (a *DecisionAgent) BuildCheckpointSummary(req *server.AgentRequest, history string) (string, error) {
	history = strings.TrimSpace(history)
	if history == "" {
		return "", nil
	}
	prompt := fmt.Sprintf(`请将以下医疗对话压缩成短期记忆摘要，要求：
1) 保留核心症状、持续时间、就诊目标、关键医生/科室信息；
2) 保留过敏史/慢病/禁忌等安全信息；
3) 只输出摘要正文，不输出JSON。

对话：
%s`, history)
	resp, err := a.LlmCall(req, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(LlmRespDeal(resp)), nil
}

func RespJsonError(c *gin.Context, code, msg, stc string, data interface{}) {
	c.JSON(200, map[string]interface{}{
		"code":           code,
		"message":        msg,
		"sys_track_code": stc,
		"data":           data,
	})
}

func RespJsonSuccess(c *gin.Context, stc string, data interface{}) {
	c.JSON(200, map[string]interface{}{
		"code":           "success",
		"message":        "执行成功",
		"sys_track_code": stc,
		"data":           data,
	})
}

func (a *DecisionAgent) GetRedis(c *gin.Context, req *server.AgentRequest) *powerai.SessionValue {
	_ = c
	session, err := a.App.GetShortMemory(req.ConversationId)
	if err != nil || session == nil {
		if createErr := a.App.CreateShortMemory(req); createErr != nil {
			xlog.LogErrorF(req.SysTrackCode, "send_msg", "create short memory", "创建短期记忆失败", createErr)
		}
		session, err = a.App.GetShortMemory(req.ConversationId)
		if err != nil || session == nil {
			session = &powerai.SessionValue{}
		}
	}
	if session.FlowContext == nil {
		session.FlowContext = &powerai.FlowContext{}
	}
	if session.GlobalState == nil {
		session.GlobalState = &powerai.GlobalState{}
	}
	if session.GlobalState.Entities == nil && session.GlobalState.Shared != nil {
		session.GlobalState.Entities = session.GlobalState.Shared
	}
	if session.GlobalState.Shared == nil && session.GlobalState.Entities != nil {
		session.GlobalState.Shared = session.GlobalState.Entities
	}
	return session
}

func (a *DecisionAgent) GetAgentByDomain(domainId string) ([]*AgentRegistryModel, error) {
	var agents []*AgentRegistryModel
	sqlQuery := `
		SELECT 
			id, 
			agent_key, 
			domain_id, 
			agent_name, 
			description, 
			mcp_tool_name, 
			is_active 
		FROM 
			ai_agent_registry 
		WHERE 
			domain_id = $1 
		AND 
			is_active = 'true' 
	`
	err := a.App.DBQueryMultiple(&agents, sqlQuery, domainId)
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (a *DecisionAgent) GetHistoryDialogue(req *server.AgentRequest) (string, error) {
	queryReq := &powerai.MemoryQueryRequest{
		ConversationID:      req.ConversationId,
		EnterpriseID:        req.EnterpriseId,
		PatientID:           req.UserId,
		Query:               req.Query,
		TokenThresholdRatio: 0.75,
		RecentTurns:         8,
	}
	ctx, err := a.App.QueryMemoryContext(queryReq)
	if err != nil {
		return "", err
	}
	if ctx.ShouldCheckpointSummary {
		summary, summaryErr := a.BuildCheckpointSummary(req, ctx.FullHistory)
		if summaryErr == nil && strings.TrimSpace(summary) != "" {
			_ = a.App.CheckpointShortMemory(req.ConversationId, summary, queryReq.RecentTurns)
			ctx, err = a.App.QueryMemoryContext(queryReq)
			if err != nil {
				return "", err
			}
		}
	}
	return ctx.History, nil
}

func (a *DecisionAgent) GetAgentConfigByKey(agentKey string) (*AgentRegistryModel, error) {
	var agent AgentRegistryModel
	sqlQuery := `
		SELECT 
			id, 
			agent_key, 
			domain_id, 
			agent_name, 
			description, 
			mcp_tool_name, 
			is_active 
		FROM 
			ai_agent_registry 
		WHERE 
			agent_key = $1 
		AND 
			is_active = 'true' 
		LIMIT 1
	`
	err := a.App.DBQuerySingle(&agent, sqlQuery, agentKey)
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (a *DecisionAgent) RerankDeal(searchResult [][]milvus_mw.SearchResult, enterpriseId string, query string) ([]milvus_mw.SearchResult, error) {
	if len(searchResult) == 0 || len(searchResult[0]) == 0 {
		return nil, nil
	}
	hits := searchResult[0]
	docs := make([]string, 0, len(hits))
	for _, hit := range hits {
		if desc, ok := hit.Data["intent_description"]; ok {
			docs = append(docs, desc)
		} else {
			docs = append(docs, "")
		}
	}
	newScores, err := a.App.RerankResults(enterpriseId, query, docs)
	if err != nil {
		return nil, fmt.Errorf("rerank failed: %w", err)
	}
	for i := range hits {
		hits[i].Score = float32(newScores[i])
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	return hits, nil
}
