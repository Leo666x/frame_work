package milvus_mw

import (
	"context"
	"errors"
	"fmt"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"strings"
	"time"
)

type Config struct {
	Addr     string
	Username string
	Password string
	Timeout  time.Duration
}

type Milvus struct {
	client client.Client
	config *Config
}

func New(c *Config) (*Milvus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	cli, err := client.NewClient(ctx, client.Config{
		Address:  c.Addr,
		Username: c.Username,
		Password: c.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus at %s: %w", c.Addr, err)
	}
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer checkCancel()
	_, err = cli.ListCollections(checkCtx)
	if err != nil {
		// 如果检查失败，关闭已经创建的 client，防止资源泄露
		_ = cli.Close()
		return nil, fmt.Errorf("milvus connection established but health check failed: %w", err)
	}
	return &Milvus{client: cli, config: c}, nil
}

// DynamicInsert 原子化插入函数
//
// 入参说明：
// ctx: 上下文
// collectionName: 集合名称
// scalarColumns: 标量列数据，Key为字段名，Value为字符串形式的列数据列表。例如：{"age": ["10", "20"], "name": ["A", "B"]}
// vectorColumns: 向量列数据，Key为字段名，Value为二维浮点数组。例如：{"vec": [[0.1, 0.2], [0.3, 0.4]]}
// 出参说明：
// error: 如果列长度不一致或插入失败，返回 error

func (m *Milvus) DynamicInsert(
	ctx context.Context,
	collectionName string,
	scalarColumns map[string][]string,
	vectorColumns map[string][][]float32,
) error {

	// 1. 边界检查
	if len(scalarColumns) == 0 && len(vectorColumns) == 0 {
		return errors.New("input data is empty")
	}

	// 2. 动态构建 Milvus 所需的 entity.Column 列表
	// 我们不知道有多少列，所以需要动态 slice
	milvusColumns := make([]entity.Column, 0, len(scalarColumns)+len(vectorColumns))

	// 用于校验所有列的行数是否对齐（Consistency Check）
	var rowCount int = -1

	// --- 处理标量数据 (全部按 VarChar 处理) ---
	for fieldName, data := range scalarColumns {
		currentLen := len(data)

		// 第一次记录行数
		if rowCount == -1 {
			rowCount = currentLen
		} else if rowCount != currentLen {
			return fmt.Errorf("column length mismatch: field '%s' has %d rows, expected %d", fieldName, currentLen, rowCount)
		}

		// 核心操作：将 []string 包装成 Milvus VarChar Column
		// 注意：这要求 Milvus 集合中对应的字段类型必须是 VarChar
		col := entity.NewColumnVarChar(fieldName, data)
		milvusColumns = append(milvusColumns, col)
	}

	// --- 处理向量数据 ---
	for fieldName, data := range vectorColumns {
		currentLen := len(data)

		if rowCount == -1 {
			rowCount = currentLen
		} else if rowCount != currentLen {
			return fmt.Errorf("vector column length mismatch: field '%s' has %d rows, expected %d", fieldName, currentLen, rowCount)
		}

		if currentLen == 0 {
			continue
		}

		// 获取向量维度 (dim)
		dim := len(data[0])

		// 核心操作：将 [][]float32 包装成 Milvus FloatVector Column
		col := entity.NewColumnFloatVector(fieldName, dim, data)
		milvusColumns = append(milvusColumns, col)
	}

	// 3. 执行原子插入
	// PartitionName 传空字符串 "" 表示使用默认分区
	_, err := m.client.Insert(ctx, collectionName, "", milvusColumns...)
	if err != nil {
		return fmt.Errorf("milvus insert failed: %w", err)
	}

	return nil
}

// DeleteVectorsByIDs 根据主键 ID 列表批量删除向量
//
// 入参：
// ctx: 上下文
// collectionName: 集合名称
// pkFieldName: 主键字段名 (例如 "doc_id" 或 "id")
// ids: 要删除的主键 ID 列表 (全部视为 string 处理)
func (m *Milvus) DeleteVectorsByIDs(ctx context.Context, collectionName string, pkFieldName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// 1. 构造删除表达式 (Expression)
	// Milvus 删除语法: field_name in ["id1", "id2", ...]
	// 注意: 字符串类型的 ID 必须用单引号包裹
	var sb strings.Builder
	sb.WriteString(pkFieldName)
	sb.WriteString(" in [")

	for i, id := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		// 安全起见，防止 SQL 注入类的拼接风险，简单转义单引号
		safeID := strings.ReplaceAll(id, "'", "\\'")
		sb.WriteString(fmt.Sprintf("'%s'", safeID))
	}
	sb.WriteString("]")
	expr := sb.String()

	// 2. 执行删除
	// partitionName 传 "" 表示在所有分区中删除
	err := m.client.Delete(ctx, collectionName, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete vectors from milvus: %w", err)
	}

	return nil
}

