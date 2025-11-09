// internal/storage/qdrant_client_impl.go
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// qdrantClientImpl Qdrant 客户端实现
type qdrantClientImpl struct {
	config     *QdrantConfig
	httpClient *http.Client
	baseURL    string
	cache      CacheService // 向量查询结果缓存
}

// NewQdrantClient 创建 Qdrant 客户端
func NewQdrantClient(config *QdrantConfig) (QdrantClient, error) {
	return NewQdrantClientWithCache(config, nil)
}

// NewQdrantClientWithCache 创建带缓存的 Qdrant 客户端
func NewQdrantClientWithCache(config *QdrantConfig, cache CacheService) (QdrantClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API Key 不能为空")
	}

	var baseURL string

	// 优先使用 Endpoint（Qdrant Cloud）
	if config.Endpoint != "" {
		baseURL = config.Endpoint
	} else if config.Host != "" {
		// 使用 Host + Port（自托管）
		if config.Port <= 0 {
			config.Port = 6333 // 默认端口
		}
		scheme := "http"
		if config.UseTLS {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s:%d", scheme, config.Host, config.Port)
	} else {
		return nil, fmt.Errorf("必须提供 Endpoint 或 Host")
	}

	// 设置默认配置值
	if config.ShardNumber <= 0 {
		config.ShardNumber = DefaultShardNumber
	}
	if config.ReplicationFactor <= 0 {
		config.ReplicationFactor = DefaultReplicationFactor
	}
	if config.HnswM <= 0 {
		config.HnswM = DefaultHnswM
	}
	if config.HnswEfConstruction <= 0 {
		config.HnswEfConstruction = DefaultHnswEfConstruction
	}
	if config.OptimizationInterval <= 0 {
		config.OptimizationInterval = DefaultOptimizationInterval
	}

	client := &qdrantClientImpl{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		cache:   cache,
	}

	return client, nil
}

// InitializeCollection 初始化单个共享 Collection
func (c *qdrantClientImpl) InitializeCollection(ctx context.Context) error {
	// 1. 检查 Collection 是否已存在
	exists, err := c.collectionExists(ctx)
	if err != nil {
		return fmt.Errorf("检查 collection 失败: %w", err)
	}

	if exists {
		// Collection 已存在，跳过创建
		return nil
	}

	// 2. 创建共享 Collection（使用配置的优化参数）
	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     VectorDim,
			"distance": "Cosine",
		},
		"shard_number":       c.config.ShardNumber,       // 根据租户数量调整
		"replication_factor": c.config.ReplicationFactor, // 高可用配置
		"hnsw_config": map[string]interface{}{
			"m":                   c.config.HnswM,             // 控制图的连接度
			"ef_construction":     c.config.HnswEfConstruction, // 构建时的搜索深度
			"full_scan_threshold": 10000,                      // 全扫描阈值
		},
		"optimizers_config": map[string]interface{}{
			"deleted_threshold":    0.2,  // 删除阈值
			"vacuum_min_vector_number": 1000, // 最小向量数量
			"default_segment_number": c.config.ShardNumber, // 默认段数量
		},
	}

	if err := c.createCollection(ctx, createReq); err != nil {
		return fmt.Errorf("创建 collection 失败: %w", err)
	}

	// 3. 创建租户标识索引（is_tenant=true）
	if err := c.createFieldIndex(ctx, "tenant_id", "keyword", true); err != nil {
		return fmt.Errorf("创建租户索引失败: %w", err)
	}

	// 4. 创建其他 payload 索引
	indexes := []struct {
		field string
		typ   string
	}{
		{"session_id", "keyword"},
		{"memory_type", "keyword"},
		{"created_at", "datetime"},
	}

	for _, idx := range indexes {
		if err := c.createFieldIndex(ctx, idx.field, idx.typ, false); err != nil {
			return fmt.Errorf("创建索引 %s 失败: %w", idx.field, err)
		}
	}

	return nil
}

