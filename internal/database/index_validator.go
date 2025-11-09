package database

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// IndexValidator 索引验证器
type IndexValidator struct {
	db *gorm.DB
}

// NewIndexValidator 创建索引验证器
func NewIndexValidator(db *gorm.DB) *IndexValidator {
	return &IndexValidator{db: db}
}

// IndexInfo 索引信息
type IndexInfo struct {
	TableName  string   `json:"tableName"`
	IndexName  string   `json:"indexName"`
	ColumnName string   `json:"columnName"`
	IsUnique   bool     `json:"isUnique"`
	IsPrimary  bool     `json:"isPrimary"`
	IndexType  string   `json:"indexType"`
	Columns    []string `json:"columns"`
}

// MissingIndex 缺失的索引
type MissingIndex struct {
	TableName   string   `json:"tableName"`
	Columns     []string `json:"columns"`
	Reason      string   `json:"reason"`
	Recommended bool     `json:"recommended"`
}

// ValidateIndexes 验证所有必需的索引是否存在
func (v *IndexValidator) ValidateIndexes(ctx context.Context) ([]MissingIndex, error) {
	missingIndexes := make([]MissingIndex, 0)

	// 定义所有必需的索引
	requiredIndexes := v.getRequiredIndexes()

	// 获取现有索引
	existingIndexes, err := v.GetAllIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取现有索引失败: %w", err)
	}

	// 检查每个必需的索引
	for _, required := range requiredIndexes {
		if !v.indexExists(required, existingIndexes) {
			missingIndexes = append(missingIndexes, required)
		}
	}

	return missingIndexes, nil
}

// GetAllIndexes 获取所有索引信息
func (v *IndexValidator) GetAllIndexes(ctx context.Context) ([]IndexInfo, error) {
	var indexes []IndexInfo

	query := `
		SELECT 
			t.tablename AS table_name,
			i.indexname AS index_name,
			a.attname AS column_name,
			i.indexdef LIKE '%UNIQUE%' AS is_unique,
			c.contype = 'p' AS is_primary,
			am.amname AS index_type
		FROM pg_indexes i
		JOIN pg_class pc ON pc.relname = i.indexname
		JOIN pg_am am ON am.oid = pc.relam
		JOIN pg_attribute a ON a.attrelid = pc.oid
		JOIN pg_tables t ON t.tablename = i.tablename
		LEFT JOIN pg_constraint c ON c.conname = i.indexname
		WHERE t.schemaname = 'public'
		ORDER BY t.tablename, i.indexname, a.attnum
	`

	rows, err := v.db.WithContext(ctx).Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询索引信息失败: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*IndexInfo)

	for rows.Next() {
		var tableName, indexName, columnName, indexType string
		var isUnique, isPrimary bool

		if err := rows.Scan(&tableName, &indexName, &columnName, &isUnique, &isPrimary, &indexType); err != nil {
			return nil, fmt.Errorf("扫描索引信息失败: %w", err)
		}

		key := fmt.Sprintf("%s.%s", tableName, indexName)
		if idx, exists := indexMap[key]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[key] = &IndexInfo{
				TableName:  tableName,
				IndexName:  indexName,
				ColumnName: columnName,
				IsUnique:   isUnique,
				IsPrimary:  isPrimary,
				IndexType:  indexType,
				Columns:    []string{columnName},
			}
		}
	}

	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, nil
}

// GetTableIndexes 获取指定表的索引
func (v *IndexValidator) GetTableIndexes(ctx context.Context, tableName string) ([]IndexInfo, error) {
	allIndexes, err := v.GetAllIndexes(ctx)
	if err != nil {
		return nil, err
	}

	tableIndexes := make([]IndexInfo, 0)
	for _, idx := range allIndexes {
		if idx.TableName == tableName {
			tableIndexes = append(tableIndexes, idx)
		}
	}

	return tableIndexes, nil
}

