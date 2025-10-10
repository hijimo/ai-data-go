package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"your-project/internal/model"
	"your-project/internal/repository"
)

// MigrationService 数据迁移服务接口
type MigrationService interface {
	// 导出项目数据
	ExportProject(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*repository.ProjectExportData, error)
	
	// 导入项目数据
	ImportProject(ctx context.Context, targetProjectID uuid.UUID, data *repository.ProjectImportData, userID uuid.UUID) (*model.Task, error)
	
	// 获取项目数据统计
	GetProjectStats(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*repository.ProjectDataStats, error)
	
	// 创建数据迁移任务
	CreateMigrationTask(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID, options *repository.ImportOptions, userID uuid.UUID) (*model.Task, error)
	
	// 获取迁移任务状态
	GetMigrationTaskStatus(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) (*model.Task, error)
	
	// 取消迁移任务
	CancelMigrationTask(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) error
}

// migrationService 数据迁移服务实现
type migrationService struct {
	projectRepo repository.ProjectRepository
	db          *gorm.DB
}

// NewMigrationService 创建数据迁移服务实例
func NewMigrationService(projectRepo repository.ProjectRepository, db *gorm.DB) MigrationService {
	return &migrationService{
		projectRepo: projectRepo,
		db:          db,
	}
}

// ExportProject 导出项目数据
func (s *migrationService) ExportProject(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*repository.ProjectExportData, error) {
	// 验证用户权限
	project, err := s.projectRepo.GetByIDWithMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}

	if !project.CanRead(userID) {
		return nil, fmt.Errorf("用户无权限访问该项目")
	}

	// 导出项目数据
	exportData, err := s.projectRepo.ExportProjectData(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("导出项目数据失败: %w", err)
	}

	return exportData, nil
}

// ImportProject 导入项目数据
func (s *migrationService) ImportProject(ctx context.Context, targetProjectID uuid.UUID, data *repository.ProjectImportData, userID uuid.UUID) (*model.Task, error) {
	// 验证用户权限
	project, err := s.projectRepo.GetByIDWithMembers(ctx, targetProjectID)
	if err != nil {
		return nil, fmt.Errorf("获取目标项目失败: %w", err)
	}

	if !project.CanWrite(userID) {
		return nil, fmt.Errorf("用户无权限修改该项目")
	}

	// 创建导入任务
	inputData := map[string]interface{}{
		"target_project_id": targetProjectID,
		"import_data":       data,
		"user_id":           userID,
	}

	inputDataJSON, _ := json.Marshal(inputData)
	var inputDataMap map[string]interface{}
	json.Unmarshal(inputDataJSON, &inputDataMap)

	task := &model.Task{
		ProjectID:   &targetProjectID,
		TaskType:    model.TaskTypeDataMigration,
		Status:      model.TaskStatusProcessing,
		Progress:    0,
		InputData:   inputDataMap,
		OutputData:  make(map[string]interface{}),
		StartedAt:   &[]time.Time{time.Now()}[0],
		CreatedAt:   time.Now(),
	}

	// 保存任务到数据库
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建导入任务失败: %w", err)
	}

	// 异步执行导入
	go s.executeImportTask(context.Background(), task.ID, targetProjectID, data)

	return task, nil
}

// GetProjectStats 获取项目数据统计
func (s *migrationService) GetProjectStats(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*repository.ProjectDataStats, error) {
	// 验证用户权限
	project, err := s.projectRepo.GetByIDWithMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("获取项目失败: %w", err)
	}

	if !project.CanRead(userID) {
		return nil, fmt.Errorf("用户无权限访问该项目")
	}

	// 获取项目数据统计
	stats, err := s.projectRepo.GetProjectDataStats(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("获取项目数据统计失败: %w", err)
	}

	return stats, nil
}

