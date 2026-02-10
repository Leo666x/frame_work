# Intent Routing Mapping (Layer1-4)

This document maps the current agent list and multi-intent strategy to the
Layer1-4 pipeline. It is designed to be stable for future expansion.

## Core Principles
1) Layer1/Layer2 enforce hard constraints (low cost, low drift).
2) Layer3 is the state machine for multi-turn intent shift/add/clarify.
3) Layer4 only does candidate generation + bounded selection.
4) Output stays backward compatible (agent_code), while frames/intent_ops
   carry multi-intent semantics.

## Agents (Current)
- power-ai-agent-triage: symptom/condition/uncertain medical routing
- power-ai-agent-dept-direct: explicit department entity
- power-ai-agent-doc-direct: explicit doctor name entity
- power-ai-agent-smartCS: admin/logistics/geo/rules/basic calculators
- power-ai-agent-payment: explicit payment actions
- power-ai-agent-queue: queue/waiting status
- power-ai-agent-drug: drug usage/contraindications/interaction/image
- power-ai-agent-report: report interpretation (example of future agent)

## Layer1: Fast Rule Match (Hard Hits)
Goal: capture clear, stable intents with deterministic rules.

Recommended Layer1 targets:
- payment (e.g., "缴费/付款/支付/结算")
- queue (e.g., "排队/候诊/叫号/过号")
- drug (e.g., "用法用量/能否一起吃/药盒图片")
- doc-direct (explicit doctor name)
- dept-direct (explicit department name)
- report (e.g., "化验单/血常规/影像报告/指标解释")

Strategy:
- Hit -> output frames.secondary + intent_ops=add (default).
- Exceptions: payment/emergency can use shift(primary).

Extension:
- New agent: add rule + agent_code + as_secondary + interruptible.

## Layer2: Safety Audit (Global Gate)
Goal: block or force-shift for prohibited or emergency content.

Strategy:
- SAFE -> continue
- PROHIBITED / ILLEGAL_MEDICAL -> block
- EMERGENCY -> shift(primary) to emergency guidance

Extension:
- Version safety policy per tenant.

## Layer3: Context Router (State Machine)
Goal: decide continue/add/shift/clarify for multi-turn dialog.

Inputs:
- Primary frame state (slots/missing_slots)
- Recent history (1-3 turns)
- Current user input
- Layer1 hits (insert signals)

Operations (minimal v1):
- continue(primary): slot fill/confirm/repair
- add(secondary): "顺便/另外"
- shift(primary): explicit topic switch
- clarify: ambiguous input

Strategy:
- Rule-first (turn_type detection).
- LLM as fallback for unclear cases.

Extension:
- Replace LLM gate with small model; interface unchanged.

## Layer4: Supervisor Dispatch (Candidates + Bounded Choice)
Goal: route uncertain intents using domain + candidates + bounded select.

Domains:
- medical_service: triage/dept/doc/report/drug
- admin_service: smartCS/payment/queue

Strategy:
- Domain select -> candidate list (RAG + metadata)
- If top1 score high -> direct select
- Else LLM chooses ONLY from candidate list

Extension:
- New agent: register domain + description + embedding
- Add rerank fusion without changing interface

## Output Contract (Backward Compatible)
- agent_code: kept for legacy consumers (primary)
- frames.primary / frames.secondary: multi-intent structure
- intent_ops: continue/add/shift/clarify/ood
- meta.layer_hit: layer1|layer2|layer3|layer4|ask

