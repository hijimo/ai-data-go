package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Task 任务模型
type Task struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID   *uuid.UUID     `json:"project_id" gorm:"type:uuid"`
	TaskType    string         `json:"task_type" gorm:"not null;size:50" validate:"required"`
	Status      int            `json:"status" gorm:"default:0"`    // 0:处理中, 1:已完成, 2:失败, 3:已中断
	Progress    int            `json:"progress" gorm:"default:0"`  // 进度百分比
	InputData   map[string]any `json:"input_data" gorm:"type:jsonb;not null"`
	OutputData  map[string]any `json:"output_data" gorm:"type:jsonb;default:'{}'"`
	ErrorMessage *string       `json:"error_message" gorm:"type:text"`
	StartedAt   *time.Time     `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	IsDeleted   bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt   *time.Time     `json:"deleted_at"`
	CreatedAt   time.Time      `json:"created_at"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// TrainingJob 训练任务模型
type TrainingJob struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID     uuid.UUID      `json:"project_id" gorm:"type:uuid;not null"`
	Name          string         `json:"name" gorm:"not null;size:255" validate:"required,min=1,max=255"`
	DatasetPath   string         `json:"dataset_path" gorm:"not null;size:500" validate:"required"` // OSS路径
	Config        map[string]any `json:"config" gorm:"type:jsonb;not null"`
	ExternalJobID *string        `json:"external_job_id" gorm:"size:255"` // 外部训练平台的任务ID
	Status        int            `json:"status" gorm:"default:0"`         // 0:提交中, 1:训练中, 2:已完成, 3:失败
	Progress      int            `json:"progress" gorm:"default:0"`       // 进度百分比
	Result        map[string]any `json:"result" gorm:"type:jsonb;default:'{}'"`
	IsDeleted     bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt     *time.Time     `json:"deleted_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// 任务类型常量
const (
	TaskTypeDocumentProcess = "document_process"
	TaskTypeQuestionGen     = "question_generate"
	TaskTypeAnswerGen       = "answer_generate"
	TaskTypeVectorIndex     = "vector_index"
	TaskTypeDatasetExport   = "dataset_export"
	TaskTypeModelTrain      = "model_train"
	TaskTypeDataMigration   = "data_migration"
)

// 任务状态常量
const (
	TaskStatusProcessing = 0 // 处理中
	TaskStatusCompleted  = 1 // 已完成
	TaskStatusFailed     = 2 // 失败
	TaskStatusCancelled  = 3 // 已中断
)

// 训练任务状态常量
const (
	TrainingJobStatusSubmitting = 0 // 提交中
	TrainingJobStatusTraining   = 1 // 训练中
	TrainingJobStatusCompleted  = 2 // 已完成
	TrainingJobStatusFailed     = 3 // 失败
)

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

// TableName 指定表名
func (TrainingJob) TableName() string {
	return "training_jobs"
}

// BeforeCreate GORM钩子 - 创建前
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (tj *TrainingJob) BeforeCreate(tx *gorm.DB) error {
	if tj.ID == uuid.Nil {
		tj.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除任务
func (t *Task) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	t.IsDeleted = true
	t.DeletedAt = &now
	return tx.Save(t).Error
}

// SoftDelete 软删除训练任务
func (tj *TrainingJob) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	tj.IsDeleted = true
	tj.DeletedAt = &now
	return tx.Save(tj).Error
}

// IsCompleted 检查任务是否已完成
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

// IsFailed 检查任务是否失败
func (t *Task) IsFailed() bool {
	return t.Status == TaskStatusFailed
}

// IsProcessing 检查任务是否正在处理
func (t *Task) IsProcessing() bool {
	return t.Status == TaskStatusProcessing
}

// MarkAsCompleted 标记任务为已完成
func (t *Task) MarkAsCompleted(tx *gorm.DB) error {
	now := time.Now()
	t.Status = TaskStatusCompleted
	t.Progress = 100
	t.CompletedAt = &now
	return tx.Save(t).Error
}

// MarkAsFailed 标记任务为失败
func (t *Task) MarkAsFailed(tx *gorm.DB, errorMsg string) error {
	now := time.Now()
	t.Status = TaskStatusFailed
	t.ErrorMessage = &errorMsg
	t.CompletedAt = &now
	return tx.Save(t).Error
}

// UpdateProgress 更新任务进度
func (t *Task) UpdateProgress(tx *gorm.DB, progress int) error {
	t.Progress = progress
	return tx.Save(t).Error
}