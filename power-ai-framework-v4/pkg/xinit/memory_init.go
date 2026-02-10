package xinit

import (
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xconfig"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlock"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xmemory"
)

// ============================================================================
// 记忆管理初始化器
// ============================================================================

// MemoryInitResult 记忆管理初始化结果
type MemoryInitResult struct {
	Config        *xconfig.MemoryConfig
	LockManager   *xlock.SessionLockManager
	MessageBuilder *xmemory.MessageBuilder
	Error         error
}

// InitMemoryManager 初始化记忆管理所需的所有工具类
// 返回:
//   - *MemoryInitResult: 初始化结果
//
// 使用场景:
//   - 在 AgentApp 初始化时调用
//
// 初始化内容:
//   - 配置加载器
//   - 会话锁管理器
//   - 消息历史构建器
func InitMemoryManager() *MemoryInitResult {
	// 初始化配置
	config, err := xconfig.LoadConfig(xconfig.GetConfigPath())
	if err != nil {
		return &MemoryInitResult{
			Error: fmt.Errorf("failed to load config: %w", err),
		}
	}

	// 初始化锁管理器
	lockManager := xlock.NewSessionLockManager()

	// 初始化消息构建器
	messageBuilder := xmemory.NewMessageBuilder(
		config.EstimatedMessageChars,
		config.EstimatedWindowMessageChars,
	)

	return &MemoryInitResult{
		Config:        config,
		LockManager:   lockManager,
		MessageBuilder: messageBuilder,
		Error:         nil,
	}
}

// GetConfig 获取记忆管理配置
// 这是一个便捷方法，用于在代码中快速访问配置
//
// 返回:
//   - *xconfig.MemoryConfig: 配置实例
func GetConfig() *xconfig.MemoryConfig {
	return xconfig.GetConfig()
}

// GetLockManager 获取会话锁管理器
// 这是一个便捷方法，用于在代码中快速访问锁管理器
//
// 返回:
//   - *xlock.SessionLockManager: 锁管理器实例
func GetLockManager() *xlock.SessionLockManager {
	// 注意：这个函数需要在初始化后才能使用
	// 实际使用时，应该从 AgentApp 中获取
	return nil
}

// GetMessageBuilder 获取消息历史构建器
// 这是一个便捷方法，用于在代码中快速访问消息构建器
//
// 返回:
//   - *xmemory.MessageBuilder: 消息构建器实例
func GetMessageBuilder() *xmemory.MessageBuilder {
	// 注意：这个函数需要在初始化后才能使用
	// 实际使用时，应该从 AgentApp 中获取
	return nil
}
