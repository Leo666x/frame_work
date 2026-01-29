package weaviate_mw

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

type Weaviate struct {
	client *weaviate.Client
	config *Config
}
type Config struct {
	Host   string
	Scheme string
	ApiKey string
}

func New(c *Config) (*Weaviate, error) {
	client, err := weaviate.NewClient(weaviate.Config{
		Host:       c.Host,
		Scheme:     c.Scheme,
		AuthConfig: auth.ApiKey{Value: c.ApiKey},
	})
	if err != nil {
		return nil, err
	}
	return &Weaviate{client: client, config: c}, nil
}

func (w *Weaviate) check() error {
	if w.client == nil {
		return fmt.Errorf("weaviate初始化失败,endpoints：%s", w.config.Host)
	}
	return nil
}

func (w *Weaviate) Insert(className string, records []map[string]string, vectors [][]float32) ([]string, error) {
	if err := w.check(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	batcher := w.client.Batch().ObjectsBatcher()

	var objs []*models.Object

	for i, rec := range records {
		obj := &models.Object{
			Class:      className,
			Properties: rec,
			Vector:     vectors[i],
		}
		objs = append(objs, obj)

	}
	batcher = batcher.WithObjects(objs...)

	res, err := batcher.Do(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(res))
	for i, r := range res {
		ids[i] = r.ID.String()
	}
	return ids, nil
}

// HybridSearch 在 Weaviate 上执行混合检索，返回原始结果
func (w *Weaviate) HybridSearch(className, query string, vector []float32, returnFields []string, topK int, alpha float32) ([]map[string]interface{}, error) {
	if err := w.check(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	// 构建查询字段
	var gqlFields []graphql.Field
	for _, f := range returnFields {
		gqlFields = append(gqlFields, graphql.Field{Name: f})
	}
	gqlFields = append(gqlFields, graphql.Field{
		Name:   "_additional",
		Fields: []graphql.Field{{Name: "score"}},
	})
	// 执行混合检索
	resp, err := w.client.GraphQL().Get().
		WithClassName(className).
		WithFields(gqlFields...).
		WithHybrid(w.client.GraphQL().HybridArgumentBuilder().
			WithQuery(query).WithVector(vector).WithAlpha(alpha),
		).
		WithLimit(topK).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	rawGet, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		var errMsg string
		for _, em := range resp.Errors {
			errMsg += fmt.Sprintf("message: %s, path: %v, locations: %v\n", em.Message, em.Path, em.Locations)
		}
		return nil, fmt.Errorf("GraphQL 响应格式错误：缺少 Get, error: %s", errMsg)
	}
	items, ok := rawGet[className].([]interface{})
	if !ok {
		return nil, fmt.Errorf("未找到 %q 结果", className)
	}
	out := make([]map[string]interface{}, len(items))
	for i, it := range items {
		out[i] = it.(map[string]interface{})
	}
	return out, nil
}

// DeleteClass 删除整个 class 及其所有数据
func (w *Weaviate) DeleteClass(className string) error {
	if err := w.check(); err != nil {
		return err
	}
	ctx := context.Background()
	if err := w.client.Schema().ClassDeleter().
		WithClassName(className).
		Do(ctx); err != nil {
		return fmt.Errorf("删除知识库 %q 失败: %w", className, err)
	}
	return nil
}

// EnsureClassExists 校验 Weaviate 中指定 class 是否已创建
func (w *Weaviate) EnsureClassExists(className string) error {
	if err := w.check(); err != nil {
		return err
	}
	ctx := context.Background()
	schemaRes, err := w.client.Schema().Getter().Do(ctx)
	if err != nil {
		return fmt.Errorf("获取 schema 失败: %w", err)
	}
	for _, class := range schemaRes.Classes {
		if class.Class == className {
			return nil
		}
	}
	return fmt.Errorf("知识库 class %q 不存在", className)
}

// HybridSearchAllInclude 混合检索 可过滤条件 V5版本
func (w *Weaviate) HybridSearchAllInclude(
	className string,
	query string,
	vector []float32,
	returnFields []string,
	topK int,
	alpha float32,
	include map[string][]string, // 例如: {"doc_id": {"fadfse1213","xxx"}}
) ([]map[string]interface{}, error) {
	ctx := context.Background()
	if err := w.check(); err != nil {
		return nil, err
	}
	// 构建查询字段
	var gqlFields []graphql.Field
	for _, f := range returnFields {
		gqlFields = append(gqlFields, graphql.Field{Name: f})
	}

	// 2) 组装返回字段
	fields := append([]graphql.Field{}, gqlFields...)
	fields = append(fields, graphql.Field{
		Name: "_additional",
		Fields: []graphql.Field{
			{Name: "score"},
		},
	})

	// 3) 构造 where 过滤器（“包含条件”）
	// 语义：对于同一字段的多个值：Equal OR Equal；多个字段之间：AND
	var where *filters.WhereBuilder
	if len(include) > 0 {
		and := filters.Where().WithOperator(filters.And)
		andOps := make([]*filters.WhereBuilder, 0, len(include))

		for field, values := range include {
			// 针对该字段，构建 OR(values)
			switch len(values) {
			case 0:
				// 跳过空列表
				continue
			case 1:
				// 单值：直接 Equal
				andOps = append(andOps,
					filters.Where().
						WithPath([]string{field}).
						WithOperator(filters.Equal).
						WithValueText(values[0]),
				)
			default:
				// 多值：OR
				orNode := filters.Where().WithOperator(filters.Or)
				orOps := make([]*filters.WhereBuilder, 0, len(values))
				for _, v := range values {
					orOps = append(orOps,
						filters.Where().
							WithPath([]string{field}).
							WithOperator(filters.Equal).
							WithValueText(v),
					)
				}
				andOps = append(andOps, orNode.WithOperands(orOps))
			}
		}
		if len(andOps) > 0 {
			where = and.WithOperands(andOps)
		}
	}

	// 4) hybrid 参数
	hy := w.client.GraphQL().HybridArgumentBuilder().WithAlpha(alpha)
	if query != "" {
		hy = hy.WithQuery(query)
	}
	if vector != nil {
		hy = hy.WithVector(vector)
	}

	// 5) 执行查询
	get := w.client.GraphQL().Get().
		WithClassName(className).
		WithFields(fields...).
		WithHybrid(hy).
		WithLimit(topK)

	if where != nil {
		get = get.WithWhere(where)
	}

	resp, err := get.Do(ctx)
	if err != nil {
		return nil, err
	}
	// 先看 GraphQL 层面的错误
	if len(resp.Errors) > 0 {
		b, _ := json.Marshal(resp.Errors)
		return nil, fmt.Errorf("GraphQL 返回 errors: %s", string(b))
	}

	// 6) 解析响应
	rawGet, ok := resp.Data["Get"].(map[string]interface{})
	if !ok {
		b, _ := json.Marshal(resp.Data)
		return nil, fmt.Errorf("GraphQL 响应格式错误：缺少 Get, data=%s", string(b))
	}
	items, ok := rawGet[className].([]interface{})
	if !ok {
		return nil, fmt.Errorf("未找到 %q 结果", className)
	}

	out := make([]map[string]interface{}, len(items))
	for i, it := range items {
		obj, ok := it.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("结果项类型错误：index=%d", i)
		}
		out[i] = obj
	}
	return out, nil
}
