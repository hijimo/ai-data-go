package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"your-project/internal/model"
)

// ProjectRepository 项目仓库接口
type ProjectRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, project *model.Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	GetByIDWithMembers(ctx context.Context, id uuid.UUID) (*model.Project, error)
	Update(ctx context.Context, project *model.Project) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.Project, int64, error)

	// 成员管理
	AddMember(ctx context.Context, member *model.ProjectMember) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error
	GetMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error)

	// 数据迁移相关
	ExportProjectData(ctx context.Context, projectID uuid.UUID) (*ProjectExportData, error)
	ImportProjectData(ctx context.Context, targetProjectID uuid.UUID, data *ProjectImportData) error
	GetProjectDataStats(ctx context.Context, projectID uuid.UUID) (*ProjectDataStats, error)
}

// ProjectExportData 项目导出数据结构
type ProjectExportData struct {
	Project          *model.Project          `json:"project"`
	Members          []*model.ProjectMember  `json:"members"`
	Files            []*FileExportData       `json:"files"`
	Agents           []*AgentExportData      `json:"agents"`
	ChatSessions     []*ChatSessionExportData `json:"chat_sessions"`
	Questions        []*QuestionExportData   `json:"questions"`
	VectorIndexes    []*VectorIndexExportData `json:"vector_indexes"`
	TrainingJobs     []*TrainingJobExportData `json:"training_jobs"`
	ExportedAt       time.Time               `json:"exported_at"`
	ExportVersion    string                  `json:"export_version"`
}

// ProjectImportData 项目导入数据结构
type ProjectImportData struct {
	Files            []*FileExportData       `json:"files"`
	Agents           []*AgentExportData      `json:"agents"`
	ChatSessions     []*ChatSessionExportData `json:"chat_sessions"`
	Questions        []*QuestionExportData   `json:"questions"`
	VectorIndexes    []*VectorIndexExportData `json:"vector_indexes"`
	TrainingJobs     []*TrainingJobExportData `json:"training_jobs"`
	ImportOptions    *ImportOptions          `json:"import_options"`
}

// ImportOptions 导入选项
type ImportOptions struct {
	IncludeFiles         bool `json:"include_files"`
	IncludeAgents        bool `json:"include_agents"`
	IncludeChatSessions  bool `json:"include_chat_sessions"`
	IncludeQuestions     bool `json:"include_questions"`
	IncludeVectorIndexes bool `json:"include_vector_indexes"`
	IncludeTrainingJobs  bool `json:"include_training_jobs"`
	OverwriteExisting    bool `json:"overwrite_existing"`
}

// ProjectDataStats 项目数据统计
type ProjectDataStats struct {
	ProjectID        uuid.UUID `json:"project_id"`
	FilesCount       int64     `json:"files_count"`
	ChunksCount      int64     `json:"chunks_count"`
	AgentsCount      int64     `json:"agents_count"`
	ChatSessionsCount int64    `json:"chat_sessions_count"`
	QuestionsCount   int64     `json:"questions_count"`
	AnswersCount     int64     `json:"answers_count"`
	VectorIndexesCount int64   `json:"vector_indexes_count"`
	TrainingJobsCount int64    `json:"training_jobs_count"`
	TotalSize        int64     `json:"total_size"` // 文件总大小（字节）
}

// 导出数据子结构
type FileExportData struct {
	ID             uuid.UUID                `json:"id"`
	Name           string                   `json:"name"`
	OriginalName   string                   `json:"original_name"`
	MimeType       string                   `json:"mime_type"`
	Size           int64                    `json:"size"`
	SHA256         string                   `json:"sha256"`
	OSSPath        string                   `json:"oss_path"`
	UploaderID     uuid.UUID                `json:"uploader_id"`
	Status         int                      `json:"status"`
	Metadata       map[string]interface{}   `json:"metadata"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
	Versions       []*DocumentVersionExportData `json:"versions"`
}

type DocumentVersionExportData struct {
	ID          uuid.UUID              `json:"id"`
	Version     int                    `json:"version"`
	ChunkConfig map[string]interface{} `json:"chunk_config"`
	ChunkCount  int                    `json:"chunk_count"`
	Status      int                    `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	Chunks      []*ChunkExportData     `json:"chunks"`
}

type ChunkExportData struct {
	ID               uuid.UUID              `json:"id"`
	Sequence         int                    `json:"sequence"`
	Content          string                 `json:"content"`
	Metadata         map[string]interface{} `json:"metadata"`
	EmbeddingStatus  int                    `json:"embedding_status"`
	CreatedAt        time.Time              `json:"created_at"`
	VectorRecords    []*VectorRecordExportData `json:"vector_records"`
}

type VectorRecordExportData struct {
	ID             uuid.UUID `json:"id"`
	ExternalID     string    `json:"external_id"`
	EmbeddingModel string    `json:"embedding_model"`
	CreatedAt      time.Time `json:"created_at"`
}

type AgentExportData struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	Description  *string                `json:"description"`
	SystemPrompt *string                `json:"system_prompt"`
	LLMModelID   uuid.UUID              `json:"llm_model_id"`
	Tools        []interface{}          `json:"tools"`
	Config       map[string]interface{} `json:"config"`
	CreatedBy    uuid.UUID              `json:"created_by"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type ChatSessionExportData struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	AgentID   *uuid.UUID             `json:"agent_id"`
	Title     *string                `json:"title"`
	Context   map[string]interface{} `json:"context"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Messages  []*ChatMessageExportData `json:"messages"`
}

