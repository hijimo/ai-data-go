package handler

import (
	"net/http"

	"ai-knowledge-platform/internal/kms"
	"ai-knowledge-platform/internal/middleware"
	"github.com/gin-gonic/gin"
)

// KMSHandler KMS处理器
type KMSHandler struct {
	kmsManager    *kms.Manager
	secretManager *kms.SecretManager
}

// NewKMSHandler 创建KMS处理器
func NewKMSHandler(kmsManager *kms.Manager, secretManager *kms.SecretManager) *KMSHandler {
	return &KMSHandler{
		kmsManager:    kmsManager,
		secretManager: secretManager,
	}
}

// EncryptRequest 加密请求结构
type EncryptRequest struct {
	Plaintext string `json:"plaintext" binding:"required"`
	Provider  string `json:"provider,omitempty"`
}

// EncryptResponse 加密响应结构
type EncryptResponse struct {
	Ciphertext string `json:"ciphertext"`
	Provider   string `json:"provider"`
}

// DecryptRequest 解密请求结构
type DecryptRequest struct {
	Ciphertext string `json:"ciphertext" binding:"required"`
	Provider   string `json:"provider,omitempty"`
}

// DecryptResponse 解密响应结构
type DecryptResponse struct {
	Plaintext string `json:"plaintext"`
	Provider  string `json:"provider"`
}

