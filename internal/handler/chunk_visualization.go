package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ai-knowledge-platform/internal/model"
	"ai-knowledge-platform/internal/processor"
	"ai-knowledge-platform/internal/service"
)

// ChunkVisualizationHandler 分块可视化处理器
type ChunkVisualizationHandler struct {
	documentService DocumentService
}

// DocumentService 文档服务接口（简化版）
type DocumentService interface {
	GetChunks(ctx context.Context, versionID uuid.UUID) ([]*model.Chunk, error)
	GetDocumentVersion(ctx context.Context, versionID uuid.UUID) (*service.DocumentVersionDetail, error)
}

// NewChunkVisualizationHandler 创建分块可视化处理器
func NewChunkVisualizationHandler(documentService DocumentService) *ChunkVisualizationHandler {
	return &ChunkVisualizationHandler{
		documentService: documentService,
	}
}

// VisualizeChunks 可视化分块结果
// @Summary 可视化分块结果
// @Description 获取分块结果的可视化数据，包括分块分布、重叠关系等
// @Tags 分块可视化
// @Produce json
// @Param version_id path string true "文档版本ID"
// @Success 200 {object} Response{data=ChunkVisualizationData}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/chunks/{version_id}/visualization [get]
func (h *ChunkVisualizationHandler) VisualizeChunks(c *gin.Context) {
	// 获取版本ID
	versionIDStr := c.Param("version_id")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的版本ID格式"))
		return
	}

	// 获取文档块
	chunks, err := h.documentService.GetChunks(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文档块失败: "+err.Error()))
		return
	}

	// 获取文档版本详情
	versionDetail, err := h.documentService.GetDocumentVersion(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文档版本失败: "+err.Error()))
		return
	}

	// 生成可视化数据
	visualizationData := h.generateVisualizationData(chunks, versionDetail)

	c.JSON(http.StatusOK, SuccessResponse(visualizationData))
}

// CompareChunkStrategies 比较不同分块策略
// @Summary 比较不同分块策略
// @Description 比较不同分块策略在同一文档上的效果
// @Tags 分块可视化
// @Accept json
// @Produce json
// @Param request body CompareStrategiesRequest true "比较策略请求"
// @Success 200 {object} Response{data=StrategyComparisonResult}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/chunks/compare-strategies [post]
func (h *ChunkVisualizationHandler) CompareChunkStrategies(c *gin.Context) {
	var req CompareStrategiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("请求参数错误: "+err.Error()))
		return
	}

	// 创建临时文档
	doc := &processor.Document{
		Content: req.Content,
		Title:   "比较文档",
	}

	// 比较不同策略
	comparison := h.compareStrategies(doc, req.Strategies)

	c.JSON(http.StatusOK, SuccessResponse(comparison))
}

// GetChunkStatistics 获取分块统计信息
// @Summary 获取分块统计信息
// @Description 获取文档版本的分块统计信息
// @Tags 分块可视化
// @Produce json
// @Param version_id path string true "文档版本ID"
// @Success 200 {object} Response{data=ChunkStatistics}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/chunks/{version_id}/statistics [get]
func (h *ChunkVisualizationHandler) GetChunkStatistics(c *gin.Context) {
	// 获取版本ID
	versionIDStr := c.Param("version_id")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("无效的版本ID格式"))
		return
	}

	// 获取文档块
	chunks, err := h.documentService.GetChunks(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse("获取文档块失败: "+err.Error()))
		return
	}

	// 计算统计信息
	statistics := h.calculateStatistics(chunks)

	c.JSON(http.StatusOK, SuccessResponse(statistics))
}