type ChatMessageExportData struct {
	ID        uuid.UUID              `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

type QuestionExportData struct {
	ID           uuid.UUID     `json:"id"`
	ChunkID      *uuid.UUID    `json:"chunk_id"`
	Content      string        `json:"content"`
	QuestionType *string       `json:"question_type"`
	Tags         []interface{} `json:"tags"`
	Difficulty   int           `json:"difficulty"`
	Status       int           `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	Answers      []*AnswerExportData `json:"answers"`
}

type AnswerExportData struct {
	ID           uuid.UUID  `json:"id"`
	Content      string     `json:"content"`
	Reasoning    *string    `json:"reasoning"`
	LLMModelID   *uuid.UUID `json:"llm_model_id"`
	QualityScore *float64   `json:"quality_score"`
	IsReviewed   bool       `json:"is_reviewed"`
	CreatedAt    time.Time  `json:"created_at"`
}

type VectorIndexExportData struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Provider  string                 `json:"provider"`
	Config    map[string]interface{} `json:"config"`
	Status    int                    `json:"status"`
	Stats     map[string]interface{} `json:"stats"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type TrainingJobExportData struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	DatasetPath   string                 `json:"dataset_path"`
	Config        map[string]interface{} `json:"config"`
	ExternalJobID *string                `json:"external_job_id"`
	Status        int                    `json:"status"`
	Progress      int                    `json:"progress"`
	Result        map[string]interface{} `json:"result"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// projectRepository 项目仓库实现
type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository 创建项目仓库实例
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

// Create 创建项目
func (r *projectRepository) Create(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

// GetByID 根据ID获取项目
func (r *projectRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetByIDWithMembers 根据ID获取项目（包含成员）
func (r *projectRepository) GetByIDWithMembers(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Preload("Members", "is_deleted = ?", false).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// Update 更新项目
func (r *projectRepository) Update(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

// SoftDelete 软删除项目
func (r *projectRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
			"updated_at": now,
		}).Error
}

// List 获取项目列表
func (r *projectRepository) List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.Project, int64, error) {
	var projects []*model.Project
	var total int64

	// 获取总数
	err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("owner_id = ? AND is_deleted = ?", ownerID, false).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取项目列表
	err = r.db.WithContext(ctx).
		Where("owner_id = ? AND is_deleted = ?", ownerID, false).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&projects).Error
	if err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// AddMember 添加项目成员
func (r *projectRepository) AddMember(ctx context.Context, member *model.ProjectMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// RemoveMember 移除项目成员
func (r *projectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND is_deleted = ?", projectID, userID, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
		}).Error
}

// UpdateMemberRole 更新成员角色
func (r *projectRepository) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	return r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND is_deleted = ?", projectID, userID, false).
		Update("role", role).Error
}

// GetMembers 获取项目成员列表
func (r *projectRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*model.ProjectMember, error) {
	var members []*model.ProjectMember
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}// Ge
tProjectDataStats 获取项目数据统计
func (r *projectRepository) GetProjectDataStats(ctx context.Context, projectID uuid.UUID) (*ProjectDataStats, error) {
	stats := &ProjectDataStats{
		ProjectID: projectID,
	}

	// 统计文件数量和总大小
	err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Select("COUNT(*) as count, COALESCE(SUM(size), 0) as total_size").
		Row().
		Scan(&stats.FilesCount, &stats.TotalSize)
	if err != nil {
		return nil, fmt.Errorf("统计文件数据失败: %w", err)
	}

	// 统计文档块数量
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(c.id) 
		FROM chunks c
		JOIN document_versions dv ON c.document_version_id = dv.id
		JOIN files f ON dv.file_id = f.id
		WHERE f.project_id = ? AND c.is_deleted = ? AND dv.is_deleted = ? AND f.is_deleted = ?
	`, projectID, false, false, false).
		Row().
		Scan(&stats.ChunksCount)
	if err != nil {
		return nil, fmt.Errorf("统计文档块数据失败: %w", err)
	}

	// 统计Agent数量
	err = r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Count(&stats.AgentsCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计Agent数据失败: %w", err)
	}

	// 统计对话会话数量
	err = r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Count(&stats.ChatSessionsCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计对话会话数据失败: %w", err)
	}

	// 统计问题数量
	err = r.db.WithContext(ctx).
		Model(&model.Question{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Count(&stats.QuestionsCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计问题数据失败: %w", err)
	}

	// 统计答案数量
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(a.id)
		FROM answers a
		JOIN questions q ON a.question_id = q.id
		WHERE q.project_id = ? AND a.is_deleted = ? AND q.is_deleted = ?
	`, projectID, false, false).
		Row().
		Scan(&stats.AnswersCount)
	if err != nil {
		return nil, fmt.Errorf("统计答案数据失败: %w", err)
	}

	// 统计向量索引数量
	err = r.db.WithContext(ctx).
		Model(&model.VectorIndex{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Count(&stats.VectorIndexesCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计向量索引数据失败: %w", err)
	}

	// 统计训练任务数量
	err = r.db.WithContext(ctx).
		Model(&model.TrainingJob{}).
		Where("project_id = ? AND is_deleted = ?", projectID, false).
		Count(&stats.TrainingJobsCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计训练任务数据失败: %w", err)
	}

	return stats, nil
}

// ExportProjectData 导出项目数据
func (r *projectRepository) ExportProjectData(ctx context.Context, projectID uuid.UUID) (*ProjectExportData, error) {
	exportData := &ProjectExportData{
		ExportedAt:    time.Now(),
		ExportVersion: "1.0",
	}

	// 导出项目基本信息
	project, err := r.GetByIDWithMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("获取项目信息失败: %w", err)
	}
	exportData.Project = project
	exportData.Members = project.Members

	// 导出文件数据
	files, err := r.exportFiles(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出文件数据失败: %w", err)
	}
	exportData.Files = files

	// 导出Agent数据
	agents, err := r.exportAgents(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出Agent数据失败: %w", err)
	}
	exportData.Agents = agents

	// 导出对话会话数据
	chatSessions, err := r.exportChatSessions(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出对话会话数据失败: %w", err)
	}
	exportData.ChatSessions = chatSessions

	// 导出问题数据
	questions, err := r.exportQuestions(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出问题数据失败: %w", err)
	}
	exportData.Questions = questions

	// 导出向量索引数据
	vectorIndexes, err := r.exportVectorIndexes(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出向量索引数据失败: %w", err)
	}
	exportData.VectorIndexes = vectorIndexes

	// 导出训练任务数据
	trainingJobs, err := r.exportTrainingJobs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出训练任务数据失败: %w", err)
	}
	exportData.TrainingJobs = trainingJobs

	return exportData, nil
}

