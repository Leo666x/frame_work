# 后台服务 vs Agent 职责边界图

> 目的：清晰区分“后台基础设施服务”和“Agent 业务服务”的职责

```mermaid
graph TD
  %% ===== Backend Services =====
  subgraph BACKEND[后台基础设施服务 (Docker / 独立进程)]
    E[etcd
配置中心 + 服务发现]
    P[Postgres
会话/消息/系统配置]
    R[Redis
短期记忆/缓存]
    W[Weaviate/Milvus
向量检索]
    M[MinIO
对象存储]
  end

  %% ===== Agent =====
  subgraph AGENT[Agent 服务 (业务逻辑进程)]
    A[AgentApp
业务入口 + 路由]
    A1[意图识别 / LLM / 规则逻辑]
    A2[数据访问封装
powerai_db.go]
    A3[短期记忆
powerai_short_memory.go]
    A4[向量检索封装
powerai_weaviate.go / powerai_milvus.go]
    A5[对象存储封装
powerai_minio.go]
  end

  %% ===== Connections =====
  A --> E
  A2 --> P
  A3 --> R
  A4 --> W
  A5 --> M

  %% ===== Notes =====
  N[核心结论
后台提供能力
Agent 调用能力做业务]:::note
  N -.-> A

  classDef note fill:#fff3cd,stroke:#f0ad4e,color:#000;
```

## 一句话总结
- **后台服务**：只提供基础能力（数据库/缓存/向量库/对象存储/配置中心）。
- **Agent 服务**：只做业务逻辑，通过 SDK/客户端连接后台能力。

## 在本项目中的体现
- 后台服务不会在 Agent 里启动，只会被连接：`powerai_internal.go`
- Agent 业务调用封装在 `powerai_*.go`

如果你需要，我可以继续补一张“部署视角（两台机器 + Docker + Agent）”的图。
