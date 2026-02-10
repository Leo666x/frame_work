package powerai

import (
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xdatetime"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xmemory"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xuid"
)

// ============================================================================
// 数据结构定义
// ============================================================================

// MemoryQueryRequest 记忆查询请求
// 用于查询会话的记忆上下文
type MemoryQueryRequest struct {
	ConversationID      string  // 会话ID（必填）
	EnterpriseID        string  // 企业ID
	PatientID           string  // 患者ID
	Query               string  // 当前用户查询（用于计算Token）
	TokenThresholdRatio float64 // Token阈值比例（默认0.75）
	RecentTurns         int     // 保留轮数（默认8）
	ModelContextWindow  int     // 模型上下文窗口（默认16000）
}

// MemoryContext 记忆上下文
// 返回给调用者的完整记忆上下文信息
type MemoryContext struct {
	ConversationID          string       // 会话ID
	Mode                    string       // 当前使用的模式: FULL_HISTORY / SUMMARY_N
	Session                 *SessionValue // 完整的会话状态
	History                 string       // 最终返回的对话历史（用于LLM）
	FullHistory             string       // 完整对话历史（用于摘要生成）
	EstimatedTokens         int          // 预估Token数量
	TokenRatio              float64      // Token占用比例
	ShouldCheckpointSummary bool         // 是否需要触发摘要
}

// MemoryWriteRequest 记忆写入请求
// 用于记录一次对话轮次
type MemoryWriteRequest struct {
	ConversationID string // 会话ID（必填）
	UserID         string // 用户ID
	AgentCode      string // 智能体代码
	UserQuery      string // 用户查询
	AgentResponse  string // 智能体响应
}

// MemoryWriteResult 记忆写入结果
// 返回写入操作的结果
type MemoryWriteResult struct {
	ConversationID string // 会话ID
	Mode           string // 当前记忆模式
	UpdatedAt      int64  // 更新时间戳
}

// SessionFinalizeRequest 会话结束请求
// 用于结束会话并创建最终Checkpoint
type SessionFinalizeRequest struct {
	ConversationID string // 会话ID
	Summary        string // 会话摘要
	RecentTurns    int    // 保留轮数
}

// MedicalFact 医疗事实
// 用于存储医疗相关的结构化信息（预留）
type MedicalFact struct {
	FactType   string  // 事实类型
	FactValue  string  // 事实值
	Confidence float64 // 置信度（0-1）
	Source     string  // 来源
}

// UserPreferenceMemory 用户偏好记忆
// 用于存储用户的偏好信息（预留）
type UserPreferenceMemory struct {
	Preference string // 偏好内容
	Source     string // 来源
}

// FactUpsertRequest 事实插入/更新请求
type FactUpsertRequest struct {
	ConversationID string        // 会话ID
	Facts          []*MedicalFact // 医疗事实列表
}

// PreferenceUpsertRequest 偏好插入/更新请求
type PreferenceUpsertRequest struct {
	ConversationID string                 // 会话ID
	Preferences    []*UserPreferenceMemory // 偏好列表
}

// ============================================================================
// 核心API函数
// ============================================================================

