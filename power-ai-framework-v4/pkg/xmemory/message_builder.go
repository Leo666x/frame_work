package xmemory

import (
	"strings"
)

// ============================================================================
// 消息历史构建器
// ============================================================================

// MessageBuilder 消息历史构建器
type MessageBuilder struct {
	estimatedMessageChars    int
	estimatedWindowMessageChars int
}

// NewMessageBuilder 创建消息历史构建器
func NewMessageBuilder(estimatedMessageChars, estimatedWindowMessageChars int) *MessageBuilder {
	return &MessageBuilder{
		estimatedMessageChars:     estimatedMessageChars,
		estimatedWindowMessageChars: estimatedWindowMessageChars,
	}
}

// AIMessage AI消息接口
type AIMessage interface {
	GetQuery() string
	GetAnswer() string
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// BuildHistoryFromMessages 从消息列表构建对话历史文本
// 参数:
//   - messages: AI消息列表
// 返回:
//   - string: 对话历史文本
//
// 格式:
//   用户: xxx
//   AI: xxx
func (b *MessageBuilder) BuildHistoryFromMessages(messages []AIMessage) string {
	if len(messages) == 0 {
		return ""
	}

	// 性能优化：预分配容量
	estimatedSize := len(messages) * b.estimatedMessageChars
	builder := strings.Builder{}
	builder.Grow(estimatedSize)

	for _, msg := range messages {
		if msg == nil {
			continue
		}

		userMessage := strings.TrimSpace(msg.GetQuery())
		agentMessage := strings.TrimSpace(msg.GetAnswer())

		// 添加用户消息
		if userMessage != "" {
			builder.WriteString("用户: ")
			builder.WriteString(userMessage)
			builder.WriteString("\n")
		}

		// 添加智能体消息
		if agentMessage != "" {
			builder.WriteString("AI: ")
			builder.WriteString(agentMessage)
			builder.WriteString("\n")
		}
	}

	return strings.TrimSpace(builder.String())
}

// ComposeSummaryAndRecent 组合摘要和最近消息
// 构建"历史摘要 + 最近N轮对话"的文本
//
// 参数:
//   - summary: 历史摘要
//   - messages: 最近N轮消息列表
// 返回:
//   - string: 组合后的文本
//
// 格式:
//   历史摘要: xxx
//   用户: xxx
//   AI: xxx
func (b *MessageBuilder) ComposeSummaryAndRecent(summary string, messages []*Message) string {
	// 性能优化：预分配容量
	estimatedSize := len(summary) + len(messages)*b.estimatedWindowMessageChars
	builder := strings.Builder{}
	builder.Grow(estimatedSize)

	// 添加摘要
	summary = strings.TrimSpace(summary)
	if summary != "" {
		builder.WriteString("历史摘要: ")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}

	// 添加最近N轮消息
	for _, msg := range messages {
		if msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}

		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "user" {
			builder.WriteString("用户: ")
		} else {
			builder.WriteString("AI: ")
		}
		builder.WriteString(strings.TrimSpace(msg.Content))
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

// BuildRecentMessages 从消息列表中提取最近N轮消息
// 参数:
//   - messages: AI消息列表
//   - recentTurns: 保留轮数
// 返回:
//   - []*Message: 最近N轮消息列表
func (b *MessageBuilder) BuildRecentMessages(messages []AIMessage, recentTurns int) []*Message {
	if len(messages) == 0 {
		return nil
	}

	start := len(messages) - recentTurns
	if start < 0 {
		start = 0
	}

	// 预分配容量
	recent := make([]*Message, 0, recentTurns*2)

	for _, msg := range messages[start:] {
		if msg == nil {
			continue
		}

		userMessage := strings.TrimSpace(msg.GetQuery())
		agentMessage := strings.TrimSpace(msg.GetAnswer())

		// 添加用户消息
		if userMessage != "" {
			recent = append(recent, &Message{
				Role:    "user",
				Content: userMessage,
			})
		}

		// 添加智能体消息
		if agentMessage != "" {
			recent = append(recent, &Message{
				Role:    "assistant",
				Content: agentMessage,
			})
		}
	}

	return recent
}

// EstimateTokenCount 估算文本的Token数量
// 参数:
//   - text: 文本内容
// 返回:
//   - int: Token数量
//
// 注意事项:
//   - 使用简化估算：字符数 / 4
//   - 适用于中文场景
//   - 如果文本为空，返回0
//   - 如果计算结果为0，返回1
func EstimateTokenCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	runeCount := len([]rune(text))
	tokens := runeCount / 4
	if tokens <= 0 {
		return 1
	}
	return tokens
}

// ExtractAgentAnswer 从智能体响应中提取纯文本答案
// 参数:
//   - answer: 智能体响应（可能是JSON格式）
// 返回:
//   - string: 纯文本答案
//
// 注意事项:
//   - 如果响应是JSON格式，提取 data.msg 字段
//   - 否则直接返回原始响应
func ExtractAgentAnswer(answer string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return ""
	}

	// 简单的JSON解析
	if strings.HasPrefix(answer, "{") && strings.HasSuffix(answer, "}") {
		// 尝试提取 data.msg 字段
		dataStart := strings.Index(answer, "\"data\"")
		if dataStart != -1 {
			msgStart := strings.Index(answer[dataStart:], "\"msg\"")
			if msgStart != -1 {
				msgStart = dataStart + msgStart + 6 // 跳过 "msg":
				// 查找第一个引号
				firstQuote := strings.Index(answer[msgStart:], "\"")
				if firstQuote != -1 {
					msgStart += firstQuote + 1
					// 查找第二个引号
					secondQuote := strings.Index(answer[msgStart:], "\"")
					if secondQuote != -1 {
						return strings.TrimSpace(answer[msgStart : msgStart+secondQuote])
					}
				}
			}
		}
	}

	// 不是JSON格式，直接返回
	return answer
}
