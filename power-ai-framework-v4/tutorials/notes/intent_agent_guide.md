# 意图识别 Agent 操作文档（规则路由版）

> 目标：做一个独立的“意图识别 Agent”，根据规则决定 **调用哪个 Agent / 哪个 Router**。本指南以最小可用实现为目标，便于你后续替换规则和扩展能力。

生成时间：2026-01-26

---

## 1. 适用范围
- 你要新增一个 **独立的意图识别 Agent**（不依赖其他 Agent）。
- 该 Agent 的职责是：**解析用户请求 → 产出路由决策**。
- 决策结果可返回给上游，也可由它直接代调用。

---

## 2. 总体流程（简述）
1. 新建一个 `decision-agent` 服务（独立进程）。
2. 使用框架 `NewAgent` 启动，并注册 `/send_msg` 路由。
3. 在 `send_msg` handler 内做规则匹配。
4. 输出 `{target_agent, target_router, confidence, reason}`。

---

## 3. 最小可用骨架（推荐）

### 3.1 main 入口
```go
package main

import (
    "context"
    "github.com/gin-gonic/gin"
    "orgine.com/ai-team/power-ai-framework-v4"
)

func main() {
    manifest := `{
        "code":"power-ai-decision",
        "name":"Decision Agent",
        "version":"1.0.0",
        "description":"rule-based router"
    }`

    app, err := powerai.NewAgent(
        manifest,
        powerai.WithSendMsgRouter(sendMsg),
        powerai.WithOnShutDown(func(ctx context.Context) {
            // TODO: cleanup
        }),
    )
    if err != nil {
        panic(err)
    }

    app.Run()
}

func sendMsg(c *gin.Context) {
    req, resp, event, ok := powerai.DoValidateAgentRequest(c, "power-ai-decision")
    if !ok { return }

    decision := routeDecision(req.Query)

    _ = event.WriteAgentResponseStruct(resp, map[string]interface{}{
        "target_agent":  decision.AgentCode,
        "target_router": decision.Router,
        "confidence":    decision.Score,
        "reason":        decision.Reason,
    })

    event.Done(resp)
}
```

### 3.2 规则函数
```go
type DecisionResult struct {
    AgentCode string
    Router    string
    Score     float64
    Reason    string
}

func routeDecision(query string) DecisionResult {
    if strings.Contains(query, "挂号") || strings.Contains(query, "预约") {
        return DecisionResult{
            AgentCode: "power-ai-appointment",
            Router:    "send_msg",
            Score:     0.85,
            Reason:    "命中预约关键词",
        }
    }
    if strings.Contains(query, "报告") || strings.Contains(query, "影像") {
        return DecisionResult{
            AgentCode: "power-ai-report",
            Router:    "send_msg",
            Score:     0.80,
            Reason:    "命中报告关键词",
        }
    }
    return DecisionResult{
        AgentCode: "power-ai-general",
        Router:    "send_msg",
        Score:     0.50,
        Reason:    "默认兜底",
    }
}
```

---

## 4. 路由规则与注意事项
- **路由路径规则**：`agent_code` 中的 `-` 会被替换成 `/`。
  - `power-ai-decision` → `/power/ai/decision/send_msg`
- 建议统一使用 `send_msg`，便于对齐框架调用链。
- 决策输出要包含 `target_agent + target_router`，方便上游按结果调用。

---

## 5. 扩展方式（推荐顺序）
1. **先加规则**：简单 if/contains。
2. **再加配置化**：把规则放到 etcd 或配置文件。
3. **再加 LLM**：通过 `powerai_model.go` 统一调模型做分类。

---

## 6. 两种调用模式

### 模式 A：只返回决策（低耦合）
- 意图 Agent 返回目标路由
- 上游自己调用目标 Agent

### 模式 B：代理调用（高耦合）
- 意图 Agent 直接调用目标 Agent
- 推荐用 `SyncCallAgent` 或 `CallAgentProxy`

---

## 7. 常见坑
- manifest 必须包含 4 个字段，否则启动失败。
- 未注册 send_msg 会导致调用失败。
- 规则不命中必须有兜底，否则返回空。

---

## 8. 你后续可做的升级
- 加入置信度阈值
- 做多轮意图判断（history）
- 记录决策日志（入库）
- 加“fallback agent”兜底

---

> 如果你希望我把这套文档升级为“可配置规则版本 + etcd 规则加载模板”，告诉我你偏向的规则结构（JSON or YAML）。
