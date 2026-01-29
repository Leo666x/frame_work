package powerai

import (
	xsql "database/sql"
	"fmt"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/pgsql"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xdatetime"
	"orgine.com/ai-team/power-ai-framework-v4/pkg/xuid"
	"strings"
	"time"
)

// AIConversation 对应 ai_conversation 表
type AIConversation struct {
	ConversationID   xsql.NullString `db:"conversation_id"`
	ConversationName xsql.NullString `db:"conversation_name"`
	UserID           xsql.NullString `db:"user_id"`
	Channel          xsql.NullString `db:"channel"`
	ChannelApp       xsql.NullString `db:"channel_app"`
	EnterpriseID     xsql.NullString `db:"enterprise_id"`
	CreateTime       time.Time       `db:"create_time"`
	UpdateTime       time.Time       `db:"update_time"`
	ExtendedField    xsql.NullString `db:"extended_field"`
	Messages         []*AIMessage
}

// AIMessage 对应 ai_message 表
type AIMessage struct {
	MessageID      xsql.NullString `db:"message_id"`
	ConversationID string          `db:"conversation_id"`
	Query          xsql.NullString `db:"query"`
	Answer         xsql.NullString `db:"answer"`
	Rating         xsql.NullString `db:"rating"`
	Inputs         xsql.NullString `db:"inputs"`
	Errors         xsql.NullString `db:"errors"`
	AgentCode      xsql.NullString `db:"agent_code"`
	FileID         xsql.NullString `db:"file_id"`
	CreateTime     time.Time       `db:"create_time"`
	UpdateTime     time.Time       `db:"update_time"`
	ExtendedField  xsql.NullString `db:"extended_field"`
}

// AISystemConfig 对应 ai_system_config 表
type AISystemConfig struct {
	ConfCode    xsql.NullString `db:"conf_code"`
	ConfName    xsql.NullString `db:"conf_name"`
	ConfContent xsql.NullString `db:"conf_content"`
	ConfType    xsql.NullString `db:"conf_type"`
	CreateTime  time.Time       `db:"create_time"`
	UpdateTime  time.Time       `db:"update_time"`
}

func (a *AgentApp) GetPgSqlClient() (*pgsql_mw.PgSql, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pgsql == nil {
		client, err := initPgSql(a.etcd)
		if err != nil {
			return nil, err
		}
		a.pgsql = client
	}
	return a.pgsql, nil
}

// QueryConversationById 根据conversationID查询会话，返回单结果
func (a *AgentApp) QueryConversationById(conversationID string) (*AIConversation, error) {

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID不能为空")
	}

	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}

	sql := `select conversation_id,conversation_name,user_id,channel,channel_app,enterprise_id,create_time,update_time,extended_field from ai_conversation where conversation_id = $1`
	r := &AIConversation{}
	if err := client.QuerySingle(r, sql, conversationID); err != nil {
		return nil, err
	}
	return r, nil
}

// QueryConversationWithMessageById 根据conversationID查询会话并且附带消息内容，返回单结果
func (a *AgentApp) QueryConversationWithMessageById(conversationID string) (*AIConversation, error) {

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}

	convSql := `select conversation_id,conversation_name,user_id,channel,channel_app,enterprise_id,create_time,update_time,extended_field from ai_conversation where conversation_id = $1`
	r := &AIConversation{}
	if err := client.QuerySingle(r, convSql, conversationID); err != nil {
		return nil, err
	}

	msgSql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field from ai_message where conversation_id = $1 ORDER BY create_time DESC`

	var m []*AIMessage
	if err := client.QueryMultiple(&m, msgSql, conversationID); err != nil {
		return nil, err
	}
	r.Messages = m
	return r, nil
}

// QueryMessageByMessageId 根据messageID查询消息内容，返回单结果
func (a *AgentApp) QueryMessageByMessageId(messageID string) (*AIMessage, error) {

	if messageID == "" {
		return nil, fmt.Errorf("messageID不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}

	sql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field from ai_message where message_id = $1`
	r := &AIMessage{}
	if err := client.QuerySingle(r, sql, messageID); err != nil {
		return nil, err
	}
	return r, nil
}

