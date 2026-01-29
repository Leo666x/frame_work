package powerai

import (
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/etcd"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xcache"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"time"
)

// ***************************************************************************************************************
//
//	AgentConfig 用于加载管理智能体配置
//  一、通用配置数据
//     prefix : /agent/config/_general_config_
//     fullKey: /agent/config/_general_config_/企业ID/智能体编号/key
//     value: {
//              "key":"",
//              "value":"",
//              "name":"",
//              "agent_code":"",
//              "classify":"",
//              "modify_from":"",
//              "remark":""
//              "conf_type":"",
//              "update_time":""
//    }
//
//  二、意图配置数据
//     prefix : /agent/config/_decision_config_
//     fullKey: /agent/config/_decision_config_/企业ID/智能体编号/intention_category
//     value: {
//              "key":"",
//              "value":"",
//              "name":"",
//              "agent_code":"",
//              "classify":"",
//              "modify_from":"",
//              "remark":""
//              "conf_type":"",
//              "update_time":""
//    }

//  三、系统配置数据/system/config/_internal_/default/powermop_db
//     prefix : /system/config/_internal_
//     fullKey: /system/config/_internal_/企业ID/powermop_db
//     value: {
//              "key":"",
//              "value":"",
//              "name":"",
//              "agent_code":"",
//              "classify":"",
//              "modify_from":"",
//              "remark":""
//              "conf_type":"",
//              "update_time":""
//    }
//
//
// ***************************************************************************************************************

type Config struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Name       string `json:"name"`
	Remark     string `json:"remark"`
	Classify   string `json:"classify"`
	AgentCode  string `json:"agent_code"`
	ModifyFrom string `json:"modify_from"`
	ConfType   string `json:"conf_type"`
	UpdateTime string `json:"update_time"`
}

type AgentConfig struct {
	etcd            *etcd_mw.Etcd
	configs         *xcache.Cache[string, *Config]
	changeCallbacks []func(key string)
}

func newAgentConfig(etcd *etcd_mw.Etcd, agentCode string, defaultConfigs map[string]*Config, changeCallbacks []func(key string)) *AgentConfig {
	a := &AgentConfig{
		etcd:            etcd,
		configs:         xcache.NewCache[string, *Config](),
		changeCallbacks: changeCallbacks,
	}
	go func() {
		for !a.registerAgentDefaultConfig(defaultConfigs, agentCode) {
			time.Sleep(5 * time.Second)
		}
		time.Sleep(5 * time.Second)

		// 监听 /agent/config/_general_config_/智能体编号
		go a.watch(GetAgentConfigPrefixKey(agentCode))
		// 监听 /system/config/_internal_
		go a.watch(GetSystemConfigPrefixKey())

		// 如果智能体是意图分类智能体那么才开启监听，防止数据过多
		if agentCode == PowerAiDecision {
			// 监听 /agent/config/_decision_config_
			go a.watch(GetAgentDecisionPrefixKey())
		}

	}()

	return a
}

// 注册智能体配置 无需对比etcd上面已存在配置
func (a *AgentConfig) registerAgentDefaultConfig(defaultConfigs map[string]*Config, agentCode string) bool {
	if a.etcd == nil {
		xlog.LogErrorF("10000", "agent-config", "merge", "etcd 未初始化", nil)
		return false
	}
	for k, v := range defaultConfigs {
		// 组装etcd存储的key,注册到默认配置中
		etcdKey := GetAgentConfigFullKey(v.Classify, "", agentCode, k)
		a.configs.Set(etcdKey, v)
		// 从etcd获取配置
		b, err := json.Marshal(v)
		if err != nil {
			xlog.LogErrorF("10000", "agent-config", "register", fmt.Sprintf("将[%s]提交到etcd", etcdKey), err)
			continue
		}
		err = a.etcd.Set(etcdKey, string(b))
		if err != nil {
			xlog.LogErrorF("10000", "agent-config", "register", fmt.Sprintf("将[%s]提交到etcd", etcdKey), err)
			continue
		}
		xlog.LogInfoF("10000", "agent-config", "merge", fmt.Sprintf("从etcd获取[%s]配置数据成功", etcdKey))
	}
	return true
}

func (a *AgentConfig) getConfigFromEtcdAndCache(key string) *Config {

	// 1.查询企业配置是否存在
	c, b := a.configs.Get(key)
	if b {
		// 2.如果缓存存在直接返回
		return c
	}
	// 3.如果缓存不存在，去查询etcd是否存在
	v, err := a.etcd.Get(key)
	if err != nil {
		xlog.LogErrorF("10000", "agent-config", "get", fmt.Sprintf("从etcd获取[%s]配置数据失败,原因：%v", key, err), nil)
		return nil
	}
	if v == nil {
		xlog.LogErrorF("10000", "agent-config", "get", fmt.Sprintf("从etcd获取[%s]配置数据失败,原因：%v", key, "查询数据为空"), nil)
		return nil
	}
	// 4.成功查询到，设置到缓存
	c = &Config{}
	err = json.Unmarshal([]byte(v.Value), c)
	if err != nil {
		xlog.LogErrorF("10000", "agent-config", "get", fmt.Sprintf("将etcd获取[%s]配置转换结构体", key), err)
		return nil
	}
	a.configs.Set(key, c)
	return c
}

// 配置监听
func (a *AgentConfig) watch(prefixKey string) {
	rch, err := a.etcd.WatchPrefixKey(prefixKey)
	if err != nil {
		xlog.LogErrorF("10000", "agent-config", "watch", fmt.Sprintf("监听[%s]失败", prefixKey), err)
		return
	}
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if ev.Type == clientv3.EventTypePut {
				key := string(ev.Kv.Key)
				xlog.LogInfoF("10000", "agent-config", "update", fmt.Sprintf("更新[%s],数据:%s", key, string(ev.Kv.Value)))
				// 配置更新
				c := &Config{}
				err = json.Unmarshal(ev.Kv.Value, c)
				if err != nil {
					xlog.LogErrorF("10000", "agent-config", "update", fmt.Sprintf("将etcd获取[%s]配置转换结构体", key), err)
				} else {
					a.configs.Set(key, c)
				}
			} else if ev.Type == clientv3.EventTypeDelete {
				// 配置删除
				key := string(ev.Kv.Key)
				xlog.LogInfoF("10000", "agent-config", "delete", fmt.Sprintf("删除[%s]", key))
				a.configs.Delete(key)
			}
			// 发出通知
			for _, l := range a.changeCallbacks {
				l(string(ev.Kv.Key))
			}
		}
	}
}
