package powerai

import (
	"encoding/json"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
)

// SessionValue 对应 Redis Value 的顶层结构
type SessionValue struct {
	Meta           *MetaInfo       `json:"meta"`
	FlowContext    *FlowContext    `json:"flow_context"`
	MessageContext *MessageContext `json:"message_context"`
	GlobalState    *GlobalState    `json:"global_state"`
	UserSnapshot   *UserProfile    `json:"user_snapshot"`
}

type MetaInfo struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	UpdatedAt      int64  `json:"updated_at"`
}

type FlowContext struct {
	CurrentAgentKey string `json:"current_agent_key"`
	LastBotMessage  string `json:"last_bot_message"`
	TurnCount       int    `json:"turn_count"`
}

type MessageContext struct {
	Summary        string     `json:"summary"`
	WindowMessages []*Message `json:"window_messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GlobalState 全局共享状态
type GlobalState struct {
	// ===============================
	// 1. 公共协议区 (Router 和 Supervisor 的决策依据)
	// ===============================
	// 这里的字段必须是全院通用的"官方语言"。
	// Router 只需要看这里，不需要去遍历 AgentSlots。
	Shared *SharedEntities `json:"shared"`

	// ===============================
	// 2. 智能体私有槽位 (Agent 独享的记忆)
	// ===============================
	// Key: AgentKey (如 "triage_agent", "report_agent")
	// Value: 对应 Agent 的专属结构体 (存入 Redis 时为 map[string]interface{})
	AgentSlots map[string]interface{} `json:"agent_slots,omitempty"`

	// ===============================
	// 3. 流程控制
	// ===============================
	CurrentIntent string         `json:"current_intent,omitempty"`
	PendingAction *PendingAction `json:"pending_action,omitempty"`
}

// SharedEntities 公共实体 (路由器的罗盘)
type SharedEntities struct {
	// 只有最核心、最通用的实体才放这里
	SymptomSummary string `json:"symptom_summary"`
	Disease        string `json:"disease,omitempty"`       // 疾病
	TargetDept     string `json:"target_dept,omitempty"`   // 目标科室
	TargetDoctor   string `json:"target_doctor,omitempty"` // 目标医生
	IntentTag      string `json:"intent_tag,omitempty"`    // 意图标签 (如 "book_ticket", "consult")
}

// PendingAction 挂起操作详情
type PendingAction struct {
	// ToolName: 准备调用的工具名
	// 示例: "create_payment_order"
	ToolName string `json:"tool_name"`

	// ToolParams: 准备好的参数
	// 示例: {"amount": 100, "bill_id": "123"}
	ToolParams map[string]interface{} `json:"tool_params"`

	// Reason: 挂起原因
	// 示例: "waiting_for_user_confirmation"
	Reason string `json:"reason"`
}
type UserProfile struct {
	// UserID: 用户唯一标识
	UserID string `json:"user_id"`

	// Name: 用户称呼
	// 示例: "李大爷"
	Name string `json:"name,omitempty"`

	// ===============================
	// 1. 安全红线数据 (必须注入 System Prompt)
	// ===============================

	// Allergies: 过敏史
	// 关键消费者: DrugAgent (开药禁忌), PreConsultAgent
	// 示例: ["青霉素", "芒果"]
	Allergies []string `json:"allergies,omitempty"`

	// ===============================
	// 2. 医疗背景数据 (辅助决策)
	// ===============================

	// ChronicDiseases: 慢病史
	// 关键消费者: TriageAgent (慢病开药通道匹配)
	// 示例: ["高血压", "2型糖尿病"]
	ChronicDiseases []string `json:"history,omitempty"` // 对应 JSON 中的 history 字段

	// SurgeryHistory: 手术史
	// 关键消费者: ReportAgent (影像解读参考), TriageAgent
	// 示例: ["2020年阑尾切除术"]
	SurgeryHistory []string `json:"surgery_history,omitempty"`

	// ===============================
	// 3. 偏好数据
	// ===============================

	// Preferences: 就医偏好
	// 示例: ["prefer_weekend", "expert_only"]
	Preferences []string `json:"preferences,omitempty"`
}

const ShortMemorySessionKeyPrefix = "short_term_memory:session:%s"
const expiration = 30 * 60

func (a *AgentApp) CreateShortMemory(req *server.AgentRequest) error {
	client, err := a.GetRedisClient()
	if err != nil {
		return err
	}
	key := fmt.Sprintf(ShortMemorySessionKeyPrefix, req.ConversationId)
	count, _ := client.Exists(key)
	if count > 0 {
		return nil
	}
	session := &SessionValue{
		Meta: &MetaInfo{
			ConversationID: req.ConversationId,
			UserID:         req.UserId,
			UpdatedAt:      0,
		},
		FlowContext: &FlowContext{
			CurrentAgentKey: "",
			LastBotMessage:  "",
			TurnCount:       0,
		},
		MessageContext: &MessageContext{
			Summary:        "",
			WindowMessages: nil,
		},
		GlobalState: &GlobalState{
			Shared:        nil,
			AgentSlots:    nil,
			CurrentIntent: "",
			PendingAction: nil,
		},
		UserSnapshot: &UserProfile{
			UserID:          req.UserId,
			Name:            "",
			Allergies:       nil,
			ChronicDiseases: nil,
			SurgeryHistory:  nil,
			Preferences:     nil,
		},
	}
	b, _ := json.Marshal(session)
	return client.Set(key, string(b), expiration)
}
func (a *AgentApp) GetShortMemory(conversationId string) (*SessionValue, error) {
	client, err := a.GetRedisClient()
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf(ShortMemorySessionKeyPrefix, conversationId)
	s, err := client.Get(key)
	if err != nil {
		return nil, err
	}
	session := &SessionValue{}
	err = json.Unmarshal([]byte(s), session)
	if err != nil {
		return nil, err
	}
	return session, nil
}
func (a *AgentApp) SetShortMemory(conversationId string, session *SessionValue) error {
	client, err := a.GetRedisClient()
	if err != nil {
		return err
	}
	key := fmt.Sprintf(ShortMemorySessionKeyPrefix, conversationId)
	b, _ := json.Marshal(session)
	return client.Set(key, string(b), expiration)
}
