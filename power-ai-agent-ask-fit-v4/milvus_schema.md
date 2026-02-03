# Milvus Design (ask agent, bge-m3)

Model: bge-m3, vector dim = 1024

## Collection
- name: `qa_data_get`

## Fields
- `id` (VarChar, primary key)
- `card_type` (VarChar)
- `function_name` (VarChar)
- `keywords` (VarChar, optional)
- `content` (VarChar, embedding text)
- `embedding` (FloatVector, dim=1024)

## Index/Search
- index: HNSW
- metric: IP (Inner Product)
- search param: ef = topK * 2~10

## Query Flow
1) EmbedTexts(query)
2) MilvusVectorSearch(collection, vectorField, vectors, topK, filter, outputFields)
3) Read `card_type` / `function_name`

## Output Fields
- `card_type`
- `function_name`