// UpsertVector 插入或更新向量
func (c *qdrantClientImpl) UpsertVector(ctx context.Context, req *UpsertVectorRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	// 验证必填字段
	if req.TenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}
	if req.MemoryID == uuid.Nil {
		return fmt.Errorf("记忆ID不能为空")
	}
	if req.SessionID == uuid.Nil {
		return fmt.Errorf("会话ID不能为空")
	}
	if len(req.Vector) != VectorDim {
		return fmt.Errorf("向量维度必须为 %d，当前为 %d", VectorDim, len(req.Vector))
	}

	// 构建 payload
	payload := map[string]interface{}{
		"memory_id":   req.MemoryID.String(),
		"tenant_id":   req.TenantID.String(),
		"session_id":  req.SessionID.String(),
		"memory_type": req.MemoryType,
		"importance":  req.Importance,
		"created_at":  time.Now().Format(time.RFC3339),
	}

	// 添加过期时间
	if req.ExpiresAt != nil {
		payload["expires_at"] = req.ExpiresAt.Format(time.RFC3339)
	}

	// 添加其他元数据
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			payload[k] = v
		}
	}

	// 构建 upsert 请求
	point := map[string]interface{}{
		"id":      req.MemoryID.String(),
		"vector":  req.Vector,
		"payload": payload,
	}

	upsertReq := map[string]interface{}{
		"points": []interface{}{point},
	}

	// 发送请求
	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, CollectionName)
	if err := c.doRequest(ctx, "PUT", url, upsertReq, nil); err != nil {
		return fmt.Errorf("插入向量失败: %w", err)
	}

	return nil
}

// BatchUpsertVectors 批量插入或更新向量（优化性能）
func (c *qdrantClientImpl) BatchUpsertVectors(ctx context.Context, reqs []*UpsertVectorRequest) error {
	if len(reqs) == 0 {
		return nil
	}

	// 验证所有请求
	for i, req := range reqs {
		if req == nil {
			return fmt.Errorf("请求 %d 不能为空", i)
		}
		if req.TenantID == uuid.Nil {
			return fmt.Errorf("请求 %d 的租户ID不能为空", i)
		}
		if req.MemoryID == uuid.Nil {
			return fmt.Errorf("请求 %d 的记忆ID不能为空", i)
		}
		if req.SessionID == uuid.Nil {
			return fmt.Errorf("请求 %d 的会话ID不能为空", i)
		}
		if len(req.Vector) != VectorDim {
			return fmt.Errorf("请求 %d 的向量维度必须为 %d，当前为 %d", i, VectorDim, len(req.Vector))
		}
	}

	// 构建批量 points
	points := make([]interface{}, 0, len(reqs))
	for _, req := range reqs {
		// 构建 payload
		payload := map[string]interface{}{
			"memory_id":   req.MemoryID.String(),
			"tenant_id":   req.TenantID.String(),
			"session_id":  req.SessionID.String(),
			"memory_type": req.MemoryType,
			"importance":  req.Importance,
			"created_at":  time.Now().Format(time.RFC3339),
		}

		// 添加过期时间
		if req.ExpiresAt != nil {
			payload["expires_at"] = req.ExpiresAt.Format(time.RFC3339)
		}

		// 添加其他元数据
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				payload[k] = v
			}
		}

		// 构建 point
		point := map[string]interface{}{
			"id":      req.MemoryID.String(),
			"vector":  req.Vector,
			"payload": payload,
		}
		points = append(points, point)
	}

	// 构建批量 upsert 请求
	upsertReq := map[string]interface{}{
		"points": points,
	}

	// 发送请求
	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, CollectionName)
	if err := c.doRequest(ctx, "PUT", url, upsertReq, nil); err != nil {
		return fmt.Errorf("批量插入向量失败: %w", err)
	}

	return nil
}