// QueryMessageByConversationID 根据conversationID查询消息列表，返回多值，按时间降序
func (a *AgentApp) QueryMessageByConversationID(conversationID string) ([]*AIMessage, error) {

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	sql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field from ai_message where conversation_id = $1 ORDER BY create_time DESC`
	var r []*AIMessage
	if err := client.QueryMultiple(&r, sql, conversationID); err != nil {
		return nil, err
	}
	return r, nil
}

// QueryMessageByConversationIDASC 根据conversationID查询消息列表，返回多值，按时间升序
func (a *AgentApp) QueryMessageByConversationIDASC(conversationID string) ([]*AIMessage, error) {

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID不能为空")
	}

	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	sql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field from ai_message where conversation_id = $1 ORDER BY create_time ASC`
	var r []*AIMessage
	if err := client.QueryMultiple(&r, sql, conversationID); err != nil {
		return nil, err
	}
	return r, nil
}

// QueryMessageByLimit 根据conversationID查询消息列表，限制条数
func (a *AgentApp) QueryMessageByLimit(conversationID string, limit int) ([]*AIMessage, error) {

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	sql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field from ai_message where conversation_id = $1 ORDER BY create_time DESC limit $2`
	var r []*AIMessage
	if err := client.QueryMultiple(&r, sql, conversationID, limit); err != nil {
		return nil, err
	}
	return r, nil
}

// QueryMessageByAgentCode 根据conversationID和agentCode查询消息列表，限制条数
func (a *AgentApp) QueryMessageByAgentCode(conversationID, agentCode string, limit int) ([]*AIMessage, error) {

	if conversationID == "" {
		return nil, fmt.Errorf("conversationID不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	sql := `select message_id,conversation_id,query,answer,rating,inputs,errors,agent_code,file_id,create_time, update_time,extended_field from ai_message where conversation_id = $1 and agent_code = $2 ORDER BY create_time DESC limit $3`
	var r []*AIMessage
	if err := client.QueryMultiple(&r, sql, conversationID, agentCode, limit); err != nil {
		return nil, err
	}
	return r, nil
}

// QueryMessageCount 根据conversationID查询消息数量
func (a *AgentApp) QueryMessageCount(conversationID string) (int64, error) {

	if conversationID == "" {
		return 0, fmt.Errorf("conversationID不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return 0, err
	}
	sql := `select count(*) from ai_message where conversation_id = $1 `
	var r int64
	if err := client.QuerySingle(&r, sql, conversationID); err != nil {
		return 0, err
	}
	return r, nil
}

// CreateConversation 创建会话，并且插入消息信息
func (a *AgentApp) CreateConversation(conversationName, userID, channel, channelApp, enterpriseID, query, inputs, fileID string) (string, string, error) {
	file, s, _, err := a.CreateConversationWithFile(conversationName, userID, channel, channelApp, enterpriseID,
		query, inputs,
		fileID, nil)
	return file, s, err
}

// CreateConversationWithFile 创建会话，并且插入消息信息
func (a *AgentApp) CreateConversationWithFile(conversationName, userID, channel, channelApp, enterpriseID, query,
	inputs,
	fileID string, fileIDs []string) (string, string, []string, error) {

	conversationId := xuid.UUID()
	messageId := xuid.UUID()
	timeNow := xdatetime.GetNowDateTime()
	var sqls []*pgsql_mw.TransactionSql
	sqls = append(sqls, &pgsql_mw.TransactionSql{
		SqlStatement: `INSERT INTO ai_conversation (conversation_id, conversation_name, user_id, channel, channel_app, enterprise_id, create_time, create_by, update_time, update_by) 
					   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		Args: []any{
			conversationId, conversationName, userID, channel, channelApp, enterpriseID, timeNow, "admin", timeNow, "admin",
		},
	})

	sqls = append(sqls, &pgsql_mw.TransactionSql{
		SqlStatement: `INSERT INTO ai_message (message_id,conversation_id,query,inputs,file_id,create_time, create_by, update_time,update_by)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		Args: []any{
			messageId, conversationId, query, inputs, fileID, timeNow, "admin", timeNow, "admin",
		},
	})

	var messageFileIds []string
	for _, fileID := range fileIDs {
		messageFileId := xuid.UUID()
		sqls = append(sqls, &pgsql_mw.TransactionSql{
			SqlStatement: `INSERT INTO ai_message_file (message_file_id,conversation_id,message_id,file_id,create_time, create_by, update_time,update_by)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			Args: []any{
				messageFileId, conversationId, messageId, fileID, timeNow, "admin", timeNow, "admin",
			},
		})
		messageFileIds = append(messageFileIds, messageFileId)
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return "", "", nil, err
	}
	if err := client.BatchExecTransaction(sqls); err != nil {
		return "", "", []string{}, err
	}
	return conversationId, messageId, messageFileIds, nil
}

