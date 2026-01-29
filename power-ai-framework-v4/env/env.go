package env

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xenv"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xhttp"
	"strings"
	"time"
)

var G *Env

func Init() {

	G = _default() //获取默认值

	return
}

func _default() *Env {

	var etcdEndpoints []string
	etcdEndpointsEnvString := xenv.GetEnvOrDefault("POWER_AI_ETCD_ENDPOINTS", "")
	if etcdEndpointsEnvString == "" {
		etcdEndpoints = append(etcdEndpoints,
			fmt.Sprintf("%s:%s",
				xenv.GetEnvOrDefault("POWER_AI_ETCD_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
				xenv.GetEnvOrDefault("POWER_AI_ETCD_PORT", "39001"),
			),
		)
	} else {
		etcdEndpoints = strings.Split(etcdEndpointsEnvString, ",")
	}

	return &Env{
		EtcdConfig: &EtcdConfig{
			Endpoints: etcdEndpoints,
			//port: xenv.GetEnvOrDefault("POWER_AI_ETCD_PORT", "39001"),
			//ip:   xenv.GetEnvOrDefault("POWER_AI_ETCD_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
		},
		PgsqlConfig: &PgsqlConfig{
			Username:        "postgres",
			Password:        xenv.GetEnvOrDefault("POWER_AI_POSTGRES_PASSWORD", "1qaz@WSX"),
			Database:        "powerai",
			Host:            xenv.GetEnvOrDefault("POWER_AI_POSTGRES_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
			Port:            xenv.GetEnvOrDefault("POWER_AI_POSTGRES_PORT", "39002"),
			MaxOpenConns:    xenv.GetEnvOrDefaultInt("POSTGRES_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    xenv.GetEnvOrDefaultInt("POSTGRES_MAX_IDLE_CONNS", 10),
			MaxConnLifetime: time.Duration(xenv.GetEnvOrDefaultInt("POSTGRES_MAX_CONN_LIFETIME", 30)) * time.Minute,
		},
		MinioConfig: &MinioConfig{
			Ip:   xenv.GetEnvOrDefault("POWER_AI_MINIO_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
			Port: xenv.GetEnvOrDefault("POWER_AI_MINIO_API_PORT", "39004"),
			Ak:   xenv.GetEnvOrDefault("POWER_AI_MINIO_USERNAME", "powerai"),
			Sk:   xenv.GetEnvOrDefault("POWER_AI_MINIO_PASSWORD", "1qaz@WSX"),
		},
		CommonHttpClientConfig: &xhttp.HttpClientConfig{
			Timeout:          time.Duration(xenv.GetEnvOrDefaultInt("HTTP_CLIENT_TIMEOUT", 600)) * time.Second,
			Compressed:       xenv.GetEnvOrDefaultBool("HTTP_CLIENT_COMPRESSED", false),
			HandshakeTimeout: time.Duration(xenv.GetEnvOrDefaultInt("HTTP_CLIENT_HANDSHAKE_TIMEOUT", 600)) * time.Second,
			ResponseTimeout:  time.Duration(xenv.GetEnvOrDefaultInt("HTTP_CLIENT_RESPONSE_TIMEOUT", 600)) * time.Second,
		},
		StreamHttpClientConfig: &xhttp.HttpClientConfig{
			Timeout:          time.Duration(xenv.GetEnvOrDefaultInt("HTTP_LLM_CLIENT_TIMEOUT", 600)) * time.Second,
			Compressed:       xenv.GetEnvOrDefaultBool("HTTP_LLM_CLIENT_COMPRESSED", false),
			HandshakeTimeout: time.Duration(xenv.GetEnvOrDefaultInt("HTTP_LLM_CLIENT_HANDSHAKE_TIMEOUT", 600)) * time.Second,
			ResponseTimeout:  time.Duration(xenv.GetEnvOrDefaultInt("HTTP_LLM_CLIENT_RESPONSE_TIMEOUT", 600)) * time.Second,
		},
		HttpServerConfig: &HttpServerConfig{
			Ip:   xenv.GetEnvOrDefault("IP_ADDR_DEBUG", xenv.GetEnvOrDefault("IP_ADDR", "0.0.0.0")),
			Port: xenv.GetEnvOrDefault("PORT", "40000"),
		},
		WeaviateConfig: &WeaviateConfig{
			Host: fmt.Sprintf("%s:%s",
				xenv.GetEnvOrDefault("POWER_AI_WEAVIATE_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
				xenv.GetEnvOrDefault("POWER_AI_WEAVIATE_HTTP_PORT", "39006")),
			Scheme: "http",
			ApiKey: xenv.GetEnvOrDefault("POWER_AI_WEAVIATE_AUTHENTICATION_APIKEY_ALLOWED_KEYS", "WVF5YThaHlkYwhGUSmCRgsX3tD5ngdN8pkih"),
		},
		RedisConfig: &RedisConfig{
			Addr: fmt.Sprintf("%s:%s",
				xenv.GetEnvOrDefault("POWER_AI_REDIS_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
				xenv.GetEnvOrDefault("POWER_AI_REDIS_PORT", "39009")),
			Password: "http",
		},
		MilvusConfig: &MilvusConfig{
			Addr: fmt.Sprintf("%s:%s",
				xenv.GetEnvOrDefault("POWER_AI_MILVUS_HOST", xenv.GetEnvOrDefault("IP_ADDR", "127.0.0.1")),
				xenv.GetEnvOrDefault("POWER_AI_MILVUS_PORT", "39008")),
			Password: xenv.GetEnvOrDefault("POWER_AI_MILVUS_PASSWORD", ""),
			Username: xenv.GetEnvOrDefault("POWER_AI_MILVUS_USERNAME", ""),
			Timeout:  time.Duration(xenv.GetEnvOrDefaultInt("POWER_AI_MILVUS__TIMEOUT", 10)) * time.Second,
		},
	}
}

type Env struct {
	EtcdConfig             *EtcdConfig
	PgsqlConfig            *PgsqlConfig
	MinioConfig            *MinioConfig
	CommonHttpClientConfig *xhttp.HttpClientConfig
	StreamHttpClientConfig *xhttp.HttpClientConfig
	HttpServerConfig       *HttpServerConfig
	WeaviateConfig         *WeaviateConfig
	RedisConfig            *RedisConfig
	MilvusConfig           *MilvusConfig
}

type EtcdConfig struct {
	Endpoints []string
}

type PgsqlConfig struct {
	Username        string
	Password        string
	Database        string
	Host            string
	Port            string
	MaxOpenConns    int
	MaxIdleConns    int
	MaxConnLifetime time.Duration
}

type MinioConfig struct {
	Ip   string
	Port string
	Ak   string
	Sk   string
}

type HttpServerConfig struct {
	Ip   string
	Port string
}

type WeaviateConfig struct {
	Host   string
	Scheme string
	ApiKey string
}
type MilvusConfig struct {
	Addr     string
	Username string
	Password string
	Timeout  time.Duration
}
type RedisConfig struct {
	Addr     string
	Password string
}

func FmtToYml(i interface{}) string {
	b, _ := yaml.Marshal(i)
	return string(b)
}
