# 即问即办结构对比图（重构前 vs 重构后）

```mermaid
graph TD
  %% ===== Old: Before =====
  subgraph OLD[旧结构：重构前]
    O0[send_msg] --> O1[校验 patient_id]
    O1 --> O2[智能客服回答
GextractQAXHYYAnswer]
    O2 --> O3[二分类 LLM
是否“即问即办”]
    O3 -->|否| O3a[直接返回 answer]
    O3 -->|是| O4[16类 LLM 分类]
    O4 --> O5[Weaviate 检索
ReadKnowledge(Qa_data_get)]
    O5 --> O6[返回卡片]
  end

  %% ===== New: After =====
  subgraph NEW[新结构：重构后]
    N0[send_msg] --> N1[校验 + 规则短路]
    N1 --> N2[二分类 LLM
+ 置信度阈值]
    N2 -->|低置信| N2a[兜底回复]
    N2 -->|高置信| N3[多类意图分类
输出结构化结果]
    N3 --> N4[Milvus 检索
Search(Collection: qa_data_get)]
    N4 --> N5[卡片结果 + 文本]
  end

  %% ===== Highlights =====
  H[改动亮点
- 规则短路降低成本
- 阈值兜底降低误判
- Weaviate -> Milvus
- 输出结构更稳定]:::note
  H -.-> OLD
  H -.-> NEW

  classDef note fill:#fff3cd,stroke:#f0ad4e,color:#000;
```
