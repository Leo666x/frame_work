package powerai

import (
	"orgine.com/ai-team/power-ai-framework-v4/middleware/redis"
)

func (a *AgentApp) GetRedisClient() (*redis_mw.Redis, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.redis == nil {
		client, err := initRedis(a.etcd)
		if err != nil {
			return nil, err
		}
		a.redis = client
	}
	return a.redis, nil
}
