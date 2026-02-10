# 多轮对话与意图识别对接规划（v1.1）

目标：在多意图、多轮对话场景下，保证“路由清晰、执行可控、便于改动”，并形成可回收的任务闭环。

---

## 1) 角色与边界
- **Decision（意图识别）**
  - 输入：用户 Query + 记忆（M1/M2/M3）
  - 内部：意图解析（分段+实体+规则/LLM，后续可替换为小模型）
  - 输出：frames + intent_ops + medical_focus_id + safety

- **Task Orchestrator（任务编排）**
  - 负责执行顺序、调度、回收
  - 将 frame 转为可执行任务

- **Agent Executor（业务智能体）**
  - 执行具体业务逻辑
  - 返回结果、状态、槽位更新

- **Memory Store（M1/M2/M3）**
  - 记录上下文、帧摘要与结构化状态

- **Safety Gate（并行安全检查）**
  - 与意图解析并行执行
  - 具备中断权限：直接返回特定值/提示

---

## 2) 任务生命周期（Task Lifecycle）
状态建议：
- **queued**：等待执行
- **active**：正在执行
- **awaiting_user**：等待用户补槽/确认
- **completed**：任务完成（可回收）
- **canceled**：被用户/系统取消

“消耗掉”规则：
- 当 Agent 返回 `status=completed` 且 `missing_slots=[]` → 任务可回收
- 如果 Agent 返回 `awaiting_user` → 不可回收，保留在 M3
- 用户主动取消 → 直接 canceled 并回收

---

## 3) 对接接口（最小契约）

### 3.1 Dispatch（发给业务方）
```
TaskDispatch {
  conversation_id,
  task_id,
  agent_code,
  priority,
  parent_task_id,      // 允许任务链
  slots,               // 结构化输入
  context_summary,     // 简要上下文（M3）
  memory_ref           // M2/M3 索引引用
}
```

### 3.2 Result（业务方返回）
```
TaskResult {
  task_id,
  status: completed|awaiting_user|canceled,
  slots_update,
  missing_slots,
  handoff_suggestion,  // 只建议，不强制改路由
  user_response        // 回答或澄清
}
```

---

## 4) 多轮对话执行逻辑（推荐）
0) **Safety Gate 并行检查**
   - 若触发 BLOCK/EMERGENCY → 直接返回，不进入后续
1) Decision 生成 frames + intent_ops
2) Orchestrator 生成任务队列（按 priority 排序）
3) 若有 clarify → 停止调度，仅向用户提问
4) 依次 dispatch queued 任务
5) 任务完成 → 回收；待补槽 → 保留
6) 每次任务结果更新 M3/M2（结果 + 摘要）

---

## 5) 优先级与抢占（必要规则）
优先级建议：
1) **安全/紧急**（终止一切任务）
2) **医疗 focus**（唯一 focus）
3) **支付/排队**（高优先级 admin）
4) **其他插入任务**

抢占规则：
- payment/queue 可短暂插入，不允许替换 medical focus

---

## 6) 记忆对接位置（M1/M2/M3）
- **M1 全量历史**：默认加载全量对话；当 token 达阈值后切换为 `summary + 最近N轮`
- **M2 向量摘要**：任务完成或进入 awaiting_user 时写入
- **M3 结构状态**：每次 TaskResult 都要更新

---

## 7) 变更与扩展的闭环
新增智能体时，需完成：
1) AgentSpec（意图边界/记忆策略/样例）
2) Decision 规则/候选配置
3) 回放样本集新增
4) Orchestrator 增加路由规则（如优先级/可插入）

---

## 8) 复杂场景示例（多意图）
用户：“我头痛想挂号，顺便问停车怎么收费，前面还有几个人？”

Decision 输出：
- triage → focus (medical)
- smartCS → queued (admin)
- queue → queued (admin)

Orchestrator 执行：
1) dispatch triage
2) triage 完成/补槽后
3) dispatch smartCS
4) dispatch queue

若 triage 返回 awaiting_user（缺 duration）：
→ 暂停其他任务，等待用户补槽后再继续

---

## 9) 流程图（含并行 Safety Gate）
```mermaid
graph TD
  A[User Query] --> N[Normalize]
  N --> P[Intent Parsing: segmentation + entity + rule/LLM]
  N --> S[Safety Gate (parallel)]

  S -->|BLOCK/EMERGENCY| R0[Return Safety Response]
  S -->|SAFE| P

  P --> M3[M3 Structured State Align]
  M3 -->|bind| F[Bind to Frame]
  M3 -->|no bind| M1[M1 Full History]
  M1 --> M2[M2 Session Embedding Match]
  M2 -->|match| F
  M2 -->|no match| L4[Candidate Routing]

  F --> R[Relation Judge]
  L4 --> R

  R --> O[Generate intent_ops]
  O --> X[Update Frames + Memory]
  X --> OUT[Return IntentContext]
```