// exportFiles 导出文件数据
func (r *projectRepository) exportFiles(ctx context.Context, projectID uuid.UUID) ([]*FileExportData, error) {
	var files []*FileExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, original_name, mime_type, size, sha256, oss_path, 
			   uploader_id, status, metadata, created_at, updated_at
		FROM files 
		WHERE project_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, projectID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var file FileExportData
		var metadataJSON []byte
		
		err := rows.Scan(
			&file.ID, &file.Name, &file.OriginalName, &file.MimeType,
			&file.Size, &file.SHA256, &file.OSSPath, &file.UploaderID,
			&file.Status, &metadataJSON, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &file.Metadata); err != nil {
				file.Metadata = make(map[string]interface{})
			}
		} else {
			file.Metadata = make(map[string]interface{})
		}

		// 导出文档版本数据
		versions, err := r.exportDocumentVersions(ctx, file.ID)
		if err != nil {
			return nil, fmt.Errorf("导出文件 %s 的版本数据失败: %w", file.ID, err)
		}
		file.Versions = versions

		files = append(files, &file)
	}

	return files, nil
}

// exportDocumentVersions 导出文档版本数据
func (r *projectRepository) exportDocumentVersions(ctx context.Context, fileID uuid.UUID) ([]*DocumentVersionExportData, error) {
	var versions []*DocumentVersionExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, version, chunk_config, chunk_count, status, created_at
		FROM document_versions 
		WHERE file_id = ? AND is_deleted = ?
		ORDER BY version ASC
	`, fileID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version DocumentVersionExportData
		var chunkConfigJSON []byte
		
		err := rows.Scan(
			&version.ID, &version.Version, &chunkConfigJSON,
			&version.ChunkCount, &version.Status, &version.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析chunk_config JSON
		if len(chunkConfigJSON) > 0 {
			if err := json.Unmarshal(chunkConfigJSON, &version.ChunkConfig); err != nil {
				version.ChunkConfig = make(map[string]interface{})
			}
		} else {
			version.ChunkConfig = make(map[string]interface{})
		}

		// 导出文档块数据
		chunks, err := r.exportChunks(ctx, version.ID)
		if err != nil {
			return nil, fmt.Errorf("导出版本 %s 的块数据失败: %w", version.ID, err)
		}
		version.Chunks = chunks

		versions = append(versions, &version)
	}

	return versions, nil
}

// exportChunks 导出文档块数据
func (r *projectRepository) exportChunks(ctx context.Context, versionID uuid.UUID) ([]*ChunkExportData, error) {
	var chunks []*ChunkExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, sequence, content, metadata, embedding_status, created_at
		FROM chunks 
		WHERE document_version_id = ? AND is_deleted = ?
		ORDER BY sequence ASC
	`, versionID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chunk ChunkExportData
		var metadataJSON []byte
		
		err := rows.Scan(
			&chunk.ID, &chunk.Sequence, &chunk.Content,
			&metadataJSON, &chunk.EmbeddingStatus, &chunk.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &chunk.Metadata); err != nil {
				chunk.Metadata = make(map[string]interface{})
			}
		} else {
			chunk.Metadata = make(map[string]interface{})
		}

		// 导出向量记录数据
		vectorRecords, err := r.exportVectorRecords(ctx, chunk.ID)
		if err != nil {
			return nil, fmt.Errorf("导出块 %s 的向量记录失败: %w", chunk.ID, err)
		}
		chunk.VectorRecords = vectorRecords

		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

