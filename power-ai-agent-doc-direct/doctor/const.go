package doctor

import powerai "orgine.com/ai-team/power-ai-framework-v4"

const (
	ErrorCallLlm         = "cl-err" //备注: 调用大模型错误
	ErrorMemory          = "memory-err"
	ErrorKword           = "kword-err"    //
	ErrorRedis           = "cl-redis-err" //备注: 调用大模型错误
	ErrorPsql            = "psql-err"     //备注: 调用大模型错误
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

type DoctorAgent struct {
	App          *powerai.AgentApp
	resultConfig map[string]interface{} //智能体配置
}

// ExtractionResult 内部提取结果
type ExtractionResult struct {
	DoctorName string `json:"doctor_name"`
	IsFullName bool   `json:"is_full_name"`
}

type OutReponse struct {
	ClassiFication string     `json:"classification"`
	UnKnown        string     `json:"unknown"`
	Data           dataStruct `json:"data"`
}

type dataStruct struct {
	Msg     interface{} `json:"msg"`
	EndFlag string      `json:"endflag"`
}
type DocResponse struct {
	Msg        interface{} `json:"msg"`
	EndFlag    string      `json:"endflag"`
	Type       string      `json:"type"`
	DeptList   interface{} `json:"dept_list"`
	DocList    interface{} `json:"doc_list"`
	Answers    interface{} `json:"answers"`
	PreConsult interface{} `json:"pre_consult"`
}

// DoctorSlots 找医生智能体的私有状态
type DoctorSlots struct {
	// 1. 搜索上下文
	TargetName string `json:"target_name"` // 用户想找的名字 (e.g. "张伟")
	Intent     string `json:"intent"`      // FIND(挂号) / INTRO(介绍)

	// 2. 消歧上下文 (关键)
	// 当搜出多个同名医生时，暂存列表，等待用户选择
	CandidateDocs []map[string]interface{} `json:"candidate_docs"`

	// 3. 状态锁
	// 枚举: "idle" (空闲), "waiting_for_selection" (等待选人), "waiting_for_name" (等待补全名字)
	Status string `json:"status"`
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

// SelectionResult 对应 LLM 的 JSON 输出
type SelectionResult struct {
	Reason        string `json:"reason"`
	SelectedIndex int    `json:"selected_index"`
}