// CreateMessage 创建消息内容
func (a *AgentApp) CreateMessage(conversationID, query, inputs, fileId string) (string, error) {
	file, _, err := a.CreateMessageWithFile(conversationID, query, inputs, fileId, nil)
	return file, err
}

func (a *AgentApp) CreateMessageWithFile(conversationID, query, inputs, fileId string, fileIDs []string) (string, []string, error) {

	var msgFileIDs []string
	if conversationID == "" {
		return "", msgFileIDs, fmt.Errorf("conversationID不能为空")
	}
	messageId := xuid.UUID()
	timeNow := xdatetime.GetNowDateTime()
	var sqls []*pgsql_mw.TransactionSql

	sqls = append(sqls, &pgsql_mw.TransactionSql{
		SqlStatement: `INSERT INTO ai_message (message_id,conversation_id,query,inputs,file_id,create_time, create_by, update_time,update_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		Args: []any{
			messageId, conversationID, query, inputs, fileId, timeNow, "admin", timeNow, "admin",
		},
	})

	var messageFileIds []string
	for _, fileID := range fileIDs {
		messageFileId := xuid.UUID()
		sqls = append(sqls, &pgsql_mw.TransactionSql{
			SqlStatement: `INSERT INTO ai_message_file (message_file_id,conversation_id,message_id,file_id,create_time, create_by, update_time,update_by)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			Args: []any{
				messageFileId, conversationID, messageId, fileID, timeNow, "admin", timeNow, "admin",
			},
		})
		messageFileIds = append(messageFileIds, messageFileId)
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return "", nil, err
	}
	if err := client.BatchExecTransaction(sqls); err != nil {
		return "", []string{}, err
	}
	return messageId, msgFileIDs, nil
}

// UpdateMessage 更新消息
func (a *AgentApp) UpdateMessage(messageID, answer, rating, inputs, errors, agentCode, fileID string) error {

	if messageID == "" {
		return fmt.Errorf("messageID不能为空")
	}

	// 构造动态更新语句
	var setClauses []string
	var values []any
	if answer != "" {
		setClauses = append(setClauses, "answer = ?")
		values = append(values, answer)
	}
	if rating != "" {
		setClauses = append(setClauses, "rating = ?")
		values = append(values, rating)
	}
	if inputs != "" {
		setClauses = append(setClauses, "inputs = ?")
		values = append(values, inputs)
	}
	if errors != "" {
		setClauses = append(setClauses, "errors = ?")
		values = append(values, errors)
	}
	if agentCode != "" {
		setClauses = append(setClauses, "agent_code = ?")
		values = append(values, agentCode)
	}
	if fileID != "" {
		setClauses = append(setClauses, "file_id = ?")
		values = append(values, fileID)
	}

	// 检查是否有需要更新的字段
	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setClauses = append(setClauses, "update_time = ?")
	values = append(values, xdatetime.GetNowDateTime())
	values = append(values, messageID)

	// 构造 SET 子句
	setClause := ""
	if len(setClauses) > 0 {
		for i, clause := range setClauses {
			if i == 0 {
				setClause = strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", i+1))
			} else {
				setClause += ", " + strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", i+1))
			}

		}
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return err
	}
	// 构造完整的 SQL 更新语句
	sql := fmt.Sprintf("UPDATE ai_message SET %s WHERE message_id = $%d", setClause, len(setClauses)+1)
	_, err = client.Exec(sql, values...)
	return err
}