// DeleteVectorsByExpression 根据自定义表达式删除向量 (高级用法)
//
// 例如: expression = "knowledge_id == 'kb_001'"
func (m *Milvus) DeleteVectorsByExpression(ctx context.Context, collectionName string, expression string) error {
	if expression == "" {
		return fmt.Errorf("delete expression cannot be empty")
	}

	err := m.client.Delete(ctx, collectionName, "", expression)
	if err != nil {
		return fmt.Errorf("failed to execute delete expression in milvus: %w", err)
	}
	return nil
}

// DropCollection 删除整个集合
//
// 适用于直接清空整个知识库的场景
func (m *Milvus) DropCollection(ctx context.Context, collectionName string) error {
	// 检查是否存在
	has, err := m.client.HasCollection(ctx, collectionName)
	if err != nil {
		return err
	}
	if !has {
		return nil // 不存在则视为删除成功
	}

	err = m.client.DropCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to drop collection %s: %w", collectionName, err)
	}
	return nil
}

// SearchResult 封装单条搜索结果
type SearchResult struct {
	ID    string            `json:"id"`    // 主键 ID
	Score float32           `json:"score"` // 相似度分数
	Data  map[string]string `json:"data"`  // 返回的标量字段数据 (key:列名, value:值)
}

// MilvusVectorSearch 执行向量相似度搜索
//
// 入参:
// ctx: 上下文
// collectionName: 集合名
// vectorFieldName: 向量字段名 (如 "doc_embedding")
// queryVectors: 查询向量 (支持批量搜索，通常是一条 [][]float32{{...}})
// topK: 返回多少条结果
// filterExpr: 标量过滤表达式 (空字符串表示不过滤，例如 "dept_level == '2'")
// outputFields: 需要返回的标量字段列表 (如 ["doc_name", "doc_content"])
func (m *Milvus) MilvusVectorSearch(
	ctx context.Context,
	collectionName string,
	vectorFieldName string,
	queryVectors [][]float32,
	topK int,
	filterExpr string,
	outputFields []string,
) ([][]SearchResult, error) {

	// 1. 准备搜索参数
	// 针对 bge-m3，metricType 必须为 IP (Inner Product)
	// NewIndexHNSWSearchParam(ef) 中的 ef 参数决定搜索精度，通常设为 topK 的 2-10 倍
	sp, err := entity.NewIndexHNSWSearchParam(topK * 2)
	if err != nil {
		return nil, err
	}
	searchVectors := make([]entity.Vector, len(queryVectors))
	for i, v := range queryVectors {
		searchVectors[i] = entity.FloatVector(v)
	}

	// 2. 执行搜索
	searchResult, err := m.client.Search(
		ctx,
		collectionName,
		[]string{}, // partitionNames (留空表示搜索所有分区)
		filterExpr, // 过滤表达式
		outputFields,
		searchVectors,
		vectorFieldName,
		entity.IP,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}

	// 3. 解析结果 (列式转行式)
	finalResults := make([][]SearchResult, len(searchResult))

	for i, res := range searchResult {
		resultCount := res.ResultCount // 实际搜索到的条数
		rowResults := make([]SearchResult, 0, resultCount)

		// 获取 ID 列和 分数列
		// 注意：根据你的建表逻辑，ID 是 VarChar，所以使用 IDs.(*entity.ColumnVarChar)
		// 如果不确定类型，可以使用通用处理，这里为了性能假设是 VarChar
		var idList []string

		// 处理 ID 列
		if col, ok := res.IDs.(*entity.ColumnVarChar); ok {
			idList = col.Data()
		} else if col, ok := res.IDs.(*entity.ColumnInt64); ok {
			// 兼容 Int64 ID 的情况
			data := col.Data()
			idList = make([]string, len(data))
			for idx, v := range data {
				idList[idx] = fmt.Sprintf("%d", v)
			}
		} else {
			return nil, fmt.Errorf("unsupported result ID type")
		}

		scores := res.Scores // []float32

		// 遍历每一行命中结果
		for j := 0; j < resultCount; j++ {
			row := SearchResult{
				ID:    idList[j],
				Score: scores[j],
				Data:  make(map[string]string),
			}

			// 提取 OutputFields (标量数据)
			// Milvus 返回的 Fields 是一个 Column 切片
			for _, fieldCol := range res.Fields {
				fieldName := fieldCol.Name()

				// 提取第 j 行的数据
				valStr, err := extractColumnValueAsString(fieldCol, j)
				if err != nil {
					// 仅打印日志或忽略，不阻断流程
					// fmt.Printf("warning: extract field %s failed: %v\n", fieldName, err)
					valStr = ""
				}
				row.Data[fieldName] = valStr
			}

			rowResults = append(rowResults, row)
		}
		finalResults[i] = rowResults
	}

	return finalResults, nil
}

