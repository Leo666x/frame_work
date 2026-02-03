# Response Abstraction (legacy + new)

## New Response (structured)
```json
{
  "text": "string",
  "intent": "string",
  "confidence": 0.85,
  "reason": "string",
  "target_agent": "power-ai-xxx",
  "target_router": "send_msg",
  "cards": [
    {"card_type": "card_x", "function_name": "xxx"}
  ],
  "endflag": "true",
  "type": "card"
}
```

## Legacy Response (backward compatible)
```json
{
  "go_url": "",
  "endflag": "true",
  "type": "card",
  "list": ["card_type"]
}
```

## Adapter
- `BuildNew` produces the structured response.
- `BuildLegacy` maps to the legacy `list` output.