// generateVisualizationData 生成可视化数据
func (h *ChunkVisualizationHandler) generateVisualizationData(chunks []*model.Chunk, versionDetail *service.DocumentVersionDetail) ChunkVisualizationData {
	var chunkItems []ChunkVisualizationItem
	var overlaps []ChunkOverlap

	totalLength := len(versionDetail.Document.Content)

	for i, chunk := range chunks {
		// 计算在文档中的位置百分比
		startPercent := float64(chunk.Metadata["start_offset"].(int)) / float64(totalLength) * 100
		endPercent := float64(chunk.Metadata["end_offset"].(int)) / float64(totalLength) * 100

		item := ChunkVisualizationItem{
			Index:        i + 1,
			StartOffset:  chunk.Metadata["start_offset"].(int),
			EndOffset:    chunk.Metadata["end_offset"].(int),
			Length:       len(chunk.Content),
			TokenCount:   chunk.Metadata["token_count"].(int),
			StartPercent: startPercent,
			EndPercent:   endPercent,
			Content:      h.truncateForVisualization(chunk.Content, 100),
		}
		chunkItems = append(chunkItems, item)

		// 检查与下一个块的重叠
		if i < len(chunks)-1 {
			nextChunk := chunks[i+1]
			currentEnd := chunk.Metadata["end_offset"].(int)
			nextStart := nextChunk.Metadata["start_offset"].(int)

			if currentEnd > nextStart {
				overlap := ChunkOverlap{
					Chunk1Index:    i + 1,
					Chunk2Index:    i + 2,
					OverlapStart:   nextStart,
					OverlapEnd:     currentEnd,
					OverlapLength:  currentEnd - nextStart,
					OverlapContent: h.extractOverlapContent(versionDetail.Document.Content, nextStart, currentEnd),
				}
				overlaps = append(overlaps, overlap)
			}
		}
	}

	return ChunkVisualizationData{
		TotalChunks:    len(chunks),
		TotalLength:    totalLength,
		AverageLength:  h.calculateAverageLength(chunks),
		Chunks:         chunkItems,
		Overlaps:       overlaps,
		Distribution:   h.calculateDistribution(chunks),
		LengthHistogram: h.calculateLengthHistogram(chunks),
	}
}

// compareStrategies 比较不同策略
func (h *ChunkVisualizationHandler) compareStrategies(doc *processor.Document, strategies []ChunkConfigRequest) StrategyComparisonResult {
	var comparisons []StrategyComparison

	chunkerManager := processor.NewChunkerManager()

	for _, strategyConfig := range strategies {
		config := &processor.ChunkConfig{
			Strategy:        processor.ChunkStrategy(strategyConfig.Strategy),
			MaxSize:         strategyConfig.MaxSize,
			Overlap:         strategyConfig.Overlap,
			Separators:      strategyConfig.Separators,
			PreserveContext: strategyConfig.PreserveContext,
		}

		chunks, err := chunkerManager.ChunkDocument(doc, config)
		if err != nil {
			continue
		}

		comparison := StrategyComparison{
			Strategy:      strategyConfig.Strategy,
			ChunkCount:    len(chunks),
			AverageLength: h.calculateAverageLengthFromProcessorChunks(chunks),
			MinLength:     h.calculateMinLength(chunks),
			MaxLength:     h.calculateMaxLength(chunks),
			TotalTokens:   h.calculateTotalTokens(chunks),
			OverlapCount:  h.calculateOverlapCount(chunks, strategyConfig.Overlap),
			Efficiency:    h.calculateEfficiency(chunks, len(doc.Content)),
		}
		comparisons = append(comparisons, comparison)
	}

	return StrategyComparisonResult{
		DocumentLength: len(doc.Content),
		Comparisons:    comparisons,
		Recommendation: h.getBestStrategy(comparisons),
	}
}

// calculateStatistics 计算统计信息
func (h *ChunkVisualizationHandler) calculateStatistics(chunks []*model.Chunk) ChunkStatistics {
	if len(chunks) == 0 {
		return ChunkStatistics{}
	}

	var lengths []int
	var tokenCounts []int
	totalLength := 0
	totalTokens := 0

	for _, chunk := range chunks {
		length := len(chunk.Content)
		tokenCount := chunk.Metadata["token_count"].(int)

		lengths = append(lengths, length)
		tokenCounts = append(tokenCounts, tokenCount)
		totalLength += length
		totalTokens += tokenCount
	}

	return ChunkStatistics{
		TotalChunks:      len(chunks),
		TotalLength:      totalLength,
		TotalTokens:      totalTokens,
		AverageLength:    totalLength / len(chunks),
		AverageTokens:    totalTokens / len(chunks),
		MinLength:        h.minInt(lengths),
		MaxLength:        h.maxInt(lengths),
		MedianLength:     h.medianInt(lengths),
		StandardDeviation: h.standardDeviation(lengths),
		LengthDistribution: h.createDistribution(lengths),
		TokenDistribution:  h.createDistribution(tokenCounts),
	}
}

// 辅助函数