// exportVectorRecords 导出向量记录数据
func (r *projectRepository) exportVectorRecords(ctx context.Context, chunkID uuid.UUID) ([]*VectorRecordExportData, error) {
	var records []*VectorRecordExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, external_id, embedding_model, created_at
		FROM vector_records 
		WHERE chunk_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, chunkID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record VectorRecordExportData
		err := rows.Scan(
			&record.ID, &record.ExternalID,
			&record.EmbeddingModel, &record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

// exportAgents 导出Agent数据
func (r *projectRepository) exportAgents(ctx context.Context, projectID uuid.UUID) ([]*AgentExportData, error) {
	var agents []*AgentExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, description, system_prompt, llm_model_id, 
			   tools, config, created_by, created_at, updated_at
		FROM agents 
		WHERE project_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, projectID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var agent AgentExportData
		var toolsJSON, configJSON []byte
		
		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.Description, &agent.SystemPrompt,
			&agent.LLMModelID, &toolsJSON, &configJSON, &agent.CreatedBy,
			&agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析tools JSON
		if len(toolsJSON) > 0 {
			if err := json.Unmarshal(toolsJSON, &agent.Tools); err != nil {
				agent.Tools = make([]interface{}, 0)
			}
		} else {
			agent.Tools = make([]interface{}, 0)
		}

		// 解析config JSON
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &agent.Config); err != nil {
				agent.Config = make(map[string]interface{})
			}
		} else {
			agent.Config = make(map[string]interface{})
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

// exportChatSessions 导出对话会话数据
func (r *projectRepository) exportChatSessions(ctx context.Context, projectID uuid.UUID) ([]*ChatSessionExportData, error) {
	var sessions []*ChatSessionExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, agent_id, title, context, created_at, updated_at
		FROM chat_sessions 
		WHERE project_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, projectID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var session ChatSessionExportData
		var contextJSON []byte
		
		err := rows.Scan(
			&session.ID, &session.UserID, &session.AgentID, &session.Title,
			&contextJSON, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析context JSON
		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &session.Context); err != nil {
				session.Context = make(map[string]interface{})
			}
		} else {
			session.Context = make(map[string]interface{})
		}

		// 导出消息数据
		messages, err := r.exportChatMessages(ctx, session.ID)
		if err != nil {
			return nil, fmt.Errorf("导出会话 %s 的消息失败: %w", session.ID, err)
		}
		session.Messages = messages

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// exportChatMessages 导出对话消息数据
func (r *projectRepository) exportChatMessages(ctx context.Context, sessionID uuid.UUID) ([]*ChatMessageExportData, error) {
	var messages []*ChatMessageExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, role, content, metadata, created_at
		FROM chat_messages 
		WHERE session_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, sessionID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message ChatMessageExportData
		var metadataJSON []byte
		
		err := rows.Scan(
			&message.ID, &message.Role, &message.Content,
			&metadataJSON, &message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &message.Metadata); err != nil {
				message.Metadata = make(map[string]interface{})
			}
		} else {
			message.Metadata = make(map[string]interface{})
		}

		messages = append(messages, &message)
	}

	return messages, nil
}

// exportQuestions 导出问题数据
func (r *projectRepository) exportQuestions(ctx context.Context, projectID uuid.UUID) ([]*QuestionExportData, error) {
	var questions []*QuestionExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, chunk_id, content, question_type, tags, difficulty, status, created_at
		FROM questions 
		WHERE project_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, projectID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question QuestionExportData
		var tagsJSON []byte
		
		err := rows.Scan(
			&question.ID, &question.ChunkID, &question.Content, &question.QuestionType,
			&tagsJSON, &question.Difficulty, &question.Status, &question.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析tags JSON
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &question.Tags); err != nil {
				question.Tags = make([]interface{}, 0)
			}
		} else {
			question.Tags = make([]interface{}, 0)
		}

		// 导出答案数据
		answers, err := r.exportAnswers(ctx, question.ID)
		if err != nil {
			return nil, fmt.Errorf("导出问题 %s 的答案失败: %w", question.ID, err)
		}
		question.Answers = answers

		questions = append(questions, &question)
	}

	return questions, nil
}

// exportAnswers 导出答案数据
func (r *projectRepository) exportAnswers(ctx context.Context, questionID uuid.UUID) ([]*AnswerExportData, error) {
	var answers []*AnswerExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, content, reasoning, llm_model_id, quality_score, is_reviewed, created_at
		FROM answers 
		WHERE question_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, questionID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var answer AnswerExportData
		err := rows.Scan(
			&answer.ID, &answer.Content, &answer.Reasoning, &answer.LLMModelID,
			&answer.QualityScore, &answer.IsReviewed, &answer.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		answers = append(answers, &answer)
	}

	return answers, nil
}

// exportVectorIndexes 导出向量索引数据
func (r *projectRepository) exportVectorIndexes(ctx context.Context, projectID uuid.UUID) ([]*VectorIndexExportData, error) {
	var indexes []*VectorIndexExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, provider, config, status, stats, created_at, updated_at
		FROM vector_indexes 
		WHERE project_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, projectID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var index VectorIndexExportData
		var configJSON, statsJSON []byte
		
		err := rows.Scan(
			&index.ID, &index.Name, &index.Provider,
			&configJSON, &index.Status, &statsJSON,
			&index.CreatedAt, &index.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析config JSON
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &index.Config); err != nil {
				index.Config = make(map[string]interface{})
			}
		} else {
			index.Config = make(map[string]interface{})
		}

		// 解析stats JSON
		if len(statsJSON) > 0 {
			if err := json.Unmarshal(statsJSON, &index.Stats); err != nil {
				index.Stats = make(map[string]interface{})
			}
		} else {
			index.Stats = make(map[string]interface{})
		}

		indexes = append(indexes, &index)
	}

	return indexes, nil
}

