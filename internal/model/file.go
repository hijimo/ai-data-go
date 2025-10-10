package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// File 文件模型
type File struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID    uuid.UUID      `json:"project_id" gorm:"type:uuid;not null"`
	Name         string         `json:"name" gorm:"not null;size:255" validate:"required,min=1,max=255"`
	OriginalName string         `json:"original_name" gorm:"not null;size:255" validate:"required,min=1,max=255"`
	MimeType     string         `json:"mime_type" gorm:"not null;size:100" validate:"required"`
	Size         int64          `json:"size" gorm:"not null" validate:"required,min=1"`
	SHA256       string         `json:"sha256" gorm:"not null;size:64" validate:"required,len=64"`
	OSSPath      string         `json:"oss_path" gorm:"not null;size:500" validate:"required"`
	UploaderID   uuid.UUID      `json:"uploader_id" gorm:"type:uuid;not null"`
	Status       int            `json:"status" gorm:"default:0"` // 0:上传中, 1:已完成, 2:处理失败
	Metadata     map[string]any `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	IsDeleted    bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt    *time.Time     `json:"deleted_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// 关联关系
	Project  *Project           `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Versions []DocumentVersion  `json:"versions,omitempty" gorm:"foreignKey:FileID"`
}

// DocumentVersion 文档版本模型
type DocumentVersion struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	FileID      uuid.UUID      `json:"file_id" gorm:"type:uuid;not null"`
	Version     int            `json:"version" gorm:"not null;default:1"`
	ChunkConfig map[string]any `json:"chunk_config" gorm:"type:jsonb;not null"`
	ChunkCount  int            `json:"chunk_count" gorm:"default:0"`
	Status      int            `json:"status" gorm:"default:0"` // 0:处理中, 1:已完成, 2:失败
	IsDeleted   bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt   *time.Time     `json:"deleted_at"`
	CreatedAt   time.Time      `json:"created_at"`

	// 关联关系
	File   *File   `json:"file,omitempty" gorm:"foreignKey:FileID"`
	Chunks []Chunk `json:"chunks,omitempty" gorm:"foreignKey:DocumentVersionID"`
}

// Chunk 文档块模型
type Chunk struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	DocumentVersionID uuid.UUID      `json:"document_version_id" gorm:"type:uuid;not null"`
	Sequence          int            `json:"sequence" gorm:"not null"`
	Content           string         `json:"content" gorm:"type:text;not null" validate:"required"`
	Metadata          map[string]any `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	EmbeddingStatus   int            `json:"embedding_status" gorm:"default:0"` // 0:未处理, 1:已完成, 2:失败
	IsDeleted         bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt         *time.Time     `json:"deleted_at"`
	CreatedAt         time.Time      `json:"created_at"`

	// 关联关系
	DocumentVersion *DocumentVersion `json:"document_version,omitempty" gorm:"foreignKey:DocumentVersionID"`
	VectorRecords   []VectorRecord   `json:"vector_records,omitempty" gorm:"foreignKey:ChunkID"`
}

// VectorRecord 向量记录模型
type VectorRecord struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ChunkID        uuid.UUID  `json:"chunk_id" gorm:"type:uuid;not null"`
	VectorIndexID  uuid.UUID  `json:"vector_index_id" gorm:"type:uuid;not null"`
	ExternalID     string     `json:"external_id" gorm:"not null;size:255" validate:"required"`
	EmbeddingModel string     `json:"embedding_model" gorm:"not null;size:100" validate:"required"`
	IsDeleted      bool       `json:"is_deleted" gorm:"default:false"`
	DeletedAt      *time.Time `json:"deleted_at"`
	CreatedAt      time.Time  `json:"created_at"`

	// 关联关系
	Chunk       *Chunk       `json:"chunk,omitempty" gorm:"foreignKey:ChunkID"`
	VectorIndex *VectorIndex `json:"vector_index,omitempty" gorm:"foreignKey:VectorIndexID"`
}

// VectorIndex 向量索引模型
type VectorIndex struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID uuid.UUID      `json:"project_id" gorm:"type:uuid;not null"`
	Name      string         `json:"name" gorm:"not null;size:255" validate:"required,min=1,max=255"`
	Provider  string         `json:"provider" gorm:"not null;size:50;default:'adbpg'" validate:"required"`
	Config    map[string]any `json:"config" gorm:"type:jsonb;not null"`
	Status    int            `json:"status" gorm:"default:0"` // 0:创建中, 1:可用, 2:错误
	Stats     map[string]any `json:"stats" gorm:"type:jsonb;default:'{}'"`
	IsDeleted bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt *time.Time     `json:"deleted_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// 关联关系
	Project       *Project       `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	VectorRecords []VectorRecord `json:"vector_records,omitempty" gorm:"foreignKey:VectorIndexID"`
}

// 文件状态常量
const (
	FileStatusUploading = 0 // 上传中
	FileStatusCompleted = 1 // 已完成
	FileStatusFailed    = 2 // 处理失败
)

// 文档版本状态常量
const (
	DocumentVersionStatusProcessing = 0 // 处理中
	DocumentVersionStatusCompleted  = 1 // 已完成
	DocumentVersionStatusFailed     = 2 // 失败
)

// 嵌入状态常量
const (
	EmbeddingStatusPending   = 0 // 未处理
	EmbeddingStatusCompleted = 1 // 已完成
	EmbeddingStatusFailed    = 2 // 失败
)

// 向量索引状态常量
const (
	VectorIndexStatusCreating = 0 // 创建中
	VectorIndexStatusReady    = 1 // 可用
	VectorIndexStatusError    = 2 // 错误
)

// TableName 指定表名
func (File) TableName() string {
	return "files"
}

// TableName 指定表名
func (DocumentVersion) TableName() string {
	return "document_versions"
}

// TableName 指定表名
func (Chunk) TableName() string {
	return "chunks"
}

// TableName 指定表名
func (VectorRecord) TableName() string {
	return "vector_records"
}

// TableName 指定表名
func (VectorIndex) TableName() string {
	return "vector_indexes"
}

// BeforeCreate GORM钩子 - 创建前
func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (dv *DocumentVersion) BeforeCreate(tx *gorm.DB) error {
	if dv.ID == uuid.Nil {
		dv.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (c *Chunk) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (vr *VectorRecord) BeforeCreate(tx *gorm.DB) error {
	if vr.ID == uuid.Nil {
		vr.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (vi *VectorIndex) BeforeCreate(tx *gorm.DB) error {
	if vi.ID == uuid.Nil {
		vi.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除文件
func (f *File) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	f.IsDeleted = true
	f.DeletedAt = &now
	return tx.Save(f).Error
}

// SoftDelete 软删除文档版本
func (dv *DocumentVersion) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	dv.IsDeleted = true
	dv.DeletedAt = &now
	return tx.Save(dv).Error
}

// SoftDelete 软删除文档块
func (c *Chunk) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	c.IsDeleted = true
	c.DeletedAt = &now
	return tx.Save(c).Error
}

// SoftDelete 软删除向量记录
func (vr *VectorRecord) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	vr.IsDeleted = true
	vr.DeletedAt = &now
	return tx.Save(vr).Error
}

// SoftDelete 软删除向量索引
func (vi *VectorIndex) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	vi.IsDeleted = true
	vi.DeletedAt = &now
	return tx.Save(vi).Error
}