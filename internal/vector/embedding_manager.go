package vector

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EmbeddingManager 向量化管理器
type EmbeddingManager struct {
	providers map[string]EmbeddingProvider
	configs   map[string]*EmbeddingConfig
	mu        sync.RWMutex
	factory   EmbeddingProviderFactory
}

// EmbeddingProviderFactory 向量化提供商工厂接口
type EmbeddingProviderFactory interface {
	CreateProvider(config *EmbeddingConfig) (EmbeddingProvider, error)
	SupportedProviders() []EmbeddingProviderType
}

// NewEmbeddingManager 创建向量化管理器
func NewEmbeddingManager(factory EmbeddingProviderFactory) *EmbeddingManager {
	return &EmbeddingManager{
		providers: make(map[string]EmbeddingProvider),
		configs:   make(map[string]*EmbeddingConfig),
		factory:   factory,
	}
}

// RegisterProvider 注册向量化提供商
func (m *EmbeddingManager) RegisterProvider(name string, config *EmbeddingConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	
	provider, err := m.factory.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 如果已存在同名提供商，先关闭旧的（如果有Close方法）
	if oldProvider, exists := m.providers[name]; exists {
		if closer, ok := oldProvider.(interface{ Close() error }); ok {
			closer.Close()
		}
	}
	
	m.providers[name] = provider
	m.configs[name] = config
	
	return nil
}

// GetProvider 获取向量化提供商
func (m *EmbeddingManager) GetProvider(name string) (EmbeddingProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	
	return provider, nil
}

// RemoveProvider 移除向量化提供商
func (m *EmbeddingManager) RemoveProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return fmt.Errorf("provider not found: %s", name)
	}
	
	// 如果提供商有Close方法，调用它
	if closer, ok := provider.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("failed to close provider: %w", err)
		}
	}
	
	delete(m.providers, name)
	delete(m.configs, name)
	
	return nil
}

// ListProviders 列出所有注册的提供商
func (m *EmbeddingManager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	
	return names
}

// GetProviderConfig 获取提供商配置
func (m *EmbeddingManager) GetProviderConfig(name string) (*EmbeddingConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[name]
	if !exists {
		return nil, fmt.Errorf("provider config not found: %s", name)
	}
	
	return config, nil
}

// HealthCheck 检查所有提供商的健康状态
func (m *EmbeddingManager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := make(map[string]error)
	for name, provider := range m.providers {
		results[name] = provider.HealthCheck(ctx)
	}
	
	return results
}

// EmbedWithProvider 使用指定提供商生成向量
func (m *EmbeddingManager) EmbedWithProvider(ctx context.Context, providerName string, text string) ([]float32, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	
	return provider.Embed(ctx, text)
}

// EmbedBatchWithProvider 使用指定提供商批量生成向量
func (m *EmbeddingManager) EmbedBatchWithProvider(ctx context.Context, providerName string, texts []string) ([][]float32, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	
	return provider.EmbedBatch(ctx, texts)
}

// Close 关闭所有提供商连接
func (m *EmbeddingManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	for name, provider := range m.providers {
		if closer, ok := provider.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				lastErr = fmt.Errorf("failed to close provider %s: %w", name, err)
			}
		}
	}
	
	// 清空所有提供商
	m.providers = make(map[string]EmbeddingProvider)
	m.configs = make(map[string]*EmbeddingConfig)
	
	return lastErr
}

// DefaultEmbeddingProviderFactory 默认向量化提供商工厂
type DefaultEmbeddingProviderFactory struct{}

