package powerai

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"orgine.com/ai-team/power-ai-framework-v4/env"
	etcd_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/etcd"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/milvus"
	minio_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/minio"
	pgsql_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/pgsql"
	redis_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/redis"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
	weaviate_mw "orgine.com/ai-team/power-ai-framework-v4/middleware/weaviate"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xconfig"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xdefense"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xinit"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlock"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xlog"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xmemory"
	"orgine.com/ai-team/power-ai-framework-v4/tools"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const Ver = "v1.0.27"

type AgentApp struct {
	Manifest    *Manifest
	HttpServer  *server.HttpServer
	OnShutdown  func(ctx context.Context)
	etcd        *etcd_mw.Etcd
	pgsql       *pgsql_mw.PgSql
	redis       *redis_mw.Redis
	minio       *minio_mw.Minio
	weaviate    *weaviate_mw.Weaviate
	milvus      *milvus_mw.Milvus
	agentConfig *AgentConfig
	agentClient *AgentClient
	mu          sync.Mutex
	// 记忆管理相关字段
	memoryConfig      *xconfig.MemoryConfig
	sessionLockMgr    *xlock.SessionLockManager
	sessionNormalizer *xdefense.SessionNormalizer
	messageBuilder    *xmemory.MessageBuilder
}

type Manifest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

func (a *AgentApp) Run() {

	go func() {
		if err := a.HttpServer.RunServer(env.G.HttpServerConfig.Ip, env.G.HttpServerConfig.Port); err != nil {
			xlog.LogErrorF("10000", "httpserver", "init", fmt.Sprintf("服务[%s:%s]启动失败", env.G.HttpServerConfig.Ip, env.G.HttpServerConfig.Port), err)
		}
	}()

	go a.agentClient.register(
		env.G.HttpServerConfig.Ip,
		env.G.HttpServerConfig.Port,
		a.Manifest.Code,
		a.Manifest.Name,
		a.Manifest.Version)

	// signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			a.HttpServer.StopServer()
			if a.OnShutdown != nil {
				a.OnShutdown(context.Background())
			}
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}

func NewAgent(manifest string, opts ...Option) (*AgentApp, error) {
	mf, err := initManifest(manifest)
	if err != nil {
		return nil, err
	}
	newOpts := newOptions(opts)
	// 配置初始化环境变量
	env.Init()
	// 工具初始化
	tools.Init()
	// 初始化 etcd
	etcd, err := initEtcd()
	if err != nil {
		return nil, fmt.Errorf("init etcd middleware err:%s", err.Error())
	}

	// 初始化记忆管理工具类
	memoryInitResult := xinit.InitMemoryManager()
	if memoryInitResult.Error != nil {
		xlog.LogWarnF("INIT", "NewAgent", "InitMemoryManager",
			fmt.Sprintf("failed to init memory manager: %v, using default config", memoryInitResult.Error))
		// 使用默认配置
		memoryInitResult.Config = xconfig.GetConfig()
		memoryInitResult.LockManager = xlock.NewSessionLockManager()
		memoryInitResult.MessageBuilder = xmemory.NewMessageBuilder(200, 100)
	}

	a := &AgentApp{
		Manifest:    mf,
		HttpServer:  server.New(),
		OnShutdown:  newOpts.OnShutDown,
		etcd:        etcd,
		agentConfig: newAgentConfig(etcd, mf.Code, newOpts.DefaultConfigs, newOpts.ConfigChangeCallbacks),
		agentClient: newAgentClient(etcd),
		// 记忆管理相关字段
		memoryConfig:      memoryInitResult.Config,
		sessionLockMgr:    memoryInitResult.LockManager,
		sessionNormalizer: xdefense.NewSessionNormalizer(memoryInitResult.Config.MemoryModeFullHistory),
		messageBuilder:    memoryInitResult.MessageBuilder,
	}

	// 生成base_url
	baseUrl := strings.ReplaceAll(mf.Code, "-", "/")
	a.HttpServer.GET(fmt.Sprintf("/%s/health", baseUrl), a.health)
	a.HttpServer.GET(fmt.Sprintf("/%s/version", baseUrl), a.version)
	for k, v := range newOpts.PostRouters {
		a.HttpServer.POST(fmt.Sprintf("/%s/%s", baseUrl, k), v)
	}
	for k, v := range newOpts.GetRouters {
		a.HttpServer.GET(fmt.Sprintf("/%s/%s", baseUrl, k), v)
	}
	return a, nil
}

func (a *AgentApp) health(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"code":    server.ResultSuccess.Code,
		"message": server.ResultSuccess.Message,
	})
}

func (a *AgentApp) version(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"code":    server.ResultSuccess.Code,
		"message": server.ResultSuccess.Message,
		"data": map[string]string{
			"agent_code":        a.Manifest.Code,
			"agent_name":        a.Manifest.Name,
			"agent_version":     a.Manifest.Version,
			"framework_version": Ver,
		},
	})
}
