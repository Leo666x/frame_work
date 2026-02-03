package main

import (
	_ "embed"
	"os"
	triage "power-ai-agent-triage/triage"

	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
)

//go:embed manifest.json
var manifest string

func main() {

	// 本地调试模式，设置环境变量
	os.Setenv("IP_ADDR_DEBUG", "192.168.5.21")
	// ，重点：只有本地开发调试的时候，要设置

	_ = os.Setenv("PORT", "39601")               // 你自己智能体服务的端口号
	_ = os.Setenv("POWER_AI_ETCD_PORT", "39001") // 研发环境的etcd端口
	_ = os.Setenv("IP_ADDR", "192.168.0.98")     // 研发环境的etcd地址
	ag := triage.TriageAgent{}
	app, err := powerai.NewAgent(
		manifest,
		powerai.WithDefaultConfigs(initConf()), // 初始化配置
		powerai.WithSendMsgRouter(ag.SendMsg),  // 这个路由智能体必须添加，用于接收用户发来的消息，并且定义
		//powerai.WithCustomPostRouter("demo_post", ag.DemoPost), // 添加自定义Post路由
		//powerai.WithCustomPostRouter("demo_get", ag.DemoGet),   // 添加自定义Get路由
	)
	if err != nil {
		xlog.LogErrorF("10000", "create agent", "create agent", "create agent", err)
		return
	}
	ag.App = app
	ag.App.Run()

}

