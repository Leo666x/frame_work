package pgsql_mw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"math"
)

type PgSql struct {
	client *sqlx.DB
	config *Config
}

type Config struct {
	Username string
	Password string
	Database string
	Host     string
	Port     string
}

func New(c *Config) (*PgSql, error) {
	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		c.Username, c.Password, c.Database, c.Host, c.Port,
	)
	client, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return &PgSql{client: client, config: c}, nil
}

// Pagination 分页查询结果
type Pagination struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
}

type TransactionSql struct {
	SqlStatement string
	Args         []any
}

func (p *PgSql) check() error {
	if p.client == nil {
		return fmt.Errorf("postgres初始化失败,endpoints：%s:%s", p.config.Host, p.config.Port)
	}
	return nil
}

// QuerySingle 获取单一对象
func (p *PgSql) QuerySingle(dest interface{}, sqlWhere string, args ...interface{}) error {
	if err := p.check(); err != nil {
		return err
	}
	return p.client.Get(dest, sqlWhere, args...)
}

// QueryMultiple 获取多行对象
func (p *PgSql) QueryMultiple(dest interface{}, sqlWhere string, args ...interface{}) error {
	if err := p.check(); err != nil {
		return err
	}
	return p.client.Select(dest, sqlWhere, args...)
}

// QueryByPaginate 执行分页查询
func (p *PgSql) QueryByPaginate(dest interface{}, sqlWhere string, page, pageSize int, args ...interface{}) (*Pagination, error) {
	if err := p.check(); err != nil {
		return nil, err
	}
	// 验证参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 执行数据查询
	dataQuery := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", sqlWhere, len(args)+1, len(args)+2)

	// 添加分页参数
	paginationArgs := make([]interface{}, len(args)+2)
	copy(paginationArgs, args)
	paginationArgs[len(args)] = pageSize
	paginationArgs[len(args)+1] = (page - 1) * pageSize

	ctx := context.Background()
	// 执行数据查询
	err := p.client.SelectContext(ctx, dest, dataQuery, paginationArgs...)
	if err != nil {
		return nil, fmt.Errorf("分页查询失败: %w", err)
	}

	// 执行计数查询
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS subquery", sqlWhere)
	var totalItems int64
	err = p.client.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, fmt.Errorf("获取总记录数失败: %w", err)
	}
	// 计算分页信息
	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	return &Pagination{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
	}, nil
}

// Exec 执行SQL语句
func (p *PgSql) Exec(sqlWhere string, args ...interface{}) (sql.Result, error) {
	if err := p.check(); err != nil {
		return nil, err
	}
	return p.client.Exec(sqlWhere, args...)
}

// BatchExecTransaction 批量执行
func (p *PgSql) BatchExecTransaction(ts []*TransactionSql) error {
	if err := p.check(); err != nil {
		return err
	}
	if ts == nil || len(ts) == 0 {
		return errors.New("SQL语句列表不能为空")
	}
	ctx := context.Background()
	// 开始事务
	tx, err := p.client.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	for i, s := range ts {
		// 执行SQL
		_, err := tx.ExecContext(ctx, s.SqlStatement, s.Args...)
		if err != nil {
			return fmt.Errorf("执行第 %d 条SQL失败: %w\nSQL: %s", i+1, err, s)
		}
	}
	return nil
}
