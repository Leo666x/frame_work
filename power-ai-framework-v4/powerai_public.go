package powerai

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xdatetime"
	"strings"
)

func doValidateAgentRequest(c *gin.Context) (*server.AgentRequest, *server.ErrorCode) {
	msg, err := ioutil.ReadAll(c.Request.Body)

	if err != nil {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s:无法获取请求内容", server.ResultError.Message)}
	}
	msgStr := string(msg)
	req := &server.AgentRequest{}
	err = json.Unmarshal([]byte(msgStr), req)
	//1.必备条件验证
	if err != nil {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-JSON解析错误,错误：%v", server.ResultError.Message, err)}
	}
	if req == nil {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-不是正确的报文结构", server.ResultError.Message)}
	}
	if req.SysTrackCode == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{sys_track_code}为空", server.InvalidParam.Message)}
	}

	// 校验用户query是否为空  req.Files != nil 图片解读时，query可为空
	if req.Files == nil {
		if req.Query == "" {
			return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{query}为空", server.InvalidParam.Message)}
		}
	}

	if req.UserId == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{user_id}为空", server.InvalidParam.Message)}
	}
	if req.Channel == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{channel}为空", server.InvalidParam.Message)}
	}
	if req.ChannelApp == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{channel_app}为空", server.InvalidParam.Message)}
	}
	if req.EnterpriseId == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{enterprise_id}为空", server.InvalidParam.Message)}
	}
	if req.ConversationId == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{conversation_id}为空", server.InvalidParam.Message)}
	}
	if req.MessageId == "" {
		return nil, &server.ErrorCode{Code: server.ResultError.Code, Message: fmt.Sprintf("%s-{message_id}为空", server.InvalidParam.Message)}
	}
	return req, nil
}

func DoValidateAgentRequest(c *gin.Context, agentCode string) (*server.AgentRequest, *server.AgentResponse, *server.SSEEvent, bool) {
	event := NewHttpStreamEvent(c)
	req, errCode := doValidateAgentRequest(c)
	if errCode != nil {
		_ = event.WriteAgentResponseError(nil, errCode.Code, errCode.Message)
		return nil, nil, nil, false
	}
	return req, BuildAgentResponse(req, agentCode), event, true
}

func BuildAgentResponse(req *server.AgentRequest, agentCode string) *server.AgentResponse {
	return &server.AgentResponse{
		ConversationId: req.ConversationId,
		MessageId:      req.MessageId,
		CreatedAt:      xdatetime.GetNowDateTimeNano(),
		User:           req.UserId,
		Channel:        req.Channel,
		ChannelApp:     req.ChannelApp,
		EnterpriseId:   req.EnterpriseId,
		SysTrackCode:   req.SysTrackCode,
		AgentCode:      agentCode,
	}
}

func NewHttpStreamEvent(c *gin.Context) *server.SSEEvent {
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	return &server.SSEEvent{Context: c}
}
func RespJsonError(c *gin.Context, code, msg, stc string, data interface{}) {
	c.JSON(200, map[string]interface{}{
		"code":           code,
		"message":        msg,
		"sys_track_code": stc,
		"data":           data,
	})
}

func RespJsonSuccess(c *gin.Context, stc string, data interface{}) {
	c.JSON(200, map[string]interface{}{
		"code":           "success",
		"message":        "执行成功",
		"sys_track_code": stc,
		"data":           data,
	})
}

func ParseAgentResponse(b []byte) (string, bool) {

	// 去掉
	s := strings.TrimSpace(string(b))
	if s == "[Done]" {
		return "", true
	}
	return strings.TrimPrefix(s, "data:"), false
}