// SearchVectors 向量检索（带缓存）
func (c *qdrantClientImpl) SearchVectors(ctx context.Context, req *SearchVectorRequest) ([]*VectorSearchResult, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 验证必填字段
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("租户ID不能为空")
	}
	if len(req.QueryVector) != VectorDim {
		return nil, fmt.Errorf("查询向量维度必须为 %d，当前为 %d", VectorDim, len(req.QueryVector))
	}
	if req.TopK <= 0 {
		req.TopK = 5 // 默认返回5个结果
	}

	// 尝试从缓存获取结果
	if c.cache != nil {
		cacheKey := c.buildSearchCacheKey(req)
		var cachedResults []*VectorSearchResult
		if err := c.cache.Get(ctx, cacheKey, &cachedResults); err == nil {
			return cachedResults, nil
		}
	}

	// 构建过滤条件（强制包含租户ID）
	filter := c.buildFilter(req)

	// 构建搜索请求
	searchReq := map[string]interface{}{
		"vector":       req.QueryVector,
		"limit":        req.TopK,
		"with_payload": true,
		"with_vector":  false,
	}

	// 添加过滤条件
	if filter != nil {
		searchReq["filter"] = filter
	}

	// 添加分数阈值
	if req.MinScore > 0 {
		searchReq["score_threshold"] = req.MinScore
	}

	// 发送请求
	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, CollectionName)
	var response struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := c.doRequest(ctx, "POST", url, searchReq, &response); err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	// 转换结果
	results := make([]*VectorSearchResult, 0, len(response.Result))
	for _, item := range response.Result {
		result, err := c.parseSearchResult(item.ID, item.Score, item.Payload)
		if err != nil {
			// 记录错误但继续处理其他结果
			continue
		}
		results = append(results, result)
	}

	// 缓存结果（30分钟）
	if c.cache != nil && len(results) > 0 {
		cacheKey := c.buildSearchCacheKey(req)
		_ = c.cache.Set(ctx, cacheKey, results, 30*time.Minute)
	}

	return results, nil
}

// DeleteVector 删除向量
func (c *qdrantClientImpl) DeleteVector(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}
	if memoryID == uuid.Nil {
		return fmt.Errorf("记忆ID不能为空")
	}

	// 构建删除请求（包含租户验证）
	deleteReq := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"key": "tenant_id",
					"match": map[string]interface{}{
						"value": tenantID.String(),
					},
				},
				map[string]interface{}{
					"key": "memory_id",
					"match": map[string]interface{}{
						"value": memoryID.String(),
					},
				},
			},
		},
	}

	// 发送请求
	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, CollectionName)
	if err := c.doRequest(ctx, "POST", url, deleteReq, nil); err != nil {
		return fmt.Errorf("删除向量失败: %w", err)
	}

	return nil
}

// DeleteByFilter 按条件批量删除
func (c *qdrantClientImpl) DeleteByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	// 构建删除条件，强制包含租户ID
	conditions := []interface{}{
		map[string]interface{}{
			"key": "tenant_id",
			"match": map[string]interface{}{
				"value": tenantID.String(),
			},
		},
	}

	// 添加用户提供的过滤条件
	for field, value := range filter {
		conditions = append(conditions, map[string]interface{}{
			"key": field,
			"match": map[string]interface{}{
				"value": value,
			},
		})
	}

	// 构建删除请求
	deleteReq := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": conditions,
		},
	}

	// 发送请求
	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, CollectionName)
	if err := c.doRequest(ctx, "POST", url, deleteReq, nil); err != nil {
		return fmt.Errorf("批量删除向量失败: %w", err)
	}

	return nil
}

// UpdatePayload 更新 payload
func (c *qdrantClientImpl) UpdatePayload(ctx context.Context, tenantID, memoryID uuid.UUID, payload map[string]interface{}) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}
	if memoryID == uuid.Nil {
		return fmt.Errorf("记忆ID不能为空")
	}

	// 确保不能修改租户ID
	delete(payload, "tenant_id")

	// 构建更新请求
	updateReq := map[string]interface{}{
		"points": []string{memoryID.String()},
		"payload": payload,
	}

	// 发送请求
	url := fmt.Sprintf("%s/collections/%s/points/payload", c.baseURL, CollectionName)
	if err := c.doRequest(ctx, "PUT", url, updateReq, nil); err != nil {
		return fmt.Errorf("更新 payload 失败: %w", err)
	}

	return nil
}

// OptimizeCollection 优化 Collection（compact、reindex）
func (c *qdrantClientImpl) OptimizeCollection(ctx context.Context) error {
	// 执行优化操作
	url := fmt.Sprintf("%s/collections/%s/optimizer", c.baseURL, CollectionName)
	
	optimizeReq := map[string]interface{}{
		"optimize": true,
	}

	if err := c.doRequest(ctx, "POST", url, optimizeReq, nil); err != nil {
		return fmt.Errorf("优化 collection 失败: %w", err)
	}

	return nil
}

