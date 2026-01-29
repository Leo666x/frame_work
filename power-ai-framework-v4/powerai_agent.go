package powerai

import (
	"encoding/json"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xhttp"
	"orgine.com/ai-team/power-ai-framework-v4/tools"
)

// SyncCallAgent 同步调用智能体
func (a *AgentApp) SyncCallAgent(agentCode string, request *server.AgentRequest, handler xhttp.HttpRequestResponseFunc) string {
	if agentCode == "" {
		handler(nil, fmt.Errorf("agentCode不能为空"))
		return ""
	}
	addr, err := a.agentClient.get(agentCode)
	if err != nil {
		handler(nil, fmt.Errorf("获取智能体[%s]实例信息失败,失败：%v", agentCode, err))
		return ""
	}
	b, _ := json.Marshal(request)
	url := GetAgentSendMsgUrl(addr, agentCode)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   b,
	}
	tools.SendReqBySyncRespStream(r, handler)
	return url
}

// AsyncCallAgent 异步调用智能体
func (a *AgentApp) AsyncCallAgent(agentCode string, request *server.AgentRequest, handler xhttp.HttpRequestResponseFunc) string {
	if agentCode == "" {
		handler(nil, fmt.Errorf("agentCode不能为空"))
		return ""
	}
	addr, err := a.agentClient.get(agentCode)
	if err != nil {
		handler(nil, fmt.Errorf("获取智能体[%s]实例信息失败,失败：%v", agentCode, err))
		return ""
	}

	b, _ := json.Marshal(request)
	url := GetAgentSendMsgUrl(addr, agentCode)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   b,
	}
	tools.SendReqByAsyncRespStream(r, handler)
	return url
}

// CallAgentProxy 调用智能体代理
func (a *AgentApp) CallAgentProxy(agentCode string, request *server.AgentRequest) (string, error) {
	if agentCode == "" {
		return "", fmt.Errorf("agentCode不能为空")
	}
	if request.MethodName == "" {
		return "", fmt.Errorf("methodName不能为空")
	}

	addr, err := a.agentClient.get(agentCode)
	if err != nil {
		return "", fmt.Errorf("获取智能体[%s]实例信息失败,失败：%v", agentCode, err)
	}

	b, _ := json.Marshal(request)
	url := GetAgentProxyUrl(addr, agentCode, request.MethodName)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   b,
	}
	return tools.SendReqByRespString(r)
}

type SendBoxResponse struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	SysTrackCode string `json:"sys_track_code"`
	Data         string `json:"data"`
}

// CallSendBox 代理执行lua脚本，返回字符串
func (a *AgentApp) CallSendBox(sysTrackCode, agentCode, enterpriseId, scriptId string, params interface{}) (*SendBoxResponse, string, error) {
	request := map[string]interface{}{
		"agent_code":     agentCode,
		"sys_track_code": sysTrackCode,
		"enterprise_id":  enterpriseId,
		"script_id":      scriptId,
		"params":         params,
	}
	b, _ := json.Marshal(request)

	r, url, err := a.CallAgentByHttp(PowerAiAgentSendBox, b)
	if err != nil {
		return nil, url, err
	}
	sbr := &SendBoxResponse{}
	err = json.Unmarshal([]byte(r), sbr)
	if err != nil {
		return nil, url, fmt.Errorf("invoke sendbox success,resp to json err: %v", err)
	}

	return sbr, url, nil

}

func (a *AgentApp) CallAgentByHttp(agentCode string, request []byte) (string, string, error) {
	if agentCode == "" {
		return "", "", fmt.Errorf("agentCode不能为空")
	}
	addr, err := a.agentClient.get(agentCode)

	if err != nil {
		return "", "", fmt.Errorf("获取智能体[%s]实例信息失败,失败：%v", agentCode, err)
	}

	url := GetAgentSendMsgUrl(addr, agentCode)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   request,
	}
	resp, err := tools.SendReqByRespString(r)
	return resp, url, err
}

type MultipleEnterprise struct {
	Id   string `json:"enterprise_id"`
	Name string `json:"enterprise_name"`
}

