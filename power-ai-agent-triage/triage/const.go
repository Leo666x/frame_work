package triage

import powerai "orgine.com/ai-team/power-ai-framework-v4"

const (
	ErrorCallLlm         = "cl-err"       //备注: 调用大模型错误
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

type TriageAgent struct {
	App          *powerai.AgentApp
	resultConfig map[string]interface{} //智能体配置
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
type DeptResponse struct {
	Msg        interface{} `json:"msg"`
	EndFlag    string      `json:"endflag"`
	Type       string      `json:"type"`
	DeptList   interface{} `json:"dept_list"`
	DocList    interface{} `json:"doc_list"`
	Answers    interface{} `json:"answers"`
	PreConsult interface{} `json:"pre_consult"`
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

// TriageSlots TriageAgent 的私有状态
type TriageSlots struct {
	DiagnosisRound    int      `json:"diagnosis_round"`    // 当前回合数
	HistorySummary    string   `json:"history_summary"`    // 问诊历史摘要 (对应 JSON 中的 symptom_summary)
	SuspectedDiseases []string `json:"suspected_diseases"` // 疑似疾病
	Status            string   `json:"status"`             // 状态锁

	// [新增] 症状属性表：用于存储 "主诉": "头疼", "部位": "xx" 这种细粒度信息
	SymptomAttributes map[string]string `json:"symptom_attributes"`
}

// 内部意图识别结果
type InternalIntent struct {
	Intent     string `json:"intent"`      // DIAGNOSIS / SERVICE / EXPERT / INTRO
	KeyEntity  string `json:"key_entity"`  // 提取的实体
	EntityType string `json:"entity_type"` // DISEASE / SYMPTOM (仅 DIAGNOSIS 有效)
}

type ResLlmCallGuide struct {
	Question    []string `json:"question"`
	Answers     []string `json:"answers"`
	Msg         []string `json:"msg"`
	DiseaseList []string `json:"disease_list"`
}

type DeptHistory struct {
	DiagnoseCount map[string]int64 `json:"diagnose_count"`
	Sort          string           `json:"sort"`
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
