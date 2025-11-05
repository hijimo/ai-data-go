package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"genkit-ai-service/internal/logger"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// qdrantVectorService Qdrant 向量服务实现，集成 Google AI 嵌入模型
type qdrantVectorService struct {
	config         *VectorServiceConfig
	httpClient     *http.Client
	qdrantClient   *qdrant.Client
	logger         logger.Logger
	dimension      int
	apiURL         string
	collectionName string
}

// googleAIEmbedRequest Google AI 嵌入请求
type googleAIEmbedRequest struct {
	Model   string               `json:"model"`
	Content googleAIEmbedContent `json:"content"`
}

// googleAIEmbedContent 嵌入内容
type googleAIEmbedContent struct {
	Parts []googleAIEmbedPart `json:"parts"`
}

// googleAIEmbedPart 嵌入部分
type googleAIEmbedPart struct {
	Text string `json:"text"`
}

// googleAIEmbedResponse Google AI 嵌入响应
type googleAIEmbedResponse struct {
	Embedding googleAIEmbedding `json:"embedding"`
}

// googleAIEmbedding 嵌入向量
type googleAIEmbedding struct {
	Values []float32 `json:"values"`
}

// NewGoogleAIVectorService 创建 Google AI 向量服务，集成 Qdrant
func NewGoogleAIVectorService(config *VectorServiceConfig, log logger.Logger) (VectorService, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API 密钥不能为空")
	}

	if config.QdrantEndpoint == "" {
		return nil, fmt.Errorf("Qdrant 端点不能为空")
	}

	if config.QdrantAPIKey == "" {
		return nil, fmt.Errorf("Qdrant API 密钥不能为空")
	}

	if config.Model == "" {
		config.Model = "text-embedding-004"
	}

	if config.Dimension == 0 {
		config.Dimension = 768
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.CollectionName == "" {
		config.CollectionName = "conversation_memories"
	}

	// 创建 Qdrant 客户端
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host:   config.QdrantEndpoint,
		APIKey: config.QdrantAPIKey,
		UseTLS: true,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Qdrant 客户端失败: %w", err)
	}

	service := &qdrantVectorService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		qdrantClient:   qdrantClient,
		logger:         log,
		dimension:      config.Dimension,
		apiURL:         fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s", config.Model, config.APIKey),
		collectionName: config.CollectionName,
	}

	log.Info("Qdrant 向量服务初始化成功", logger.Fields{
		"model":          config.Model,
		"dimension":      config.Dimension,
		"qdrantEndpoint": config.QdrantEndpoint,
		"collectionName": config.CollectionName,
	})

	return service, nil
}

// GenerateEmbedding 生成单个文本的向量
func (s *qdrantVectorService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("文本不能为空")
	}

	startTime := time.Now()

	var lastErr error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			s.logger.WarnContext(ctx, "向量生成失败，准备重试", logger.Fields{
				"attempt": attempt,
				"backoff": backoff.String(),
				"error":   lastErr.Error(),
			})
			time.Sleep(backoff)
		}

		reqBody := googleAIEmbedRequest{
			Model: fmt.Sprintf("models/%s", s.config.Model),
			Content: googleAIEmbedContent{
				Parts: []googleAIEmbedPart{{Text: text}},
			},
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			lastErr = fmt.Errorf("序列化请求失败: %w", err)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("创建请求失败: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("发送请求失败: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("API 返回错误: %d, %s", resp.StatusCode, string(body))
			continue
		}

		var embedResp googleAIEmbedResponse
		if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %w", err)
			continue
		}

		if len(embedResp.Embedding.Values) == 0 {
			lastErr = fmt.Errorf("API 返回空向量")
			continue
		}

		duration := time.Since(startTime)
		s.logger.InfoContext(ctx, "向量生成成功", logger.Fields{
			"textLength": len(text),
			"dimension":  len(embedResp.Embedding.Values),
			"duration":   duration.String(),
			"attempt":    attempt + 1,
		})

		return embedResp.Embedding.Values, nil
	}

	s.logger.ErrorContext(ctx, "向量生成失败，已达最大重试次数", logger.Fields{
		"maxRetries": s.config.MaxRetries,
		"error":      lastErr.Error(),
	})

	return nil, fmt.Errorf("生成向量失败: %w", lastErr)
}

// GenerateEmbeddings 批量生成文本向量
func (s *qdrantVectorService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	startTime := time.Now()
	totalTexts := len(texts)

	s.logger.InfoContext(ctx, "开始批量生成向量", logger.Fields{
		"totalTexts": totalTexts,
		"batchSize":  s.config.BatchSize,
	})

	var allVectors [][]float32
	for i := 0; i < totalTexts; i += s.config.BatchSize {
		end := i + s.config.BatchSize
		if end > totalTexts {
			end = totalTexts
		}

		batch := texts[i:end]
		batchNum := (i / s.config.BatchSize) + 1

		s.logger.InfoContext(ctx, "处理批次", logger.Fields{
			"batchNum":  batchNum,
			"batchSize": len(batch),
			"progress":  fmt.Sprintf("%d/%d", end, totalTexts),
		})

		vectors, err := s.generateBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("批次 %d 生成失败: %w", batchNum, err)
		}

		allVectors = append(allVectors, vectors...)
	}

	duration := time.Since(startTime)
	s.logger.InfoContext(ctx, "批量向量生成完成", logger.Fields{
		"totalTexts":   totalTexts,
		"totalVectors": len(allVectors),
		"duration":     duration.String(),
		"avgPerText":   duration / time.Duration(totalTexts),
	})

	return allVectors, nil
}

