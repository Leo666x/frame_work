package main

import (
	_ "embed"
	"encoding/json"
	powerai "orgine.com/ai-team/power-ai-framework"
	"orgine.com/power-ai-agent-ask/ask"
)

//go:embed manifest.json
var manifest string

func main() {
	// 本地调试模式，设置环境变量 os.Setenv("IP_ADDR_DEBUG", "ip地址，为你自己开发电脑的ip")，重点：只有本地开发调试的时候，要设置
	//_ = os.Setenv("PORT", "9527") // 你自己智能体服务的端口号
	//_ = os.Setenv("IP_ADDR_DEBUG", "192.168.5.2")
	//_ = os.Setenv("POWER_AI_ETCD_PORT", "39001") // 研发环境的etcd端口
	//_ = os.Setenv("IP_ADDR", "192.168.0.121")    // 研发环境的etcd地址

	// 基础信息
	appInfo := &powerai.AgentAppInfo{}
	_ = json.Unmarshal([]byte(manifest), appInfo)

	// 配置初始化
	appInfo.DefaultConf = initConf()

	// 初始化逻辑体
	ag := &ask.AgentAsk{}
	// 初始化appAgent
	app := powerai.NewAgentApp(appInfo, ag.SendMsg)

	ag.App = app
	ag.App.Run()
}

func initConf() map[string]*powerai.Conf {
	conf := make(map[string]*powerai.Conf)

	conf["question_classfication_prompt"] = &powerai.Conf{
		Key:       "question_classfication_prompt",
		Value:     extractCardtypeSystemPrompt,
		Name:      "即问即办问题分类提示词",
		AgentCode: "power-ai-agent-ask",
		Classify:  powerai.GeneralConfClassify,
		Remark:    "大模型提取时间参数提示词，可以在prompt.go文件中进行修改，用于提高大模型根据【门诊缴费清单咨询】返回【正确时间范围】的准确率",
	}
	// 添加大模型提取时间参数的提示词配置
	conf["double_class_prompt"] = &powerai.Conf{
		Key:       "double_class_prompt",
		Value:     doubleClssPrompt,
		Name:      "二分类意图识别提示词",
		AgentCode: "power-ai-agent-ask",
		Classify:  powerai.GeneralConfClassify,
		Remark:    "",
	}

	conf["intention_category"] = &powerai.Conf{ // intention_category 固定不得修改
		Key:       "intention_category", // intention_category 固定不得修改
		Value:     getDecision(),
		Name:      "即问即办意图识别配置",
		AgentCode: "power-ai-agent-ask",
		Classify:  powerai.DecisionConfClassify,
		Remark:    "在decision.go文件中，修改【意图识别】提示词，提高识别用户【门诊缴费清单咨询】意图的识别准确率", //这是注意事项，根据实际情况进项填写，主要是指导，这个value如何配置
	}

	// 默认字段配置
	//conf["qa_default_reply"] = &powerai.Conf{
	//	Key:       "qa_default_reply",
	//	Value:     "实在抱歉(⊙﹏⊙)，对于您刚刚提出的问题，我目前还在学习中~如果您想要使用即问即办的相关功能，您可以尝试问我诸如:在线问诊，导航，轮椅租赁等问题。", // 默认查询最近6个月数据
	//	Name:      "找医院默认返回字段",
	//	AgentCode: "power-ai-agent-ask",
	//	Classify:  powerai.GeneralConfClassify,
	//	Remark:    "配置默认查询的时间范围，填写整数，单位是月，例如：8，表示为默认查询近8个月的数据",
	//}

	conf["return_solid_text"] = &powerai.Conf{
		Key:       "return_solid_text",
		Value:     "您好，已帮您找到服务相关内容，请点击卡片使用相应功能",
		Name:      "默认响应文本",
		AgentCode: "power-ai-agent-ask",
		Classify:  powerai.GeneralConfClassify,
		Remark:    "",
	}

	conf["qa_xhyy_agent_QAXHYYExtraction_prompt"] = &powerai.Conf{
		Key:       "qa_xhyy_agent_QAXHYYExtraction_prompt",
		Value:     QAXHYYExtractionPrompt, // 这是配置定义 字符串
		Name:      "智能客服提示词",              //这是配置名称 字符串
		AgentCode: "power-ai-agent-qa-xhyy",
		Classify:  powerai.GeneralConfClassify, //通用配置 无需修改
		Remark:    "智能客服提示词",
	}

	//智能客服提示词
	conf["qa_xhyy_agent_topk_conf"] = &powerai.Conf{
		Key:       "qa_xhyy_agent_topk_conf",
		Value:     "10",     // 这是配置定义 字符串
		Name:      "最多显示条数", //这是配置名称 字符串
		AgentCode: "power-ai-agent-qa-xhyy",
		Classify:  powerai.GeneralConfClassify, //通用配置 无需修改
		Remark:    "智能客服提示词",
	}

	return conf

}
