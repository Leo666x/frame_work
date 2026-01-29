package powerai

import (
	"fmt"
	"strings"
)

const (

	// AgentInstancePrefixKey 服务注册信息key前缀
	AgentInstancePrefixKey = "/service/instance/"

	SystemConfigPrefixKey = "/system/config/_internal_"

	AgentConfigPrefixKey = "/agent/config"

	DecisionConfigClassify = "_decision_config_"

	GeneralConfigClassify = "_general_config_"

	AgentListKey = "agent_list"

	AgentDecisionIntentionKey = "intention_category"
	PowerAiDecision           = "power-ai-decision"
	PowerAiAgentSendBox       = "power-ai-agent-sendbox"
)

// GetServiceInstancePrefixKey /service/instance/{agent_code}
func GetServiceInstancePrefixKey(agentCode string) string {
	return fmt.Sprintf("%s%s", AgentInstancePrefixKey, agentCode)
}

// GetServiceInstanceFullKey /service/instance/{agent_code}/ip:port
func GetServiceInstanceFullKey(agentCode, ip, port string) string {
	return fmt.Sprintf("%s%s/%s:%s", AgentInstancePrefixKey, agentCode, ip, port)
}

func GetSystemConfigFullKey(enterpriseId, key string) string {
	//eg:/system/config/_internal_/default/powermop_db
	if enterpriseId == "" {
		enterpriseId = "default"

	}
	return fmt.Sprintf("%s/%s/%s", SystemConfigPrefixKey, enterpriseId, key)
}

func GetAgentGeneralConfigFullKey(enterpriseId, agentCode, key string) string {
	return GetAgentConfigFullKey(GeneralConfigClassify, enterpriseId, agentCode, key)
}
func GetAgentDecisionConfigFullKey(enterpriseId, agentCode, key string) string {
	return GetAgentConfigFullKey(DecisionConfigClassify, enterpriseId, agentCode, key)
}

func GetAgentDecisionPrefixKey() string {
	return fmt.Sprintf("%s/%s", AgentConfigPrefixKey, DecisionConfigClassify)
}

func GetAgentConfigPrefixKey(code string) string {
	return fmt.Sprintf("%s/%s/%s", AgentConfigPrefixKey, GeneralConfigClassify, code)
}

func GetSystemConfigPrefixKey() string {
	return SystemConfigPrefixKey
}

func GetAgentSendMsgUrl(addr, agentCode string) string {

	return fmt.Sprintf("http://%s/%s/send_msg", addr, strings.ReplaceAll(agentCode, "-", "/"))
}

func GetAgentProxyUrl(addr, agentCode, methodName string) string {
	return fmt.Sprintf("http://%s/%s/%s", addr, strings.ReplaceAll(agentCode, "-", "/"), methodName)
}

func GetAgentConfigFullKey(classify, enterpriseId, code, key string) string {
	// eg:/agent/config/_general_config_/power-ai-agent-register/default/OrderList-script-ge
	if enterpriseId == "" {
		enterpriseId = "default"
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", AgentConfigPrefixKey, classify, code, enterpriseId, key)
}
