// internal/storage/async_vector_generator.go
package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
)

// VectorGenerateTask 向量生成任务
type VectorGenerateTask struct {
	TenantID   uuid.UUID              // 租户ID
	MemoryID   uuid.UUID              // 记忆ID
	SessionID  uuid.UUID              // 会话ID
	MemoryType string                 // 记忆类型
	Content    string                 // 内容
	Importance float32                // 重要性
	ExpiresAt  *time.Time             // 过期时间
	Metadata   map[string]interface{} // 元数据
	Callback   func(error)            // 回调函数
}

// VectorGenerator 向量生成器接口
type VectorGenerator interface {
	// GenerateEmbedding 生成文本向量
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	// GenerateBatchEmbeddings 批量生成向量
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// AsyncVectorGenerator 异步向量生成器
type AsyncVectorGenerator struct {
	generator    VectorGenerator
	qdrantClient QdrantClient
	taskQueue    chan *VectorGenerateTask
	batchQueue   chan *VectorGenerateTask
	config       *AsyncGeneratorConfig
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	running      bool
}

// AsyncGeneratorConfig 异步生成器配置
type AsyncGeneratorConfig struct {
	// 任务队列大小
	QueueSize int
	// 工作协程数量
	WorkerCount int
	// 批量队列大小
	BatchQueueSize int
	// 批量大小
	BatchSize int
	// 批量间隔（秒）
	BatchInterval int
	// 任务超时（秒）
	TaskTimeout int
}

// NewAsyncVectorGenerator 创建异步向量生成器
func NewAsyncVectorGenerator(
	generator VectorGenerator,
	qdrantClient QdrantClient,
	config *AsyncGeneratorConfig,
) *AsyncVectorGenerator {
	if config == nil {
		config = &AsyncGeneratorConfig{
			QueueSize:      1000,
			WorkerCount:    5,
			BatchQueueSize: 500,
			BatchSize:      50,
			BatchInterval:  5,
			TaskTimeout:    30,
		}
	}

	return &AsyncVectorGenerator{
		generator:    generator,
		qdrantClient: qdrantClient,
		taskQueue:    make(chan *VectorGenerateTask, config.QueueSize),
		batchQueue:   make(chan *VectorGenerateTask, config.BatchQueueSize),
		config:       config,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动异步生成器
func (g *AsyncVectorGenerator) Start(ctx context.Context) error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return fmt.Errorf("异步生成器已经在运行")
	}
	g.running = true
	g.mu.Unlock()

	// 启动工作协程
	for i := 0; i < g.config.WorkerCount; i++ {
		g.wg.Add(1)
		go g.worker(ctx, i)
	}

	// 启动批量处理协程
	g.wg.Add(1)
	go g.batchWorker(ctx)

	logger.Info("异步向量生成器已启动", logger.Fields{
		"worker_count":     g.config.WorkerCount,
		"queue_size":       g.config.QueueSize,
		"batch_size":       g.config.BatchSize,
		"batch_interval":   g.config.BatchInterval,
	})

	return nil
}

// Stop 停止异步生成器
func (g *AsyncVectorGenerator) Stop() error {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return fmt.Errorf("异步生成器未运行")
	}
	g.running = false
	g.mu.Unlock()

	// 发送停止信号
	close(g.stopChan)

	// 关闭任务队列
	close(g.taskQueue)
	close(g.batchQueue)

	// 等待所有 goroutine 完成
	g.wg.Wait()

	logger.Info("异步向量生成器已停止")
	return nil
}

// SubmitTask 提交向量生成任务
func (g *AsyncVectorGenerator) SubmitTask(task *VectorGenerateTask) error {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return fmt.Errorf("异步生成器未运行")
	}
	g.mu.Unlock()

	select {
	case g.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// SubmitBatchTask 提交批量任务（优先使用批量处理）
func (g *AsyncVectorGenerator) SubmitBatchTask(task *VectorGenerateTask) error {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return fmt.Errorf("异步生成器未运行")
	}
	g.mu.Unlock()

	select {
	case g.batchQueue <- task:
		return nil
	default:
		// 批量队列满了，降级到普通队列
		return g.SubmitTask(task)
	}
}

// GetQueueSize 获取队列大小
func (g *AsyncVectorGenerator) GetQueueSize() (int, int) {
	return len(g.taskQueue), len(g.batchQueue)
}

// worker 工作协程
func (g *AsyncVectorGenerator) worker(ctx context.Context, workerID int) {
	defer g.wg.Done()

	logger.Debug("工作协程启动", logger.Fields{
		"worker_id": workerID,
	})

	for {
		select {
		case <-g.stopChan:
			logger.Debug("工作协程停止", logger.Fields{
				"worker_id": workerID,
			})
			return
		case task, ok := <-g.taskQueue:
			if !ok {
				logger.Debug("任务队列已关闭", logger.Fields{
					"worker_id": workerID,
				})
				return
			}

			// 处理任务
			g.processTask(ctx, task, workerID)
		}
	}
}

