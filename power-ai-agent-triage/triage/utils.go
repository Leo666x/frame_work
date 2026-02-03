package triage

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
	"github.com/mitchellh/mapstructure"
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
func (a *TriageAgent) InitReportConfig(rsp *server.AgentResponse, event *server.SSEEvent, StrConfig map[string]interface{}) {
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

func (a *TriageAgent) LlmCall(req *server.AgentRequest, prompt string) (string, error) {

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

func (a *TriageAgent) GetRedis(c *gin.Context, req *server.AgentRequest) *powerai.SessionValue {

	//// --- [模拟数据开始] ---
	//// 仅用于测试 Layer 3 逻辑：如果 Redis 里没数据，强制写入一个模拟的“上一轮状态”
	//// 场景：上一轮是报告解读智能体，刚说完“白细胞高”
	//currentAgentKey := "power-ai-agent-triage"
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

func (a *TriageAgent) GetHistoryDialogue(req *server.AgentRequest) (string, string, error) {
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

func LoadAgentSlots(agentSlots map[string]interface{}, agentCode string, triageSlots *TriageSlots) error {

	if err := mapstructure.Decode(agentSlots[agentCode], triageSlots); err != nil {
		return err
	}
	return nil
}
func (a *TriageAgent) RerankDeal(searchResult [][]milvus_mw.SearchResult, enterpriseId, query, matchField string) ([]milvus_mw.SearchResult, error) {
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

func (a *TriageAgent) Updatememory(req *server.AgentRequest, resp *server.AgentResponse, session *powerai.SessionValue, aiMsg, msgHistory string, mySlots *TriageSlots) error {

	msgHistory = fmt.Sprintf("%s\nuser:%s\nai:%s", msgHistory, req.Query, aiMsg)
	prompt := strings.NewReplacer("DIALOGUE_HISTORY", msgHistory).Replace(SymptomMemoryPrompt)
	resLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return err
	}
	respStr := LlmRespDeal(resLlm)
	var data ExtractedData
	if err := json.Unmarshal([]byte(respStr), &data); err != nil {
		return err
	}
	err = MergeExtractedDataToSession(session, &data, a.App.Manifest.Code, aiMsg, mySlots)
	if err != nil {
		return err
	}
	err = a.App.SetShortMemory(req.ConversationId, session)
	if err != nil {
		return err
	}
	return nil

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

// 辅助函数：将字符串切片格式化为字符串，处理 nil 和空切片
func formatMedicalHistory(items []string) string {
	// Go 语言中，len(nil) 也是 0，所以这里同时覆盖了 nil 和 空切片 []string{} 的情况
	if len(items) == 0 {
		return "无" // 显式告诉大模型没有相关病史，防止幻觉
	}
	// 使用中文顿号或逗号连接，方便大模型阅读
	return strings.Join(items, "、")
}

// MergeExtractedDataToSession 将提取的数据合并入 Session
// currentAgentKey 通常为 "power-ai-agent-triage"
func MergeExtractedDataToSession(sess *powerai.SessionValue, data *ExtractedData, currentAgentKey, aiMsg string, mySlots *TriageSlots) error {

	// ====================================================
	// 1. 处理私有记忆 (TriageSlots)
	// ====================================================

	// a. 更新症状属性 (Map Merge)
	// 将新提取的属性合并进去，覆盖旧的
	if mySlots.SymptomAttributes == nil {
		mySlots.SymptomAttributes = make(map[string]string)
	}
	for k, v := range data.SymptomAttributes {
		// 过滤无效值
		if v != "" && v != "null" && v != "未提及" {
			mySlots.SymptomAttributes[k] = v
		}
	}

	// C. 更新疑似疾病列表
	if len(data.DiagnosisResult.Disease) > 0 {
		mySlots.SuspectedDiseases = data.DiagnosisResult.Disease
	}

	// D. 更新历史摘要
	if data.DiagnosisResult.SymptomSummary != "" {
		mySlots.HistorySummary = data.DiagnosisResult.SymptomSummary
	}

	// E. 回写私有状态到 GlobalState
	sess.GlobalState.AgentSlots[currentAgentKey] = mySlots

	// ====================================================
	// 2. 处理公共记忆 (GlobalState.Shared)
	// ====================================================

	// 无论是否确诊，都更新摘要，方便 Router 查看
	if data.DiagnosisResult.SymptomSummary != "" {
		sess.GlobalState.Shared.SymptomSummary = data.DiagnosisResult.SymptomSummary
	}
	sess.FlowContext.CurrentAgentKey = currentAgentKey
	sess.FlowContext.LastBotMessage = aiMsg
	// 如果确诊了，更新 Disease 和 TargetDept
	if data.DiagnosisResult.IsDiagnosed {
		// 填入疾病
		if len(data.DiagnosisResult.Disease) > 0 {
			// 简单处理：取第一个，或者逗号拼接
			sess.GlobalState.Shared.Disease = data.DiagnosisResult.Disease[0]
		}

		// 此时通常会触发 TriageAgent 内部的逻辑去查科室，查完后填入
		// sess.GlobalState.Shared.TargetDept = "..." (由后续逻辑填充)

	}

	return nil
}
