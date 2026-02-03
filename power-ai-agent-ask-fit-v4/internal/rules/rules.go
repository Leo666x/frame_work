package rules

type Rule struct {
	Name       string
	Keywords   []string
	AgentCode  string
	Router     string
	Confidence float64
	Reason     string
}

type RuleSet struct {
	Rules             []Rule
	DefaultIntent     string
	DefaultAgentCode  string
	DefaultRouter     string
	DefaultConfidence float64
	DefaultReason     string
}

func DefaultRules() RuleSet {
	return RuleSet{
		Rules: []Rule{
			{
				Name:       "appointment",
				Keywords:   []string{"挂号", "预约", "约个号"},
				AgentCode:  "power-ai-appointment",
				Router:     "send_msg",
				Confidence: 0.85,
				Reason:     "预约关键词命中",
			},
			{
				Name:       "report",
				Keywords:   []string{"报告", "影像", "检验", "化验"},
				AgentCode:  "power-ai-report",
				Router:     "send_msg",
				Confidence: 0.80,
				Reason:     "报告关键词命中",
			},
		},
		DefaultIntent:     "general",
		DefaultAgentCode:  "power-ai-agent-ask-fit",
		DefaultRouter:     "send_msg",
		DefaultConfidence: 0.50,
		DefaultReason:     "默认兜底",
	}
}

const (
	DoubleClassPromptKey      = "double_class_prompt"
	MultiClassPromptKey       = "question_classfication_prompt"
	DecisionConfigKey         = "intention_category"
	ReturnSolidTextKey        = "return_solid_text"
	QAXHYYPromptKey           = "qa_xhyy_agent_QAXHYYExtraction_prompt"
	QAXHYYTopKKey             = "qa_xhyy_agent_topk_conf"
)

const DefaultReturnSolidText = "您好，已帮您找到服务相关内容，请点击卡片使用相应功能"
