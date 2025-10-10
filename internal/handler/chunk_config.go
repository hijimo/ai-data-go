package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-knowledge-platform/internal/processor"
)

// ChunkConfigHandler 分块配置处理器
type ChunkConfigHandler struct{}

// NewChunkConfigHandler 创建分块配置处理器
func NewChunkConfigHandler() *ChunkConfigHandler {
	return &ChunkConfigHandler{}
}

// GetDefaultConfigs 获取默认分块配置
// @Summary 获取默认分块配置
// @Description 获取各种分块策略的默认配置
// @Tags 分块配置
// @Produce json
// @Success 200 {object} Response{data=map[string]ChunkConfigTemplate}
// @Router /api/v1/chunk-configs/defaults [get]
func (h *ChunkConfigHandler) GetDefaultConfigs(c *gin.Context) {
	configs := map[string]ChunkConfigTemplate{
		"fixed_size": {
			Name:        "固定长度分块",
			Description: "按固定字符数分割文档，适用于大多数文档类型",
			Strategy:    "fixed_size",
			MaxSize:     1500,
			Overlap:     150,
			Separators:  []string{"\n\n", "\n", "。", "！", "？", ".", "!", "?"},
			PreserveContext: true,
			UseCase:     "通用文档处理，平衡分块大小和语义完整性",
		},
		"semantic": {
			Name:        "语义分块",
			Description: "基于段落和语义边界分割文档，保持内容的逻辑完整性",
			Strategy:    "semantic",
			MaxSize:     2000,
			Overlap:     100,
			Separators:  []string{"\n\n", "\n"},
			PreserveContext: true,
			UseCase:     "文章、报告等结构化文档，需要保持语义完整性",
		},
		"structure": {
			Name:        "结构化分块",
			Description: "基于文档结构（标题、章节）分割，适用于有明确层级结构的文档",
			Strategy:    "structure",
			MaxSize:     3000,
			Overlap:     0,
			Separators:  []string{},
			PreserveContext: true,
			UseCase:     "技术文档、学术论文等有明确章节结构的文档",
		},
		"code": {
			Name:        "代码分块",
			Description: "基于代码结构（函数、类、方法）分割，适用于代码文档",
			Strategy:    "code",
			MaxSize:     2500,
			Overlap:     50,
			Separators:  []string{"\n\n", "\n"},
			PreserveContext: true,
			UseCase:     "代码文档、API文档等包含大量代码的文档",
		},
	}

	c.JSON(http.StatusOK, SuccessResponse(configs))
}

// ValidateConfig 验证分块配置
// @Summary 验证分块配置
// @Description 验证分块配置的有效性
// @Tags 分块配置
// @Accept json
// @Produce json
// @Param config body ChunkConfigRequest true "分块配置"
// @Success 200 {object} Response{data=ConfigValidationResult}
// @Failure 400 {object} Response
// @Router /api/v1/chunk-configs/validate [post]
func (h *ChunkConfigHandler) ValidateConfig(c *gin.Context) {
	var config ChunkConfigRequest
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("请求参数错误: "+err.Error()))
		return
	}

	result := h.validateChunkConfig(&config)
	c.JSON(http.StatusOK, SuccessResponse(result))
}

// GetRecommendedConfig 获取推荐配置
// @Summary 获取推荐配置
// @Description 基于文档类型和内容特征推荐最适合的分块配置
// @Tags 分块配置
// @Accept json
// @Produce json
// @Param request body RecommendConfigRequest true "推荐配置请求"
// @Success 200 {object} Response{data=ChunkConfigRecommendation}
// @Failure 400 {object} Response
// @Router /api/v1/chunk-configs/recommend [post]
func (h *ChunkConfigHandler) GetRecommendedConfig(c *gin.Context) {
	var req RecommendConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("请求参数错误: "+err.Error()))
		return
	}

	recommendation := h.recommendConfig(&req)
	c.JSON(http.StatusOK, SuccessResponse(recommendation))
}

