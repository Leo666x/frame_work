# Milvus Ingest Example (bge-m3)

Note: This is a logical example. Real ingest requires your actual data and embeddings.

## 1) qa_data_get (card collection)

Fields:
- id (string)
- card_type (string)
- function_name (string)
- content (string)
- embedding ([]float32, dim=1024)

```go
// texts -> embeddings
vecs, _ := app.EmbedTexts(enterpriseId, []string{content})

scalar := map[string][]string{
  "id": {"card_001"},
  "card_type": {"card_type_x"},
  "function_name": {"住院清单"},
  "content": {"住院清单 住院费用明细 住院花了多少钱"},
}
vector := map[string][][]float32{
  "embedding": {vecs[0]},
}

_ = milvus.DynamicInsert(ctx, "qa_data_get", scalar, vector)
```

## 2) QAXHYY (knowledge collection)

Fields:
- id (string)
- q (string)
- a (string)
- content (string)
- embedding ([]float32, dim=1024)

```go
vecs, _ := app.EmbedTexts(enterpriseId, []string{content})

scalar := map[string][]string{
  "id": {"qa_001"},
  "q": {"儿科在几楼？"},
  "a": {"您好，儿科位于医院的3楼。"},
  "content": {"儿科在几楼 儿科位置 3楼"},
}
vector := map[string][][]float32{
  "embedding": {vecs[0]},
}

_ = milvus.DynamicInsert(ctx, "QAXHYY", scalar, vector)
```

## Notes
- bge-m3 vector dim = 1024
- collection field `embedding` must be FloatVector(1024)
