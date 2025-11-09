// internal/storage/qdrant_optimizer.go
package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
)

// QdrantOptimizer Qdrant 优化器
type QdrantOptimizer struct {
	client   QdrantClient
	config   *OptimizerConfig
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	// 优化间隔（小时）
	OptimizationInterval int
	// 是否启用自动优化
	EnableAutoOptimization bool
	// 批量插入队列大小
	BatchQueueSize int
	// 批量插入间隔（秒）
	BatchInterval int
	// 批量大小
	BatchSize int
}

// NewQdrantOptimizer 创建 Qdrant 优化器
func NewQdrantOptimizer(client QdrantClient, config *OptimizerConfig) *QdrantOptimizer {
	if config == nil {
		config = &OptimizerConfig{
			OptimizationInterval:   24,   // 默认24小时
			EnableAutoOptimization: true,
			BatchQueueSize:         1000,
			BatchInterval:          5,   // 默认5秒
			BatchSize:              100, // 默认100条
		}
	}

	return &QdrantOptimizer{
		client:   client,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start 启动优化器
func (o *QdrantOptimizer) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("优化器已经在运行")
	}
	o.running = true
	o.mu.Unlock()

	// 启动定期优化
	if o.config.EnableAutoOptimization {
		o.wg.Add(1)
		go o.runPeriodicOptimization(ctx)
	}

	logger.Info("Qdrant 优化器已启动", logger.Fields{
		"optimization_interval": o.config.OptimizationInterval,
		"auto_optimization":     o.config.EnableAutoOptimization,
	})

	return nil
}

// Stop 停止优化器
func (o *QdrantOptimizer) Stop() error {
	o.mu.Lock()
	if !o.running {
		o.mu.Unlock()
		return fmt.Errorf("优化器未运行")
	}
	o.running = false
	o.mu.Unlock()

	// 发送停止信号
	close(o.stopChan)

	// 等待所有 goroutine 完成
	o.wg.Wait()

	logger.Info("Qdrant 优化器已停止")
	return nil
}

// OptimizeNow 立即执行优化
func (o *QdrantOptimizer) OptimizeNow(ctx context.Context) error {
	logger.Info("开始优化 Qdrant Collection")

	startTime := time.Now()

	// 获取优化前的信息
	infoBefore, err := o.client.GetCollectionInfo(ctx)
	if err != nil {
		logger.Error("获取 Collection 信息失败", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	logger.Info("优化前的 Collection 信息", logger.Fields{
		"status":               infoBefore.Status,
		"vectors_count":        infoBefore.VectorsCount,
		"indexed_vectors_count": infoBefore.IndexedVectorsCount,
		"points_count":         infoBefore.PointsCount,
		"segments_count":       infoBefore.SegmentsCount,
	})

	// 执行优化
	if err := o.client.OptimizeCollection(ctx); err != nil {
		logger.Error("优化 Collection 失败", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	// 等待优化完成（最多等待5分钟）
	optimizationCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := o.waitForOptimization(optimizationCtx); err != nil {
		logger.Warn("等待优化完成超时", logger.Fields{
			"error": err.Error(),
		})
	}

	// 获取优化后的信息
	infoAfter, err := o.client.GetCollectionInfo(ctx)
	if err != nil {
		logger.Error("获取优化后的 Collection 信息失败", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	duration := time.Since(startTime)

	logger.Info("优化完成", logger.Fields{
		"duration_seconds":     duration.Seconds(),
		"vectors_count":        infoAfter.VectorsCount,
		"indexed_vectors_count": infoAfter.IndexedVectorsCount,
		"points_count":         infoAfter.PointsCount,
		"segments_count":       infoAfter.SegmentsCount,
		"segments_reduced":     infoBefore.SegmentsCount - infoAfter.SegmentsCount,
	})

	return nil
}

// GetCollectionStats 获取 Collection 统计信息
func (o *QdrantOptimizer) GetCollectionStats(ctx context.Context) (*CollectionStats, error) {
	info, err := o.client.GetCollectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	stats := &CollectionStats{
		Status:              info.Status,
		VectorsCount:        info.VectorsCount,
		IndexedVectorsCount: info.IndexedVectorsCount,
		PointsCount:         info.PointsCount,
		SegmentsCount:       info.SegmentsCount,
		IndexingProgress:    0,
	}

	// 计算索引进度
	if info.VectorsCount > 0 {
		stats.IndexingProgress = float64(info.IndexedVectorsCount) / float64(info.VectorsCount) * 100
	}

	return stats, nil
}

// runPeriodicOptimization 定期优化
func (o *QdrantOptimizer) runPeriodicOptimization(ctx context.Context) {
	defer o.wg.Done()

	ticker := time.NewTicker(time.Duration(o.config.OptimizationInterval) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopChan:
			logger.Info("停止定期优化")
			return
		case <-ticker.C:
			logger.Info("开始定期优化")
			if err := o.OptimizeNow(ctx); err != nil {
				logger.Error("定期优化失败", logger.Fields{
					"error": err.Error(),
				})
			}
		}
	}
}

// waitForOptimization 等待优化完成
func (o *QdrantOptimizer) waitForOptimization(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := o.client.GetCollectionInfo(ctx)
			if err != nil {
				return err
			}

			// 检查是否优化完成（所有向量都已索引）
			if info.VectorsCount == info.IndexedVectorsCount {
				return nil
			}

			logger.Debug("等待优化完成", logger.Fields{
				"indexed_vectors": info.IndexedVectorsCount,
				"total_vectors":   info.VectorsCount,
				"progress":        float64(info.IndexedVectorsCount) / float64(info.VectorsCount) * 100,
			})
		}
	}
}

// CollectionStats Collection 统计信息
type CollectionStats struct {
	Status              string  // Collection 状态
	VectorsCount        int64   // 向量数量
	IndexedVectorsCount int64   // 已索引向量数量
	PointsCount         int64   // 点数量
	SegmentsCount       int     // 段数量
	IndexingProgress    float64 // 索引进度（百分比）
}