// validateChunkConfig 验证分块配置
func (h *ChunkConfigHandler) validateChunkConfig(config *ChunkConfigRequest) ConfigValidationResult {
	result := ConfigValidationResult{
		IsValid:  true,
		Warnings: []string{},
		Errors:   []string{},
		Suggestions: []string{},
	}

	// 验证策略
	validStrategies := []string{"fixed_size", "semantic", "structure", "code"}
	isValidStrategy := false
	for _, strategy := range validStrategies {
		if config.Strategy == strategy {
			isValidStrategy = true
			break
		}
	}
	if !isValidStrategy {
		result.IsValid = false
		result.Errors = append(result.Errors, "不支持的分块策略: "+config.Strategy)
	}

	// 验证最大长度
	if config.MaxSize < 100 {
		result.IsValid = false
		result.Errors = append(result.Errors, "最大长度不能小于100字符")
	} else if config.MaxSize > 10000 {
		result.Warnings = append(result.Warnings, "最大长度过大，可能影响处理效率")
	}

	// 验证重叠设置
	if config.Overlap < 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "重叠长度不能为负数")
	} else if config.Overlap >= config.MaxSize {
		result.IsValid = false
		result.Errors = append(result.Errors, "重叠长度不能大于或等于最大长度")
	} else if config.Overlap > config.MaxSize/2 {
		result.Warnings = append(result.Warnings, "重叠长度过大，建议不超过最大长度的50%")
	}

	// 提供建议
	if config.Strategy == "fixed_size" && config.MaxSize > 2000 {
		result.Suggestions = append(result.Suggestions, "固定长度分块建议使用1000-2000字符的块大小")
	}

	if config.Strategy == "semantic" && config.Overlap > 200 {
		result.Suggestions = append(result.Suggestions, "语义分块通常不需要太大的重叠，建议100-200字符")
	}

	if config.Strategy == "structure" && config.Overlap > 0 {
		result.Suggestions = append(result.Suggestions, "结构化分块通常不需要重叠，因为基于自然的章节边界")
	}

	return result
}

// recommendConfig 推荐配置
func (h *ChunkConfigHandler) recommendConfig(req *RecommendConfigRequest) ChunkConfigRecommendation {
	// 基于文档类型和特征推荐配置
	var recommendedStrategy string
	var maxSize int
	var overlap int
	var reason string

	switch req.DocumentType {
	case "markdown", "html":
		if req.HasStructure {
			recommendedStrategy = "structure"
			maxSize = 2500
			overlap = 0
			reason = "文档具有明确的结构层级，建议使用结构化分块"
		} else {
			recommendedStrategy = "semantic"
			maxSize = 2000
			overlap = 100
			reason = "标记语言文档建议使用语义分块保持内容完整性"
		}
	case "code":
		recommendedStrategy = "code"
		maxSize = 2000
		overlap = 50
		reason = "代码文档建议使用代码分块，基于函数和类结构分割"
	case "pdf", "docx":
		if req.HasStructure {
			recommendedStrategy = "structure"
			maxSize = 3000
			overlap = 0
			reason = "结构化文档建议基于章节分块"
		} else {
			recommendedStrategy = "semantic"
			maxSize = 2000
			overlap = 150
			reason = "长文档建议使用语义分块，保持段落完整性"
		}
	default:
		recommendedStrategy = "fixed_size"
		maxSize = 1500
		overlap = 150
		reason = "通用文档建议使用固定长度分块，平衡效果和性能"
	}

	// 根据文档长度调整
	if req.DocumentLength > 50000 {
		maxSize = min(maxSize+500, 4000)
		reason += "；文档较长，适当增加块大小"
	} else if req.DocumentLength < 5000 {
		maxSize = max(maxSize-300, 800)
		reason += "；文档较短，适当减小块大小"
	}

	return ChunkConfigRecommendation{
		RecommendedConfig: ChunkConfigTemplate{
			Name:        getStrategyName(recommendedStrategy),
			Description: getStrategyDescription(recommendedStrategy),
			Strategy:    recommendedStrategy,
			MaxSize:     maxSize,
			Overlap:     overlap,
			Separators:  getDefaultSeparators(recommendedStrategy),
			PreserveContext: true,
			UseCase:     reason,
		},
		Confidence: h.calculateConfidence(req),
		Reason:     reason,
		Alternatives: h.getAlternativeConfigs(recommendedStrategy, req),
	}
}

