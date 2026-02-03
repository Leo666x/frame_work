package department

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	milvus_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"regexp"
	"sort"
	"strings"
)

// 初始化配置参数
func (a *DeptAgent) InitReportConfig(rsp *server.AgentResponse, event *server.SSEEvent, StrConfig map[string]interface{}) {
	reportConfig := make(map[string]interface{})
	for key, value := range StrConfig {
		timeExtractionPromptConf := a.App.GetAgentConfig(key, rsp.EnterpriseId)
		if timeExtractionPromptConf == nil {
			xlog.LogErrorF(rsp.SysTrackCode, "send_msg", "etcd配置获取", fmt.Sprintf("[%s]-%s获取为空", a.App.Manifest.Code, value), nil)
			_ = event.WriteAgentResponseError(rsp, server.Unavailable.Code, fmt.Sprintf("[%s]-%s获取为空", a.App.Manifest.Code, value))
			event.Done(rsp)
		}
		reportConfig[key] = timeExtractionPromptConf.Value
	}
	a.resultConfig = reportConfig
}

func LlmRespDeal(resContent string) string {

	resContentRep := strings.ReplaceAll(resContent, "```json", "")
	resContentRep = strings.ReplaceAll(resContentRep, "```", "")
	resContentRep = strings.ReplaceAll(resContentRep, "<think>\n\n</think>\n\n", "")

	return resContentRep
}

func (a *DeptAgent) ReadKnowledge(req *server.AgentRequest, query, className, matchField string, outputFields []string, topK int) ([]milvus_mw.SearchResult, error) {
	milvusCli, err := a.App.GetMilvusClient()
	if err != nil {
		return nil, err
	}
	// 在向量库中搜索描述最匹配的 n 个 Agent
	ctx := context.Background()

	queryVectors, err := a.App.EmbedTexts(req.EnterpriseId, []string{query}) // 查询向量  [][]float32{{...}}
	if err != nil {
		return nil, err
	}
	vectorFieldName := "emb_field" // 向量字段名 (如 "doc_embedding")
	filterExpr := ""
	searchResult, err := milvusCli.MilvusVectorSearch(ctx, className, vectorFieldName, queryVectors, topK, filterExpr, outputFields)
	if err != nil {
		return nil, fmt.Errorf("RAG search failed: %w", err)
	}

	// 重排序
	candidates, err := a.RerankDeal(searchResult, req.EnterpriseId, req.Query, matchField)
	if err != nil {
		return nil, err
	}
	return candidates, nil

}
func LlmRespDealGuide(resContent string) string {

	resContentRep := strings.ReplaceAll(resContent, "```json", "")
	resContentRep = strings.ReplaceAll(resContentRep, "```", "")
	resContentRep = strings.ReplaceAll(resContentRep, "<think>\n\n</think>\n\n", "")

	resContentRep = strings.Replace(resContentRep, "你：", "", 1)
	resContentRep = strings.Replace(resContentRep, "\"round\": 1,", "", 1)
	resContentRep = strings.Replace(resContentRep, "\"round\":1,", "", 1)
	resContentRep = strings.Replace(resContentRep, "\"\"answer\"", "\"answer\"", 1)

	re := regexp.MustCompile(`(?s)\{.*\}`)
	resContentRep = re.FindString(resContentRep)

	return resContentRep
}

func (a *DeptAgent) LlmCall(req *server.AgentRequest, prompt string) (string, error) {

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
		return "", fmt.Errorf("-未成功请求调用大模型: %w", err)
	}

	resRaw := xjson.Get(res, "choices.0.message.content")
	return resRaw.String(), nil
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

