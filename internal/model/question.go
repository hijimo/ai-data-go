package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Question 问题模型
type Question struct {
	ID           uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID    uuid.UUID     `json:"project_id" gorm:"type:uuid;not null"`
	ChunkID      *uuid.UUID    `json:"chunk_id" gorm:"type:uuid"`
	Content      string        `json:"content" gorm:"type:text;not null" validate:"required"`
	QuestionType *string       `json:"question_type" gorm:"size:50"` // factual, reasoning, application
	Tags         []interface{} `json:"tags" gorm:"type:jsonb;default:'[]'"`
	Difficulty   int           `json:"difficulty" gorm:"default:1" validate:"min=1,max=5"` // 1-5难度等级
	Status       int           `json:"status" gorm:"default:0"`                            // 0:待审核, 1:已审核, 2:已拒绝
	IsDeleted    bool          `json:"is_deleted" gorm:"default:false"`
	DeletedAt    *time.Time    `json:"deleted_at"`
	CreatedAt    time.Time     `json:"created_at"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Chunk   *Chunk   `json:"chunk,omitempty" gorm:"foreignKey:ChunkID"`
	Answers []Answer `json:"answers,omitempty" gorm:"foreignKey:QuestionID"`
}

// Answer 答案模型
type Answer struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	QuestionID   uuid.UUID  `json:"question_id" gorm:"type:uuid;not null"`
	Content      string     `json:"content" gorm:"type:text;not null" validate:"required"`
	Reasoning    *string    `json:"reasoning" gorm:"type:text"` // CoT推理过程
	LLMModelID   *uuid.UUID `json:"llm_model_id" gorm:"type:uuid"`
	QualityScore *float64   `json:"quality_score" gorm:"type:decimal(3,2)"` // 质量评分 0-1
	IsReviewed   bool       `json:"is_reviewed" gorm:"default:false"`
	IsDeleted    bool       `json:"is_deleted" gorm:"default:false"`
	DeletedAt    *time.Time `json:"deleted_at"`
	CreatedAt    time.Time  `json:"created_at"`

	// 关联关系
	Question *Question `json:"question,omitempty" gorm:"foreignKey:QuestionID"`
	LLMModel *LLMModel `json:"llm_model,omitempty" gorm:"foreignKey:LLMModelID"`
}

// 问题类型常量
const (
	QuestionTypeFactual     = "factual"
	QuestionTypeReasoning   = "reasoning"
	QuestionTypeApplication = "application"
)

// 问题状态常量
const (
	QuestionStatusPending  = 0 // 待审核
	QuestionStatusApproved = 1 // 已审核
	QuestionStatusRejected = 2 // 已拒绝
)

// 难度等级常量
const (
	DifficultyVeryEasy = 1
	DifficultyEasy     = 2
	DifficultyMedium   = 3
	DifficultyHard     = 4
	DifficultyVeryHard = 5
)

// TableName 指定表名
func (Question) TableName() string {
	return "questions"
}

// TableName 指定表名
func (Answer) TableName() string {
	return "answers"
}

// BeforeCreate GORM钩子 - 创建前
func (q *Question) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (a *Answer) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除问题
func (q *Question) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	q.IsDeleted = true
	q.DeletedAt = &now
	return tx.Save(q).Error
}

// SoftDelete 软删除答案
func (a *Answer) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	a.IsDeleted = true
	a.DeletedAt = &now
	return tx.Save(a).Error
}