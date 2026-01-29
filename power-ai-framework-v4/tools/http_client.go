package tools

import (
	"encoding/json"
	"net/http"
	"orgine.com/ai-team/power-ai-framework-v4/env"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xhttp"
)

var DefaultCommonHttpClient *xhttp.HttpClient
var StreamCommonHttpClient *xhttp.HttpClient

func Init() {
	DefaultCommonHttpClient = xhttp.NewHttpClientWithConfig(&xhttp.HttpClientConfig{
		Timeout:          env.G.CommonHttpClientConfig.Timeout,
		SSLEnabled:       env.G.CommonHttpClientConfig.SSLEnabled,
		TLSConfig:        env.G.CommonHttpClientConfig.TLSConfig,
		Compressed:       env.G.CommonHttpClientConfig.Compressed,
		HandshakeTimeout: env.G.CommonHttpClientConfig.HandshakeTimeout,
		ResponseTimeout:  env.G.CommonHttpClientConfig.ResponseTimeout,
		Verbose:          env.G.CommonHttpClientConfig.Verbose,
		Proxy:            env.G.CommonHttpClientConfig.Proxy,
	})
	StreamCommonHttpClient = xhttp.NewHttpClientWithConfig(&xhttp.HttpClientConfig{
		Timeout:          env.G.StreamHttpClientConfig.Timeout,
		SSLEnabled:       env.G.StreamHttpClientConfig.SSLEnabled,
		TLSConfig:        env.G.StreamHttpClientConfig.TLSConfig,
		Compressed:       env.G.StreamHttpClientConfig.Compressed,
		HandshakeTimeout: env.G.StreamHttpClientConfig.HandshakeTimeout,
		ResponseTimeout:  env.G.StreamHttpClientConfig.ResponseTimeout,
		Verbose:          env.G.StreamHttpClientConfig.Verbose,
		Proxy:            env.G.StreamHttpClientConfig.Proxy,
	})
}

// ***************************************************************************************************************
//  http客户端相关方法
// ***************************************************************************************************************

// SendReqByRespDownload http请求下载
func SendReqByRespDownload(request *xhttp.HttpRequest, saveFilePath string) error {
	return DefaultCommonHttpClient.SendReqByRespDownload(request, saveFilePath)
}

// SendReqByAsyncRespStream http请求流式响应 异步结果返回
func SendReqByAsyncRespStream(request *xhttp.HttpRequest, handler xhttp.HttpRequestResponseFunc) {
	StreamCommonHttpClient.SendReqByAsyncRespStream(request, handler)
}

// SendReqBySyncRespStream http请求流式响应 同步结果返回
func SendReqBySyncRespStream(request *xhttp.HttpRequest, handler xhttp.HttpRequestResponseFunc) {
	StreamCommonHttpClient.SendReqBySyncRespStream(request, handler)
}

// SendReqByRespStruct http请求，将响应转换成结构体
func SendReqByRespStruct(request *xhttp.HttpRequest, target any) error {
	return DefaultCommonHttpClient.SendReqByRespStruct(request, target)
}

// SendReqByRespString http请求，将响应转换成string
func SendReqByRespString(request *xhttp.HttpRequest) (string, error) {
	return DefaultCommonHttpClient.SendReqByRespString(request)
}

// SendReqByRespHttpResponse http请求，返回原生http响应
func SendReqByRespHttpResponse(request *xhttp.HttpRequest) (*http.Response, error) {
	return DefaultCommonHttpClient.SendReqByRespHttpResponse(request)
}

// SyncCallAgentByStream 同步调用智能体
func SyncCallAgentByStream(url string, request *server.AgentRequest, handler xhttp.HttpRequestResponseFunc) string {
	b, _ := json.Marshal(request)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   b,
	}

	StreamCommonHttpClient.SendReqBySyncRespStream(r, handler)
	return url
}

// AsyncCallAgentByStream 异步调用智能体
func asyncCallAgentByStream(url string, request *server.AgentRequest, handler xhttp.HttpRequestResponseFunc) string {
	b, _ := json.Marshal(request)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   b,
	}
	StreamCommonHttpClient.SendReqByAsyncRespStream(r, handler)
	return url
}

// CallAgentByHttp 非流式调用智能体
func CallAgentByHttp(url string, request *server.AgentRequest) (string, string, error) {
	b, _ := json.Marshal(request)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   b,
	}
	resp, err := StreamCommonHttpClient.SendReqByRespString(r)
	return resp, url, err
}