func (a *AgentApp) InsertSystemConfig(confCode, confName, confContent, confType string) error {
	if confCode == "" {
		return fmt.Errorf("confCode不能为空")
	}
	if confName == "" {
		return fmt.Errorf("confName不能为空")
	}
	if confContent == "" {
		return fmt.Errorf("confContent不能为空")
	}
	if confType == "" {
		return fmt.Errorf("confType不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return err
	}
	_, err = client.Exec(`INSERT INTO ai_system_config (conf_code,conf_name,conf_content,conf_type,create_time, create_by, update_time,update_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		confCode,
		confName,
		confContent,
		confType,
		xdatetime.GetNowDateTime(),
		"admin",
		xdatetime.GetNowDateTime(),
		"admin",
	)
	return err
}

func (a *AgentApp) UpdateSystemConfig(confCode, confContent string) error {
	if confCode == "" {
		return fmt.Errorf("confCode不能为空")
	}

	if confContent == "" {
		return fmt.Errorf("confContent不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return err
	}
	_, err = client.Exec(`update ai_system_config set  conf_content= $1, update_time =$2 where conf_code = $3`,
		confCode,
		confContent,
		xdatetime.GetNowDateTime(),
	)
	return err
}

func (a *AgentApp) QueryAllSystemConfig() ([]*AISystemConfig, error) {
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	sql := `select conf_code,conf_name,conf_content,conf_type,create_time,update_time from ai_system_config `
	var r []*AISystemConfig
	if err := client.QueryMultiple(&r, sql); err != nil {
		return nil, err
	}
	return r, nil
}

func (a *AgentApp) QuerySystemConfigByCode(confCode string) (*AISystemConfig, error) {
	if confCode == "" {
		return nil, fmt.Errorf("confCode不能为空")
	}
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	sql := `select conf_code,conf_name,conf_content,conf_type,create_time,update_time from ai_system_config where conf_code = $1`
	r := &AISystemConfig{}
	if err := client.QuerySingle(r, sql, confCode); err != nil {
		return nil, err
	}
	return r, nil
}

// DBQuerySingle 查询单值
func (a *AgentApp) DBQuerySingle(dest interface{}, sqlWhere string, args ...interface{}) error {
	client, err := a.GetPgSqlClient()
	if err != nil {
		return err
	}
	return client.QuerySingle(dest, sqlWhere, args...)
}

// DBQueryMultiple 获取多行对象
func (a *AgentApp) DBQueryMultiple(dest interface{}, sqlWhere string, args ...interface{}) error {
	client, err := a.GetPgSqlClient()
	if err != nil {
		return err
	}
	return client.QueryMultiple(dest, sqlWhere, args...)
}

// DBQueryByPaginate 执行分页查询
func (a *AgentApp) DBQueryByPaginate(dest interface{}, sqlWhere string, page, pageSize int, args ...interface{}) (*pgsql_mw.Pagination, error) {
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	return client.QueryByPaginate(dest, sqlWhere, page, pageSize, args...)
}

// DBExec 执行SQL语句
func (a *AgentApp) DBExec(sqlWhere string, args ...interface{}) (xsql.Result, error) {
	client, err := a.GetPgSqlClient()
	if err != nil {
		return nil, err
	}
	return client.Exec(sqlWhere, args...)
}

// DBBatchExecTransaction 批量执行
func (a *AgentApp) DBBatchExecTransaction(ts []*pgsql_mw.TransactionSql) error {
	client, err := a.GetPgSqlClient()
	if err != nil {
		return err
	}
	return client.BatchExecTransaction(ts)
}
