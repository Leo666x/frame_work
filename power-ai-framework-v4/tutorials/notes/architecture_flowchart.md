graph TD
  %% ============ Entry & Boot ============
  A["NewAgent(manifest, opts...)<br/>(powerai.go)"] --> A1["initManifest<br/>校验 code/name/version/description"]
  A --> A2["env.Init()<br/>加载环境配置"]
  A --> A3["tools.Init()<br/>HTTP 客户端初始化"]
  A --> A4["initEtcd()<br/>连接 etcd"]
  A --> A5["newAgentConfig()<br/>配置缓存 + watch"]
  A --> A6["newAgentClient()<br/>服务发现 + watch"]
  A --> A7["注册路由<br/>health/version + custom"]

  %% ============ Run & Server ============
  A7 --> B["Run()<br/>启动服务 + 注册 + 信号监听"]
  B --> B1["HttpServer.RunServer<br/>Gin 启动"]
  B --> B2["agentClient.register<br/>服务注册 + KeepAlive"]

  %% ============ Request Path ============
  C["HTTP Request<br/>/{agent_code}/send_msg"] --> D["Handler (gin)<br/>自定义路由"]
  D --> D1["DoValidateAgentRequest<br/>校验请求 + SSE 构造"]
  D1 --> D2["SSEEvent.Write...<br/>流式输出"]
  D2 --> D3["SSEEvent.Done<br/>结束"]

  %% ============ Config & Discovery ============
  E["etcd<br/>/service/instance/*<br/>/agent/config/*<br/>/system/config/*"] --> A5
  E --> A6

  %% ============ Core Services ============
  D1 --> F["AgentApp 能力层"]

  F --> F1["Call Agent<br/>Sync/Async/Proxy<br/>(powerai_agent.go)"]
  F --> F2["DB<br/>Postgres<br/>(powerai_db.go)"]
  F --> F3["Short Memory<br/>Redis<br/>(powerai_short_memory.go)"]
  F --> F4["Model<br/>LLM/Embedding/Rerank<br/>(powerai_model.go)"]
  F --> F5["Object Storage<br/>MinIO<br/>(powerai_minio.go)"]
  F --> F6["Vector Search<br/>Weaviate/Milvus<br/>(powerai_weaviate.go / powerai_milvus.go)"]

  %% ============ Middlewares ============
  F2 --> G1["middleware/pgsql"]
  F3 --> G2["middleware/redis"]
  F5 --> G3["middleware/minio"]
  F6 --> G4["middleware/weaviate"]
  F6 --> G5["middleware/milvus"]
  F1 --> G6["middleware/etcd"]

  %% ============ Tools ============
  F4 --> H1["tools/http_llm.go<br/>LLM/Embedding/Rerank 调用"]
  F1 --> H2["tools/http_client.go<br/>通用 HTTP 调用"]

  %% ============ Notes ============
  N1["潜在问题<br/>- 无 Recovery 中间件<br/>- etcd 依赖强<br/>- 短期记忆结构偏医疗<br/>- think 过滤"]
  N1 -.-> B