// exportTrainingJobs 导出训练任务数据
func (r *projectRepository) exportTrainingJobs(ctx context.Context, projectID uuid.UUID) ([]*TrainingJobExportData, error) {
	var jobs []*TrainingJobExportData

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, dataset_path, config, external_job_id, 
			   status, progress, result, created_at, updated_at
		FROM training_jobs 
		WHERE project_id = ? AND is_deleted = ?
		ORDER BY created_at ASC
	`, projectID, false).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var job TrainingJobExportData
		var configJSON, resultJSON []byte
		
		err := rows.Scan(
			&job.ID, &job.Name, &job.DatasetPath, &configJSON, &job.ExternalJobID,
			&job.Status, &job.Progress, &resultJSON, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析config JSON
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &job.Config); err != nil {
				job.Config = make(map[string]interface{})
			}
		} else {
			job.Config = make(map[string]interface{})
		}

		// 解析result JSON
		if len(resultJSON) > 0 {
			if err := json.Unmarshal(resultJSON, &job.Result); err != nil {
				job.Result = make(map[string]interface{})
			}
		} else {
			job.Result = make(map[string]interface{})
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}// Imp
ortProjectData 导入项目数据
func (r *projectRepository) ImportProjectData(ctx context.Context, targetProjectID uuid.UUID, data *ProjectImportData) error {
	// 开启事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 验证目标项目是否存在
		var project model.Project
		if err := tx.Where("id = ? AND is_deleted = ?", targetProjectID, false).First(&project).Error; err != nil {
			return fmt.Errorf("目标项目不存在: %w", err)
		}

		// 根据导入选项导入不同类型的数据
		if data.ImportOptions.IncludeFiles && len(data.Files) > 0 {
			if err := r.importFiles(ctx, tx, targetProjectID, data.Files, data.ImportOptions.OverwriteExisting); err != nil {
				return fmt.Errorf("导入文件数据失败: %w", err)
			}
		}

		if data.ImportOptions.IncludeAgents && len(data.Agents) > 0 {
			if err := r.importAgents(ctx, tx, targetProjectID, data.Agents, data.ImportOptions.OverwriteExisting); err != nil {
				return fmt.Errorf("导入Agent数据失败: %w", err)
			}
		}

		if data.ImportOptions.IncludeChatSessions && len(data.ChatSessions) > 0 {
			if err := r.importChatSessions(ctx, tx, targetProjectID, data.ChatSessions, data.ImportOptions.OverwriteExisting); err != nil {
				return fmt.Errorf("导入对话会话数据失败: %w", err)
			}
		}

		if data.ImportOptions.IncludeQuestions && len(data.Questions) > 0 {
			if err := r.importQuestions(ctx, tx, targetProjectID, data.Questions, data.ImportOptions.OverwriteExisting); err != nil {
				return fmt.Errorf("导入问题数据失败: %w", err)
			}
		}

		if data.ImportOptions.IncludeVectorIndexes && len(data.VectorIndexes) > 0 {
			if err := r.importVectorIndexes(ctx, tx, targetProjectID, data.VectorIndexes, data.ImportOptions.OverwriteExisting); err != nil {
				return fmt.Errorf("导入向量索引数据失败: %w", err)
			}
		}

		if data.ImportOptions.IncludeTrainingJobs && len(data.TrainingJobs) > 0 {
			if err := r.importTrainingJobs(ctx, tx, targetProjectID, data.TrainingJobs, data.ImportOptions.OverwriteExisting); err != nil {
				return fmt.Errorf("导入训练任务数据失败: %w", err)
			}
		}

		return nil
	})
}

// importFiles 导入文件数据
func (r *projectRepository) importFiles(ctx context.Context, tx *gorm.DB, projectID uuid.UUID, files []*FileExportData, overwrite bool) error {
	for _, fileData := range files {
		// 检查文件是否已存在（基于SHA256）
		var existingFile model.File
		err := tx.Where("project_id = ? AND sha256 = ? AND is_deleted = ?", projectID, fileData.SHA256, false).First(&existingFile).Error
		
		if err == nil {
			// 文件已存在
			if !overwrite {
				continue // 跳过已存在的文件
			}
			// 更新现有文件
			existingFile.Name = fileData.Name
			existingFile.OriginalName = fileData.OriginalName
			existingFile.MimeType = fileData.MimeType
			existingFile.Size = fileData.Size
			existingFile.OSSPath = fileData.OSSPath
			existingFile.Status = fileData.Status
			existingFile.UpdatedAt = time.Now()
			
			metadataJSON, _ := json.Marshal(fileData.Metadata)
			if err := tx.Model(&existingFile).Updates(map[string]interface{}{
				"name":          existingFile.Name,
				"original_name": existingFile.OriginalName,
				"mime_type":     existingFile.MimeType,
				"size":          existingFile.Size,
				"oss_path":      existingFile.OSSPath,
				"status":        existingFile.Status,
				"metadata":      string(metadataJSON),
				"updated_at":    existingFile.UpdatedAt,
			}).Error; err != nil {
				return fmt.Errorf("更新文件 %s 失败: %w", fileData.Name, err)
			}
			
			// 导入文档版本
			if err := r.importDocumentVersions(ctx, tx, existingFile.ID, fileData.Versions, overwrite); err != nil {
				return fmt.Errorf("导入文件 %s 的版本数据失败: %w", fileData.Name, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新文件
			newFile := model.File{
				ID:           uuid.New(),
				ProjectID:    projectID,
				Name:         fileData.Name,
				OriginalName: fileData.OriginalName,
				MimeType:     fileData.MimeType,
				Size:         fileData.Size,
				SHA256:       fileData.SHA256,
				OSSPath:      fileData.OSSPath,
				UploaderID:   fileData.UploaderID,
				Status:       fileData.Status,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			
			metadataJSON, _ := json.Marshal(fileData.Metadata)
			if err := tx.Exec(`
				INSERT INTO files (id, project_id, name, original_name, mime_type, size, sha256, oss_path, uploader_id, status, metadata, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newFile.ID, newFile.ProjectID, newFile.Name, newFile.OriginalName, newFile.MimeType,
				newFile.Size, newFile.SHA256, newFile.OSSPath, newFile.UploaderID, newFile.Status,
				string(metadataJSON), newFile.CreatedAt, newFile.UpdatedAt).Error; err != nil {
				return fmt.Errorf("创建文件 %s 失败: %w", fileData.Name, err)
			}
			
			// 导入文档版本
			if err := r.importDocumentVersions(ctx, tx, newFile.ID, fileData.Versions, overwrite); err != nil {
				return fmt.Errorf("导入文件 %s 的版本数据失败: %w", fileData.Name, err)
			}
		} else {
			return fmt.Errorf("检查文件 %s 是否存在时出错: %w", fileData.Name, err)
		}
	}
	return nil
}

