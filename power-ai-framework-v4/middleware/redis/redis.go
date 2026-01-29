package redis_mw

import (
	"github.com/go-redis/redis/v7"
	"time"
)

type Redis struct {
	client *redis.Client
	config *Config
}
type Config struct {
	Addr     string
	Password string
}

func New(c *Config) (*Redis, error) {

	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       0,
	})
	return &Redis{client: client, config: c}, nil
}

// Set 基础新增：设置key-value，支持过期时间，已存在则覆盖
// expiration：过期时间（秒），0表示永不过期
func (r *Redis) Set(key string, value any, expiration int64) error {
	exp := time.Duration(expiration) * time.Second
	if expiration <= 0 {
		exp = 0 // 永不过期
	}
	return r.client.Set(key, value, exp).Err()
}

// SetNX 不存在则新增：仅当key不存在时设置，返回是否设置成功
// expiration：过期时间（秒），0表示永不过期
func (r *Redis) SetNX(key string, value any, expiration int64) (bool, error) {
	exp := time.Duration(expiration) * time.Second
	if expiration <= 0 {
		exp = 0 // 永不过期
	}
	return r.client.SetNX(key, value, exp).Result()
}

// Get 查询指定key的value，返回字符串结果
// 若key不存在，返回空字符串和对应的错误（redis.Nil）
func (r *Redis) Get(key string) (string, error) {
	return r.client.Get(key).Result()
}

// Exists 查询一个或多个key是否存在，返回存在的key数量
func (r *Redis) Exists(keys ...string) (int64, error) {
	return r.client.Exists(keys...).Result()
}

// Del 批量删除指定key，返回成功删除的key数量
func (r *Redis) Del(keys ...string) (int64, error) {
	return r.client.Del(keys...).Result()
}

// Close 关闭Redis连接，释放资源
func (r *Redis) Close() error {
	return r.client.Close()
}
