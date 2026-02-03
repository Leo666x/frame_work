package decision

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"
)

// 快速关键词/正则匹配
// Layer1_FastPatternMatch 极速匹配
func (a *DecisionAgent) Layer1_FastPatternMatch(input string) *Layer1Result {
	// 0. 基础清洗
	input = strings.TrimSpace(input)
	if input == "" {
		return &Layer1Result{Hit: false}
	}

	// 获取当前 input 的字符数 (注意：Go 的 len() 是字节数，中文要用 RuneCount)
	inputLen := utf8.RuneCountInString(input)

	err := a.LoadRules()
	if err != nil {

	}
	// 1. 遍历内存中的规则 (已有序)
	for _, rule := range memoryRules {

		// --- 条件检查 1: 长度限制 ---
		// 如果配置了 max_len 且当前输入超长，直接跳过该规则
		if rule.MaxLenCondition > 0 && inputLen > rule.MaxLenCondition {
			continue
		}

		matched := false

		// --- 核心检查: 模式匹配 ---
		switch rule.Original.MatchType {
		case "regex":
			// 使用预编译好的正则对象
			if rule.CompiledRegex != nil && rule.CompiledRegex.MatchString(input) {
				matched = true
			}
		case "keyword":
			// 简单包含匹配
			// 进阶优化：支持 "停车场|车位" 这种多关键词写法
			keywords := strings.Split(rule.Original.Pattern, "|")
			for _, kw := range keywords {
				if input == kw {
					matched = true
					break
				}
			}
		}

		// --- 命中处理 ---
		if matched {
			// 返回命中的结果
			return &Layer1Result{
				Hit:        true,
				ActionType: rule.Original.ActionType,
				Content:    rule.Original.ActionContent,
			}
		}
	}

	// 所有规则都未命中
	return &Layer1Result{Hit: false}
}

// 全局缓存变量
var memoryRules []*CachedRule

// LoadRules 从数据库加载并预处理
func (a *DecisionAgent) LoadRules() error {
	var dbRules []FastRuleDBModel

	// 1. 查库
	// sqlx 的 Select 方法需要完整的 SQL 语句
	// 假设表名为 ai_fast_match_rule
	sqlQuery := `
        SELECT id, match_type, pattern, condition_param, action_type, action_content, priority 
        FROM ai_fast_match_rule 
        ORDER BY priority DESC
    `

	// 调用您封装的 DBQueryMultiple
	if err := a.App.DBQueryMultiple(&dbRules, sqlQuery); err != nil {
		return err
	}

	var tempRules []*CachedRule

	// 2. 遍历并预处理 (逻辑保持不变)
	for _, r := range dbRules {
		cached := &CachedRule{
			Original:        r,
			MaxLenCondition: -1, // 默认无限制
		}

		// A. 解析额外条件 (JSON -> Int)
		if r.ConditionParam != "" && r.ConditionParam != "{}" {
			var params map[string]interface{}
			// 忽略 JSON 解析错误，避免单条数据错误影响整体加载
			if err := json.Unmarshal([]byte(r.ConditionParam), &params); err == nil {
				if val, ok := params["max_len"]; ok {
					// 注意 JSON 数字转 interface{} 通常是 float64
					cached.MaxLenCondition = int(val.(float64))
				}
			}
		}

		// B. 预编译正则 (如果是 REGEX 类型)
		if r.MatchType == "regex" {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				// 建议记录日志，告知哪条规则有问题
				// log.Printf("规则 ID %a 正则编译失败: %v, 跳过", r.ID, err)
				continue
			}
			cached.CompiledRegex = re
		}

		tempRules = append(tempRules, cached)
	}

	// 3. 替换内存缓存 (原子操作)
	// 注意：这里需要确保 memoryRules 所在的包能被访问，或者通过 Setter 方法设置
	memoryRules = tempRules
	return nil
}
