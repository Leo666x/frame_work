# 意图分解/分类/关系统一规范（v1.1）

目的：在医疗多意图场景下，统一“怎么拆句、怎么分类、怎么建关系”，保证准确率与可扩展性。

---

## 1) 术语定义

- **Intent Segment（意图片段）**：一句话里可独立执行的任务单元。
- **Intent Class（意图类别）**：映射到具体智能体（agent_code）。
- **Relation（意图关系）**：意图片段之间的结构关系（互斥/插入/并列/转移建议）。

---

## 2) 意图分解规则（Segmentation）
> 实现层面可将“分段 + 实体抽取 + 规则/LLM 判断”合并为**意图解析模块**；后续可替换为小模型，不影响接口。

### 2.1 基础切分信号
- 并列连接词：`并且/另外/还有/顺便/再问/同时`
- 语义转折：`但是/不过/先不/不说这个`
- 动作切换：`挂号/找医生/缴费/排队/问路/查报告/用药`

### 2.2 不切分的情况
- 同一任务的补充信息：`时间/地点/人名/科室/症状描述`
- 同一任务的确认/否定：`是/不是/对/不对/改成`

### 2.3 输出格式（示例）
Input:
“我头痛想挂号，顺便问停车怎么收费，还有李四医生明天出诊吗”
Output segments:
1) “头痛想挂号”
2) “停车怎么收费”
3) “李四医生明天出诊吗”

---

## 3) 意图分类体系（Classification）
> 当前分类由“规则 + LLM”共同判断；后续可替换为分类模型（接口不变）。

### 3.1 Domain → Task → Agent

**medical_service**
- triage：症状/疾病归属/项目咨询
- dept-direct：明确科室名
- doc-direct：明确医生名
- drug：药品用法/禁忌/相互作用/识图
- report：化验/影像指标解读

**admin_service**
- smartCS：流程/位置/医保/规则类咨询
- payment：明确支付动作
- queue：排队/候诊进度

### 3.2 分类优先规则（必执行）
1) 明确医生名 → doc-direct
2) 明确科室名 → dept-direct
3) 明确症状/疾病 → triage
4) 明确支付动作 → payment
5) 排队/候诊 → queue
6) 医保/流程/地点 → smartCS
7) 药物咨询 → drug
8) 报告解读 → report

> 注：doc vs dept 同句出现必须澄清（见关系规则）

---

## 4) 意图关系类型（Relations）

### 4.1 关系定义
- **互斥（Exclusive）**：不能同时成立，需澄清
  - doc-direct vs dept-direct
- **插入（Insertion）**：次要任务，不打断主任务
  - smartCS / queue / ask
- **并列（Parallel）**：可以并行，需排序
  - payment + queue
  - triage + smartCS
- **转移建议（Handoff）**：由 Agent 结果发出，非识别阶段的强绑定
  - report -> dept-direct（建议）

### 4.2 关系输出规则
- 互斥 → 输出 clarify
- 插入 → 医疗 focus 不变，任务入队
- 并列 → 医疗 focus + 队列（按优先级排序）
- 转移建议 → Agent 输出时写入 handoff，不改变识别结果

---

## 5) 分类与关系示例（最小集）

1) “我头痛想挂号，顺便问停车怎么收费”
- Segments: 2
- triage (focus) + smartCS (queued)
- Relation: insertion

2) “我要挂心内科，李四医生明天出诊吗”
- Segments: 2
- dept-direct vs doc-direct
- Relation: exclusive → clarify

3) “报告显示白细胞高，挂哪科”
- Segments: 2
- report + triage
- Relation: parallel → medical focus=triage, report queued

4) “我要缴费，另外前面还有几个人”
- Segments: 2
- payment (focus) + queue (queued)
- Relation: parallel

---

## 6) 最小落地要求
- 每轮输出：segments + class + relations
- 多意图必须写入关系类型
- 互斥场景必须澄清
- 医疗优先级高于行政类

