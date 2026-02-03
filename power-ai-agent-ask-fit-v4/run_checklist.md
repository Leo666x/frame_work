# power-ai-agent-ask-fit-v4 运行检查清单

## 1) 环境变量
- `IP_ADDR` 或 `IP_ADDR_DEBUG`
- `PORT`
- `POWER_AI_ETCD_HOST` / `POWER_AI_ETCD_PORT`
- `POWER_AI_MILVUS_HOST` / `POWER_AI_MILVUS_PORT`
- `POWER_AI_MILVUS_USERNAME` / `POWER_AI_MILVUS_PASSWORD`
- 其它（如 Redis/MinIO/Weaviate）若未使用可忽略

## 2) etcd 配置（可选）
- 若使用 etcd 服务发现/配置：
  - 确保 `/service/instance/*` 正常
  - 如需覆盖模型/提示词/配置，请写入：
    - `/agent/config/_general_config_/{agent_code}/{enterprise}/{key}`
    - `/agent/config/_decision_config_/{agent_code}/{enterprise}/intention_category`

## 3) Milvus 准备
- Collection: `qa_data_get`
- Collection: `QAXHYY`
- 向量字段名：`embedding`
- 维度：1024（bge-m3）
- 输出字段：
  - `qa_data_get`: `card_type`, `function_name`
  - `QAXHYY`: `q`, `a`

## 4) LLM 配置
- 系统模型配置需存在（etcd 或系统配置）
  - `system-llm`
  - `system-text-embedding`

## 5) 启动检查
- 服务是否注册到 etcd
- `/power/ai/agent/ask/fit/send_msg` 路由可访问
- LLM 调用与 Milvus 检索是否正常返回

## 6) 最小请求校验
- `inputs.patient_id` 必须存在
- `sys_track_code` / `conversation_id` / `message_id` 等必填字段

---

> 如果你需要，我可以补一份：
> - 环境变量示例
> - 最小请求 JSON
> - Milvus 建表/入库脚本
