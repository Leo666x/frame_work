package xdefense

import (
	"strings"
	"time"
)

// ============================================================================
// 会话状态规范化
// ============================================================================

// SessionNormalizer 会话状态规范化器
type SessionNormalizer struct {
	defaultMode string
}

// NewSessionNormalizer 创建会话状态规范化器
func NewSessionNormalizer(defaultMode string) *SessionNormalizer {
	return &SessionNormalizer{
		defaultMode: defaultMode,
	}
}

// NormalizeString 规范化字符串
// 确保字符串不为空，如果为空则返回默认值
func (n *SessionNormalizer) NormalizeString(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

// NormalizeStringSlice 规范化字符串切片
// 确保切片不为 nil，如果为 nil 则返回空切片
func (n *SessionNormalizer) NormalizeStringSlice(slice []string) []string {
	if slice == nil {
		return []string{}
	}
	return slice
}

// NormalizeInt 规范化整数
// 确保整数不小于最小值
func (n *SessionNormalizer) NormalizeInt(value, minValue int) int {
	if value < minValue {
		return minValue
	}
	return value
}

// NormalizeTimestamp 规范化时间戳
// 确保时间戳有效，如果无效则返回当前时间
func (n *SessionNormalizer) NormalizeTimestamp(timestamp int64) int64 {
	if timestamp <= 0 {
		return time.Now().Unix()
	}
	return timestamp
}

// ValidateLength 验证字符串长度
// 检查字符串长度是否在允许范围内
func (n *SessionNormalizer) ValidateLength(value string, maxLength int) bool {
	if value == "" {
		return true
	}
	return len(value) <= maxLength
}

// ValidateAgentCode 验证智能体代码格式
// 确保智能体代码只包含允许的字符
func (n *SessionNormalizer) ValidateAgentCode(code string) bool {
	if code == "" {
		return false
	}

	for _, c := range code {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// ValidateUUID 验证UUID格式
// 简单验证UUID的格式是否正确
func (n *SessionNormalizer) ValidateUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	// TODO: 添加更严格的UUID格式验证
	return true
}

// IsDuplicateKeyError 判断是否是主键冲突错误
// 用于识别UUID重复导致的数据库错误
func (n *SessionNormalizer) IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "23505")
}

// SafeString 安全获取字符串
// 防止空指针异常
func (n *SessionNormalizer) SafeString(ptr *string, defaultValue string) string {
	if ptr == nil {
		return defaultValue
	}
	if strings.TrimSpace(*ptr) == "" {
		return defaultValue
	}
	return strings.TrimSpace(*ptr)
}

// SafeInt 安全获取整数
// 防止空指针异常
func (n *SessionNormalizer) SafeInt(ptr *int, defaultValue int) int {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// SafeBool 安全获取布尔值
// 防止空指针异常
func (n *SessionNormalizer) SafeBool(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}
