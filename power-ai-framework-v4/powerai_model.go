package powerai

import (
	"encoding/json"
	"fmt"
	"io"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xhttp"
	"orgine.com/ai-team/power-ai-framework-v4/tools"
)

const (
	SYSTEM_MODEL_LLM            = "system-llm"            //系统llm模型
	SYSTEM_MODEL_OCR            = "system-OCR"            //系统ocr模型
	SYSTEM_MODEL_ASR            = "system-asr"            //系统asr模型
	SYSTEM_MODEL_RERANK         = "system-rerank"         //系统rerank模型
	SYSTEM_MODEL_TEXT_EMBEDDING = "system-text-embedding" //系统text-embedding模型
	SYSTEM_MODEL_SPEECH2TEXT    = "system-speech2text"    //系统语音转文本模型
	SYSTEM_MODEL_TTS            = "system-tts"            //系统文本转语言模型
)

type SystemModel struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// GetSystemLlmConfig 获取系统llm模型
func (a *AgentApp) GetSystemLlmConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_LLM)
}

// GetSystemOCRConfig 系统ocr模型
func (a *AgentApp) GetSystemOCRConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_OCR)
}

// GetSystemAsrConfig 系统ocr模型
func (a *AgentApp) GetSystemAsrConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_ASR)
}

// GetSystemRerankConfig 系统rerank模型
func (a *AgentApp) GetSystemRerankConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_RERANK)
}

// GetSystemEmbeddingConfig 系统text-embedding模型
func (a *AgentApp) GetSystemEmbeddingConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_TEXT_EMBEDDING)
}

// GetSystemSpeech2textConfig 系统语音转文本模型
func (a *AgentApp) GetSystemSpeech2textConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_SPEECH2TEXT)
}

// GetSystemTtsConfig 系统文本转语言模型
func (a *AgentApp) GetSystemTtsConfig(enterpriseId string) (*SystemModel, error) {
	return a.getModelConfig(enterpriseId, SYSTEM_MODEL_TTS)
}

// getModelConfig 获取系统模型配置
func (a *AgentApp) getModelConfig(enterpriseId, confCode string) (*SystemModel, error) {
	v := a.GetSystemConfig(enterpriseId, confCode)
	if v == nil {
		return nil, fmt.Errorf("未查询到企业[%s],模型[%s]对应的值", enterpriseId, confCode)
	}

	sm := &SystemModel{}
	if err := json.Unmarshal([]byte(v.Value), sm); err != nil {
		return nil, fmt.Errorf("企业[%s],模型[%s]已查到对应的值，但json转换失败:%v", enterpriseId, confCode, err)
	}
	return sm, nil
}

// AsyncStreamCallSystemLLM 异步流式请求大语言模型 url, key, modelName string
func (a *AgentApp) AsyncStreamCallSystemLLM(enterpriseId string, request map[string]interface{}, handler xhttp.HttpRequestResponseFunc) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_LLM)

	if err != nil {
		handler(nil, err)
		return
	}

	tools.AsyncStreamCallSystemLLM(c.URL, c.Key, c.Name, request, handler)
}

// SyncStreamCallSystemLLM 同步流式请求大语言模型 url, key, modelName string,
func (a *AgentApp) SyncStreamCallSystemLLM(enterpriseId string, request map[string]interface{}, handler xhttp.HttpRequestResponseFunc) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_LLM)

	if err != nil {
		handler(nil, err)
		return
	}
	tools.SyncStreamCallSystemLLM(c.URL, c.Key, c.Name, request, handler)
}

// SyncCallSystemLLM 同步非流式请求大语言模型 url, key, modelName string,
func (a *AgentApp) SyncCallSystemLLM(enterpriseId string, request map[string]interface{}) (string, error) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_LLM)
	if err != nil {
		return "", err
	}
	return tools.SyncCallSystemLLM(c.URL, c.Key, c.Name, request)
}

// SyncStreamCallOCRLLM 同步非流式请求大语言模型 url, key, modelName string,
func (a *AgentApp) SyncStreamCallOCRLLM(enterpriseId string, request map[string]interface{}) (string, error) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_OCR)
	if err != nil {
		return "", err
	}
	return tools.SyncCallSystemLLM(c.URL, c.Key, c.Name, request)
}

func (a *AgentApp) SyncCallSystemASRFromReader(enterpriseId, filename string, fileReader io.Reader) (string, error) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_ASR)
	if err != nil {
		return "", err
	}
	return tools.SyncCallSystemASRFromReader(c.URL, c.Key, c.Name, filename, fileReader)
}

// AsyncRequestCallSystemTextEmbedding 异步请求TextEmbedding模型
func (a *AgentApp) AsyncRequestCallSystemTextEmbedding(enterpriseId string, request map[string]interface{}, handler xhttp.HttpRequestResponseFunc) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_TEXT_EMBEDDING)
	if err != nil {
		handler(nil, err)
		return
	}
	tools.AsyncRequestCallSystemTextEmbedding(c.URL, c.Key, c.Name, request, handler)
}

// SyncRequestCallSystemTextEmbedding 同步请求TextEmbedding模型
func (a *AgentApp) SyncRequestCallSystemTextEmbedding(enterpriseId string, request map[string]interface{}) (string, error) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_TEXT_EMBEDDING)
	if err != nil {
		return "", err
	}
	return tools.SyncRequestCallSystemTextEmbedding(c.URL, c.Key, c.Name, request)
}

// SyncRequestCallSystemRerank 同步请求Rerank模型
func (a *AgentApp) SyncRequestCallSystemRerank(enterpriseId string, request map[string]interface{}) (string, error) {
	c, err := a.getModelConfig(enterpriseId, SYSTEM_MODEL_RERANK)
	if err != nil {
		return "", err
	}
	return tools.SyncRequestCallSystemRerank(c.URL, c.Key, c.Name, request)
}

// EmbedTexts 调用 bge-m3 接口将 texts 向量化
func (a *AgentApp) EmbedTexts(enterpriseId string, texts []string) ([][]float32, error) {
	req := map[string]interface{}{"input": texts}
	raw, err := a.SyncRequestCallSystemTextEmbedding(enterpriseId, req)
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
		return nil, fmt.Errorf("embedding 数量 %d 与文本数 %d 不匹配 详情：%+v", len(resp.Data), len(texts), resp.Data)
	}
	vecs := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		vecs[i] = d.Embedding
	}
	return vecs, nil
}

// RerankResults 调用 bge-reranker-v2-m3 接口，对 docs 做重排序
func (a *AgentApp) RerankResults(enterpriseId string, query string, docs []string) ([]float64, error) {
	req := map[string]interface{}{"query": query, "documents": docs}
	raw, err := a.SyncRequestCallSystemRerank(enterpriseId, req)
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
	for _, r := range resp.Results {
		idx := r.Index
		scores[idx] = r.RelevanceScore
	}
	return scores, nil
}
