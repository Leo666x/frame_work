package powerai

import (
	"orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
)

func (a *AgentApp) GetMilvusClient() (*milvus_mw.Milvus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.minio == nil {
		client, err := initMilvus(a.etcd)
		if err != nil {
			return nil, err
		}
		a.milvus = client
	}
	return a.milvus, nil
}
