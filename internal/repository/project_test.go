package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"your-project/internal/model"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(
		&model.Project{},
		&model.ProjectMember{},
		&model.File{},
		&model.DocumentVersion{},
		&model.Chunk{},
		&model.VectorRecord{},
		&model.VectorIndex{},
		&model.Agent{},
		&model.LLMProvider{},
		&model.LLMModel{},
		&model.ChatSession{},
		&model.ChatMessage{},
		&model.Question{},
		&model.Answer{},
		&model.Task{},
		&model.TrainingJob{},
	)
	require.NoError(t, err)

	return db
}

// createTestProject 创建测试项目
func createTestProject(t *testing.T, db *gorm.DB) *model.Project {
	project := &model.Project{
		ID:          uuid.New(),
		Name:        "测试项目",
		Description: stringPtr("这是一个测试项目"),
		OwnerID:     uuid.New(),
		Settings:    make(map[string]any),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := db.Create(project).Error
	require.NoError(t, err)

	return project
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}

func TestProjectRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	project := &model.Project{
		Name:        "新项目",
		Description: stringPtr("项目描述"),
		OwnerID:     uuid.New(),
		Settings:    make(map[string]any),
	}

	err := repo.Create(context.Background(), project)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, project.ID)
}

func TestProjectRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建测试项目
	originalProject := createTestProject(t, db)

	// 获取项目
	project, err := repo.GetByID(context.Background(), originalProject.ID)
	assert.NoError(t, err)
	assert.Equal(t, originalProject.ID, project.ID)
	assert.Equal(t, originalProject.Name, project.Name)
}

func TestProjectRepository_GetProjectDataStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建测试项目
	project := createTestProject(t, db)

	// 创建测试数据
	// 创建文件
	file := &model.File{
		ID:           uuid.New(),
		ProjectID:    project.ID,
		Name:         "test.txt",
		OriginalName: "test.txt",
		MimeType:     "text/plain",
		Size:         1024,
		SHA256:       "abcd1234",
		OSSPath:      "/files/test.txt",
		UploaderID:   project.OwnerID,
		Status:       model.FileStatusCompleted,
		Metadata:     make(map[string]any),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := db.Create(file).Error
	require.NoError(t, err)

	// 创建Agent
	// 首先创建LLM提供商和模型
	provider := &model.LLMProvider{
		ID:           uuid.New(),
		Name:         "测试提供商",
		ProviderType: "openai",
		Config:       make(map[string]any),
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.Create(provider).Error
	require.NoError(t, err)

	llmModel := &model.LLMModel{
		ID:          uuid.New(),
		ProviderID:  provider.ID,
		ModelName:   "gpt-4",
		DisplayName: "GPT-4",
		ModelType:   "chat",
		Config:      make(map[string]any),
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
	err = db.Create(llmModel).Error
	require.NoError(t, err)

	agent := &model.Agent{
		ID:           uuid.New(),
		ProjectID:    project.ID,
		Name:         "测试Agent",
		Description:  stringPtr("测试Agent描述"),
		SystemPrompt: stringPtr("你是一个测试助手"),
		LLMModelID:   llmModel.ID,
		Tools:        make([]interface{}, 0),
		Config:       make(map[string]any),
		CreatedBy:    project.OwnerID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.Create(agent).Error
	require.NoError(t, err)

	// 获取项目数据统计
	stats, err := repo.GetProjectDataStats(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.Equal(t, project.ID, stats.ProjectID)
	assert.Equal(t, int64(1), stats.FilesCount)
	assert.Equal(t, int64(1024), stats.TotalSize)
	assert.Equal(t, int64(1), stats.AgentsCount)
}

func TestProjectRepository_ExportProjectData(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建测试项目
	project := createTestProject(t, db)

	// 添加项目成员
	member := &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: project.ID,
		UserID:    uuid.New(),
		Role:      "member",
		CreatedAt: time.Now(),
	}
	err := db.Create(member).Error
	require.NoError(t, err)

	// 导出项目数据
	exportData, err := repo.ExportProjectData(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.NotNil(t, exportData)
	assert.Equal(t, project.ID, exportData.Project.ID)
	assert.Equal(t, project.Name, exportData.Project.Name)
	assert.Len(t, exportData.Members, 1)
	assert.Equal(t, member.UserID, exportData.Members[0].UserID)
	assert.Equal(t, "1.0", exportData.ExportVersion)
}

func TestProjectRepository_ImportProjectData(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建目标项目
	targetProject := createTestProject(t, db)

	// 准备导入数据
	importData := &ProjectImportData{
		Files:         []*FileExportData{},
		Agents:        []*AgentExportData{},
		ChatSessions:  []*ChatSessionExportData{},
		Questions:     []*QuestionExportData{},
		VectorIndexes: []*VectorIndexExportData{},
		TrainingJobs:  []*TrainingJobExportData{},
		ImportOptions: &ImportOptions{
			IncludeFiles:         true,
			IncludeAgents:        true,
			IncludeChatSessions:  true,
			IncludeQuestions:     true,
			IncludeVectorIndexes: true,
			IncludeTrainingJobs:  true,
			OverwriteExisting:    false,
		},
	}

	// 执行导入
	err := repo.ImportProjectData(context.Background(), targetProject.ID, importData)
	assert.NoError(t, err)
}

func TestProjectRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建测试项目
	project := createTestProject(t, db)

	// 软删除项目
	err := repo.SoftDelete(context.Background(), project.ID)
	assert.NoError(t, err)

	// 验证项目已被软删除
	_, err = repo.GetByID(context.Background(), project.ID)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// 验证数据库中记录仍存在但标记为已删除
	var deletedProject model.Project
	err = db.Unscoped().Where("id = ?", project.ID).First(&deletedProject).Error
	assert.NoError(t, err)
	assert.True(t, deletedProject.IsDeleted)
	assert.NotNil(t, deletedProject.DeletedAt)
}

func TestProjectRepository_AddMember(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建测试项目
	project := createTestProject(t, db)

	// 添加成员
	member := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    uuid.New(),
		Role:      "member",
	}

	err := repo.AddMember(context.Background(), member)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, member.ID)

	// 验证成员已添加
	members, err := repo.GetMembers(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, member.UserID, members[0].UserID)
}

func TestProjectRepository_RemoveMember(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	// 创建测试项目
	project := createTestProject(t, db)

	// 添加成员
	member := &model.ProjectMember{
		ProjectID: project.ID,
		UserID:    uuid.New(),
		Role:      "member",
	}
	err := repo.AddMember(context.Background(), member)
	require.NoError(t, err)

	// 移除成员
	err = repo.RemoveMember(context.Background(), project.ID, member.UserID)
	assert.NoError(t, err)

	// 验证成员已被移除
	members, err := repo.GetMembers(context.Background(), project.ID)
	assert.NoError(t, err)
	assert.Len(t, members, 0)
}

func TestProjectRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db)

	ownerID := uuid.New()

	// 创建多个测试项目
	for i := 0; i < 5; i++ {
		project := &model.Project{
			ID:          uuid.New(),
			Name:        fmt.Sprintf("项目%d", i+1),
			Description: stringPtr(fmt.Sprintf("项目%d描述", i+1)),
			OwnerID:     ownerID,
			Settings:    make(map[string]any),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := db.Create(project).Error
		require.NoError(t, err)
	}

	// 获取项目列表
	projects, total, err := repo.List(context.Background(), ownerID, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, projects, 5)

	// 测试分页
	projects, total, err = repo.List(context.Background(), ownerID, 2, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, projects, 2)

	projects, total, err = repo.List(context.Background(), ownerID, 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, projects, 2)
}