// SecretRequest 敏感信息请求结构
type SecretRequest struct {
	Name        string            `json:"name" binding:"required"`
	Type        kms.SecretType    `json:"type" binding:"required"`
	Description string            `json:"description,omitempty"`
	Value       string            `json:"value" binding:"required"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Provider    string            `json:"provider,omitempty"`
}

// Encrypt 加密数据
// @Summary 加密数据
// @Description 使用KMS加密敏感数据
// @Tags KMS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body EncryptRequest true "加密请求"
// @Success 200 {object} EncryptResponse "加密成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Failure 500 {object} map[string]interface{} "加密失败"
// @Router /api/v1/kms/encrypt [post]
func (h *KMSHandler) Encrypt(c *gin.Context) {
	var req EncryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}
	
	var ciphertext string
	var err error
	var providerName string
	
	if req.Provider != "" {
		// 使用指定提供商
		ciphertext, err = h.kmsManager.EncryptWithProvider(c.Request.Context(), req.Provider, req.Plaintext)
		providerName = req.Provider
	} else {
		// 使用默认提供商
		ciphertext, err = h.kmsManager.Encrypt(c.Request.Context(), req.Plaintext)
		provider, _ := h.kmsManager.GetDefaultProvider()
		if provider != nil {
			providerName = string(provider.GetProviderType())
		}
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "加密失败",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, EncryptResponse{
		Ciphertext: ciphertext,
		Provider:   providerName,
	})
}

// Decrypt 解密数据
// @Summary 解密数据
// @Description 使用KMS解密敏感数据
// @Tags KMS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body DecryptRequest true "解密请求"
// @Success 200 {object} DecryptResponse "解密成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Failure 500 {object} map[string]interface{} "解密失败"
// @Router /api/v1/kms/decrypt [post]
func (h *KMSHandler) Decrypt(c *gin.Context) {
	var req DecryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}
	
	var plaintext string
	var err error
	var providerName string
	
	if req.Provider != "" {
		// 使用指定提供商
		plaintext, err = h.kmsManager.DecryptWithProvider(c.Request.Context(), req.Provider, req.Ciphertext)
		providerName = req.Provider
	} else {
		// 使用默认提供商
		plaintext, err = h.kmsManager.Decrypt(c.Request.Context(), req.Ciphertext)
		provider, _ := h.kmsManager.GetDefaultProvider()
		if provider != nil {
			providerName = string(provider.GetProviderType())
		}
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "解密失败",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, DecryptResponse{
		Plaintext: plaintext,
		Provider:  providerName,
	})
}

// EncryptSecret 加密敏感信息
// @Summary 加密敏感信息
// @Description 加密并存储敏感信息
// @Tags KMS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body SecretRequest true "敏感信息请求"
// @Success 200 {object} kms.EncryptedSecret "加密成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Failure 500 {object} map[string]interface{} "加密失败"
// @Router /api/v1/kms/secrets/encrypt [post]
func (h *KMSHandler) EncryptSecret(c *gin.Context) {
	var req SecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}
	
	// 创建敏感信息对象
	secret := &kms.Secret{
		ID:          "", // 将由SecretManager生成
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Value:       req.Value,
		Metadata:    req.Metadata,
		Tags:        req.Tags,
	}
	
	// 验证敏感信息
	if err := h.secretManager.ValidateSecret(secret); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "敏感信息验证失败",
			"message": err.Error(),
		})
		return
	}
	
	var encryptedSecret *kms.EncryptedSecret
	var err error
	
	if req.Provider != "" {
		// 使用指定提供商
		encryptedSecret, err = h.secretManager.EncryptSecretWithProvider(c.Request.Context(), req.Provider, secret)
	} else {
		// 使用默认提供商
		encryptedSecret, err = h.secretManager.EncryptSecret(c.Request.Context(), secret)
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "加密敏感信息失败",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, encryptedSecret)
}

// DecryptSecret 解密敏感信息
// @Summary 解密敏感信息
// @Description 解密敏感信息
// @Tags KMS
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body kms.EncryptedSecret true "加密的敏感信息"
// @Success 200 {object} kms.Secret "解密成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Failure 500 {object} map[string]interface{} "解密失败"
// @Router /api/v1/kms/secrets/decrypt [post]
func (h *KMSHandler) DecryptSecret(c *gin.Context) {
	var encryptedSecret kms.EncryptedSecret
	if err := c.ShouldBindJSON(&encryptedSecret); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}
	
	secret, err := h.secretManager.DecryptSecret(c.Request.Context(), &encryptedSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "解密敏感信息失败",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, secret)
}

// ListProviders 列出KMS提供商
// @Summary 列出KMS提供商
// @Description 获取所有已注册的KMS提供商列表
// @Tags KMS
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "提供商列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Router /api/v1/kms/providers [get]
func (h *KMSHandler) ListProviders(c *gin.Context) {
	providers := h.kmsManager.ListProviders()
	
	// 获取默认提供商
	var defaultProvider string
	if provider, err := h.kmsManager.GetDefaultProvider(); err == nil {
		defaultProvider = string(provider.GetProviderType())
	}
	
	c.JSON(http.StatusOK, gin.H{
		"providers":        providers,
		"default_provider": defaultProvider,
	})
}

// HealthCheck KMS健康检查
// @Summary KMS健康检查
// @Description 检查所有KMS提供商的健康状态
// @Tags KMS
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "健康检查结果"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Router /api/v1/kms/health [get]
func (h *KMSHandler) HealthCheck(c *gin.Context) {
	results := h.kmsManager.HealthCheckAll(c.Request.Context())
	
	// 统计健康状态
	healthy := 0
	unhealthy := 0
	healthStatus := make(map[string]string)
	
	for provider, err := range results {
		if err == nil {
			healthy++
			healthStatus[provider] = "healthy"
		} else {
			unhealthy++
			healthStatus[provider] = err.Error()
		}
	}
	
	overallStatus := "healthy"
	if unhealthy > 0 {
		overallStatus = "degraded"
		if healthy == 0 {
			overallStatus = "unhealthy"
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"overall_status": overallStatus,
		"providers":      healthStatus,
		"summary": gin.H{
			"total":     len(results),
			"healthy":   healthy,
			"unhealthy": unhealthy,
		},
	})
}

// GenerateDataKey 生成数据密钥
// @Summary 生成数据密钥
// @Description 生成用于数据加密的密钥
// @Tags KMS
// @Security BearerAuth
// @Produce json
// @Param key_spec query string false "密钥规格" default(AES_256)
// @Param provider query string false "KMS提供商"
// @Success 200 {object} kms.DataKey "数据密钥"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 403 {object} map[string]interface{} "权限不足"
// @Failure 500 {object} map[string]interface{} "生成失败"
// @Router /api/v1/kms/data-key [post]
func (h *KMSHandler) GenerateDataKey(c *gin.Context) {
	keySpec := c.Query("key_spec")
	if keySpec == "" {
		keySpec = "AES_256"
	}
	
	providerName := c.Query("provider")
	
	var dataKey *kms.DataKey
	var err error
	
	if providerName != "" {
		// 使用指定提供商
		provider, err := h.kmsManager.GetProvider(providerName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "提供商不存在",
				"message": err.Error(),
			})
			return
		}
		dataKey, err = provider.GenerateDataKey(c.Request.Context(), keySpec)
	} else {
		// 使用默认提供商
		dataKey, err = h.kmsManager.GenerateDataKey(c.Request.Context(), keySpec)
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "生成数据密钥失败",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, dataKey)
}