package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xhttp"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xjson"
	"regexp"
	"strings"
	"sync/atomic"
)

// AsyncStreamCallSystemLLM 异步流式请求大语言模型
func AsyncStreamCallSystemLLM(url, key, modelName string, request map[string]interface{}, handler xhttp.HttpRequestResponseFunc) {
	request["model"] = modelName
	request["stream"] = true
	r := noThinkLLMReq(url, key, request)
	asyncStream(r, handler)
}

// SyncStreamCallSystemLLM 同步流式请求大语言模型
func SyncStreamCallSystemLLM(url, key, modelName string, request map[string]interface{}, handler xhttp.HttpRequestResponseFunc) {
	request["model"] = modelName
	request["stream"] = true
	r := noThinkLLMReq(url, key, request)
	syncStream(r, handler)
}

// SyncCallSystemLLM 同步非流式请求大语言模型
func SyncCallSystemLLM(url, key, modelName string, request map[string]interface{}) (string, error) {
	request["model"] = modelName
	request["stream"] = false
	request["enable_thinking"] = false
	r := noThinkLLMReq(url, key, request)
	return syncRequest(r)
}

// AsyncRequestCallSystemTextEmbedding 异步请求TextEmbedding模型
func AsyncRequestCallSystemTextEmbedding(url, key, modelName string, request map[string]interface{}, handler xhttp.HttpRequestResponseFunc) {
	go func() {
		request["model"] = modelName
		body, _ := json.Marshal(request)
		r := &xhttp.HttpRequest{
			RawURL: url,
			Method: "POST",
			Body:   body,
			Headers: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", key)},
			},
		}
		rtn, err := syncRequest(r)
		if err != nil {
			handler(nil, err)
		} else {
			handler([]byte(rtn), err)
		}
	}()
}

// SyncRequestCallSystemTextEmbedding 同步请求TextEmbedding模型
func SyncRequestCallSystemTextEmbedding(url, key, modelName string, request map[string]interface{}) (string, error) {
	request["model"] = modelName
	body, _ := json.Marshal(request)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   body,
		Headers: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", key)},
		},
	}
	return syncRequest(r)
}

// SyncRequestCallSystemRerank 同步请求Rerank模型
func SyncRequestCallSystemRerank(url, key, modelName string, request map[string]interface{}) (string, error) {
	request["model"] = modelName
	body, _ := json.Marshal(request)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   body,
		Headers: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", key)},
		},
	}
	return syncRequest(r)
}