// CreateProvider 创建向量化提供商实例
func (f *DefaultEmbeddingProviderFactory) CreateProvider(config *EmbeddingConfig) (EmbeddingProvider, error) {
	switch config.Provider {
	case EmbeddingProviderOpenAI, EmbeddingProviderAzure, EmbeddingProviderQianwen, 
		 EmbeddingProviderBaichuan, EmbeddingProviderZhipu:
		return NewHTTPEmbeddingClient(config)
	case EmbeddingProviderLocal:
		return nil, fmt.Errorf("local embedding provider not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// SupportedProviders 返回支持的提供商列表
func (f *DefaultEmbeddingProviderFactory) SupportedProviders() []EmbeddingProviderType {
	return []EmbeddingProviderType{
		EmbeddingProviderOpenAI,
		EmbeddingProviderAzure,
		EmbeddingProviderQianwen,
		EmbeddingProviderBaichuan,
		EmbeddingProviderZhipu,
	}
}

// NewDefaultEmbeddingProviderFactory 创建默认向量化提供商工厂
func NewDefaultEmbeddingProviderFactory() EmbeddingProviderFactory {
	return &DefaultEmbeddingProviderFactory{}
}

// AsyncEmbeddingProcessor 异步向量化处理器
type AsyncEmbeddingProcessor struct {
	manager    *EmbeddingManager
	taskQueue  chan *EmbeddingTask
	workers    int
	stopCh     chan struct{}
	taskStore  map[string]*EmbeddingTask
	taskMutex  sync.RWMutex
}

// NewAsyncEmbeddingProcessor 创建异步向量化处理器
func NewAsyncEmbeddingProcessor(manager *EmbeddingManager, workers int) *AsyncEmbeddingProcessor {
	return &AsyncEmbeddingProcessor{
		manager:   manager,
		taskQueue: make(chan *EmbeddingTask, 1000),
		workers:   workers,
		stopCh:    make(chan struct{}),
		taskStore: make(map[string]*EmbeddingTask),
	}
}

// Start 启动异步处理器
func (p *AsyncEmbeddingProcessor) Start() {
	for i := 0; i < p.workers; i++ {
		go p.worker()
	}
}

// Stop 停止异步处理器
func (p *AsyncEmbeddingProcessor) Stop() {
	close(p.stopCh)
}

// SubmitTask 提交向量化任务
func (p *AsyncEmbeddingProcessor) SubmitTask(providerName string, texts []string) (*EmbeddingTask, error) {
	task := &EmbeddingTask{
		ID:        generateTaskID(),
		Texts:     texts,
		Status:    TaskStatusPending,
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// 存储任务
	p.taskMutex.Lock()
	p.taskStore[task.ID] = task
	p.taskMutex.Unlock()
	
	// 添加提供商信息到任务中（通过扩展字段）
	task.Result = [][]float32{} // 临时存储提供商名称的hack
	
	// 提交到队列
	select {
	case p.taskQueue <- task:
		return task, nil
	default:
		return nil, fmt.Errorf("task queue is full")
	}
}

// GetTask 获取任务状态
func (p *AsyncEmbeddingProcessor) GetTask(taskID string) (*EmbeddingTask, error) {
	p.taskMutex.RLock()
	defer p.taskMutex.RUnlock()
	
	task, exists := p.taskStore[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	
	return task, nil
}

// worker 工作协程
func (p *AsyncEmbeddingProcessor) worker() {
	for {
		select {
		case task := <-p.taskQueue:
			p.processTask(task)
		case <-p.stopCh:
			return
		}
	}
}

// processTask 处理任务
func (p *AsyncEmbeddingProcessor) processTask(task *EmbeddingTask) {
	// 更新任务状态
	p.updateTaskStatus(task.ID, TaskStatusProcessing, 0)
	
	ctx := context.Background()
	
	// 这里需要从任务中获取提供商名称，实际实现中应该在任务结构中添加该字段
	providerName := "default" // 临时硬编码
	
	// 批量处理文本
	batchSize := 10 // 可配置
	var allEmbeddings [][]float32
	
	for i := 0; i < len(task.Texts); i += batchSize {
		end := i + batchSize
		if end > len(task.Texts) {
			end = len(task.Texts)
		}
		
		batch := task.Texts[i:end]
		embeddings, err := p.manager.EmbedBatchWithProvider(ctx, providerName, batch)
		if err != nil {
			p.updateTaskError(task.ID, err.Error())
			return
		}
		
		allEmbeddings = append(allEmbeddings, embeddings...)
		
		// 更新进度
		progress := int(float64(end) / float64(len(task.Texts)) * 100)
		p.updateTaskStatus(task.ID, TaskStatusProcessing, progress)
	}
	
	// 完成任务
	p.completeTask(task.ID, allEmbeddings)
}

// updateTaskStatus 更新任务状态
func (p *AsyncEmbeddingProcessor) updateTaskStatus(taskID string, status TaskStatus, progress int) {
	p.taskMutex.Lock()
	defer p.taskMutex.Unlock()
	
	if task, exists := p.taskStore[taskID]; exists {
		task.Status = status
		task.Progress = progress
		task.UpdatedAt = time.Now()
	}
}

// updateTaskError 更新任务错误
func (p *AsyncEmbeddingProcessor) updateTaskError(taskID string, errorMsg string) {
	p.taskMutex.Lock()
	defer p.taskMutex.Unlock()
	
	if task, exists := p.taskStore[taskID]; exists {
		task.Status = TaskStatusFailed
		task.Error = errorMsg
		task.UpdatedAt = time.Now()
	}
}

// completeTask 完成任务
func (p *AsyncEmbeddingProcessor) completeTask(taskID string, embeddings [][]float32) {
	p.taskMutex.Lock()
	defer p.taskMutex.Unlock()
	
	if task, exists := p.taskStore[taskID]; exists {
		task.Status = TaskStatusCompleted
		task.Progress = 100
		task.Result = embeddings
		task.UpdatedAt = time.Now()
	}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}