// generateBatch 生成单个批次的向量
func (s *qdrantVectorService) generateBatch(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))

	for i, text := range texts {
		vector, err := s.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("生成第 %d 个向量失败: %w", i+1, err)
		}
		vectors[i] = vector
	}

	return vectors, nil
}

// GetEmbeddingDimension 获取向量维度
func (s *qdrantVectorService) GetEmbeddingDimension() int {
	return s.dimension
}


// EnsureCollection 确保集合存在，如果不存在则创建
func (s *qdrantVectorService) EnsureCollection(ctx context.Context) error {
	// 检查集合是否存在
	exists, err := s.qdrantClient.CollectionExists(ctx, s.collectionName)
	if err != nil {
		return fmt.Errorf("检查集合是否存在失败: %w", err)
	}

	if exists {
		s.logger.InfoContext(ctx, "集合已存在", logger.Fields{
			"collectionName": s.collectionName,
		})
		return nil
	}

	// 创建集合，配置多租户优化
	err = s.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: s.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(s.dimension),
			Distance: qdrant.Distance_Cosine,
		}),
		// 多租户优化配置
		HnswConfig: &qdrant.HnswConfigDiff{
			PayloadM: qdrant.PtrOf(uint64(16)), // 为每个租户构建独立索引
			M:        qdrant.PtrOf(uint64(0)),  // 禁用全局索引
		},
	})

	if err != nil {
		return fmt.Errorf("创建集合失败: %w", err)
	}

	// 创建租户ID索引以优化查询性能
	_, err = s.qdrantClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: s.collectionName,
		FieldName:      "tenant_id",
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
	})

	if err != nil {
		s.logger.WarnContext(ctx, "创建租户ID索引失败", logger.Fields{
			"error": err.Error(),
		})
	}

	// 创建会话ID索引
	_, err = s.qdrantClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: s.collectionName,
		FieldName:      "session_id",
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
	})

	if err != nil {
		s.logger.WarnContext(ctx, "创建会话ID索引失败", logger.Fields{
			"error": err.Error(),
		})
	}

	s.logger.InfoContext(ctx, "集合创建成功", logger.Fields{
		"collectionName": s.collectionName,
		"dimension":      s.dimension,
	})

	return nil
}

// StoreVector 存储向量到 Qdrant（支持多租户隔离）
func (s *qdrantVectorService) StoreVector(ctx context.Context, req *StoreVectorRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	if req.PointID == "" {
		return fmt.Errorf("点ID不能为空")
	}

	if req.TenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	if len(req.Vector) == 0 {
		return fmt.Errorf("向量不能为空")
	}

	// 构建 payload，包含租户ID用于多租户隔离
	payload := map[string]interface{}{
		"tenant_id":  req.TenantID.String(),
		"session_id": req.SessionID.String(),
		"content":    req.Content,
	}

	// 合并用户提供的元数据
	for k, v := range req.Metadata {
		payload[k] = v
	}

	// 创建点
	point := &qdrant.PointStruct{
		Id:      qdrant.NewIDNum(hashPointID(req.PointID)),
		Vectors: qdrant.NewVectors(req.Vector...),
		Payload: qdrant.NewValueMap(payload),
	}

	// 插入点
	_, err := s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "存储向量失败", logger.Fields{
			"pointID":  req.PointID,
			"tenantID": req.TenantID.String(),
			"error":    err.Error(),
		})
		return fmt.Errorf("存储向量失败: %w", err)
	}

	s.logger.InfoContext(ctx, "向量存储成功", logger.Fields{
		"pointID":   req.PointID,
		"tenantID":  req.TenantID.String(),
		"sessionID": req.SessionID.String(),
	})

	return nil
}

// StoreVectors 批量存储向量到 Qdrant
func (s *qdrantVectorService) StoreVectors(ctx context.Context, reqs []*StoreVectorRequest) error {
	if len(reqs) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(reqs))

	for i, req := range reqs {
		if req.PointID == "" || req.TenantID == uuid.Nil || len(req.Vector) == 0 {
			return fmt.Errorf("第 %d 个请求无效", i+1)
		}

		payload := map[string]interface{}{
			"tenant_id":  req.TenantID.String(),
			"session_id": req.SessionID.String(),
			"content":    req.Content,
		}

		for k, v := range req.Metadata {
			payload[k] = v
		}

		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(hashPointID(req.PointID)),
			Vectors: qdrant.NewVectors(req.Vector...),
			Payload: qdrant.NewValueMap(payload),
		}
	}

	_, err := s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Points:         points,
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "批量存储向量失败", logger.Fields{
			"count": len(reqs),
			"error": err.Error(),
		})
		return fmt.Errorf("批量存储向量失败: %w", err)
	}

	s.logger.InfoContext(ctx, "批量向量存储成功", logger.Fields{
		"count": len(reqs),
	})

	return nil
}