// EmbedTexts 调用 bge-m3 接口将 texts 向量化
func EmbedTexts(url, key, modelName string, texts []string) ([][]float32, error) {
	req := map[string]interface{}{"input": texts}
	raw, err := SyncRequestCallSystemTextEmbedding(url, key, modelName, req)
	if err != nil {
		return nil, fmt.Errorf("embedding 调用失败: %v", err)
	}
	var resp struct {
		Data []struct{ Embedding []float32 } `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("解析 embedding 结果失败: %w", err)
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embedding 数量 %d 与文本数 %d 不匹配", len(resp.Data), len(texts))
	}
	vecs := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		vecs[i] = d.Embedding
	}
	return vecs, nil
}

// RerankResults 调用 bge-reranker-v2-m3 接口，对 docs 做重排序
func RerankResults(url, key, modelName string, query string, docs []string) ([]float64, error) {
	req := map[string]interface{}{"query": query, "documents": docs}
	raw, err := SyncRequestCallSystemRerank(url, key, modelName, req)
	if err != nil {
		return nil, fmt.Errorf("重排序调用失败: %w", err)
	}
	var resp struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("解析重排序结果失败: %w", err)
	}
	scores := make([]float64, len(resp.Results))
	for i, r := range resp.Results {
		scores[i] = r.RelevanceScore
	}
	return scores, nil
}

func noThinkLLMReq(url, key string, request map[string]interface{}) *xhttp.HttpRequest {
	if request["messages"] != nil {
		messages := request["messages"].([]interface{})
		for i := range messages {

			switch msg := messages[i].(type) {
			case map[string]interface{}:
				if msg["role"] == "user" {
					if content, ok := msg["content"].(string); ok {
						msg["content"] = content + "/no_think"
					}
				}
			case map[string]string:
				if msg["role"] == "user" {
					msg["content"] = msg["content"] + "/no_think"
				}
			}
		}
	}
	body, _ := json.Marshal(request)
	r := &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   body,
		Headers: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", key)},
		},
	}
	return r
}

// asyncStream 异步流式请求
func asyncStream(request *xhttp.HttpRequest, handler xhttp.HttpRequestResponseFunc) {
	var index atomic.Int64
	StreamCommonHttpClient.SendReqByAsyncRespStream(request, func(bytes []byte, err error) bool {
		index.Add(1)
		// 如果错误，之直接返回
		if err != nil {
			return handler(bytes, err)
		}
		// 如果bytes为空，则直接返回
		if bytes == nil {
			return handler(bytes, err)
		}
		if index.Load() <= 4 {
			content := xjson.Get(string(bytes), "choices.0.delta.content")
			if content.String() == "<think>" || content.String() == "\n\n" || content.String() == "</think>" {
				//直接跳过
				return true
			} else {
				return handler(bytes, nil)
			}
		}
		return handler(bytes, nil)
	})
}

// syncStream 同步流式请求
func syncStream(request *xhttp.HttpRequest, handler xhttp.HttpRequestResponseFunc) {
	var index atomic.Int64
	StreamCommonHttpClient.SendReqBySyncRespStream(request, func(bytes []byte, err error) bool {
		index.Add(1)
		// 如果错误，之直接返回
		if err != nil {
			return handler(bytes, err)
		}
		// 如果bytes为空，则直接返回
		if bytes == nil {
			return handler(bytes, err)
		}
		if index.Load() <= 4 {
			content := xjson.Get(string(bytes), "choices.0.delta.content")
			if content.String() == "<think>" || content.String() == "\n\n" || content.String() == "</think>" {
				return true
			} else {
				return handler(bytes, nil)
			}
		}

		return handler(bytes, err)
	})
}

// syncRequest 同步请求
func syncRequest(r *xhttp.HttpRequest) (string, error) {

	resp, err := StreamCommonHttpClient.SendReqByRespString(r)
	// 如果错误，则直接返回
	if err != nil {
		return "", err
	}

	content := xjson.Get(resp, "choices.0.message.content")
	contentStr := strings.TrimSpace(content.String())
	newContent := regexp.MustCompile(`(?s)<think>.*?</think>\s*`).ReplaceAllString(contentStr, "")
	newContent = regexp.MustCompile("<think>\n").ReplaceAllString(newContent, "")
	newResp, err := xjson.Set(resp, "choices.0.message.content", newContent)
	// 如果重置失败，则返回原始内容
	if err != nil {
		return resp, nil
	} else {
		// 重置成功，返回新内容
		return newResp, nil
	}
}
func SyncCallSystemASRFromReader(url, key, modelName, filename string, fileReader io.Reader) (string, error) {
	r, err := buildASRReqFromReader(url, key, modelName, filename, fileReader)
	if err != nil {
		return "", err
	}
	return StreamCommonHttpClient.SendReqByRespString(r)
}

func buildASRReqFromReader(url, key, modelName, filename string, fileReader io.Reader) (*xhttp.HttpRequest, error) {
	if modelName == "" {
		return nil, fmt.Errorf("modelName is empty")
	}
	if filename == "" {
		filename = "audio"
	}
	if fileReader == nil {
		return nil, fmt.Errorf("fileReader is nil")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// OpenAI 格式必需字段：model
	if err := writer.WriteField("model", modelName); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("write field model failed: %w", err)
	}

	// file 字段
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("create form file failed: %w", err)
	}
	if _, err := io.Copy(part, fileReader); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("copy file failed: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer failed: %w", err)
	}

	return &xhttp.HttpRequest{
		RawURL: url,
		Method: "POST",
		Body:   buf.Bytes(),
		Headers: map[string][]string{
			"accept":        {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", key)},
			"Content-Type":  {writer.FormDataContentType()},
		},
	}, nil
}