// GetMultipleEnterprise 根据企业ID进行查询
// /system/config/_internal_/企业ID/multiple-enterprise-list
// {
// "key": "multiple-enterprise-list",
// "value": "[ {"enterprise_id":"testid_a","enterprise_name":"testname_a"}, {"enterprise_id":"testid_b","enterprise_name":"testname_b"} ]",
// "name": "企业院区列表",
// "remark": "企业院区列表",
// "classify": "general_config",
// "agent_code": "system",
// "modify_from": "",
// "conf_type": "json"
// }
func (a *AgentApp) GetMultipleEnterprise(enterpriseId string) []*MultipleEnterprise {
	var m []*MultipleEnterprise

	// 1.先查询内存是否存在
	cc := a.GetSystemConfig("multiple-enterprise-list", enterpriseId)
	if cc != nil {
		err := json.Unmarshal([]byte(cc.Value), &m)
		if err != nil {
			return m
		}
	}
	return nil
}

// GetAgentConfig 获取智能体配置
func (a *AgentApp) GetAgentConfig(key, enterpriseId string) *Config {
	enterpriseIdKey := GetAgentGeneralConfigFullKey(enterpriseId, a.Manifest.Code, key)
	c := a.agentConfig.getConfigFromEtcdAndCache(enterpriseIdKey)
	if c != nil {
		return c
	}
	return a.agentConfig.getConfigFromEtcdAndCache(GetAgentGeneralConfigFullKey("", a.Manifest.Code, key))
}

// GetConfigByAgent 根据指定智能体代码获取智能体配置
func (a *AgentApp) GetConfigByAgent(key, enterpriseId, agentCode string) *Config {
	enterpriseIdKey := GetAgentGeneralConfigFullKey(enterpriseId, agentCode, key)
	c := a.agentConfig.getConfigFromEtcdAndCache(enterpriseIdKey)
	if c != nil {
		return c
	}
	return a.agentConfig.getConfigFromEtcdAndCache(GetAgentGeneralConfigFullKey("", agentCode, key))
}

func (a *AgentApp) GetSystemConfig(key, enterpriseId string) *Config {
	enterpriseIdKey := GetSystemConfigFullKey(enterpriseId, key)
	c := a.agentConfig.getConfigFromEtcdAndCache(enterpriseIdKey)
	if c != nil {
		return c
	}
	return a.agentConfig.getConfigFromEtcdAndCache(GetSystemConfigFullKey("", key))
}

func (a *AgentApp) GetConfigFromEtcdAndCache(key string) *Config {
	return a.agentConfig.getConfigFromEtcdAndCache(key)
}

type Decision struct {
	Category       string `json:"category"`
	Content        string `json:"content"`
	Identification string `json:"identification"`
}

func (a *AgentApp) GetAllOnlineAgent() []string {
	return a.agentClient.instances.Keys()
}

// GetDecisions 获取所有意图列表
func (a *AgentApp) GetDecisions(enterpriseId string) ([]*Decision, error) {
	var decisionConfigs []*Config
	for _, agentCode := range a.GetAllOnlineAgent() {
		// /agent/config/_decision_config_/企业ID/智能体编号/intention_category
		enterpriseIdKey := GetAgentDecisionConfigFullKey(enterpriseId, agentCode, AgentDecisionIntentionKey)
		c := a.agentConfig.getConfigFromEtcdAndCache(enterpriseIdKey)
		decisionConfigs = append(decisionConfigs, c)

	}
	var ds []*Decision
	for _, c := range decisionConfigs {
		var d Decision
		err := json.Unmarshal([]byte(c.Value), &d)
		if err != nil {
			return nil, fmt.Errorf("parse decision conf err: %w", err)
		}
		ds = append(ds, &d)
	}
	return ds, nil
}

type DecisionAgentResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Classification string      `json:"classification"`
		Unknown        string      `json:"unknown"`
		Data           interface{} `json:"data"`
	} `json:"data"`
}

// CallDecisionAgent 同步调用智能体
func (a *AgentApp) CallDecisionAgent(request *server.AgentRequest) (*DecisionAgentResponse, string, error) {
	b, _ := json.Marshal(request)
	r, url, err := a.CallAgentByHttp(PowerAiDecision, b)
	if err != nil {
		return nil, url, err
	}

	resp := &DecisionAgentResponse{}
	err = json.Unmarshal([]byte(r), &resp)
	if err != nil {
		return nil, url, err
	}
	return resp, url, nil
}