// batchWorker 批量处理协程
func (g *AsyncVectorGenerator) batchWorker(ctx context.Context) {
	defer g.wg.Done()

	ticker := time.NewTicker(time.Duration(g.config.BatchInterval) * time.Second)
	defer ticker.Stop()

	batch := make([]*VectorGenerateTask, 0, g.config.BatchSize)

	processBatch := func() {
		if len(batch) == 0 {
			return
		}

		logger.Debug("处理批量任务", logger.Fields{
			"batch_size": len(batch),
		})

		g.processBatchTasks(ctx, batch)
		batch = batch[:0] // 清空批次
	}

	for {
		select {
		case <-g.stopChan:
			// 处理剩余任务
			processBatch()
			logger.Debug("批量处理协程停止")
			return
		case task, ok := <-g.batchQueue:
			if !ok {
				// 处理剩余任务
				processBatch()
				logger.Debug("批量队列已关闭")
				return
			}

			batch = append(batch, task)

			// 达到批量大小，立即处理
			if len(batch) >= g.config.BatchSize {
				processBatch()
			}
		case <-ticker.C:
			// 定时处理
			processBatch()
		}
	}
}

// processTask 处理单个任务
func (g *AsyncVectorGenerator) processTask(ctx context.Context, task *VectorGenerateTask, workerID int) {
	startTime := time.Now()

	// 设置超时
	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(g.config.TaskTimeout)*time.Second)
	defer cancel()

	// 生成向量
	vector, err := g.generator.GenerateEmbedding(taskCtx, task.Content)
	if err != nil {
		logger.Error("生成向量失败", logger.Fields{
			"worker_id": workerID,
			"memory_id": task.MemoryID.String(),
			"error":     err.Error(),
		})
		if task.Callback != nil {
			task.Callback(err)
		}
		return
	}

	// 插入向量
	req := &UpsertVectorRequest{
		TenantID:   task.TenantID,
		MemoryID:   task.MemoryID,
		SessionID:  task.SessionID,
		MemoryType: task.MemoryType,
		Vector:     vector,
		Importance: task.Importance,
		ExpiresAt:  task.ExpiresAt,
		Metadata:   task.Metadata,
	}

	if err := g.qdrantClient.UpsertVector(taskCtx, req); err != nil {
		logger.Error("插入向量失败", logger.Fields{
			"worker_id": workerID,
			"memory_id": task.MemoryID.String(),
			"error":     err.Error(),
		})
		if task.Callback != nil {
			task.Callback(err)
		}
		return
	}

	duration := time.Since(startTime)

	logger.Debug("向量生成成功", logger.Fields{
		"worker_id":        workerID,
		"memory_id":        task.MemoryID.String(),
		"duration_ms":      duration.Milliseconds(),
	})

	if task.Callback != nil {
		task.Callback(nil)
	}
}

// processBatchTasks 批量处理任务
func (g *AsyncVectorGenerator) processBatchTasks(ctx context.Context, tasks []*VectorGenerateTask) {
	if len(tasks) == 0 {
		return
	}

	startTime := time.Now()

	// 设置超时
	batchCtx, cancel := context.WithTimeout(ctx, time.Duration(g.config.TaskTimeout)*time.Second)
	defer cancel()

	// 提取所有内容
	contents := make([]string, len(tasks))
	for i, task := range tasks {
		contents[i] = task.Content
	}

	// 批量生成向量
	vectors, err := g.generator.GenerateBatchEmbeddings(batchCtx, contents)
	if err != nil {
		logger.Error("批量生成向量失败", logger.Fields{
			"batch_size": len(tasks),
			"error":      err.Error(),
		})
		// 通知所有任务失败
		for _, task := range tasks {
			if task.Callback != nil {
				task.Callback(err)
			}
		}
		return
	}

	// 构建批量插入请求
	reqs := make([]*UpsertVectorRequest, len(tasks))
	for i, task := range tasks {
		reqs[i] = &UpsertVectorRequest{
			TenantID:   task.TenantID,
			MemoryID:   task.MemoryID,
			SessionID:  task.SessionID,
			MemoryType: task.MemoryType,
			Vector:     vectors[i],
			Importance: task.Importance,
			ExpiresAt:  task.ExpiresAt,
			Metadata:   task.Metadata,
		}
	}

	// 批量插入向量
	if err := g.qdrantClient.BatchUpsertVectors(batchCtx, reqs); err != nil {
		logger.Error("批量插入向量失败", logger.Fields{
			"batch_size": len(tasks),
			"error":      err.Error(),
		})
		// 通知所有任务失败
		for _, task := range tasks {
			if task.Callback != nil {
				task.Callback(err)
			}
		}
		return
	}

	duration := time.Since(startTime)

	logger.Info("批量向量生成成功", logger.Fields{
		"batch_size":  len(tasks),
		"duration_ms": duration.Milliseconds(),
		"avg_ms":      duration.Milliseconds() / int64(len(tasks)),
	})

	// 通知所有任务成功
	for _, task := range tasks {
		if task.Callback != nil {
			task.Callback(nil)
		}
	}
}