// calculateConfidence 计算推荐置信度
func (h *ChunkConfigHandler) calculateConfidence(req *RecommendConfigRequest) float64 {
	confidence := 0.7 // 基础置信度

	// 如果有明确的文档类型，增加置信度
	if req.DocumentType != "" && req.DocumentType != "unknown" {
		confidence += 0.2
	}

	// 如果有结构信息，增加置信度
	if req.HasStructure {
		confidence += 0.1
	}

	// 如果文档长度在合理范围内，增加置信度
	if req.DocumentLength > 1000 && req.DocumentLength < 100000 {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// getAlternativeConfigs 获取备选配置
func (h *ChunkConfigHandler) getAlternativeConfigs(recommended string, req *RecommendConfigRequest) []ChunkConfigTemplate {
	var alternatives []ChunkConfigTemplate

	strategies := []string{"fixed_size", "semantic", "structure", "code"}
	for _, strategy := range strategies {
		if strategy != recommended {
			alternatives = append(alternatives, ChunkConfigTemplate{
				Name:        getStrategyName(strategy),
				Description: getStrategyDescription(strategy),
				Strategy:    strategy,
				MaxSize:     getDefaultMaxSize(strategy),
				Overlap:     getDefaultOverlap(strategy),
				Separators:  getDefaultSeparators(strategy),
				PreserveContext: true,
				UseCase:     getStrategyUseCase(strategy),
			})
		}
	}

	return alternatives
}

// 辅助函数
func getStrategyName(strategy string) string {
	names := map[string]string{
		"fixed_size": "固定长度分块",
		"semantic":   "语义分块",
		"structure":  "结构化分块",
		"code":       "代码分块",
	}
	return names[strategy]
}

func getStrategyDescription(strategy string) string {
	descriptions := map[string]string{
		"fixed_size": "按固定字符数分割文档",
		"semantic":   "基于段落和语义边界分割",
		"structure":  "基于文档结构分割",
		"code":       "基于代码结构分割",
	}
	return descriptions[strategy]
}

func getDefaultMaxSize(strategy string) int {
	sizes := map[string]int{
		"fixed_size": 1500,
		"semantic":   2000,
		"structure":  3000,
		"code":       2500,
	}
	return sizes[strategy]
}

func getDefaultOverlap(strategy string) int {
	overlaps := map[string]int{
		"fixed_size": 150,
		"semantic":   100,
		"structure":  0,
		"code":       50,
	}
	return overlaps[strategy]
}

func getDefaultSeparators(strategy string) []string {
	separators := map[string][]string{
		"fixed_size": {"\n\n", "\n", "。", "！", "？", ".", "!", "?"},
		"semantic":   {"\n\n", "\n"},
		"structure":  {},
		"code":       {"\n\n", "\n"},
	}
	return separators[strategy]
}

func getStrategyUseCase(strategy string) string {
	useCases := map[string]string{
		"fixed_size": "通用文档处理",
		"semantic":   "文章、报告等结构化文档",
		"structure":  "技术文档、学术论文",
		"code":       "代码文档、API文档",
	}
	return useCases[strategy]
}

// 请求和响应结构体

// ChunkConfigTemplate 分块配置模板
type ChunkConfigTemplate struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Strategy        string   `json:"strategy"`
	MaxSize         int      `json:"max_size"`
	Overlap         int      `json:"overlap"`
	Separators      []string `json:"separators"`
	PreserveContext bool     `json:"preserve_context"`
	UseCase         string   `json:"use_case"`
}

// ConfigValidationResult 配置验证结果
type ConfigValidationResult struct {
	IsValid     bool     `json:"is_valid"`
	Warnings    []string `json:"warnings"`
	Errors      []string `json:"errors"`
	Suggestions []string `json:"suggestions"`
}

// RecommendConfigRequest 推荐配置请求
type RecommendConfigRequest struct {
	DocumentType   string `json:"document_type"`   // pdf, docx, markdown, html, text, code
	DocumentLength int    `json:"document_length"` // 文档长度（字符数）
	HasStructure   bool   `json:"has_structure"`   // 是否有明确的结构层级
	Language       string `json:"language"`        // 文档语言
	Purpose        string `json:"purpose"`         // 使用目的：search, qa, summary, training
}

// ChunkConfigRecommendation 分块配置推荐
type ChunkConfigRecommendation struct {
	RecommendedConfig ChunkConfigTemplate   `json:"recommended_config"`
	Confidence        float64               `json:"confidence"`        // 推荐置信度 0-1
	Reason            string                `json:"reason"`            // 推荐理由
	Alternatives      []ChunkConfigTemplate `json:"alternatives"`      // 备选配置
}

// RegisterChunkConfigRoutes 注册分块配置相关路由
func RegisterChunkConfigRoutes(r *gin.RouterGroup, handler *ChunkConfigHandler) {
	configs := r.Group("/chunk-configs")
	{
		configs.GET("/defaults", handler.GetDefaultConfigs)
		configs.POST("/validate", handler.ValidateConfig)
		configs.POST("/recommend", handler.GetRecommendedConfig)
	}
}

// max 辅助函数
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}