// QueryMemoryContext 查询记忆上下文
// 根据会话ID查询并构建适合当前对话的记忆上下文
//
// 参数:
//   - req: 记忆查询请求
// 返回:
//   - *MemoryContext: 记忆上下文
//   - error: 错误信息
//
// 使用场景:
//   - 每次处理用户消息前
//   - 需要获取对话历史时
//
// 工作流程:
//   1. 参数验证
//   2. 获取短期记忆（Redis）
//   3. 根据Checkpoint查询消息（PostgreSQL）
//   4. 构建对话历史
//   5. 计算Token占用率
//   6. 判断是否需要触发摘要
//
// 注意事项:
//   - 如果Redis读取失败，会创建默认会话状态并继续执行
//   - 如果数据库查询失败，会将messages设为nil并继续执行
//   - 这确保了系统的健壮性，不会因为单点故障导致整个流程中断
func (a *AgentApp) QueryMemoryContext(req *MemoryQueryRequest) (*MemoryContext, error) {
	// ===============================
	// 1. 参数验证
	// ===============================
	if req == nil {
		return nil, fmt.Errorf("memory query request is nil")
	}
	if req.ConversationID == "" {
		return nil, fmt.Errorf("conversation_id is empty")
	}

	// 应用默认值（使用配置）
	threshold := req.TokenThresholdRatio
	if threshold <= 0 {
		threshold = a.memoryConfig.TokenThresholdRatio
	}
	contextWindow := req.ModelContextWindow
	if contextWindow <= 0 {
		contextWindow = a.memoryConfig.ModelContextWindow
	}

	// ===============================
	// 2. 获取短期记忆（Redis）
	// ===============================
	session, err := a.GetShortMemory(req.ConversationID)
	if err != nil {
		// 降级处理：创建默认会话状态
		xlog.LogWarnF("MEMORY", "QueryMemoryContext", "GetShortMemory",
			fmt.Sprintf("failed to get short memory for conversation %s: %v, using default session", req.ConversationID, err))
		session = newDefaultSessionValue(a, req.ConversationID, req.PatientID)
	}

	mode := session.MessageContext.Mode

	// ===============================
	// 3. 根据Checkpoint查询当前段的消息（PostgreSQL）
	// ===============================
	var messages []*AIMessage
	if session.MessageContext.CheckpointMessageID != "" {
		// 从checkpoint之后查询
		messages, err = a.QueryMessageByConversationIDASCFromCheckpoint(req.ConversationID, session.MessageContext.CheckpointMessageID)
		if err != nil {
			xlog.LogWarnF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASCFromCheckpoint",
				fmt.Sprintf("failed to query messages from checkpoint %s: %v, falling back to full query", session.MessageContext.CheckpointMessageID, err))
			// 降级处理：查询全部消息
			messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
			if err != nil {
				xlog.LogWarnF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASC",
					fmt.Sprintf("failed to query all messages for conversation %s: %v", req.ConversationID, err))
				messages = nil
			}
		}
	} else {
		// 查询全部消息
		messages, err = a.QueryMessageByConversationIDASC(req.ConversationID)
		if err != nil {
			xlog.LogWarnF("MEMORY", "QueryMemoryContext", "QueryMessageByConversationIDASC",
				fmt.Sprintf("failed to query all messages for conversation %s: %v", req.ConversationID, err))
			messages = nil
		}
	}

	// ===============================
	// 4. 构建对话历史（使用工具类）
	// ===============================
	fullHistory := a.messageBuilder.BuildHistoryFromMessages(messages)

	// 先计算fullHistory的token占用率（使用工具类）
	estimatedTokens := xmemory.EstimateTokenCount(fullHistory + "\n" + req.Query)
	tokenRatio := float64(estimatedTokens) / float64(contextWindow)

	// 根据模式构建最终返回的History
	history := fullHistory
	if mode == a.memoryConfig.MemoryModeSummaryN {
		history = a.messageBuilder.ComposeSummaryAndRecent(session.MessageContext.Summary, session.MessageContext.WindowMessages)
		if history == "" {
			// 摘要为空，降级到全历史模式
			history = fullHistory
			mode = a.memoryConfig.MemoryModeFullHistory
		}
	}

	// 重新计算最终History的token占用率
	estimatedTokens = xmemory.EstimateTokenCount(history + "\n" + req.Query)
	tokenRatio = float64(estimatedTokens) / float64(contextWindow)

	// 无论什么模式，只要token超过阈值就触发摘要
	shouldCheckpoint := tokenRatio >= threshold

	return &MemoryContext{
		ConversationID:          req.ConversationID,
		Mode:                    mode,
		Session:                 session,
		History:                 history,
		FullHistory:             fullHistory,
		EstimatedTokens:         estimatedTokens,
		TokenRatio:              tokenRatio,
		ShouldCheckpointSummary: shouldCheckpoint,
	}, nil
}