// GetCollectionInfo 获取 Collection 信息（用于监控）
func (c *qdrantClientImpl) GetCollectionInfo(ctx context.Context) (*CollectionInfo, error) {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, CollectionName)
	
	var response struct {
		Result struct {
			Status            string `json:"status"`
			VectorsCount      int64  `json:"vectors_count"`
			IndexedVectorsCount int64 `json:"indexed_vectors_count"`
			PointsCount       int64  `json:"points_count"`
			SegmentsCount     int    `json:"segments_count"`
			Config            struct {
				Params struct {
					ShardNumber       int `json:"shard_number"`
					ReplicationFactor int `json:"replication_factor"`
				} `json:"params"`
				HnswConfig struct {
					M              int `json:"m"`
					EfConstruction int `json:"ef_construction"`
				} `json:"hnsw_config"`
			} `json:"config"`
		} `json:"result"`
	}

	if err := c.doRequest(ctx, "GET", url, nil, &response); err != nil {
		return nil, fmt.Errorf("获取 collection 信息失败: %w", err)
	}

	info := &CollectionInfo{
		Status:              response.Result.Status,
		VectorsCount:        response.Result.VectorsCount,
		IndexedVectorsCount: response.Result.IndexedVectorsCount,
		PointsCount:         response.Result.PointsCount,
		SegmentsCount:       response.Result.SegmentsCount,
		Config: &CollectionConfig{
			ShardNumber:        response.Result.Config.Params.ShardNumber,
			ReplicationFactor:  response.Result.Config.Params.ReplicationFactor,
			HnswM:              response.Result.Config.HnswConfig.M,
			HnswEfConstruction: response.Result.Config.HnswConfig.EfConstruction,
		},
	}

	return info, nil
}

// UpdateCollectionConfig 更新 Collection 配置（分片、副本等）
func (c *qdrantClientImpl) UpdateCollectionConfig(ctx context.Context, config *CollectionConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 更新 HNSW 配置
	if config.HnswM > 0 || config.HnswEfConstruction > 0 {
		updateReq := map[string]interface{}{
			"hnsw_config": map[string]interface{}{},
		}

		if config.HnswM > 0 {
			updateReq["hnsw_config"].(map[string]interface{})["m"] = config.HnswM
		}
		if config.HnswEfConstruction > 0 {
			updateReq["hnsw_config"].(map[string]interface{})["ef_construction"] = config.HnswEfConstruction
		}

		url := fmt.Sprintf("%s/collections/%s", c.baseURL, CollectionName)
		if err := c.doRequest(ctx, "PATCH", url, updateReq, nil); err != nil {
			return fmt.Errorf("更新 HNSW 配置失败: %w", err)
		}
	}

	// 注意：分片数量和副本数量在创建后无法修改
	// 如果需要修改，需要重新创建 Collection

	return nil
}

// Close 关闭客户端连接
func (c *qdrantClientImpl) Close() error {
	// HTTP 客户端不需要显式关闭
	return nil
}

// ========== 私有辅助方法 ==========

// collectionExists 检查 Collection 是否存在
func (c *qdrantClientImpl) collectionExists(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, CollectionName)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 404 表示不存在
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// 200 表示存在
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	// 其他状态码表示错误
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("检查 collection 失败: status=%d, body=%s", resp.StatusCode, string(body))
}

// createCollection 创建 Collection
func (c *qdrantClientImpl) createCollection(ctx context.Context, config map[string]interface{}) error {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, CollectionName)
	return c.doRequest(ctx, "PUT", url, config, nil)
}

// createFieldIndex 创建字段索引
func (c *qdrantClientImpl) createFieldIndex(ctx context.Context, field, fieldType string, isTenant bool) error {
	indexReq := map[string]interface{}{
		"field_name": field,
		"field_schema": fieldType,
	}

	// 标记为租户字段
	if isTenant {
		indexReq["is_tenant"] = true
	}

	url := fmt.Sprintf("%s/collections/%s/index", c.baseURL, CollectionName)
	return c.doRequest(ctx, "PUT", url, indexReq, nil)
}