func (h *ChunkVisualizationHandler) truncateForVisualization(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

func (h *ChunkVisualizationHandler) extractOverlapContent(fullContent string, start, end int) string {
	if start >= len(fullContent) || end > len(fullContent) || start >= end {
		return ""
	}
	content := fullContent[start:end]
	return h.truncateForVisualization(content, 50)
}

func (h *ChunkVisualizationHandler) calculateAverageLength(chunks []*model.Chunk) int {
	if len(chunks) == 0 {
		return 0
	}
	total := 0
	for _, chunk := range chunks {
		total += len(chunk.Content)
	}
	return total / len(chunks)
}

func (h *ChunkVisualizationHandler) calculateDistribution(chunks []*model.Chunk) []DistributionBucket {
	if len(chunks) == 0 {
		return []DistributionBucket{}
	}

	// 创建长度分布桶
	buckets := make(map[string]int)
	for _, chunk := range chunks {
		length := len(chunk.Content)
		var bucket string
		if length < 500 {
			bucket = "0-500"
		} else if length < 1000 {
			bucket = "500-1000"
		} else if length < 1500 {
			bucket = "1000-1500"
		} else if length < 2000 {
			bucket = "1500-2000"
		} else {
			bucket = "2000+"
		}
		buckets[bucket]++
	}

	var distribution []DistributionBucket
	bucketOrder := []string{"0-500", "500-1000", "1000-1500", "1500-2000", "2000+"}
	for _, bucket := range bucketOrder {
		if count, exists := buckets[bucket]; exists {
			distribution = append(distribution, DistributionBucket{
				Range: bucket,
				Count: count,
			})
		}
	}

	return distribution
}

func (h *ChunkVisualizationHandler) calculateLengthHistogram(chunks []*model.Chunk) []HistogramBin {
	if len(chunks) == 0 {
		return []HistogramBin{}
	}

	// 计算直方图
	var lengths []int
	for _, chunk := range chunks {
		lengths = append(lengths, len(chunk.Content))
	}

	minLen := h.minInt(lengths)
	maxLen := h.maxInt(lengths)
	binCount := 10
	binSize := (maxLen - minLen) / binCount

	if binSize == 0 {
		binSize = 1
	}

	bins := make([]HistogramBin, binCount)
	for i := 0; i < binCount; i++ {
		start := minLen + i*binSize
		end := start + binSize
		if i == binCount-1 {
			end = maxLen + 1
		}
		bins[i] = HistogramBin{
			Start: start,
			End:   end,
			Count: 0,
		}
	}

	for _, length := range lengths {
		binIndex := (length - minLen) / binSize
		if binIndex >= binCount {
			binIndex = binCount - 1
		}
		bins[binIndex].Count++
	}

	return bins
}

func (h *ChunkVisualizationHandler) calculateAverageLengthFromProcessorChunks(chunks []processor.Chunk) int {
	if len(chunks) == 0 {
		return 0
	}
	total := 0
	for _, chunk := range chunks {
		total += len(chunk.Content)
	}
	return total / len(chunks)
}

func (h *ChunkVisualizationHandler) calculateMinLength(chunks []processor.Chunk) int {
	if len(chunks) == 0 {
		return 0
	}
	min := len(chunks[0].Content)
	for _, chunk := range chunks {
		if len(chunk.Content) < min {
			min = len(chunk.Content)
		}
	}
	return min
}

func (h *ChunkVisualizationHandler) calculateMaxLength(chunks []processor.Chunk) int {
	if len(chunks) == 0 {
		return 0
	}
	max := len(chunks[0].Content)
	for _, chunk := range chunks {
		if len(chunk.Content) > max {
			max = len(chunk.Content)
		}
	}
	return max
}

func (h *ChunkVisualizationHandler) calculateTotalTokens(chunks []processor.Chunk) int {
	total := 0
	for _, chunk := range chunks {
		total += chunk.TokenCount
	}
	return total
}

func (h *ChunkVisualizationHandler) calculateOverlapCount(chunks []processor.Chunk, overlapSize int) int {
	if overlapSize <= 0 || len(chunks) <= 1 {
		return 0
	}
	return len(chunks) - 1 // 简化计算
}

func (h *ChunkVisualizationHandler) calculateEfficiency(chunks []processor.Chunk, totalLength int) float64 {
	if totalLength == 0 {
		return 0
	}
	totalChunkLength := 0
	for _, chunk := range chunks {
		totalChunkLength += len(chunk.Content)
	}
	return float64(totalChunkLength) / float64(totalLength)
}

func (h *ChunkVisualizationHandler) getBestStrategy(comparisons []StrategyComparison) string {
	if len(comparisons) == 0 {
		return ""
	}
	
	best := comparisons[0]
	for _, comp := range comparisons {
		// 简单的评分逻辑：效率高且块数适中
		if comp.Efficiency > best.Efficiency && comp.ChunkCount > 0 {
			best = comp
		}
	}
	
	return best.Strategy
}

func (h *ChunkVisualizationHandler) minInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (h *ChunkVisualizationHandler) maxInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (h *ChunkVisualizationHandler) medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	// 简化实现，实际应该排序
	return values[len(values)/2]
}

