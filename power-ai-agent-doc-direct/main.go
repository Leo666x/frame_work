package main

import (
	_ "embed"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	doctor "power-ai-agent-doc-direct/doctor"
)

//go:embed manifest.json
var manifest string

func main() {
	ag := doctor.DoctorAgent{}
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
		AgentCode: "power-ai-agent-doctor",
		Classify:  powerai.DecisionConfigClassify,
		ConfType:  "intention",
	}

	conf["register_agent_multi_enterprise_conf"] = &powerai.Config{
		Key:       "register_agent_multi_enterprise_conf",
		Value:     "[{\"id\":\"jkhb\",\"name\":\"健康湖北\"},{\"id\":\"orgine\",\"name\":\"源启智慧医院\"}]", // 这是配置定义 字符串
		Name:      "挂号智能体查知识库多院区配置",                                                                //这是配置名称 字符串
		AgentCode: "power-ai-agent-doctor",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "挂号智能体查知识库多院区配置，单院区配置一个即可",
		ConfType:  "json",
	}

	return conf

}