// WriteTurn 写入对话轮次
// 记录一次对话轮次，更新短期记忆
//
// 参数:
//   - req: 记忆写入请求
// 返回:
//   - *MemoryWriteResult: 写入结果
//   - error: 错误信息
//
// 使用场景:
//   - 每次处理完用户消息后
//   - 记录对话轮次
//
// 工作流程:
//   1. 参数验证
//   2. 获取会话锁（防止并发冲突）
//   3. 获取并更新会话状态
//   4. 保存到Redis
//   5. 释放锁
//
// 注意事项:
//   - 使用会话级锁防止并发写入冲突
//   - 如果Redis读取失败，会创建默认会话状态
//   - TurnCount 会在锁保护下递增，确保计数准确
func (a *AgentApp) WriteTurn(req *MemoryWriteRequest) (*MemoryWriteResult, error) {
	// ===============================
	// 1. 参数验证
	// ===============================
	if req == nil {
		return nil, fmt.Errorf("memory write request is nil")
	}
	if req.ConversationID == "" {
		return nil, fmt.Errorf("conversation_id is empty")
	}

	// 验证输入（使用规范化器）
	if req.UserID != "" {
		if !a.sessionNormalizer.ValidateUserID(req.UserID) {
			return nil, fmt.Errorf("user_id too long or invalid format")
		}
	}
	if req.AgentCode != "" {
		if !a.sessionNormalizer.ValidateAgentCode(req.AgentCode) {
			return nil, fmt.Errorf("invalid agent_code format")
		}
	}
	if !a.sessionNormalizer.ValidateQueryLength(req.UserQuery) {
		return nil, fmt.Errorf("user_query too long")
	}
	if !a.sessionNormalizer.ValidateResponseLength(req.AgentResponse) {
		return nil, fmt.Errorf("agent_response too long")
	}

	// ===============================
	// 2. 获取会话锁（使用工具类）
	// ===============================
	lock := a.sessionLockMgr.GetLock(req.ConversationID)
	lock.Lock()
	defer lock.Unlock()

	// ===============================
	// 3. 获取并更新会话状态
	// ===============================
	session, err := a.GetShortMemory(req.ConversationID)
	if err != nil {
		// 降级处理：创建默认会话状态
		session = newDefaultSessionValue(a, req.ConversationID, req.UserID)
	}

	// 更新用户信息
	if req.UserID != "" {
		session.Meta.UserID = req.UserID
		// 防御性编程：确保 UserSnapshot 不为 nil
		if session.UserSnapshot != nil {
			session.UserSnapshot.UserID = req.UserID
		}
	}

	// 更新流程上下文
	if req.AgentCode != "" {
		session.FlowContext.CurrentAgentKey = req.AgentCode
	}
	if req.AgentResponse != "" {
		session.FlowContext.LastBotMessage = req.AgentResponse
	}

	// 增加对话轮次计数
	session.FlowContext.TurnCount++

	// ===============================
	// 4. 保存到Redis
	// ===============================
	if err := a.SetShortMemory(req.ConversationID, session); err != nil {
		xlog.LogErrorF("MEMORY", "WriteTurn", "SetShortMemory",
			fmt.Sprintf("failed to set short memory for conversation %s: %v", req.ConversationID, err))
		return nil, err
	}

	return &MemoryWriteResult{
		ConversationID: req.ConversationID,
		Mode:           session.MessageContext.Mode,
		UpdatedAt:      session.Meta.UpdatedAt,
	}, nil
}