// buildFilter 构建过滤条件
func (c *qdrantClientImpl) buildFilter(req *SearchVectorRequest) map[string]interface{} {
	conditions := []interface{}{
		// 强制添加租户过滤条件
		map[string]interface{}{
			"key": "tenant_id",
			"match": map[string]interface{}{
				"value": req.TenantID.String(),
			},
		},
	}

	// 添加会话ID过滤
	if req.SessionID != nil && *req.SessionID != uuid.Nil {
		conditions = append(conditions, map[string]interface{}{
			"key": "session_id",
			"match": map[string]interface{}{
				"value": req.SessionID.String(),
			},
		})
	}

	// 添加记忆类型过滤
	if req.MemoryType != nil && *req.MemoryType != "" {
		conditions = append(conditions, map[string]interface{}{
			"key": "memory_type",
			"match": map[string]interface{}{
				"value": *req.MemoryType,
			},
		})
	}

	// 添加时间范围过滤
	if req.TimeRange != nil {
		conditions = append(conditions, map[string]interface{}{
			"key": "created_at",
			"range": map[string]interface{}{
				"gte": req.TimeRange.Start.Format(time.RFC3339),
				"lte": req.TimeRange.End.Format(time.RFC3339),
			},
		})
	}

	return map[string]interface{}{
		"must": conditions,
	}
}

// parseSearchResult 解析搜索结果
func (c *qdrantClientImpl) parseSearchResult(id string, score float32, payload map[string]interface{}) (*VectorSearchResult, error) {
	result := &VectorSearchResult{
		Score:   score,
		Payload: payload,
	}

	// 解析 memory_id
	if memoryIDStr, ok := payload["memory_id"].(string); ok {
		if memoryID, err := uuid.Parse(memoryIDStr); err == nil {
			result.MemoryID = memoryID
		}
	}

	// 解析 tenant_id
	if tenantIDStr, ok := payload["tenant_id"].(string); ok {
		if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
			result.TenantID = tenantID
		}
	}

	// 解析 session_id
	if sessionIDStr, ok := payload["session_id"].(string); ok {
		if sessionID, err := uuid.Parse(sessionIDStr); err == nil {
			result.SessionID = sessionID
		}
	}

	// 解析 memory_type
	if memoryType, ok := payload["memory_type"].(string); ok {
		result.MemoryType = memoryType
	}

	// 解析 importance
	if importance, ok := payload["importance"].(float64); ok {
		result.Importance = float32(importance)
	}

	// 解析 created_at
	if createdAtStr, ok := payload["created_at"].(string); ok {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			result.CreatedAt = createdAt
		}
	}

	// 解析 expires_at
	if expiresAtStr, ok := payload["expires_at"].(string); ok {
		if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			result.ExpiresAt = &expiresAt
		}
	}

	return result, nil
}

// doRequest 执行 HTTP 请求
func (c *qdrantClientImpl) doRequest(ctx context.Context, method, url string, reqBody interface{}, respBody interface{}) error {
	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("序列化请求失败: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败: status=%d, body=%s", resp.StatusCode, string(respData))
	}

	// 解析响应体
	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}

	return nil
}

// setHeaders 设置请求头
func (c *qdrantClientImpl) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	
	// 添加 API Key（如果配置了）
	if c.config.APIKey != "" {
		req.Header.Set("api-key", c.config.APIKey)
	}
}

// buildSearchCacheKey 构建搜索缓存键
func (c *qdrantClientImpl) buildSearchCacheKey(req *SearchVectorRequest) string {
	// 使用租户ID、会话ID、记忆类型和TopK构建缓存键
	// 注意：不包含向量本身，因为向量太大
	key := fmt.Sprintf("qdrant:search:%s:%d", req.TenantID.String(), req.TopK)
	
	if req.SessionID != nil {
		key += ":" + req.SessionID.String()
	}
	
	if req.MemoryType != nil {
		key += ":" + *req.MemoryType
	}
	
	if req.MinScore > 0 {
		key += fmt.Sprintf(":%.2f", req.MinScore)
	}
	
	return key
}