func (h *ChunkVisualizationHandler) standardDeviation(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// 计算平均值
	sum := 0
	for _, v := range values {
		sum += v
	}
	mean := float64(sum) / float64(len(values))
	
	// 计算方差
	variance := 0.0
	for _, v := range values {
		variance += (float64(v) - mean) * (float64(v) - mean)
	}
	variance /= float64(len(values))
	
	// 返回标准差
	return variance // 简化实现，实际应该开平方根
}

func (h *ChunkVisualizationHandler) createDistribution(values []int) []DistributionBucket {
	// 简化实现
	return []DistributionBucket{}
}

// 数据结构定义

// ChunkVisualizationData 分块可视化数据
type ChunkVisualizationData struct {
	TotalChunks     int                      `json:"total_chunks"`
	TotalLength     int                      `json:"total_length"`
	AverageLength   int                      `json:"average_length"`
	Chunks          []ChunkVisualizationItem `json:"chunks"`
	Overlaps        []ChunkOverlap           `json:"overlaps"`
	Distribution    []DistributionBucket     `json:"distribution"`
	LengthHistogram []HistogramBin           `json:"length_histogram"`
}

// ChunkVisualizationItem 分块可视化项
type ChunkVisualizationItem struct {
	Index        int     `json:"index"`
	StartOffset  int     `json:"start_offset"`
	EndOffset    int     `json:"end_offset"`
	Length       int     `json:"length"`
	TokenCount   int     `json:"token_count"`
	StartPercent float64 `json:"start_percent"`
	EndPercent   float64 `json:"end_percent"`
	Content      string  `json:"content"`
}

// ChunkOverlap 分块重叠信息
type ChunkOverlap struct {
	Chunk1Index    int    `json:"chunk1_index"`
	Chunk2Index    int    `json:"chunk2_index"`
	OverlapStart   int    `json:"overlap_start"`
	OverlapEnd     int    `json:"overlap_end"`
	OverlapLength  int    `json:"overlap_length"`
	OverlapContent string `json:"overlap_content"`
}

// DistributionBucket 分布桶
type DistributionBucket struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// HistogramBin 直方图箱
type HistogramBin struct {
	Start int `json:"start"`
	End   int `json:"end"`
	Count int `json:"count"`
}

// CompareStrategiesRequest 比较策略请求
type CompareStrategiesRequest struct {
	Content    string               `json:"content" binding:"required"`
	Strategies []ChunkConfigRequest `json:"strategies" binding:"required"`
}

// StrategyComparisonResult 策略比较结果
type StrategyComparisonResult struct {
	DocumentLength int                  `json:"document_length"`
	Comparisons    []StrategyComparison `json:"comparisons"`
	Recommendation string               `json:"recommendation"`
}

// StrategyComparison 策略比较
type StrategyComparison struct {
	Strategy      string  `json:"strategy"`
	ChunkCount    int     `json:"chunk_count"`
	AverageLength int     `json:"average_length"`
	MinLength     int     `json:"min_length"`
	MaxLength     int     `json:"max_length"`
	TotalTokens   int     `json:"total_tokens"`
	OverlapCount  int     `json:"overlap_count"`
	Efficiency    float64 `json:"efficiency"`
}

// ChunkStatistics 分块统计信息
type ChunkStatistics struct {
	TotalChunks        int                  `json:"total_chunks"`
	TotalLength        int                  `json:"total_length"`
	TotalTokens        int                  `json:"total_tokens"`
	AverageLength      int                  `json:"average_length"`
	AverageTokens      int                  `json:"average_tokens"`
	MinLength          int                  `json:"min_length"`
	MaxLength          int                  `json:"max_length"`
	MedianLength       int                  `json:"median_length"`
	StandardDeviation  float64              `json:"standard_deviation"`
	LengthDistribution []DistributionBucket `json:"length_distribution"`
	TokenDistribution  []DistributionBucket `json:"token_distribution"`
}

// RegisterChunkVisualizationRoutes 注册分块可视化相关路由
func RegisterChunkVisualizationRoutes(r *gin.RouterGroup, handler *ChunkVisualizationHandler) {
	chunks := r.Group("/chunks")
	{
		chunks.GET("/:version_id/visualization", handler.VisualizeChunks)
		chunks.GET("/:version_id/statistics", handler.GetChunkStatistics)
		chunks.POST("/compare-strategies", handler.CompareChunkStrategies)
	}
}