// CreateMigrationTask 创建数据迁移任务
func (s *migrationService) CreateMigrationTask(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID, options *repository.ImportOptions, userID uuid.UUID) (*model.Task, error) {
	// 验证源项目权限
	sourceProject, err := s.projectRepo.GetByIDWithMembers(ctx, sourceProjectID)
	if err != nil {
		return nil, fmt.Errorf("获取源项目失败: %w", err)
	}

	if !sourceProject.CanRead(userID) {
		return nil, fmt.Errorf("用户无权限访问源项目")
	}

	// 验证目标项目权限
	targetProject, err := s.projectRepo.GetByIDWithMembers(ctx, targetProjectID)
	if err != nil {
		return nil, fmt.Errorf("获取目标项目失败: %w", err)
	}

	if !targetProject.CanWrite(userID) {
		return nil, fmt.Errorf("用户无权限修改目标项目")
	}

	// 创建迁移任务
	inputData := map[string]interface{}{
		"source_project_id": sourceProjectID,
		"target_project_id": targetProjectID,
		"import_options":    options,
		"user_id":           userID,
	}

	inputDataJSON, _ := json.Marshal(inputData)
	var inputDataMap map[string]interface{}
	json.Unmarshal(inputDataJSON, &inputDataMap)

	task := &model.Task{
		ProjectID:   &targetProjectID,
		TaskType:    model.TaskTypeDataMigration,
		Status:      model.TaskStatusProcessing,
		Progress:    0,
		InputData:   inputDataMap,
		OutputData:  make(map[string]interface{}),
		StartedAt:   &[]time.Time{time.Now()}[0],
		CreatedAt:   time.Now(),
	}

	// 保存任务到数据库
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建迁移任务失败: %w", err)
	}

	// 异步执行迁移
	go s.executeMigrationTask(context.Background(), task.ID, sourceProjectID, targetProjectID, options)

	return task, nil
}

// GetMigrationTaskStatus 获取迁移任务状态
func (s *migrationService) GetMigrationTaskStatus(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) (*model.Task, error) {
	var task model.Task
	err := s.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", taskID, false).
		First(&task).Error
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	// 验证用户权限
	if task.ProjectID != nil {
		project, err := s.projectRepo.GetByIDWithMembers(ctx, *task.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("获取项目失败: %w", err)
		}

		if !project.CanRead(userID) {
			return nil, fmt.Errorf("用户无权限访问该任务")
		}
	}

	return &task, nil
}

// CancelMigrationTask 取消迁移任务
func (s *migrationService) CancelMigrationTask(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) error {
	var task model.Task
	err := s.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", taskID, false).
		First(&task).Error
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 验证用户权限
	if task.ProjectID != nil {
		project, err := s.projectRepo.GetByIDWithMembers(ctx, *task.ProjectID)
		if err != nil {
			return fmt.Errorf("获取项目失败: %w", err)
		}

		if !project.CanWrite(userID) {
			return fmt.Errorf("用户无权限取消该任务")
		}
	}

	// 只能取消正在处理的任务
	if task.Status != model.TaskStatusProcessing {
		return fmt.Errorf("只能取消正在处理的任务")
	}

	// 更新任务状态为已取消
	now := time.Now()
	err = s.db.WithContext(ctx).
		Model(&task).
		Updates(map[string]interface{}{
			"status":       model.TaskStatusCancelled,
			"completed_at": now,
		}).Error
	if err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}

	return nil
}