func initConf() map[string]*powerai.Config {
	conf := make(map[string]*powerai.Config)
	//智能体意图配置
	conf["intention_category"] = &powerai.Config{
		Key:       "intention_category",
		Value:     "",
		Name:      "导诊",
		Remark:    "",
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.DecisionConfigClassify,
		ConfType:  "intention",
	}

	// 挂号智能体获取科室名称、医生名称和关键词提示词
	conf["register_agent_get_keywords_prompt"] = &powerai.Config{ //
		Key:       "register_agent_get_keywords_prompt",
		Value:     registerAgentGetKeywords,  // 这是配置定义 字符串
		Name:      "挂号智能体获取科室名称、医生名称和关键词提示词", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		ConfType:  "prompt",
	}
	// 挂号智能体引导用户query补充关键症状信息及推断疾病提示词
	conf["register_agent_guide_prompt"] = &powerai.Config{ //
		Key:       "register_agent_guide_prompt",
		Value:     registerAgentGuidePrompt,         // 这是配置定义 字符串
		Name:      "挂号智能体引导用户query补充关键症状信息及推断疾病提示词", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		ConfType:  "prompt",
	}
	// 挂号智能体返回推荐医生数量topK
	conf["register_agent_return_doc_topK"] = &powerai.Config{ //
		Key:       "register_agent_return_doc_topK",
		Value:     "10",                // 这是配置定义 字符串
		Name:      "挂号智能体返回推荐医生数量topK", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "科室推荐返回医生数量  当推荐的医生数量小于0时，则表示没有开启推荐的医生功能",
		ConfType:  "text",
	}
	// 挂号智能体未开启医生推荐提示词
	conf["register_agent_return_doc_msg"] = &powerai.Config{ //
		Key:       "register_agent_return_doc_msg",
		Value:     "没有找到该医生。您可以直接说：我要皮肤科", // 这是配置定义 字符串
		Name:      "挂号智能体未开启医生推荐提示词",      //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体未开启医生推荐提示词",
		ConfType:  "text",
	}

	// 挂号智能体返回推荐科室数量topK
	conf["register_agent_return_dept_topK"] = &powerai.Config{ //
		Key:       "register_agent_return_dept_topK",
		Value:     "3",                 // 这是配置定义 字符串
		Name:      "挂号智能体返回推荐科室数量topK", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "科室推荐返回科室数量",
		ConfType:  "text",
	}

	// 单院区单知识库模式
	conf["register_agent_SingleKnowledge"] = &powerai.Config{ //
		Key:       "register_agent_SingleKnowledge",
		Value:     "false",     // 这是配置定义 字符串
		Name:      "单院区单知识库模式", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "单院区单知识库模式 true 开启，false 关闭（默认false）",
		ConfType:  "text",
	}

	// 挂号智能体科室挂号量历史记录数据
	conf["register_agent_dept_history"] = &powerai.Config{ //
		Key:       "register_agent_dept_history",
		Value:     "{\"diagnose_count\": {\"708\": 629,\"558\": 118666,\"102\": 18666}, \"sort\": \"false\"}", // 这是配置定义 字符串
		Name:      "疾病推科室是否按历史挂号量排序返回",                                                                        //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "数据格式：{\"diagnose_count\": {\"210120\": 544629,\"231\":2312}, \"sort\": \"true\"}。sort是否进行排序，diagnose_count历史排序数据：dept_id:n",
		ConfType:  "json",
	}

	// 挂号智能体医生推荐二次精准匹配提示词
	conf["register_agent_search_doc_by_query"] = &powerai.Config{ //
		Key:       "register_agent_search_doc_by_query",
		Value:     searchDocByQuery,     // 这是配置定义 字符串
		Name:      "挂号智能体医生推荐二次精准匹配提示词", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体医生推荐二次精准匹配提示词",
		ConfType:  "prompt",
	}

	// 挂号智能体科室推荐二次精准匹配提示词
	conf["register_agent_search_dept_by_query"] = &powerai.Config{ //
		Key:       "register_agent_search_dept_by_query",
		Value:     searchDeptByQuery,    // 这是配置定义 字符串
		Name:      "挂号智能体科室推荐二次精准匹配提示词", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体科室推荐二次精准匹配提示词",
		ConfType:  "prompt",
	}

	// 挂号智能体科室推荐二次精准匹配提示词
	conf["register_agent_deduce_illness_directly"] = &powerai.Config{ //
		Key:       "register_agent_deduce_illness_directly",
		Value:     illnessPrompt,        // 这是配置定义 字符串
		Name:      "挂号智能体多轮问答直接推导疾病提示词", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体多轮问答直接推导疾病提示词",
		ConfType:  "prompt",
	}

	// 挂号智能体预问诊总结
	conf["register_agent_pre_consultation"] = &powerai.Config{ //
		Key:       "register_agent_pre_consultation",
		Value:     "false",          // 这是配置定义 字符串
		Name:      "挂号智能体预问诊总结功能启用", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "是否启用挂号智能体预问诊总结功能，true/false",
		ConfType:  "text",
	}

	// 挂号智能体预问诊总结提示词
	conf["register_agent_pre_consultation_prompt"] = &powerai.Config{ //
		Key:       "register_agent_pre_consultation_prompt",
		Value:     PreConsultationPrompt, // 这是配置定义 字符串
		Name:      "挂号智能体预问诊总结提示词",       //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体预问诊总结提示词",
		ConfType:  "prompt",
	}

	// 挂号智能体科室筛选符合性别年龄提示词
	conf["register_agent_dept_match_of_sex_age_prompt"] = &powerai.Config{ //
		Key:       "register_agent_dept_match_of_sex_age_prompt",
		Value:     DeptMatchOfSexAgePrompt, // 这是配置定义 字符串
		Name:      "挂号智能体科室筛选符合性别年龄提示词",    //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体科室筛选符合性别年龄提示词",
		ConfType:  "prompt",
	}

	// 医生跳转地址
	conf["power-ai-agent-triage-doc-go-url"] = &powerai.Config{ //
		Key:       "power-ai-agent-triage-doc-go-url",
		Value:     docGoUrl, // 这是配置定义 字符串
		Name:      "医生跳转地址", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "医生跳转地址",
		ConfType:  "text",
	}

	// 科室跳转地址
	conf["power-ai-agent-triage-dept-go-url"] = &powerai.Config{ //
		Key:       "power-ai-agent-triage-dept-go-url",
		Value:     deptGoUrl, // 这是配置定义 字符串
		Name:      "科室跳转地址",  //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "科室跳转地址",
		ConfType:  "text",
	}

	// 获取挂号记录
	conf["power-ai-agent-triage-OrderList-script-get"] = &powerai.Config{ //
		Key:       "power-ai-agent-triage-OrderList-script-get",
		Value:     OrderListScriptGetName, // 这是配置定义 字符串
		Name:      "获取挂号记录",               //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "获取挂号记录",
		ConfType:  "lua",
	}

	conf["inputs"] = &powerai.Config{
		Key:       "inputs",
		Value:     debugInpus,      // 这是配置定义 字符串
		Name:      "调试智能体inputs参数", //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "",
		ConfType:  "json",
	}

	conf["power_ai_agent_register_doc_KB_name"] = &powerai.Config{
		Key:       "power_ai_agent_register_doc_KB_name",
		Value:     "Doc_match_recommend_doc", // 这是配置定义 字符串
		Name:      "挂号智能体找医生知识库名称",           //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体找医生知识库名称",
		ConfType:  "json",
	}

	conf["power_ai_agent_register_dept_direct_KB_name"] = &powerai.Config{
		Key:       "power_ai_agent_register_dept_direct_KB_name",
		Value:     "Dept_match_recommend_dept", // 这是配置定义 字符串
		Name:      "挂号智能体找科室知识库名称",             //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体找科室知识库名称",
		ConfType:  "json",
	}

	conf["power_ai_agent_register_dept_illness_KB_name"] = &powerai.Config{
		Key:       "power_ai_agent_register_dept_illness_KB_name",
		Value:     "Illness_match_recommend_dept_", // 这是配置定义 字符串
		Name:      "挂号智能体疾病推科室知识库名称",               //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体疾病推科室知识库名称",
		ConfType:  "json",
	}

	conf["power_ai_agent_register_dept_search_msg"] = &powerai.Config{
		Key:       "power_ai_agent_register_dept_search_msg",
		Value:     "根据您的要求，我们没有找到到相关科室，请检查一下您要挂的科室信息。比如：我要找神经内科", // 这是配置定义 字符串
		Name:      "科室二次匹配的提示语",                                  //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "科室二次匹配的提示语",
		ConfType:  "json",
	}

	conf["register_agent_multi_enterprise_conf"] = &powerai.Config{
		Key:       "register_agent_multi_enterprise_conf",
		Value:     "[{\"id\":\"jkhb\",\"name\":\"健康湖北\"},{\"id\":\"orgine\",\"name\":\"源启智慧医院\"}]", // 这是配置定义 字符串
		Name:      "挂号智能体查知识库多院区配置",                                                                //这是配置名称 字符串
		AgentCode: "power-ai-agent-triage",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体查知识库多院区配置，单院区配置一个即可",
		ConfType:  "json",
	}

	return conf

}
