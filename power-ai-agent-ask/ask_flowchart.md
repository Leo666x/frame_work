# power-ai-agent-ask 流程图（graph TD）

```mermaid
graph TD
  A[send_msg 入口] --> B[校验 inputs.patient_id]
  B -->|缺失| B1[返回错误]
  B --> C[智能客服回答
GextractQAXHYYAnswer]
  C --> C1[从 Weaviate(QAXHYY) 取知识]
  C1 --> C2[拼 Prompt]
  C2 --> C3[调用 LLM 生成 answer]

  C3 --> D[二分类判断
是否“即问即办”]
  D -->|否| D1[直接返回 answer]
  D -->|是| E[16类意图识别]

  E --> F[查向量库
ReadKnowledge(Qa_data_get)]
  F --> G[拿 card_type / function_name]
  G --> H[返回文本 + 卡片]
  H --> I[Done]
```