func (a *DeptAgent) GetRedis(c *gin.Context, req *server.AgentRequest) *powerai.SessionValue {

	//// --- [模拟数据开始] ---
	//// 仅用于测试 Layer 3 逻辑：如果 Redis 里没数据，强制写入一个模拟的“上一轮状态”
	//// 场景：上一轮是报告解读智能体，刚说完“白细胞高”
	//currentAgentKey := "power-ai-agent-department"
	//triageSlots := TriageSlots{
	//	DiagnosisRound:    1,
	//	HistorySummary:    "用户主诉头皮有多发肿物，自述可能为毛囊根鞘瘤。已询问症状性质（是否有疼痛/压痛）。",
	//	SuspectedDiseases: []string{"毛囊根鞘瘤", "皮脂腺囊肿", "脂肪瘤"},
	//	Status:            "in_diagnosis_loop",
	//}
	//
	//session, _ := a.App.GetShortMemory(req.ConversationId)
	//session.FlowContext = &powerai.FlowContext{
	//	CurrentAgentKey: currentAgentKey, // 模拟当前活跃Agent
	//	LastBotMessage:  "请问您提到的头皮上的包，是否伴有疼痛或压痛感？",
	//}
	//session.GlobalState = &powerai.GlobalState{
	//	Shared: &powerai.SharedEntities{
	//		SymptomSummary: "头皮肿物",
	//	},
	//	AgentSlots: map[string]interface{}{
	//		currentAgentKey: triageSlots,
	//	},
	//}
	//_ = a.App.SetShortMemory(req.ConversationId, session)
	//// --- [模拟数据结束] ---

	sessions, err := a.App.GetShortMemory(req.ConversationId)
	// 处理 Redis 读取结果
	if err == redis.Nil {
		// 情况 1: 没有会话历史 -> 这是一个新会话 -> 直接跳过 Layer 3，去 Layer 4
		sessions = &powerai.SessionValue{} // 空对象
	} else if err != nil {
		// 情况 2: Redis 报错
		xlog.LogErrorF(req.SysTrackCode, "send_msg", "redis get", "读取Session失败", err)
		RespJsonError(c, ErrorRedis, "记忆读取失败", req.SysTrackCode, nil)
		return nil
	}

	// 补全空指针，防止后续 Panic
	if sessions.GlobalState == nil {
		sessions.GlobalState = &powerai.GlobalState{}
	}
	if sessions.GlobalState.Shared == nil {
		sessions.GlobalState.Shared = &powerai.SharedEntities{}
	}
	if sessions.GlobalState.AgentSlots == nil {
		sessions.GlobalState.AgentSlots = make(map[string]interface{})
	}
	if sessions.FlowContext == nil {
		sessions.FlowContext = &powerai.FlowContext{}
	}
	return sessions

}

func (a *DeptAgent) GetHistoryDialogue(req *server.AgentRequest) (string, string, error) {
	// 1 准备历史对话
	messages, err := a.App.QueryMessageByConversationID(req.ConversationId)
	if err != nil {
		return "", "", err
	}

	// 按时间倒序扫描，找到第一个 end_flag=="true"  并取智能体的消息
	var filteredMsg []*powerai.AIMessage
	for _, msg := range messages {
		data := xjson.Get(msg.Answer.String, "data").String()
		endFlagStr := xjson.Get(data, "endflag").String()
		if endFlagStr == "true" {
			break
		}
		if msg.AgentCode.String == a.App.Manifest.Code {
			filteredMsg = append(filteredMsg, msg)
		}
	}

	// 历史会话组装
	var promptHistory strings.Builder
	var preConsultationDiag strings.Builder
	for i := len(filteredMsg) - 1; i >= 0; i-- {
		// 用户query
		userMessage := filteredMsg[i].Query.String

		// answers
		answers := xjson.Get(filteredMsg[i].Answer.String, "data").String()
		// msg
		agentQuestion := xjson.Get(answers, "msg").String()
		// answers
		agentAnswersRaw := xjson.Get(answers, "answers").Raw
		agentAnswersStr := strings.NewReplacer("\n", "", "\r", "", " ", "").Replace(agentAnswersRaw)
		// 提示词中，限制对话轮数标识
		round := len(filteredMsg) - i
		// 开始组装
		prompt_ := fmt.Sprintf(`
round = %d
用户：%s
你：{
		"question": ["%s"],
		"answers": %v,
		"msg": [""],
		"disease_list": [""]
	}`, round, userMessage, agentQuestion, agentAnswersStr)
		promptHistory.WriteString(prompt_)
		preConsultationDiagTmp := fmt.Sprintf(`
用户：%s
医生：%s
`, userMessage, agentQuestion)
		preConsultationDiag.WriteString(preConsultationDiagTmp)

	}

	return promptHistory.String(), preConsultationDiag.String(), nil
}

