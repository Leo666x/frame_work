package main

import (
	_ "embed"
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	v4demo "power-ai-agent-v4demo/demo"
)

//go:embed manifest.json
var manifest string

func main() {
	//_ = os.Setenv("PORT", "9527") // 你自己智能体服务的端口号
	//_ = os.Setenv("IP_ADDR_DEBUG", "192.168.5.2")
	//_ = os.Setenv("POWER_AI_ETCD_PORT", "39001") // 研发环境的etcd端口
	//_ = os.Setenv("IP_ADDR", "192.168.0.98")     // 研发环境的etcd地址
	ag := v4demo.DemoAgent{}

	app, err := powerai.NewAgent(
		manifest,
		powerai.WithOnShutDown(ag.OnShutdown),                  //智能体退出的时候回调，可有可无,函数名称可以自己定义
		powerai.WithDefaultConfigs(initConf()),                 // 初始化配置
		powerai.WithSendMsgRouter(ag.SendMsg),                  // 这个路由智能体必须添加，用于接收用户发来的消息，并且定义
		powerai.WithCustomPostRouter("demo_post", ag.DemoPost), // 添加自定义Post路由
		powerai.WithCustomPostRouter("demo_get", ag.DemoGet),   // 添加自定义Get路由
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
		Value:     getDecision(),
		Name:      "意图分类",
		Remark:    "",
		AgentCode: "power-ai-agent-v4demo",
		Classify:  powerai.DecisionConfigClassify,
		ConfType:  "intention",
	}

	conf["config_key_1"] = &powerai.Config{
		Key:       "config_key_1",
		Value:     "这是自定义配置",         // 这是配置定义 字符串
		Name:      "用户查询时间抽取提示词", //这是配置名称 字符串
		AgentCode: "power-ai-agent-v4demo",
		Classify:  powerai.GeneralConfigClassify, //通用配置 无需修改
		Remark:    "",
		ConfType:  "prompt",
	}

	return conf

}