// SearchVectors 向量相似度搜索（支持多租户隔离）
func (s *qdrantVectorService) SearchVectors(ctx context.Context, req *SearchVectorRequest) ([]*VectorSearchResult, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("租户ID不能为空")
	}

	// 如果提供了查询文本，生成向量
	queryVector := req.QueryVector
	if req.QueryText != "" {
		vector, err := s.GenerateEmbedding(ctx, req.QueryText)
		if err != nil {
			return nil, fmt.Errorf("生成查询向量失败: %w", err)
		}
		queryVector = vector
	}

	if len(queryVector) == 0 {
		return nil, fmt.Errorf("查询向量不能为空")
	}

	// 构建过滤条件 - 关键：多租户隔离
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("tenant_id", req.TenantID.String()),
		},
	}

	// 如果指定了会话ID，添加会话过滤
	if req.SessionID != nil && *req.SessionID != uuid.Nil {
		filter.Must = append(filter.Must, qdrant.NewMatch("session_id", req.SessionID.String()))
	}

	// 添加额外的过滤条件
	for key, value := range req.Filter {
		if strVal, ok := value.(string); ok {
			filter.Must = append(filter.Must, qdrant.NewMatch(key, strVal))
		}
	}

	// 设置默认值
	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	scoreThreshold := req.ScoreThreshold
	if scoreThreshold == 0 {
		scoreThreshold = 0.7
	}

	// 执行搜索
	searchResult, err := s.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint64(limit)),
		ScoreThreshold: qdrant.PtrOf(float32(scoreThreshold)),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "向量搜索失败", logger.Fields{
			"tenantID": req.TenantID.String(),
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("向量搜索失败: %w", err)
	}

	// 转换结果
	results := make([]*VectorSearchResult, len(searchResult))
	for i, point := range searchResult {
		result := &VectorSearchResult{
			PointID:  fmt.Sprintf("%d", point.Id.GetNum()),
			Score:    point.Score,
			Metadata: make(map[string]interface{}),
		}

		// 提取 payload
		if point.Payload != nil {
			for key, value := range point.Payload {
				if key == "content" {
					if strVal, ok := value.GetKind().(*qdrant.Value_StringValue); ok {
						result.Content = strVal.StringValue
					}
				} else {
					result.Metadata[key] = value
				}
			}
		}

		results[i] = result
	}

	s.logger.InfoContext(ctx, "向量搜索成功", logger.Fields{
		"tenantID":    req.TenantID.String(),
		"resultCount": len(results),
	})

	return results, nil
}

// DeleteVector 删除向量
func (s *qdrantVectorService) DeleteVector(ctx context.Context, tenantID uuid.UUID, pointID string) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	if pointID == "" {
		return fmt.Errorf("点ID不能为空")
	}

	// 删除点
	_, err := s.qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						qdrant.NewIDNum(hashPointID(pointID)),
					},
				},
			},
		},
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "删除向量失败", logger.Fields{
			"pointID":  pointID,
			"tenantID": tenantID.String(),
			"error":    err.Error(),
		})
		return fmt.Errorf("删除向量失败: %w", err)
	}

	s.logger.InfoContext(ctx, "向量删除成功", logger.Fields{
		"pointID":  pointID,
		"tenantID": tenantID.String(),
	})

	return nil
}

// DeleteVectorsByFilter 根据过滤条件删除向量（支持多租户隔离）
func (s *qdrantVectorService) DeleteVectorsByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("租户ID不能为空")
	}

	// 构建过滤条件 - 关键：多租户隔离
	qdrantFilter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("tenant_id", tenantID.String()),
		},
	}

	// 添加额外的过滤条件
	for key, value := range filter {
		if strVal, ok := value.(string); ok {
			qdrantFilter.Must = append(qdrantFilter.Must, qdrant.NewMatch(key, strVal))
		}
	}

	// 删除符合条件的点
	_, err := s.qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: qdrantFilter,
			},
		},
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "根据过滤条件删除向量失败", logger.Fields{
			"tenantID": tenantID.String(),
			"filter":   filter,
			"error":    err.Error(),
		})
		return fmt.Errorf("根据过滤条件删除向量失败: %w", err)
	}

	s.logger.InfoContext(ctx, "根据过滤条件删除向量成功", logger.Fields{
		"tenantID": tenantID.String(),
		"filter":   filter,
	})

	return nil
}

// hashPointID 将字符串ID转换为数字ID
func hashPointID(pointID string) uint64 {
	var hash uint64
	for i := 0; i < len(pointID); i++ {
		hash = hash*31 + uint64(pointID[i])
	}
	return hash
}
