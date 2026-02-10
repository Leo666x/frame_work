package xconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ============================================================================
// Memory Config
// ============================================================================

// MemoryConfig 记忆管理配置
type MemoryConfig struct {
	// Token 相关配置
	TokenThresholdRatio float64 `yaml:"token_threshold_ratio"`
	DefaultRecentTurns  int      `yaml:"default_recent_turns"`
	DefaultModelContextWindow int `yaml:"default_model_context_window"`

	// 输入验证配置
	MaxQueryLength    int `yaml:"max_query_length"`
	MaxResponseLength int `yaml:"max_response_length"`
	MaxUserIDLength   int `yaml:"max_user_id_length"`
	MaxAgentCodeLength int `yaml:"max_agent_code_length"`
	MaxSummaryLength  int `yaml:"max_summary_length"`

	// Redis 配置
	RedisKeyPrefix   string `yaml:"redis_key_prefix"`
	RedisExpiration  int    `yaml:"redis_expiration"`

	// Checkpoint 配置
	CheckpointMaxRetries int `yaml:"checkpoint_max_retries"`

	// 性能优化配置
	EstimatedMessageChars    int `yaml:"estimated_message_chars"`
	EstimatedWindowMessageChars int `yaml:"estimated_window_message_chars"`

	// 记忆模式配置
	MemoryModeFullHistory string `yaml:"memory_mode_full_history"`
	MemoryModeSummaryN    string `yaml:"memory_mode_summary_n"`

	// 日志配置
	EnableVerboseLogging bool   `yaml:"enable_verbose_logging"`
	LogLevel             string `yaml:"log_level"`
}

var (
	// 全局配置实例
	config     *MemoryConfig
	configOnce sync.Once
)

// LoadConfig 加载配置文件
// 参数:
//   - configPath: 配置文件路径
// 返回:
//   - *MemoryConfig: 配置实例
//   - error: 错误信息
func LoadConfig(configPath string) (*MemoryConfig, error) {
	configOnce.Do(func() {
		// 如果配置文件路径为空，使用默认路径
		if configPath == "" {
			configPath = "config/memory_config.yaml"
		}

		// 读取配置文件
		data, err := os.ReadFile(configPath)
		if err != nil {
			// 如果读取失败，使用默认配置
			config = getDefaultConfig()
			return
		}

		// 解析 YAML
		cfg := &MemoryConfig{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			// 如果解析失败，使用默认配置
			config = getDefaultConfig()
			return
		}

		// 验证配置
		if err := validateConfig(cfg); err != nil {
			// 如果验证失败，使用默认配置
			config = getDefaultConfig()
			return
		}

		config = cfg
	})

	return config, nil
}

// GetConfig 获取配置实例
// 返回:
//   - *MemoryConfig: 配置实例
func GetConfig() *MemoryConfig {
	if config == nil {
		// 如果配置未加载，使用默认配置
		config = getDefaultConfig()
	}
	return config
}

// ReloadConfig 重新加载配置
// 参数:
//   - configPath: 配置文件路径
// 返回:
//   - error: 错误信息
func ReloadConfig(configPath string) error {
	configOnce = sync.Once{}
	config = nil
	_, err := LoadConfig(configPath)
	return err
}

// getDefaultConfig 获取默认配置
// 返回:
//   - *MemoryConfig: 默认配置实例
func getDefaultConfig() *MemoryConfig {
	return &MemoryConfig{
		TokenThresholdRatio: 0.75,
		DefaultRecentTurns:  8,
		DefaultModelContextWindow: 16000,

		MaxQueryLength:    10000,
		MaxResponseLength: 50000,
		MaxUserIDLength:   100,
		MaxAgentCodeLength: 50,
		MaxSummaryLength:  2000,

		RedisKeyPrefix:  "short_term_memory:session:%s",
		RedisExpiration:  1800,

		CheckpointMaxRetries: 3,

		EstimatedMessageChars:    200,
		EstimatedWindowMessageChars: 100,

		MemoryModeFullHistory: "FULL_HISTORY",
		MemoryModeSummaryN:    "SUMMARY_N",

		EnableVerboseLogging: false,
		LogLevel:             "info",
	}
}

// validateConfig 验证配置
// 参数:
//   - cfg: 配置实例
// 返回:
//   - error: 错误信息
func validateConfig(cfg *MemoryConfig) error {
	// 验证 Token 阈值比例
	if cfg.TokenThresholdRatio <= 0 || cfg.TokenThresholdRatio > 1 {
		return fmt.Errorf("token_threshold_ratio must be between 0 and 1")
	}

	// 验证保留轮数
	if cfg.DefaultRecentTurns <= 0 || cfg.DefaultRecentTurns > 20 {
		return fmt.Errorf("default_recent_turns must be between 1 and 20")
	}

	// 验证模型上下文窗口
	if cfg.DefaultModelContextWindow <= 0 {
		return fmt.Errorf("default_model_context_window must be greater than 0")
	}

	// 验证输入长度限制
	if cfg.MaxQueryLength <= 0 {
		return fmt.Errorf("max_query_length must be greater than 0")
	}
	if cfg.MaxResponseLength <= 0 {
		return fmt.Errorf("max_response_length must be greater than 0")
	}
	if cfg.MaxUserIDLength <= 0 {
		return fmt.Errorf("max_user_id_length must be greater than 0")
	}
	if cfg.MaxAgentCodeLength <= 0 {
		return fmt.Errorf("max_agent_code_length must be greater than 0")
	}
	if cfg.MaxSummaryLength <= 0 {
		return fmt.Errorf("max_summary_length must be greater than 0")
	}

	// 验证 Redis 过期时间
	if cfg.RedisExpiration <= 0 {
		return fmt.Errorf("redis_expiration must be greater than 0")
	}

	// 验证 Checkpoint 重试次数
	if cfg.CheckpointMaxRetries <= 0 {
		return fmt.Errorf("checkpoint_max_retries must be greater than 0")
	}

	// 验证记忆模式
	if cfg.MemoryModeFullHistory == "" {
		return fmt.Errorf("memory_mode_full_history cannot be empty")
	}
	if cfg.MemoryModeSummaryN == "" {
		return fmt.Errorf("memory_mode_summary_n cannot be empty")
	}

	return nil
}

// GetConfigPath 获取配置文件路径
// 返回:
//   - string: 配置文件路径
func GetConfigPath() string {
	// 优先使用环境变量指定的配置文件
	if envPath := os.Getenv("MEMORY_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// 否则使用默认路径
	return filepath.Join("config", "memory_config.yaml")
}