---

## 7) 中间状态结构（带注释，v2 建议）

### 7.1 设计目的
- 可解释：每轮“为什么这样路由”可追溯
- 可扩展：新增 agent 只补充规则与记忆策略
- 可控：多意图避免强行主次，改为“医疗 focus + 队列”

### 7.2 建议结构（字段注释）
```
IntentContext {
  // 多操作列表：允许 shift + add / continue + add
  // priority 用于执行顺序（数值越小越先执行）
  intent_ops: [
    { op, target, lane, priority, reason, confidence }
  ],

  // 任务帧列表，不强制 primary/secondary
  frames: [
    {
      frame_id: "唯一ID",
      agent_code: "对应智能体",

      // role：focus 表示当前主任务；queued 表示排队任务
      // lane：medical/admin，用于“医疗只允许一个 focus”
      role: "focus|queued|insert",
      lane: "medical|admin",

      status: "active|pending|completed|canceled",
      confidence: 0.0,

      // 结构化记忆：用于 M3
      slots: {},
      missing_slots: [],

      // 记忆策略：按任务持续性设置，不再全局唯一
      memory: {
        // persist_level 仅描述策略，不强制 TTL；由记忆存储层自行管理
        persist_level: "persistent|short_term|ephemeral",
        summary: "用于 M2 对齐的摘要"
      },

      evidence: {
        turn_type: "confirm|deny|repair|slot_fill|new_request|unknown",
        signals: ["doctor_name","dept_name"],
        candidates: [{agent_code, score}]
      }
    }
  ],

  // 医疗 focus 的显式标记（防止多医疗并行）
  medical_focus_id: "frame_id",

  // 记忆策略开关（全局默认）
  memory_policy_default: {
    full_history_default: true,
    vector_alignment: true,
    structured_state: true
  },

  safety: { label, action },
  meta: { layer_hit, config_version, prompt_version },

  // 兼容字段（可选）：仅用于旧链路或灰度期
  legacy_agent_code: "agent_code"
}
```

### 7.3 intent_ops 约束与执行顺序（必须）
- **clarify 独占**：如出现 clarify，本轮其他 op 只能记录，不执行路由
- **同 lane 只允许一个 focus 操作**：medical/admin 各自最多一个 shift/continue
- **同 frame 不允许冲突**：不能同时 continue + cancel
- **允许组合**：shift + add，continue + add
- **每轮 op 数量上限**：建议 <= 3

**推荐执行顺序（priority）：**
1) safety/block  
2) clarify  
3) cancel/complete  
4) shift (focus change)  
5) continue  
6) add (secondary insert)  

### 7.4 关键约束（结构层）
- 同一时间只允许一个 medical focus
- memory 策略以 frame 为单位，不再全局唯一

---

## 8) 返回兼容策略（可选）
- **legacy_agent_code** 仅用于旧链路兼容  
- 若下游已全面升级为 frames/intent_ops，可移除该字段  
- 严禁业务逻辑依赖 legacy_agent_code 作为唯一决策  

---

## 9) 多轮对话示例（检验不会错绑）

**回合 1**  
用户：“我头痛想挂号，顺便问停车怎么收费，还有李四医生明天出诊吗？”  
分段：triage / smartCS / doc-direct  
关系：doc vs triage → **clarify**，smartCS 插入  
intent_ops：clarify + add(记录，不执行)  

**回合 2（澄清）**  
用户：“我想挂李四医生的号。”  
intent_ops：shift → doc-direct focus  
smartCS 仍在队列  

**回合 3（短句补槽）**  
用户：“明天下午的。”  
对齐：当前 focus = doc-direct，缺少 date → continue  

**回合 4（插入行政问题）**  
用户：“医保报销怎么走？”  
intent_ops：add → smartCS queued  

**结论**  
- clarify 独占避免 doc/triage 误判  
- focus+slots 确保短句对齐  
- 插入任务入队不丢失  