// AnalyzeIndexUsage 分析索引使用情况
func (v *IndexValidator) AnalyzeIndexUsage(ctx context.Context) ([]IndexUsage, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_scan,
			idx_tup_read,
			idx_tup_fetch
		FROM pg_stat_user_indexes
		WHERE schemaname = 'public'
		ORDER BY idx_scan ASC
	`

	var usages []IndexUsage
	if err := v.db.WithContext(ctx).Raw(query).Scan(&usages).Error; err != nil {
		return nil, fmt.Errorf("分析索引使用情况失败: %w", err)
	}

	return usages, nil
}

// IndexUsage 索引使用情况
type IndexUsage struct {
	SchemaName   string `json:"schemaName"`
	TableName    string `json:"tableName"`
	IndexName    string `json:"indexName"`
	ScanCount    int64  `json:"scanCount"`    // 索引扫描次数
	TuplesRead   int64  `json:"tuplesRead"`   // 读取的元组数
	TuplesFetch  int64  `json:"tuplesFetch"`  // 获取的元组数
}

// IsUnused 判断索引是否未使用
func (u *IndexUsage) IsUnused() bool {
	return u.ScanCount == 0
}

// GetUnusedIndexes 获取未使用的索引
func (v *IndexValidator) GetUnusedIndexes(ctx context.Context) ([]IndexUsage, error) {
	usages, err := v.AnalyzeIndexUsage(ctx)
	if err != nil {
		return nil, err
	}

	unusedIndexes := make([]IndexUsage, 0)
	for _, usage := range usages {
		if usage.IsUnused() && !strings.HasSuffix(usage.IndexName, "_pkey") {
			unusedIndexes = append(unusedIndexes, usage)
		}
	}

	return unusedIndexes, nil
}

// getRequiredIndexes 获取所有必需的索引定义
func (v *IndexValidator) getRequiredIndexes() []MissingIndex {
	return []MissingIndex{
		// conversation_memories 表索引
		{
			TableName:   "conversation_memories",
			Columns:     []string{"tenant_id", "session_id"},
			Reason:      "用于按租户和会话查询记忆",
			Recommended: true,
		},
		{
			TableName:   "conversation_memories",
			Columns:     []string{"memory_type"},
			Reason:      "用于按记忆类型过滤",
			Recommended: true,
		},
		{
			TableName:   "conversation_memories",
			Columns:     []string{"expires_at"},
			Reason:      "用于查询过期记忆",
			Recommended: true,
		},
		{
			TableName:   "conversation_memories",
			Columns:     []string{"created_at"},
			Reason:      "用于按创建时间排序",
			Recommended: true,
		},

		// conversation_contexts 表索引
		{
			TableName:   "conversation_contexts",
			Columns:     []string{"tenant_id"},
			Reason:      "用于按租户查询上下文配置",
			Recommended: true,
		},
		{
			TableName:   "conversation_contexts",
			Columns:     []string{"session_id"},
			Reason:      "用于按会话查询上下文配置（唯一索引）",
			Recommended: true,
		},
		{
			TableName:   "conversation_contexts",
			Columns:     []string{"updated_at"},
			Reason:      "用于查询最近更新的上下文",
			Recommended: true,
		},

		// conversation_summaries 表索引
		{
			TableName:   "conversation_summaries",
			Columns:     []string{"tenant_id", "session_id"},
			Reason:      "用于按租户和会话查询摘要",
			Recommended: true,
		},
		{
			TableName:   "conversation_summaries",
			Columns:     []string{"created_at"},
			Reason:      "用于按创建时间排序",
			Recommended: true,
		},
		{
			TableName:   "conversation_summaries",
			Columns:     []string{"session_id", "created_at"},
			Reason:      "用于获取会话的最新摘要",
			Recommended: true,
		},

		// chat_sessions 表索引（如果存在）
		{
			TableName:   "chat_sessions",
			Columns:     []string{"tenant_id"},
			Reason:      "用于按租户查询会话",
			Recommended: true,
		},
		{
			TableName:   "chat_sessions",
			Columns:     []string{"user_id"},
			Reason:      "用于按用户查询会话",
			Recommended: true,
		},
		{
			TableName:   "chat_sessions",
			Columns:     []string{"created_at"},
			Reason:      "用于按创建时间排序",
			Recommended: true,
		},

		// chat_messages 表索引（如果存在）
		{
			TableName:   "chat_messages",
			Columns:     []string{"session_id"},
			Reason:      "用于按会话查询消息",
			Recommended: true,
		},
		{
			TableName:   "chat_messages",
			Columns:     []string{"created_at"},
			Reason:      "用于按创建时间排序",
			Recommended: true,
		},

		// tenants 表索引
		{
			TableName:   "tenants",
			Columns:     []string{"domain"},
			Reason:      "用于按域名查询租户",
			Recommended: true,
		},
		{
			TableName:   "tenants",
			Columns:     []string{"status"},
			Reason:      "用于按状态过滤租户",
			Recommended: true,
		},

		// users 表索引
		{
			TableName:   "users",
			Columns:     []string{"tenant_id"},
			Reason:      "用于按租户查询用户",
			Recommended: true,
		},
		{
			TableName:   "users",
			Columns:     []string{"email"},
			Reason:      "用于按邮箱查询用户（唯一索引）",
			Recommended: true,
		},
	}
}

// indexExists 检查索引是否存在
func (v *IndexValidator) indexExists(required MissingIndex, existing []IndexInfo) bool {
	for _, idx := range existing {
		if idx.TableName != required.TableName {
			continue
		}

		// 检查列是否匹配
		if v.columnsMatch(required.Columns, idx.Columns) {
			return true
		}
	}
	return false
}

// columnsMatch 检查列是否匹配
func (v *IndexValidator) columnsMatch(required, existing []string) bool {
	if len(required) != len(existing) {
		return false
	}

	for i, col := range required {
		if col != existing[i] {
			return false
		}
	}

	return true
}

// CreateMissingIndexes 创建缺失的索引
func (v *IndexValidator) CreateMissingIndexes(ctx context.Context, missingIndexes []MissingIndex) error {
	for _, missing := range missingIndexes {
		if !missing.Recommended {
			continue
		}

		indexName := fmt.Sprintf("idx_%s_%s", missing.TableName, strings.Join(missing.Columns, "_"))
		columns := strings.Join(missing.Columns, ", ")

		sql := fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s (%s) WHERE is_deleted = false",
			indexName,
			missing.TableName,
			columns,
		)

		if err := v.db.WithContext(ctx).Exec(sql).Error; err != nil {
			return fmt.Errorf("创建索引 %s 失败: %w", indexName, err)
		}
	}

	return nil
}

// GenerateIndexReport 生成索引报告
func (v *IndexValidator) GenerateIndexReport(ctx context.Context) (*IndexReport, error) {
	// 获取所有索引
	allIndexes, err := v.GetAllIndexes(ctx)
	if err != nil {
		return nil, err
	}

	// 验证索引
	missingIndexes, err := v.ValidateIndexes(ctx)
	if err != nil {
		return nil, err
	}

	// 分析索引使用情况
	unusedIndexes, err := v.GetUnusedIndexes(ctx)
	if err != nil {
		return nil, err
	}

	return &IndexReport{
		TotalIndexes:   len(allIndexes),
		MissingIndexes: missingIndexes,
		UnusedIndexes:  unusedIndexes,
		AllIndexes:     allIndexes,
	}, nil
}

// IndexReport 索引报告
type IndexReport struct {
	TotalIndexes   int            `json:"totalIndexes"`
	MissingIndexes []MissingIndex `json:"missingIndexes"`
	UnusedIndexes  []IndexUsage   `json:"unusedIndexes"`
	AllIndexes     []IndexInfo    `json:"allIndexes"`
}

// HasIssues 检查是否有索引问题
func (r *IndexReport) HasIssues() bool {
	return len(r.MissingIndexes) > 0 || len(r.UnusedIndexes) > 0
}

// GetSummary 获取报告摘要
func (r *IndexReport) GetSummary() string {
	return fmt.Sprintf(
		"索引总数: %d, 缺失索引: %d, 未使用索引: %d",
		r.TotalIndexes,
		len(r.MissingIndexes),
		len(r.UnusedIndexes),
	)
}