// importDocumentVersions 导入文档版本数据
func (r *projectRepository) importDocumentVersions(ctx context.Context, tx *gorm.DB, fileID uuid.UUID, versions []*DocumentVersionExportData, overwrite bool) error {
	for _, versionData := range versions {
		// 检查版本是否已存在
		var existingVersion model.DocumentVersion
		err := tx.Where("file_id = ? AND version = ? AND is_deleted = ?", fileID, versionData.Version, false).First(&existingVersion).Error
		
		if err == nil {
			// 版本已存在
			if !overwrite {
				continue
			}
			// 更新现有版本
			chunkConfigJSON, _ := json.Marshal(versionData.ChunkConfig)
			if err := tx.Model(&existingVersion).Updates(map[string]interface{}{
				"chunk_config": string(chunkConfigJSON),
				"chunk_count":  versionData.ChunkCount,
				"status":       versionData.Status,
			}).Error; err != nil {
				return fmt.Errorf("更新文档版本 %d 失败: %w", versionData.Version, err)
			}
			
			// 导入文档块
			if err := r.importChunks(ctx, tx, existingVersion.ID, versionData.Chunks, overwrite); err != nil {
				return fmt.Errorf("导入版本 %d 的块数据失败: %w", versionData.Version, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新版本
			newVersionID := uuid.New()
			chunkConfigJSON, _ := json.Marshal(versionData.ChunkConfig)
			
			if err := tx.Exec(`
				INSERT INTO document_versions (id, file_id, version, chunk_config, chunk_count, status, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, newVersionID, fileID, versionData.Version, string(chunkConfigJSON),
				versionData.ChunkCount, versionData.Status, time.Now()).Error; err != nil {
				return fmt.Errorf("创建文档版本 %d 失败: %w", versionData.Version, err)
			}
			
			// 导入文档块
			if err := r.importChunks(ctx, tx, newVersionID, versionData.Chunks, overwrite); err != nil {
				return fmt.Errorf("导入版本 %d 的块数据失败: %w", versionData.Version, err)
			}
		} else {
			return fmt.Errorf("检查文档版本 %d 是否存在时出错: %w", versionData.Version, err)
		}
	}
	return nil
}

// importChunks 导入文档块数据
func (r *projectRepository) importChunks(ctx context.Context, tx *gorm.DB, versionID uuid.UUID, chunks []*ChunkExportData, overwrite bool) error {
	for _, chunkData := range chunks {
		// 检查块是否已存在
		var existingChunk model.Chunk
		err := tx.Where("document_version_id = ? AND sequence = ? AND is_deleted = ?", versionID, chunkData.Sequence, false).First(&existingChunk).Error
		
		if err == nil {
			// 块已存在
			if !overwrite {
				continue
			}
			// 更新现有块
			metadataJSON, _ := json.Marshal(chunkData.Metadata)
			if err := tx.Model(&existingChunk).Updates(map[string]interface{}{
				"content":          chunkData.Content,
				"metadata":         string(metadataJSON),
				"embedding_status": chunkData.EmbeddingStatus,
			}).Error; err != nil {
				return fmt.Errorf("更新文档块 %d 失败: %w", chunkData.Sequence, err)
			}
			
			// 导入向量记录
			if err := r.importVectorRecords(ctx, tx, existingChunk.ID, chunkData.VectorRecords, overwrite); err != nil {
				return fmt.Errorf("导入块 %d 的向量记录失败: %w", chunkData.Sequence, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新块
			newChunkID := uuid.New()
			metadataJSON, _ := json.Marshal(chunkData.Metadata)
			
			if err := tx.Exec(`
				INSERT INTO chunks (id, document_version_id, sequence, content, metadata, embedding_status, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, newChunkID, versionID, chunkData.Sequence, chunkData.Content,
				string(metadataJSON), chunkData.EmbeddingStatus, time.Now()).Error; err != nil {
				return fmt.Errorf("创建文档块 %d 失败: %w", chunkData.Sequence, err)
			}
			
			// 导入向量记录
			if err := r.importVectorRecords(ctx, tx, newChunkID, chunkData.VectorRecords, overwrite); err != nil {
				return fmt.Errorf("导入块 %d 的向量记录失败: %w", chunkData.Sequence, err)
			}
		} else {
			return fmt.Errorf("检查文档块 %d 是否存在时出错: %w", chunkData.Sequence, err)
		}
	}
	return nil
}

// importVectorRecords 导入向量记录数据
func (r *projectRepository) importVectorRecords(ctx context.Context, tx *gorm.DB, chunkID uuid.UUID, records []*VectorRecordExportData, overwrite bool) error {
	for _, recordData := range records {
		// 检查向量记录是否已存在
		var existingRecord model.VectorRecord
		err := tx.Where("chunk_id = ? AND external_id = ? AND is_deleted = ?", chunkID, recordData.ExternalID, false).First(&existingRecord).Error
		
		if err == nil {
			// 记录已存在
			if !overwrite {
				continue
			}
			// 更新现有记录
			if err := tx.Model(&existingRecord).Updates(map[string]interface{}{
				"embedding_model": recordData.EmbeddingModel,
			}).Error; err != nil {
				return fmt.Errorf("更新向量记录 %s 失败: %w", recordData.ExternalID, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新记录
			if err := tx.Exec(`
				INSERT INTO vector_records (id, chunk_id, vector_index_id, external_id, embedding_model, created_at)
				VALUES (?, ?, ?, ?, ?, ?)
			`, uuid.New(), chunkID, uuid.New(), recordData.ExternalID, recordData.EmbeddingModel, time.Now()).Error; err != nil {
				return fmt.Errorf("创建向量记录 %s 失败: %w", recordData.ExternalID, err)
			}
		} else {
			return fmt.Errorf("检查向量记录 %s 是否存在时出错: %w", recordData.ExternalID, err)
		}
	}
	return nil
}

// importAgents 导入Agent数据
func (r *projectRepository) importAgents(ctx context.Context, tx *gorm.DB, projectID uuid.UUID, agents []*AgentExportData, overwrite bool) error {
	for _, agentData := range agents {
		// 检查Agent是否已存在（基于名称）
		var existingAgent model.Agent
		err := tx.Where("project_id = ? AND name = ? AND is_deleted = ?", projectID, agentData.Name, false).First(&existingAgent).Error
		
		if err == nil {
			// Agent已存在
			if !overwrite {
				continue
			}
			// 更新现有Agent
			toolsJSON, _ := json.Marshal(agentData.Tools)
			configJSON, _ := json.Marshal(agentData.Config)
			
			if err := tx.Model(&existingAgent).Updates(map[string]interface{}{
				"description":    agentData.Description,
				"system_prompt":  agentData.SystemPrompt,
				"llm_model_id":   agentData.LLMModelID,
				"tools":          string(toolsJSON),
				"config":         string(configJSON),
				"updated_at":     time.Now(),
			}).Error; err != nil {
				return fmt.Errorf("更新Agent %s 失败: %w", agentData.Name, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新Agent
			toolsJSON, _ := json.Marshal(agentData.Tools)
			configJSON, _ := json.Marshal(agentData.Config)
			
			if err := tx.Exec(`
				INSERT INTO agents (id, project_id, name, description, system_prompt, llm_model_id, tools, config, created_by, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New(), projectID, agentData.Name, agentData.Description, agentData.SystemPrompt,
				agentData.LLMModelID, string(toolsJSON), string(configJSON), agentData.CreatedBy,
				time.Now(), time.Now()).Error; err != nil {
				return fmt.Errorf("创建Agent %s 失败: %w", agentData.Name, err)
			}
		} else {
			return fmt.Errorf("检查Agent %s 是否存在时出错: %w", agentData.Name, err)
		}
	}
	return nil
}

// importChatSessions 导入对话会话数据
func (r *projectRepository) importChatSessions(ctx context.Context, tx *gorm.DB, projectID uuid.UUID, sessions []*ChatSessionExportData, overwrite bool) error {
	for _, sessionData := range sessions {
		// 创建新会话（对话会话通常不需要检查重复）
		newSessionID := uuid.New()
		contextJSON, _ := json.Marshal(sessionData.Context)
		
		if err := tx.Exec(`
			INSERT INTO chat_sessions (id, project_id, user_id, agent_id, title, context, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, newSessionID, projectID, sessionData.UserID, sessionData.AgentID, sessionData.Title,
			string(contextJSON), time.Now(), time.Now()).Error; err != nil {
			return fmt.Errorf("创建对话会话失败: %w", err)
		}
		
		// 导入消息
		if err := r.importChatMessages(ctx, tx, newSessionID, sessionData.Messages); err != nil {
			return fmt.Errorf("导入对话会话消息失败: %w", err)
		}
	}
	return nil
}

// importChatMessages 导入对话消息数据
func (r *projectRepository) importChatMessages(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID, messages []*ChatMessageExportData) error {
	for _, messageData := range messages {
		metadataJSON, _ := json.Marshal(messageData.Metadata)
		
		if err := tx.Exec(`
			INSERT INTO chat_messages (id, session_id, role, content, metadata, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, uuid.New(), sessionID, messageData.Role, messageData.Content,
			string(metadataJSON), time.Now()).Error; err != nil {
			return fmt.Errorf("创建对话消息失败: %w", err)
		}
	}
	return nil
}

// importQuestions 导入问题数据
func (r *projectRepository) importQuestions(ctx context.Context, tx *gorm.DB, projectID uuid.UUID, questions []*QuestionExportData, overwrite bool) error {
	for _, questionData := range questions {
		// 检查问题是否已存在（基于内容）
		var existingQuestion model.Question
		err := tx.Where("project_id = ? AND content = ? AND is_deleted = ?", projectID, questionData.Content, false).First(&existingQuestion).Error
		
		if err == nil {
			// 问题已存在
			if !overwrite {
				continue
			}
			// 更新现有问题
			tagsJSON, _ := json.Marshal(questionData.Tags)
			
			if err := tx.Model(&existingQuestion).Updates(map[string]interface{}{
				"question_type": questionData.QuestionType,
				"tags":          string(tagsJSON),
				"difficulty":    questionData.Difficulty,
				"status":        questionData.Status,
			}).Error; err != nil {
				return fmt.Errorf("更新问题失败: %w", err)
			}
			
			// 导入答案
			if err := r.importAnswers(ctx, tx, existingQuestion.ID, questionData.Answers, overwrite); err != nil {
				return fmt.Errorf("导入问题答案失败: %w", err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新问题
			newQuestionID := uuid.New()
			tagsJSON, _ := json.Marshal(questionData.Tags)
			
			if err := tx.Exec(`
				INSERT INTO questions (id, project_id, chunk_id, content, question_type, tags, difficulty, status, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newQuestionID, projectID, questionData.ChunkID, questionData.Content,
				questionData.QuestionType, string(tagsJSON), questionData.Difficulty,
				questionData.Status, time.Now()).Error; err != nil {
				return fmt.Errorf("创建问题失败: %w", err)
			}
			
			// 导入答案
			if err := r.importAnswers(ctx, tx, newQuestionID, questionData.Answers, overwrite); err != nil {
				return fmt.Errorf("导入问题答案失败: %w", err)
			}
		} else {
			return fmt.Errorf("检查问题是否存在时出错: %w", err)
		}
	}
	return nil
}

// importAnswers 导入答案数据
func (r *projectRepository) importAnswers(ctx context.Context, tx *gorm.DB, questionID uuid.UUID, answers []*AnswerExportData, overwrite bool) error {
	for _, answerData := range answers {
		// 检查答案是否已存在（基于内容）
		var existingAnswer model.Answer
		err := tx.Where("question_id = ? AND content = ? AND is_deleted = ?", questionID, answerData.Content, false).First(&existingAnswer).Error
		
		if err == nil {
			// 答案已存在
			if !overwrite {
				continue
			}
			// 更新现有答案
			if err := tx.Model(&existingAnswer).Updates(map[string]interface{}{
				"reasoning":      answerData.Reasoning,
				"llm_model_id":   answerData.LLMModelID,
				"quality_score":  answerData.QualityScore,
				"is_reviewed":    answerData.IsReviewed,
			}).Error; err != nil {
				return fmt.Errorf("更新答案失败: %w", err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新答案
			if err := tx.Exec(`
				INSERT INTO answers (id, question_id, content, reasoning, llm_model_id, quality_score, is_reviewed, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New(), questionID, answerData.Content, answerData.Reasoning,
				answerData.LLMModelID, answerData.QualityScore, answerData.IsReviewed, time.Now()).Error; err != nil {
				return fmt.Errorf("创建答案失败: %w", err)
			}
		} else {
			return fmt.Errorf("检查答案是否存在时出错: %w", err)
		}
	}
	return nil
}

// importVectorIndexes 导入向量索引数据
func (r *projectRepository) importVectorIndexes(ctx context.Context, tx *gorm.DB, projectID uuid.UUID, indexes []*VectorIndexExportData, overwrite bool) error {
	for _, indexData := range indexes {
		// 检查向量索引是否已存在（基于名称）
		var existingIndex model.VectorIndex
		err := tx.Where("project_id = ? AND name = ? AND is_deleted = ?", projectID, indexData.Name, false).First(&existingIndex).Error
		
		if err == nil {
			// 索引已存在
			if !overwrite {
				continue
			}
			// 更新现有索引
			configJSON, _ := json.Marshal(indexData.Config)
			statsJSON, _ := json.Marshal(indexData.Stats)
			
			if err := tx.Model(&existingIndex).Updates(map[string]interface{}{
				"provider":   indexData.Provider,
				"config":     string(configJSON),
				"status":     indexData.Status,
				"stats":      string(statsJSON),
				"updated_at": time.Now(),
			}).Error; err != nil {
				return fmt.Errorf("更新向量索引 %s 失败: %w", indexData.Name, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新索引
			configJSON, _ := json.Marshal(indexData.Config)
			statsJSON, _ := json.Marshal(indexData.Stats)
			
			if err := tx.Exec(`
				INSERT INTO vector_indexes (id, project_id, name, provider, config, status, stats, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New(), projectID, indexData.Name, indexData.Provider,
				string(configJSON), indexData.Status, string(statsJSON), time.Now(), time.Now()).Error; err != nil {
				return fmt.Errorf("创建向量索引 %s 失败: %w", indexData.Name, err)
			}
		} else {
			return fmt.Errorf("检查向量索引 %s 是否存在时出错: %w", indexData.Name, err)
		}
	}
	return nil
}

// importTrainingJobs 导入训练任务数据
func (r *projectRepository) importTrainingJobs(ctx context.Context, tx *gorm.DB, projectID uuid.UUID, jobs []*TrainingJobExportData, overwrite bool) error {
	for _, jobData := range jobs {
		// 检查训练任务是否已存在（基于名称）
		var existingJob model.TrainingJob
		err := tx.Where("project_id = ? AND name = ? AND is_deleted = ?", projectID, jobData.Name, false).First(&existingJob).Error
		
		if err == nil {
			// 任务已存在
			if !overwrite {
				continue
			}
			// 更新现有任务
			configJSON, _ := json.Marshal(jobData.Config)
			resultJSON, _ := json.Marshal(jobData.Result)
			
			if err := tx.Model(&existingJob).Updates(map[string]interface{}{
				"dataset_path":    jobData.DatasetPath,
				"config":          string(configJSON),
				"external_job_id": jobData.ExternalJobID,
				"status":          jobData.Status,
				"progress":        jobData.Progress,
				"result":          string(resultJSON),
				"updated_at":      time.Now(),
			}).Error; err != nil {
				return fmt.Errorf("更新训练任务 %s 失败: %w", jobData.Name, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 创建新任务
			configJSON, _ := json.Marshal(jobData.Config)
			resultJSON, _ := json.Marshal(jobData.Result)
			
			if err := tx.Exec(`
				INSERT INTO training_jobs (id, project_id, name, dataset_path, config, external_job_id, status, progress, result, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New(), projectID, jobData.Name, jobData.DatasetPath,
				string(configJSON), jobData.ExternalJobID, jobData.Status, jobData.Progress,
				string(resultJSON), time.Now(), time.Now()).Error; err != nil {
				return fmt.Errorf("创建训练任务 %s 失败: %w", jobData.Name, err)
			}
		} else {
			return fmt.Errorf("检查训练任务 %s 是否存在时出错: %w", jobData.Name, err)
		}
	}
	return nil
}