// executeImportTask 执行导入任务
func (s *migrationService) executeImportTask(ctx context.Context, taskID uuid.UUID, targetProjectID uuid.UUID, data *repository.ProjectImportData) {
	// 更新任务进度
	updateProgress := func(progress int) {
		s.db.WithContext(ctx).
			Model(&model.Task{}).
			Where("id = ?", taskID).
			Update("progress", progress)
	}

	// 标记任务完成
	markCompleted := func(outputData map[string]interface{}) {
		now := time.Now()
		outputDataJSON, _ := json.Marshal(outputData)
		var outputDataMap map[string]interface{}
		json.Unmarshal(outputDataJSON, &outputDataMap)

		s.db.WithContext(ctx).
			Model(&model.Task{}).
			Where("id = ?", taskID).
			Updates(map[string]interface{}{
				"status":       model.TaskStatusCompleted,
				"progress":     100,
				"output_data":  outputDataMap,
				"completed_at": now,
			})
	}

	// 标记任务失败
	markFailed := func(errorMsg string) {
		now := time.Now()
		s.db.WithContext(ctx).
			Model(&model.Task{}).
			Where("id = ?", taskID).
			Updates(map[string]interface{}{
				"status":        model.TaskStatusFailed,
				"error_message": errorMsg,
				"completed_at":  now,
			})
	}

	// 执行导入
	updateProgress(10)
	
	err := s.projectRepo.ImportProjectData(ctx, targetProjectID, data)
	if err != nil {
		markFailed(fmt.Sprintf("导入数据失败: %v", err))
		return
	}

	updateProgress(90)

	// 生成导入报告
	outputData := map[string]interface{}{
		"imported_at":    time.Now(),
		"target_project": targetProjectID,
		"import_summary": map[string]interface{}{
			"files_count":          len(data.Files),
			"agents_count":         len(data.Agents),
			"chat_sessions_count":  len(data.ChatSessions),
			"questions_count":      len(data.Questions),
			"vector_indexes_count": len(data.VectorIndexes),
			"training_jobs_count":  len(data.TrainingJobs),
		},
	}

	markCompleted(outputData)
}

// executeMigrationTask 执行迁移任务
func (s *migrationService) executeMigrationTask(ctx context.Context, taskID uuid.UUID, sourceProjectID, targetProjectID uuid.UUID, options *repository.ImportOptions) {
	// 更新任务进度
	updateProgress := func(progress int) {
		s.db.WithContext(ctx).
			Model(&model.Task{}).
			Where("id = ?", taskID).
			Update("progress", progress)
	}

	// 标记任务完成
	markCompleted := func(outputData map[string]interface{}) {
		now := time.Now()
		outputDataJSON, _ := json.Marshal(outputData)
		var outputDataMap map[string]interface{}
		json.Unmarshal(outputDataJSON, &outputDataMap)

		s.db.WithContext(ctx).
			Model(&model.Task{}).
			Where("id = ?", taskID).
			Updates(map[string]interface{}{
				"status":       model.TaskStatusCompleted,
				"progress":     100,
				"output_data":  outputDataMap,
				"completed_at": now,
			})
	}

	// 标记任务失败
	markFailed := func(errorMsg string) {
		now := time.Now()
		s.db.WithContext(ctx).
			Model(&model.Task{}).
			Where("id = ?", taskID).
			Updates(map[string]interface{}{
				"status":        model.TaskStatusFailed,
				"error_message": errorMsg,
				"completed_at":  now,
			})
	}

	// 第一步：导出源项目数据
	updateProgress(10)
	
	exportData, err := s.projectRepo.ExportProjectData(ctx, sourceProjectID)
	if err != nil {
		markFailed(fmt.Sprintf("导出源项目数据失败: %v", err))
		return
	}

	updateProgress(50)

	// 第二步：准备导入数据
	importData := &repository.ProjectImportData{
		Files:         exportData.Files,
		Agents:        exportData.Agents,
		ChatSessions:  exportData.ChatSessions,
		Questions:     exportData.Questions,
		VectorIndexes: exportData.VectorIndexes,
		TrainingJobs:  exportData.TrainingJobs,
		ImportOptions: options,
	}

	updateProgress(60)

	// 第三步：导入到目标项目
	err = s.projectRepo.ImportProjectData(ctx, targetProjectID, importData)
	if err != nil {
		markFailed(fmt.Sprintf("导入到目标项目失败: %v", err))
		return
	}

	updateProgress(90)

	// 生成迁移报告
	outputData := map[string]interface{}{
		"migrated_at":     time.Now(),
		"source_project":  sourceProjectID,
		"target_project":  targetProjectID,
		"migration_summary": map[string]interface{}{
			"files_count":          len(exportData.Files),
			"agents_count":         len(exportData.Agents),
			"chat_sessions_count":  len(exportData.ChatSessions),
			"questions_count":      len(exportData.Questions),
			"vector_indexes_count": len(exportData.VectorIndexes),
			"training_jobs_count":  len(exportData.TrainingJobs),
		},
		"import_options": options,
	}

	markCompleted(outputData)
}