// extractColumnValueAsString 辅助函数：从列中提取指定行的数据并转为 String
// 你的框架约定所有数据入库前转 String，但 Milvus 返回时会根据 Schema 类型返回
func extractColumnValueAsString(col entity.Column, index int) (string, error) {
	if index < 0 || index >= col.Len() {
		return "", fmt.Errorf("index out of range")
	}

	// 根据 Milvus 支持的常见类型进行断言
	switch c := col.(type) {
	case *entity.ColumnVarChar:
		return c.Data()[index], nil
	case *entity.ColumnString: // 老版本兼容
		return c.Data()[index], nil
	case *entity.ColumnInt64:
		return fmt.Sprintf("%d", c.Data()[index]), nil
	case *entity.ColumnInt32:
		return fmt.Sprintf("%d", c.Data()[index]), nil
	case *entity.ColumnFloat:
		return fmt.Sprintf("%f", c.Data()[index]), nil
	case *entity.ColumnDouble:
		return fmt.Sprintf("%f", c.Data()[index]), nil
	case *entity.ColumnBool:
		return fmt.Sprintf("%t", c.Data()[index]), nil
	default:
		// 如果有 Array 或 Json 类型，需额外处理
		return "", fmt.Errorf("unsupported column type: %T", col)
	}
}

// CreateCollectionFromData 根据传入的数据动态构建 Schema 并建表
func (m *Milvus) CreateCollectionFromData(ctx context.Context, collectionName, pkField string, scalarData map[string][]string, vectorData map[string][][]float32) error {

	// 1. 构建 Schema
	schema := &entity.Schema{
		CollectionName: collectionName,
		Description:    "Auto-created by Agent Framework",
		AutoID:         false, // 我们自己生成 String UUID
		Fields:         []*entity.Field{},
	}

	// 2. 遍历标量数据添加字段
	for fieldName := range scalarData {
		// 默认所有标量都是 VarChar，长度设大一点以防万一
		field := entity.NewField().
			WithName(fieldName).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(8192) // Milvus VarChar 最大支持 65535

		// 标记主键
		if fieldName == pkField {
			field.WithIsPrimaryKey(true)
		}

		schema.Fields = append(schema.Fields, field)
	}

	// 3. 遍历向量数据添加字段
	var vectorFieldName string
	for fieldName, vecs := range vectorData {
		if len(vecs) == 0 {
			continue
		}
		dim := len(vecs[0]) // 获取向量维度

		field := entity.NewField().
			WithName(fieldName).
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(dim))

		schema.Fields = append(schema.Fields, field)
		vectorFieldName = fieldName // 记录下来用于创建索引
	}

	// 4. 创建集合
	err := m.client.CreateCollection(ctx, schema, 1) // shardNum=1
	if err != nil {
		return fmt.Errorf("create collection api failed: %w", err)
	}

	// 5. 创建索引
	if vectorFieldName != "" {
		idx, err := entity.NewIndexHNSW(entity.IP, 8, 200)
		if err != nil {
			return err
		}

		err = m.client.CreateIndex(ctx, collectionName, vectorFieldName, idx, false)
		if err != nil {
			return fmt.Errorf("create index failed: %w", err)
		}
	}

	// 6. Load Collection
	err = m.client.LoadCollection(ctx, collectionName, false)
	if err != nil {
		return fmt.Errorf("load collection failed: %w", err)
	}

	//fmt.Println("Collection created, indexed, and loaded successfully.")
	return nil
}
