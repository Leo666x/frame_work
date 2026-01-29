package server

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xaes"
)

var (
	ResultSuccess        = ErrorCode{Code: "success", Message: "执行成功"}
	ResultError          = ErrorCode{Code: "request_error", Message: "请求报文错误"}
	ConversationNotExist = ErrorCode{Code: "conversation_not_exist", Message: "对话不存在"}
	InvalidParam         = ErrorCode{Code: "invalid_param", Message: "传入参数异常"}
	Unavailable          = ErrorCode{Code: "unavailable", Message: "配置不可用"}
	ServiceError         = ErrorCode{Code: "service_error", Message: "系统错误"}
	ConversationLimit    = ErrorCode{Code: "conversation_limit", Message: "对话消息达到上限"}
	InvokeAgentError     = ErrorCode{Code: "invoke_agent_error", Message: "调用智能体错误"}
	InvokeServiceError   = ErrorCode{Code: "invoke_service_error", Message: "调用其他服务错误"}
)

var (
	EventMessage       = "message"
	EventMessageStruct = "message_struct"
	EventMessageEnd    = "message_end"
	EventError         = "error"
)

type ErrorCode struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AgentResponse struct {
	ConversationId string `json:"conversation_id"`
	MessageId      string `json:"message_id"`
	CreatedAt      string `json:"created_at"`
	User           string `json:"user"`
	Channel        string `json:"channel"`
	ChannelApp     string `json:"channel_app"`
	EnterpriseId   string `json:"enterprise_id"`
	SysTrackCode   string `json:"sys_track_code"`
	AgentCode      string `json:"agent_code"`
}

type ExtendedAgentResponse struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	*AgentResponse
}

type AgentRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`
	Query          string                 `json:"query"`
	ConversationId string                 `json:"conversation_id"`
	UserId         string                 `json:"user_id"`
	Channel        string                 `json:"channel"`
	ChannelApp     string                 `json:"channel_app"`
	EnterpriseId   string                 `json:"enterprise_id"`
	SysTrackCode   string                 `json:"sys_track_code"`
	AgentCode      string                 `json:"agent_code"`
	MessageId      string                 `json:"message_id"`
	MethodName     string                 `json:"method_name"`
	Files          []struct {
		Type   string `json:"type"`
		FileID string `json:"file_id"`
	} `json:"files"`
}

type SSEEvent struct {
	*gin.Context
}

// Done 流式响应输入完成
func (se *SSEEvent) Done(resp *AgentResponse) {
	se.WriteString(se.eventMessageEnd(resp))
}

// DoneAesEncrypt (aes加密)流式响应输入完成
func (se *SSEEvent) DoneAesEncrypt(resp *AgentResponse, secretKey string) {
	rs, _ := xaes.EncryptCBC(se.eventMessageEnd(resp), secretKey)
	se.WriteString(rs)
}

// WriteAny 流式响应写入数据
func (se *SSEEvent) WriteAny(data any) error {
	if b, err := json.Marshal(data); err == nil {
		se.WriteString(string(b))
		return nil
	} else {
		return err
	}
}

// WriteAnyAesEncrypt  (aes加密)流式响应写入数据
func (se *SSEEvent) WriteAnyAesEncrypt(data any, secretKey string) error {
	if b, err := json.Marshal(data); err == nil {
		rs, _ := xaes.EncryptCBCByte(b, secretKey)
		se.WriteString(rs)
		return nil
	} else {
		return err
	}
}

// WriteAgentResponseMessage 流式响应写入数据
func (se *SSEEvent) WriteAgentResponseMessage(resp *AgentResponse, content string) error {
	se.WriteString(se.eventMessage(resp, content))
	return nil
}

// WriteAgentResponseMessageAesEncrypt (aes加密)流式响应写入数据
func (se *SSEEvent) WriteAgentResponseMessageAesEncrypt(resp *AgentResponse, content, secretKey string) error {
	rs, _ := xaes.EncryptCBC(se.eventMessage(resp, content), secretKey)
	se.WriteString(rs)
	return nil
}

// WriteAgentResponseError 流式响应错误
func (se *SSEEvent) WriteAgentResponseError(resp *AgentResponse, code, message string) error {
	se.WriteString(se.eventMessageError(resp, code, message))
	return nil
}

// WriteAgentResponseErrorAesEncrypt (aes加密)流式响应错误
func (se *SSEEvent) WriteAgentResponseErrorAesEncrypt(resp *AgentResponse, code, message, secretKey string) error {
	rs, _ := xaes.EncryptCBC(se.eventMessageError(resp, code, message), secretKey)
	se.WriteString(rs)
	return nil
}

// WriteAgentResponseStruct 流式响应结结构体
func (se *SSEEvent) WriteAgentResponseStruct(resp *AgentResponse, structContent any) error {
	se.WriteString(se.eventMessageStruct(resp, structContent))
	return nil
}

// WriteAgentResponseStructAesEncrypt 流式响应结结构体
func (se *SSEEvent) WriteAgentResponseStructAesEncrypt(resp *AgentResponse, structContent any, secretKey string) error {
	rs, _ := xaes.EncryptCBC(se.eventMessageStruct(resp, structContent), secretKey)
	se.WriteString(rs)
	return nil
}

// WriteString 流式响应写入数据
func (se *SSEEvent) WriteString(data string) {
	se.SSEvent("", data)
	se.Writer.Flush()
}

func (se *SSEEvent) eventMessageEnd(resp *AgentResponse) string {
	r := &ExtendedAgentResponse{
		Event:         EventMessageEnd,
		AgentResponse: resp,
	}

	b, _ := json.Marshal(r)
	return string(b)
}

// eventMessage
func (se *SSEEvent) eventMessage(resp *AgentResponse, content string) string {
	r := &ExtendedAgentResponse{
		Event:         EventMessage,
		AgentResponse: resp,
		Data: map[string]string{
			"content": content,
		},
	}
	b, _ := json.Marshal(r)
	return string(b)
}

// eventMessageError 流式响应错误
func (se *SSEEvent) eventMessageError(resp *AgentResponse, code, message string) string {
	r := &ExtendedAgentResponse{
		Event:         EventError,
		AgentResponse: resp,
		Data: &ErrorCode{
			Code:    code,
			Message: message,
		},
	}
	b, _ := json.Marshal(r)
	return string(b)

}

func (se *SSEEvent) eventMessageStruct(resp *AgentResponse, structContent any) string {
	r := &ExtendedAgentResponse{
		Event:         EventMessageStruct,
		AgentResponse: resp,
		Data:          structContent,
	}
	b, _ := json.Marshal(r)
	return string(b)
}
