package department

import powerai "orgine.com/ai-team/power-ai-framework-v4"

const (
	ErrorCallLlm         = "cl-err"       //备注: 调用大模型错误
	ErrorRedis           = "cl-redis-err" //备注: 调用大模型错误
	ErrorMemory          = "memory-err"
	ErrorPsql            = "psql-err" //备注: 调用大模型错误
	RAG_TopK             = 3
	ThresholdScore       = 0.85
	ThresholdScoreDiff   = 0.15
	DecisionAgentUnknown = "agent-unknown"
	ErrorRAG             = "rag-err"      // 知识库推荐错误
	ErrorDataBase        = "database-err" // 数据库调用错误

	ReturnCardTypeGuide = "card_register_answers"    // 返回前端的卡片-引导症状
	ReturnCardTypeDept  = "card_register_department" // 返回前端的卡片-科室推荐
	ReturnCardTypeDoc   = "card_register_doctor"     // 返回前端的卡片-医生推荐

	ErrorAIMsgGet = "ai-msg-get-err" // 消息会话获取错误
	ErrorParse    = "json-parse-err"
	StreamSpeed   = 3
)

type DeptAgent struct {
	App          *powerai.AgentApp
	resultConfig map[string]interface{} //智能体配置
}

type DeptResponse struct {
	Msg        interface{} `json:"msg"`
	EndFlag    string      `json:"endflag"`
	Type       string      `json:"type"`
	DeptList   interface{} `json:"dept_list"`
	DocList    interface{} `json:"doc_list"`
	Answers    interface{} `json:"answers"`
	PreConsult interface{} `json:"pre_consult"`
}

// DeptSlots 找科室智能体的私有状态 (存入 AgentSlots)
type DeptSlots struct {
	// 1. 搜索上下文
	TargetName string `json:"target_name"` // 用户想找的科室名 (e.g. "神经内科")
	Intent     string `json:"intent"`      // FIND(挂号) / INTRO(介绍)

	// 2. 消歧上下文
	// 当搜出多个相关科室时，暂存列表，等待用户选择
	CandidateDepts []SimpleDeptInfo `json:"candidate_depts"`

	// 3. 状态锁
	// "idle", "waiting_for_selection"
	Status string `json:"status"`
}

// SimpleDeptInfo 简化的科室信息
type SimpleDeptInfo struct {
	DeptID   string `json:"dept_id"`
	DeptName string `json:"dept_name"`
}

// ExtractedData 对应 LLM 从历史对话中提取的完整 JSON 结构
type ExtractedData struct {
	// 1. 基础画像 (对应 UserProfile)
	PatientInfo ExtractedPatientInfo `json:"patient_info"`

	// 2. 症状属性表 (对应 TriageSlots.SymptomAttributes)
	// 使用 map[string]string 以应对 Prompt 中动态定义的中文 Key (如 "主诉", "部位", "性质")
	// 这样设计具有最大的灵活性，即使 Prompt 里的 Key 变了，代码也不用改
	SymptomAttributes map[string]string `json:"symptom_attributes"`

	// 3. 结果摘要 (对应 GlobalState.Shared)
	DiagnosisResult ExtractedDiagnosisResult `json:"diagnosis_result"`
}

// ExtractedPatientInfo 患者基础信息
type ExtractedPatientInfo struct {
	// 使用 string 而非 int，因为 LLM 可能输出 "25岁" 或 "未知"
	Gender string `json:"gender"`
	Age    string `json:"age"`
}

// ExtractedDiagnosisResult 诊断结论
type ExtractedDiagnosisResult struct {
	IsDiagnosed    bool     `json:"is_diagnosed"`    // 是否已确诊
	Disease        []string `json:"disease"`         // 确诊/疑似疾病列表
	SymptomSummary string   `json:"symptom_summary"` // 完整的病历摘要
}

// DeptExtractionResult 内部提取结果
type DeptExtractionResult struct {
	Intent   string `json:"intent"`
	DeptName string `json:"dept_name"`
}
