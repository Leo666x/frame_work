# power-ai-agent-ask-fit-v4 架构图（graph TD）

```mermaid
graph TD
  %% ===== Entry =====
  A[HTTP /send_msg
v4 Agent入口] --> B[DoValidateAgentRequest
校验请求 + SSE]

  %% ===== Validation =====
  B --> C[必填检查
patient_id / ids]
  C -->|失败| C1[错误返回]

  %% ===== QA Answer (Milvus + LLM) =====
  C --> D[QAXHYY 知识检索]
  D --> D1[EmbedTexts (bge-m3)]
  D1 --> D2[Milvus Search
collection: QAXHYY]
  D2 --> D3[拼接知识]
  D3 --> D4[LLM 生成 answer]

  %% ===== Intent Classification =====
  D4 --> E[二分类 LLM
是否即问即办]
  E -->|否| E1[直接返回 answer]
  E -->|是| F[多分类 LLM
意图名称]

  %% ===== Card Retrieval =====
  F --> G[Milvus 检索卡片]
  G --> G1[EmbedTexts (bge-m3)]
  G1 --> G2[Milvus Search
collection: qa_data_get]
  G2 --> H[获取 card_type/function_name]

  %% ===== Response =====
  H --> I[BuildLegacy
旧版结构输出]
  I --> J[Text + Card]
  J --> K[Done]

  %% ===== Config =====
  subgraph CONF[配置与规则]
    P1[double_class_prompt]
    P2[question_classfication_prompt]
    P3[intention_category]
    P4[return_solid_text]
    P5[qa_xhyy_agent_QAXHYYExtraction_prompt]
    P6[qa_xhyy_agent_topk_conf]
  end
  CONF --> D
  CONF --> E
  CONF --> F
  CONF --> I

  %% ===== Backend =====
  subgraph BACKEND[基础设施]
    M1[Milvus]
    M2[LLM]
    M3[etcd]
  end
  BACKEND --> D
  BACKEND --> G
  BACKEND --> CONF

```
