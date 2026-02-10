package powerai

import (
	"encoding/json"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"time"
)

// ============================================================================
// 数据结构定义
// ============================================================================

// SessionValue 对应 Redis Value 的顶层结构
// 存储会话的完整状态信息，包括元数据、流程上下文、消息上下文、全局状态和用户快照
//
// 序列化格式: JSON
// 存储位置: Redis
// Key格式: short_term_memory:session:{conversation_id}
// 过期时间: 30分钟（1800秒）
type SessionValue struct {
	Meta           *MetaInfo       `json:"meta"`           // 元信息
	FlowContext    *FlowContext    `json:"flow_context"`   // 流程上下文
	MessageContext *MessageContext `json:"message_context"` // 消息上下文（核心）
	GlobalState    *GlobalState    `json:"global_state"`   // 全局共享状态
	UserSnapshot   *UserProfile    `json:"user_snapshot"`   // 用户快照
}

// MetaInfo 会话元信息
// 存储会话的基本信息和更新时间
type MetaInfo struct {
	ConversationID string `json:"conversation_id"` // 会话唯一标识
	UserID         string `json:"user_id"`         // 用户ID
	UpdatedAt      int64  `json:"updated_at"`      // 最后更新时间戳（Unix时间戳）
}

// FlowContext 流程上下文
// 记录对话流程的状态信息
type FlowContext struct {
	CurrentAgentKey string `json:"current_agent_key"` // 当前执行的智能体代码
	LastBotMessage  string `json:"last_bot_message"`  // 最后一条AI回复
	TurnCount       int    `json:"turn_count"`       // 对话轮次计数
}

