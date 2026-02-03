package decision

import (
	"database/sql"
	"regexp"
)

const (
	ErrorCallLlm         = "cl-err"       //备注: 调用大模型错误
	ErrorRedis           = "cl-redis-err" //备注: 调用大模型错误
	ErrorPsql            = "psql-err"     //备注: 调用大模型错误
	RAG_TopK             = 3
	ThresholdScore       = 0.85
	ThresholdScoreDiff   = 0.15
	DecisionAgentUnknown = "agent-unknown"
)

// DBModel: 对应数据库表结构
type FastRuleDBModel struct {
	// 将 gorm 改为 db 标签，以便 sqlx 进行字段映射
	ID             string `db:"id"`
	MatchType      string `db:"match_type"`
	Pattern        string `db:"pattern"`
	ConditionParam string `db:"condition_param"`
	ActionType     string `db:"action_type"`
	ActionContent  string `db:"action_content"`
	Priority       int    `db:"priority"`
}

// CachedRule: 内存中使用的规则结构 (优化性能)
type CachedRule struct {
	Original        FastRuleDBModel
	CompiledRegex   *regexp.Regexp // 预编译的正则对象，如果是keyword则为nil
	MaxLenCondition int            // 解析后的长度限制，-1表示无限制
}

// 定义 Layer 1 的返回结果
type Layer1Result struct {
	Hit        bool   // 是否命中
	ActionType string // TEXT_REPLY / TOOL_CALL
	Content    string // 具体内容
}

// Session 代表一次活跃的会话上下文
// 对应 Redis Key: memory:state:{conversation_id}
type Session struct {
	// ===========================
	// 1. 基础标识 (Identity)
	// ===========================
	ConversationID string `json:"conversation_id"` // 会话ID
	UserID         string `json:"user_id"`         // 用户ID
	Channel        string `json:"channel"`         // 渠道 (如 wechat, web)

	// ===========================
	// 2. 流程控制状态 (Flow Control)
	// ===========================
	// CurrentAgentKey: 当前正在掌管对话的智能体 Key (如 "triage_agent")
	// 如果为空，说明当前是闲置状态，由 Supervisor 接管
	CurrentAgentKey string `json:"current_agent_key"`

	// LastBotMessage: 上一轮 AI 说的话
	// 用途: Layer 3 判断意图转移。例如 AI 问"哪里疼?", 用户回"头". Router 对比后判断这是 In-Context 回复。
	LastBotMessage string `json:"last_bot_message"`

	// TurnCount: 当前 Agent 连续对话的轮数
	// 用途: 防止死循环，或者在 N 轮后强制进行 Summary
	TurnCount int `json:"turn_count"`

	// LastActiveTime: 最后活跃时间 (用于 Session 超时重置)
	LastActiveTime int64 `json:"last_active_time"`

	// ===========================
	// 3. 共享数据黑板 (The Blackboard)
	// ===========================
	// 用于跨 Agent 传递结构化数据，解决 "记忆衔接" 问题
	State *GlobalState `json:"state"`

	// ===========================
	// 4. 长期画像 (Long-term Memory)
	// ===========================
	// 从 t_user_profile_memory 表中提取的高优先级信息
	// 用途: 注入到 System Prompt 中 (如过敏史)
	UserProfile *UserProfile `json:"user_profile"`
}

// GlobalState 全局共享状态槽位 (结构化病历)
type GlobalState struct {
	// 核心意图槽位
	Disease      string `json:"disease,omitempty"`       // 识别到的疾病 (如 "帕金森")
	Symptom      string `json:"symptom,omitempty"`       // 症状摘要 (如 "头痛3天")
	TargetDept   string `json:"target_dept,omitempty"`   // 目标科室 (如 "神经内科")
	TargetDoctor string `json:"target_doctor,omitempty"` // 目标医生 (如 "张三")

	// 业务流程槽位
	ServiceType string `json:"service_type,omitempty"` // 服务类型 (如 "无痛胃镜")
	ReportData  string `json:"report_data,omitempty"`  // 报告解读结论 (如 "白细胞高")

	// 临时挂起任务 (用于中断恢复)
	PendingAction *PendingAction `json:"pending_action,omitempty"`
}

// PendingAction 挂起的操作 (用于 Checkpoint/Resume)
type PendingAction struct {
	ToolName    string                 `json:"tool_name"`    // 此时正在调用的工具
	ToolParams  map[string]interface{} `json:"tool_params"`  // 参数
	NeedConfirm bool                   `json:"need_confirm"` // 是否需要用户确认 (如支付)
}

// UserProfile 用户画像快照 (只读，注入 Prompt 用)
type UserProfile struct {
	Allergies       []string `json:"allergies"`        // 过敏源 (最高优先级)
	ChronicDiseases []string `json:"chronic_diseases"` // 慢病
	SurgeryHistory  []string `json:"surgery_history"`  // 手术史
}

// AgentRegistryModel 对应数据库表 ai_agent_registry
type AgentRegistryModel struct {
	ID          sql.NullString `db:"id"`            // 对应 id
	AgentKey    sql.NullString `db:"agent_key"`     // 对应 agent_key
	DomainID    sql.NullString `db:"domain_id"`     // 对应 domain_id
	AgentName   sql.NullString `db:"agent_name"`    // 对应 agent_name
	Description sql.NullString `db:"description"`   // 对应 description
	MCPToolName sql.NullString `db:"mcp_tool_name"` // 对应 mcp_tool_name
	IsActive    sql.NullString `db:"is_active"`     // 对应 is_active
}

// 定义大模型返回结构体
type RespLlm struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type ResTargetAgent struct {
	TargetAgent string `json:"target_agent"`
}