// CheckpointShortMemory 创建Checkpoint（摘要）
// 将当前对话历史压缩为Checkpoint，包含摘要和最近N轮对话
//
// 参数:
//   - conversationID: 会话ID
//   - summary: 历史摘要文本
//   - recentTurns: 保留轮数（默认8）
// 返回:
//   - error: 错误信息
//
// 工作流程:
//   1. 参数验证
//   2. 获取会话锁（防止并发冲突）
//   3. 获取会话状态
//   4. 查询全部消息
//   5. 构建"摘要+最近N轮"内容
//   6. 插入Checkpoint消息到数据库（带重试机制）
//   7. 更新会话状态
//   8. 保存到Redis
//   9. 释放锁
//
// 注意事项:
//   - 使用会话级锁防止并发冲突
//   - 实现了重试机制，防止UUID重复
//   - Checkpoint消息存储在数据库中，不会随Redis过期
func (a *AgentApp) CheckpointShortMemory(conversationID, summary string, recentTurns int) error {
	// ===============================
	// 1. 参数验证
	// ===============================
	if conversationID == "" {
		return fmt.Errorf("conversation_id is empty")
	}
	if recentTurns <= 0 {
		recentTurns = a.memoryConfig.DefaultRecentTurns
	}

	// 验证摘要长度
	if !a.sessionNormalizer.ValidateSummaryLength(summary) {
		return fmt.Errorf("summary too long")
	}

	// ===============================
	// 2. 获取会话锁（使用工具类）
	// ===============================
	lock := a.sessionLockMgr.GetLock(conversationID)
	lock.Lock()
	defer lock.Unlock()

	// ===============================
	// 3. 获取会话状态
	// ===============================
	session, err := a.GetShortMemory(conversationID)
	if err != nil {
		session = newDefaultSessionValue(a, conversationID, "")
	}

	// ===============================
	// 4. 查询全部消息
	// ===============================
	messages, err := a.QueryMessageByConversationIDASC(conversationID)
	if err != nil {
		xlog.LogErrorF("MEMORY", "CheckpointShortMemory", "QueryMessageByConversationIDASC",
			fmt.Sprintf("failed to query messages for conversation %s: %v", conversationID, err))
		return err
	}

	// ===============================
	// 5. 构建"摘要+最近N轮"的内容（使用工具类）
	// ===============================
	recentMessages := a.messageBuilder.BuildRecentMessages(messages, recentTurns)
	summaryAndRecent := a.messageBuilder.ComposeSummaryAndRecent(summary, recentMessages)

	// ===============================
	// 6. 插入Checkpoint消息到数据库（带重试机制）
	// ===============================
	// 最多重试3次，防止UUID重复
	maxRetries := a.memoryConfig.CheckpointMaxRetries
	for i := 0; i < maxRetries; i++ {
		checkpointMessageID := xuid.UUID()
		timeNow := xdatetime.GetNowDateTime()

		// 检查message_id是否已存在
		exists, err := a.checkMessageIDExists(checkpointMessageID)
		if err != nil {
			xlog.LogErrorF("MEMORY", "CheckpointShortMemory", "checkMessageIDExists",
				fmt.Sprintf("failed to check message_id existence: %v", err))
			return fmt.Errorf("failed to check message_id existence: %w", err)
		}
		if exists {
			// UUID重复，重新生成
			continue
		}

		// 插入checkpoint消息到数据库
		sql := `INSERT INTO ai_message (message_id, conversation_id, query, answer, create_time, create_by, update_time, update_by) 
		        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = a.DBExec(sql, checkpointMessageID, conversationID, "[MEMORY_CHECKPOINT]", summaryAndRecent, timeNow, "system", timeNow, "system")

		if err != nil {
			// 检查是否是主键冲突（UUID重复）
			if a.sessionNormalizer.IsDuplicateKeyError(err) {
				xlog.LogWarnF("MEMORY", "CheckpointShortMemory", "DBExec",
					fmt.Sprintf("duplicate key error for message_id %s, retrying (%d/%d)", checkpointMessageID, i+1, maxRetries))
				continue // 重新生成ID重试
			}
			xlog.LogErrorF("MEMORY", "CheckpointShortMemory", "DBExec",
				fmt.Sprintf("failed to insert checkpoint message: %v", err))
			return fmt.Errorf("failed to insert checkpoint message: %w", err)
		}

		// 插入成功，跳出重试循环
		break
	}

	// ===============================
	// 7. 更新会话状态
	// ===============================
	session.MessageContext.Summary = summary
	session.MessageContext.WindowMessages = a.messageBuilder.BuildRecentMessages(messages, recentTurns)
	session.MessageContext.Mode = a.memoryConfig.MemoryModeSummaryN
	session.MessageContext.CheckpointMessageID = xuid.UUID()

	// ===============================
	// 8. 保存到Redis
	// ===============================
	if err := a.SetShortMemory(conversationID, session); err != nil {
		xlog.LogErrorF("MEMORY", "CheckpointShortMemory", "SetShortMemory",
			fmt.Sprintf("failed to set short memory for conversation %s: %v", conversationID, err))
		return err
	}

	return nil
}

// FinalizeSessionMemory 结束会话记忆
// 会话结束时创建最终Checkpoint
//
// 参数:
//   - req: 会话结束请求
// 返回:
//   - error: 错误信息
//
// 使用场景:
//   - 用户主动结束对话时
//   - 会话超时时
//
// 注意事项:
//   - 实际上调用 CheckpointShortMemory 实现逻辑
//   - 会话结束后，Redis中的SessionValue会随时间过期
//   - 但数据库中的Checkpoint消息会永久保存
func (a *AgentApp) FinalizeSessionMemory(req *SessionFinalizeRequest) error {
	if req == nil {
		return fmt.Errorf("session finalize request is nil")
	}
	return a.CheckpointShortMemory(req.ConversationID, req.Summary, req.RecentTurns)
}

// UpsertFacts 插入或更新医疗事实（预留接口）
// 用于存储医疗相关的结构化信息
//
// 参数:
//   - req: 事实插入/更新请求
// 返回:
//   - error: 错误信息
//
// 注意事项:
//   - 当前为空实现，预留接口供后续扩展
//   - 未来可以用于存储过敏史、病史等医疗事实
func (a *AgentApp) UpsertFacts(req *FactUpsertRequest) error {
	if req == nil {
		return fmt.Errorf("fact upsert request is nil")
	}
	// TODO: 实现医疗事实的插入/更新逻辑
	return nil
}

// UpsertPreferences 插入或更新用户偏好（预留接口）
// 用于存储用户的偏好信息
//
// 参数:
//   - req: 偏好插入/更新请求
// 返回:
//   - error: 错误信息
//
// 注意事项:
//   - 当前为空实现，预留接口供后续扩展
//   - 未来可以用于存储用户的就医偏好等
func (a *AgentApp) UpsertPreferences(req *PreferenceUpsertRequest) error {
	if req == nil {
		return fmt.Errorf("preference upsert request is nil")
	}
	// TODO: 实现用户偏好的插入/更新逻辑
	return nil
}

// ============================================================================
// 安全性验证函数
// ============================================================================

// checkMessageIDExists 检查message_id是否已存在
// 用于防止UUID重复导致的插入失败
//
// 参数:
//   - messageID: 消息ID
// 返回:
//   - bool: 是否存在
//   - error: 错误信息
func (a *AgentApp) checkMessageIDExists(messageID string) (bool, error) {
	sql := `SELECT COUNT(*) FROM ai_message WHERE message_id = $1`
	var count int
	err := a.DBQuerySingle(&count, sql, messageID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