func LoadAgentSlots(slotsMap map[string]interface{}, agentKey string, target interface{}) error {
	// 1. 防御性检查
	if slotsMap == nil {
		return nil // map 为空，target 保持零值即可
	}

	// 2. 获取原始数据
	rawSlot, exists := slotsMap[agentKey]
	if !exists {
		return nil // key 不存在，target 保持零值即可
	}

	// 3. 序列化与反序列化 (Deep Copy & Type Conversion)
	// 这一步解决了 Redis 读取出来的 map[string]interface{} 无法直接强转为 Struct 的问题
	bytes, err := json.Marshal(rawSlot)
	if err != nil {
		return fmt.Errorf("failed to marshal raw slot for agent %s: %w", agentKey, err)
	}

	if err := json.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal into target struct for agent %s: %w", agentKey, err)
	}

	return nil
}
func (a *DeptAgent) RerankDeal(searchResult [][]milvus_mw.SearchResult, enterpriseId, query, matchField string) ([]milvus_mw.SearchResult, error) {
	// 0. 基础校验：如果没有检索结果，直接返回空
	if len(searchResult) == 0 || len(searchResult[0]) == 0 {
		return nil, nil
	}

	// 获取第一组搜索结果
	hits := searchResult[0]

	// 1. 准备重排序所需的文档列表 (Docs)
	docs := make([]string, 0, len(hits))
	for _, hit := range hits {
		// 从 Data map 中提取意图描述
		if desc, ok := hit.Data[matchField]; ok {
			docs = append(docs, desc)
		} else {
			// 如果数据缺失，补空字符串占位，保证 docs 和 hits 索引一一对应
			docs = append(docs, "")
		}
	}

	// 2. 调用重排序接口
	// RerankResults 通常返回 []float64
	newScores, err := a.App.RerankResults(enterpriseId, query, docs)
	if err != nil {
		return nil, fmt.Errorf("rerank failed: %w", err)
	}

	// 3. 更新分数
	for i := range hits {
		hits[i].Score = float32(newScores[i])
	}

	// 4. 执行排序 (降序：分数高的在前)
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})

	return hits, nil
}

// HasValidContent 检查切片中是否包含非空字符
func HasValidContent(list []string) bool {
	if len(list) == 0 {
		return false
	}
	for _, s := range list {
		// 去除首尾空格后，如果还有内容，则视为有效
		if strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}
func (a *DeptAgent) Updatememory(
	req *server.AgentRequest,
	session *powerai.SessionValue,
	aiMsg string, // AI 本次生成的文本回复 (用于历史记录和 Layer3 判断)
	mySlots *DeptSlots, // 当前最新的私有状态
) error {
	agentKey := a.App.Manifest.Code

	// 1. 同步私有状态
	if session.GlobalState.AgentSlots == nil {
		session.GlobalState.AgentSlots = make(map[string]interface{})
	}
	session.GlobalState.AgentSlots[agentKey] = mySlots

	// 2. 更新流程控制
	session.FlowContext.CurrentAgentKey = agentKey
	if aiMsg == "" {
		session.FlowContext.LastBotMessage = "[科室列表卡片]"
	} else {
		session.FlowContext.LastBotMessage = aiMsg
	}

	// ====================================================
	// 3. 持久化到 Redis (I/O 操作)
	// ====================================================
	err := a.App.SetShortMemory(req.ConversationId, session)
	if err != nil {
		return err
	}

	return nil
}

// 辅助函数：将字符串切片格式化为字符串，处理 nil 和空切片
func formatMedicalHistory(items []string) string {
	// Go 语言中，len(nil) 也是 0，所以这里同时覆盖了 nil 和 空切片 []string{} 的情况
	if len(items) == 0 {
		return "无" // 显式告诉大模型没有相关病史，防止幻觉
	}
	// 使用中文顿号或逗号连接，方便大模型阅读
	return strings.Join(items, "、")
}

func buildPromptRequest(prompt string) map[string]interface{} {

	requestLlm := map[string]interface{}{
		"messages": []interface{}{
			map[string]string{
				"role":    "system",
				"content": "You are a helpful assistant.",
			},
			map[string]string{
				"role":    "user",
				"content": prompt,
			},
		},
	}
	return requestLlm
}

// StrictMatchMapList
func StrictMatchMapList(targetName, matchField string, candidates []map[string]interface{}) []map[string]interface{} {

	var matchedDocs []map[string]interface{}

	// B. 遍历候选集进行比对
	for _, v := range candidates {
		// 安全断言，防止 panic
		name, ok := v[matchField].(string)
		if !ok {
			continue
		}
		if name == targetName {
			matchedDocs = append(matchedDocs, v)
		}
	}

	return matchedDocs
}
