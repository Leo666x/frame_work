package decision

import (
	"encoding/json"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"strings"
)

type ResClassL3 struct {
	Action string `json:"action"`
}

// 判断是 "继续当前话题" 还是 "意图转移"
func (a *DecisionAgent) Layer3_ContextRouter(
	req *server.AgentRequest,
	agentName,
	description,
	msgHistory string,
) (string, error) {

	// prompt
	prompt := strings.NewReplacer(
		"AGENT_NAME", agentName,
		"AGENT_DESC", description,
		"MSG_HISTORY", msgHistory,
		"USER_QUERY", req.Query,
	).Replace(Layer3ContextRouterPrompt)
	respLlm, err := a.LlmCall(req, prompt)
	if err != nil {
		return "", fmt.Errorf("意图转移分析未成功：%w", err)
	}

	res := LlmRespDeal(respLlm)
	var resClass ResClassL3
	if err := json.Unmarshal([]byte(res), &resClass); err != nil {
		return "", err
	}

	return resClass.Action, nil
}

// 定义路由动作枚举
type RoutingAction string

const (
	// ACTION_CONTINUE: 意图未转移，继续由当前 Agent 处理
	// 场景: Agent问"哪里疼?", 用户回"肚子"
	ACTION_CONTINUE RoutingAction = "CONTINUE"

	// ACTION_INTERRUPT: 意图转移，中断当前流程，上报 Layer 4
	// 场景: Agent问"哪里疼?", 用户回"挂号费多少钱"
	ACTION_INTERRUPT RoutingAction = "INTERRUPT"

	// ACTION_REJECT: (可选) 拒识/无法判断，通常兜底策略也是上报
	ACTION_REJECT RoutingAction = "REJECT"
)

// ==========================================
// 入参结构体设计
// ==========================================

// ContextRouterRequest 封装所有路由判断所需的上下文
type ContextRouterRequest struct {
	// 1. 用户当前输入
	UserQuery string

	// 2. 静态上下文 (来自 t_sys_agent_registry)
	// 必须包含当前 Agent 的边界描述，让 Router 知道它负责什么
	CurrentAgentName string
	CurrentAgentDesc string // e.g. "负责询问患者症状、发病时间..."

	// 3. 动态上下文 (来自 Redis Session)
	// 建议包含最近 3 轮 (6条消息)
	// 格式: "AI: ...\nUser: ...\nAI: ..."
	RecentHistoryText string

	// 4. (可选) 槽位状态
	// 如果你知道 Agent 正在填哪个槽，可以提高准确率
	// e.g. "waiting_for_symptom"
	CurrentSlotState string
}

// ==========================================
// 出参结构体设计
// ==========================================

// ContextRouterResponse 路由判断结果
type ContextRouterResponse struct {
	Action RoutingAction // 继续 还是 中断

	// 理由 (用于日志审计或调试)
	// e.g. "User input matches the expected symptom description."
	Reason string

	// (高级选填) 修正后的 Input
	// 有时用户说 "不看了，换个科"，Router 可以提取出 "换个科" 传给下一层
	RefinedInput string
}