// MessageContext 消息上下文（核心）
// 控制对话历史的返回方式和内容
type MessageContext struct {
	Summary             string     `json:"summary"`               // 历史摘要文本
	WindowMessages      []*Message `json:"window_messages"`        // 最近N轮消息窗口
	Mode                string     `json:"mode,omitempty"`         // 当前模式: FULL_HISTORY / SUMMARY_N
	CheckpointMessageID string     `json:"checkpoint_message_id,omitempty"` // 当前checkpoint的消息ID（分段查询的起始点）
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`    // "user" 或 "assistant"
	Content string `json:"content"` // 消息内容
}

// GlobalState 全局共享状态
// 支持智能体间协作和状态共享
type GlobalState struct {
	// ===============================
	// 1. 公共协议区 (Router 和 Supervisor 的决策依据)
	// ===============================
	// 这里的字段必须是全院通用的"官方语言"。
	// Router 只需要看这里，不需要去遍历 AgentSlots。
	Shared   *SharedEntities `json:"shared"`    // 共享实体（兼容旧版本）
	Entities *SharedEntities `json:"entities,omitempty"` // 共享实体（新版本）

	// ===============================
	// 2. 智能体私有槽位 (Agent 独享的记忆)
	// ===============================
	// Key: AgentKey (如 "triage_agent", "report_agent")
	// Value: 对应 Agent 的专属结构体 (存入 Redis 时为 map[string]interface{})
	//
	// 使用场景:
	//   - 每个智能体可以存储自己的私有状态
	//   - 不会与其他智能体冲突
	//
	// 示例:
	//   session.GlobalState.AgentSlots["triage_agent"] = map[string]interface{}{
	//       "triage_level": "moderate",
	//       "symptoms_collected": true,
	//   }
	AgentSlots map[string]interface{} `json:"agent_slots,omitempty"`

	// ===============================
	// 3. 流程控制
	// ===============================
	CurrentIntent string         `json:"current_intent,omitempty"` // 当前意图
	PendingAction *PendingAction `json:"pending_action,omitempty"` // 挂起操作
}

// SharedEntities 公共实体（路由器的罗盘）
// 存储最核心、最通用的实体信息
type SharedEntities struct {
	SymptomSummary string `json:"symptom_summary"` // 症状摘要
	Disease        string `json:"disease,omitempty"`       // 疾病
	TargetDept     string `json:"target_dept,omitempty"`   // 目标科室
	TargetDoctor   string `json:"target_doctor,omitempty"` // 目标医生
	IntentTag      string `json:"intent_tag,omitempty"`    // 意图标签（如 "book_ticket", "consult"）
}

// PendingAction 挂起操作详情
// 用于需要用户确认或等待条件的场景
type PendingAction struct {
	ToolName   string                 `json:"tool_name"`   // 工具名称（如 "create_payment_order"）
	ToolParams map[string]interface{} `json:"tool_params"` // 工具参数（如 {"amount": 100, "bill_id": "123"}）
	Reason     string                 `json:"reason"`     // 挂起原因（如 "waiting_for_user_confirmation"）
}

// UserProfile 用户快照
// 存储用户画像信息，用于个性化服务和安全检查
type UserProfile struct {
	UserID string `json:"user_id"` // 用户唯一标识
	Name   string `json:"name,omitempty"` // 用户称呼（如 "李大爷"）

	// ===============================
	// 1. 安全红线数据 (必须注入 System Prompt)
	// ===============================
	// 这些数据涉及用户安全，必须在 AI 响应时注入到 System Prompt 中
	//
	// Allergies: 过敏史
	// 关键消费者: DrugAgent (开药禁忌), PreConsultAgent
	// 示例: ["青霉素", "芒果"]
	Allergies []string `json:"allergies,omitempty"`

	// ===============================
	// 2. 医疗背景数据 (辅助决策)
	// ===============================
	// 这些数据帮助智能体做出更准确的决策
	//
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

// ============================================================================
// 核心函数
// ============================================================================

// newDefaultSessionValue 创建默认的会话状态
// 参数:
//   - a: AgentApp 实例（用于访问配置）
//   - conversationID: 会话ID
//   - userID: 用户ID
// 返回:
//   - *SessionValue: 默认会话状态
//
// 使用场景:
//   - 新会话创建时
//   - 会话读取失败时的降级处理
func newDefaultSessionValue(a *AgentApp, conversationID, userID string) *SessionValue {
	return &SessionValue{
		Meta: &MetaInfo{
			ConversationID: conversationID,
			UserID:         userID,
			UpdatedAt:      time.Now().Unix(),
		},
		FlowContext: &FlowContext{
			CurrentAgentKey: "",
			LastBotMessage:  "",
			TurnCount:       0,
		},
		MessageContext: &MessageContext{
			Summary:        "",
			WindowMessages: nil,
			Mode:           a.memoryConfig.MemoryModeFullHistory,
		},
		GlobalState: &GlobalState{
			Shared:        nil,
			Entities:      nil,
			AgentSlots:    nil,
			CurrentIntent: "",
			PendingAction: nil,
		},
		UserSnapshot: &UserProfile{
			UserID:          userID,
			Name:            "",
			Allergies:       nil,
			ChronicDiseases: nil,
			SurgeryHistory:  nil,
			Preferences:     nil,
		},
	}
}

// CreateShortMemory 创建短期记忆
// 在对话开始时调用，为会话初始化 Redis 存储
//
// 参数:
//   - req: Agent 请求信息
// 返回:
//   - error: 错误信息
//
// 使用场景:
//   - 用户发送第一条消息时
//   - 会话初始化时
//
// 注意事项:
//   - 如果会话已存在，则不重复创建
//   - 使用 Redis 的 EXISTS 命令检查是否已存在
func (a *AgentApp) CreateShortMemory(req *server.AgentRequest) error {
	// 获取 Redis 客户端
	client, err := a.GetRedisClient()
	if err != nil {
		return fmt.Errorf("failed to get redis client: %w", err)
	}

	// 生成 Redis Key
	key := fmt.Sprintf(a.memoryConfig.RedisKeyPrefix, req.ConversationId)

	// 检查会话是否已存在
	count, _ := client.Exists(key)
	if count > 0 {
		// 会话已存在，无需重复创建
		return nil
	}

	// 创建新的会话状态
	session := newDefaultSessionValue(a, req.ConversationId, req.UserId)

	// 序列化为 JSON
	b, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// 存储到 Redis
	return client.Set(key, string(b), a.memoryConfig.RedisExpiration)
}

// GetShortMemory 获取短期记忆
// 从 Redis 读取会话状态
//
// 参数:
//   - conversationId: 会话ID
// 返回:
//   - *SessionValue: 会话状态
//   - error: 错误信息
//
// 使用场景:
//   - 处理用户消息前获取上下文
//   - 查询会话状态
//
// 注意事项:
//   - 返回的会话状态已经过规范化处理
//   - 所有嵌套指针都保证不为 nil
func (a *AgentApp) GetShortMemory(conversationId string) (*SessionValue, error) {
	// 获取 Redis 客户端
	client, err := a.GetRedisClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis client: %w", err)
	}

	// 生成 Redis Key
	key := fmt.Sprintf(a.memoryConfig.RedisKeyPrefix, conversationId)

	// 从 Redis 读取
	s, err := client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session from redis: %w", err)
	}

	// 反序列化 JSON
	session := &SessionValue{}
	err = json.Unmarshal([]byte(s), session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// 规范化会话状态（使用工具类）
	normalizedSession := a.sessionNormalizer.Normalize(session)
	return normalizedSession, nil
}

// SetShortMemory 设置短期记忆
// 将会话状态保存到 Redis
//
// 参数:
//   - conversationId: 会话ID
//   - session: 会话状态
// 返回:
//   - error: 错误信息
//
// 使用场景:
//   - 更新会话状态后
//   - 写入对话轮次后
//   - 创建 checkpoint 后
//
// 注意事项:
//   - 会话状态会先进行规范化处理
//   - 自动更新 UpdatedAt 时间戳
//   - 自动刷新过期时间
func (a *AgentApp) SetShortMemory(conversationId string, session *SessionValue) error {
	// 获取 Redis 客户端
	client, err := a.GetRedisClient()
	if err != nil {
		return fmt.Errorf("failed to get redis client: %w", err)
	}

	// 生成 Redis Key
	key := fmt.Sprintf(a.memoryConfig.RedisKeyPrefix, conversationId)

	// 规范化会话状态（使用工具类）
	session = a.sessionNormalizer.Normalize(session)

	// 更新元信息
	session.Meta.ConversationID = conversationId
	session.Meta.UpdatedAt = time.Now().Unix()

	// 序列化为 JSON
	b, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// 存储到 Redis
	return client.Set(key, string(b), a.memoryConfig.RedisExpiration)
}
