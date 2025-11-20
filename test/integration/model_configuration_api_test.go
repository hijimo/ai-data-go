package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"genkit-ai-service/internal/api"
	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestModelConfigurationAPI 模型配置API端到端测试套件
type TestModelConfigurationAPI struct {
	db           *gorm.DB
	router       http.Handler
	log          logger.Logger
	systemAdmin  *testUser
	tenantAdmin1 *testUser
	tenantAdmin2 *testUser
	tenant1      *model.Tenant
	tenant2      *model.Tenant
	config1      *model.ModelConfiguration
	config2      *model.ModelConfiguration
}

// setupModelConfigurationAPITest 设置测试环境
func setupModelConfigurationAPITest(t *testing.T) *TestModelConfigurationAPI {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "加载配置失败")

	// 初始化日志
	log := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 连接测试数据库
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	require.NoError(t, err, "连接数据库失败")

	// 清理测试数据
	cleanupModelConfigTestData(t, db)

	// 创建测试数据
	test := &TestModelConfigurationAPI{
		db:  db,
		log: log,
	}

	// 创建租户
	test.tenant1 = createTestTenant(t, db, "测试租户1")
	test.tenant2 = createTestTenant(t, db, "测试租户2")

	// 创建用户
	test.systemAdmin = createTestUser(t, db, test.tenant1.ID, "system@test.com", model.RoleSystemAdmin)
	test.tenantAdmin1 = createTestUser(t, db, test.tenant1.ID, "admin1@test.com", model.RoleTenantAdmin)
	test.tenantAdmin2 = createTestUser(t, db, test.tenant2.ID, "admin2@test.com", model.RoleTenantAdmin)

	// 创建模型配置
	test.config1 = createTestModelConfiguration(t, db, test.tenant1.ID, test.tenantAdmin1.ID, "测试配置1", "openai")
	test.config2 = createTestModelConfiguration(t, db, test.tenant2.ID, test.tenantAdmin2.ID, "测试配置2", "anthropic")

	// 初始化路由
	test.router = api.SetupRouter(cfg, db, log)

	return test
}

// teardownModelConfigurationAPITest 清理测试环境
func teardownModelConfigurationAPITest(t *testing.T, test *TestModelConfigurationAPI) {
	cleanupModelConfigTestData(t, test.db)
}

// TestCreateModelConfiguration_TenantAdmin 测试租户管理员创建模型配置
func TestCreateModelConfiguration_TenantAdmin(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 准备请求
	reqBody := map[string]interface{}{
		"name":          "新配置",
		"model":         "gpt-4",
		"modelProvider": "openai",
		"apiKey":        "sk-test123456789",
	}

	// 发送请求
	resp := test.makeRequest(t, "POST", "/api/v1/model-configurations", reqBody, test.tenantAdmin1.Token)

	// 验证响应
	assert.Equal(t, http.StatusCreated, resp.Code, "应该返回201状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	assert.NotNil(t, result["data"], "应该包含数据")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "新配置", data["name"], "配置名称应该匹配")
	assert.Equal(t, test.tenant1.ID.String(), data["tenantId"], "租户ID应该自动设置为当前租户")
	
	// 验证API密钥已脱敏
	apiKey := data["apiKey"].(string)
	assert.True(t, len(apiKey) < 20, "API密钥应该被脱敏")
	assert.Contains(t, apiKey, "****", "API密钥应该包含脱敏标记")
}

// TestCreateModelConfiguration_SystemAdmin 测试平台管理员创建模型配置（指定租户ID）
func TestCreateModelConfiguration_SystemAdmin(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 准备请求（平台管理员指定租户ID）
	reqBody := map[string]interface{}{
		"tenantId":      test.tenant2.ID.String(),
		"name":          "平台管理员创建的配置",
		"model":         "claude-3-opus",
		"modelProvider": "anthropic",
		"apiKey":        "sk-ant-test123",
	}

	// 发送请求
	resp := test.makeRequest(t, "POST", "/api/v1/model-configurations", reqBody, test.systemAdmin.Token)

	// 验证响应
	assert.Equal(t, http.StatusCreated, resp.Code, "应该返回201状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, test.tenant2.ID.String(), data["tenantId"], "租户ID应该匹配指定的租户")
}

// TestCreateModelConfiguration_TenantAdminCrossTenant 测试租户管理员尝试在其他租户创建配置（应失败）
func TestCreateModelConfiguration_TenantAdminCrossTenant(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 租户1的管理员尝试在租户2创建配置
	reqBody := map[string]interface{}{
		"tenantId":      test.tenant2.ID.String(),
		"name":          "跨租户配置",
		"model":         "gpt-4",
		"modelProvider": "openai",
		"apiKey":        "sk-test123",
	}

	resp := test.makeRequest(t, "POST", "/api/v1/model-configurations", reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestGetModelConfiguration_Success 测试获取模型配置成功
func TestGetModelConfiguration_Success(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String())
	resp := test.makeRequest(t, "GET", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, test.config1.ID.String(), data["id"], "配置ID应该匹配")
	
	// 验证API密钥已脱敏
	apiKey := data["apiKey"].(string)
	assert.Contains(t, apiKey, "****", "API密钥应该被脱敏")
}

// TestGetModelConfiguration_CrossTenantAccess 测试租户管理员尝试访问其他租户配置（应返回403）
func TestGetModelConfiguration_CrossTenantAccess(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 租户1的管理员尝试访问租户2的配置
	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config2.ID.String())
	resp := test.makeRequest(t, "GET", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestGetModelConfiguration_SystemAdminCrossTenant 测试平台管理员跨租户访问（应成功）
func TestGetModelConfiguration_SystemAdminCrossTenant(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 平台管理员访问租户2的配置
	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config2.ID.String())
	resp := test.makeRequest(t, "GET", url, nil, test.systemAdmin.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "平台管理员应该可以访问所有租户的配置")
}

// TestListModelConfigurations_TenantAdmin 测试租户管理员查询配置列表
func TestListModelConfigurations_TenantAdmin(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	resp := test.makeRequest(t, "GET", "/api/v1/model-configurations?pageNo=1&pageSize=10", nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	configs := data["data"].([]interface{})
	
	// 租户管理员只能看到自己租户的配置
	assert.Equal(t, 1, len(configs), "应该只返回当前租户的配置")
	
	config := configs[0].(map[string]interface{})
	assert.Equal(t, test.tenant1.ID.String(), config["tenantId"], "配置应该属于当前租户")
}

// TestListModelConfigurations_SystemAdmin 测试平台管理员查询所有配置
func TestListModelConfigurations_SystemAdmin(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	resp := test.makeRequest(t, "GET", "/api/v1/model-configurations?pageNo=1&pageSize=10", nil, test.systemAdmin.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	configs := data["data"].([]interface{})
	
	// 平台管理员可以看到所有租户的配置
	assert.GreaterOrEqual(t, len(configs), 2, "应该返回所有租户的配置")
}

// TestUpdateModelConfiguration_Success 测试更新模型配置成功
func TestUpdateModelConfiguration_Success(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	reqBody := map[string]interface{}{
		"name":  "更新后的配置名称",
		"model": "gpt-4-turbo",
	}

	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String())
	resp := test.makeRequest(t, "PUT", url, reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "更新后的配置名称", data["name"], "配置名称应该已更新")
	assert.Equal(t, "gpt-4-turbo", data["model"], "模型应该已更新")
}

// TestUpdateModelConfiguration_CrossTenantAccess 测试租户管理员尝试更新其他租户配置（应失败）
func TestUpdateModelConfiguration_CrossTenantAccess(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	reqBody := map[string]interface{}{
		"name": "尝试更新",
	}

	// 租户1的管理员尝试更新租户2的配置
	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config2.ID.String())
	resp := test.makeRequest(t, "PUT", url, reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestUpdateStatus_EnableDisable 测试启用/禁用功能
func TestUpdateStatus_EnableDisable(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 禁用配置
	reqBody := map[string]interface{}{
		"status": "disabled",
	}

	url := fmt.Sprintf("/api/v1/model-configurations/%s/status", test.config1.ID.String())
	resp := test.makeRequest(t, "PATCH", url, reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	// 验证配置已禁用
	getURL := fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String())
	getResp := test.makeRequest(t, "GET", getURL, nil, test.tenantAdmin1.Token)

	var result map[string]interface{}
	err := json.Unmarshal(getResp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, false, data["isEnabled"], "配置应该已禁用")

	// 重新启用配置
	reqBody["status"] = "enabled"
	resp = test.makeRequest(t, "PATCH", url, reqBody, test.tenantAdmin1.Token)
	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")
}

// TestDeleteModelConfiguration_SoftDelete 测试软删除功能
func TestDeleteModelConfiguration_SoftDelete(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 删除配置
	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String())
	resp := test.makeRequest(t, "DELETE", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	// 验证配置已被软删除（无法再获取）
	getResp := test.makeRequest(t, "GET", url, nil, test.tenantAdmin1.Token)
	assert.Equal(t, http.StatusForbidden, resp.Code, "已删除的配置应该无法访问")

	// 验证数据库中记录仍存在但标记为已删除
	var config model.ModelConfiguration
	err := test.db.Unscoped().Where("id = ?", test.config1.ID).First(&config).Error
	require.NoError(t, err, "应该能在数据库中找到记录")
	assert.True(t, config.IsDeleted, "is_deleted应该为true")
	assert.NotNil(t, config.DeletedAt, "deleted_at应该有值")
	assert.NotNil(t, config.DeletedBy, "deleted_by应该有值")
}

// TestDeleteModelConfiguration_CrossTenantAccess 测试租户管理员尝试删除其他租户配置（应失败）
func TestDeleteModelConfiguration_CrossTenantAccess(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 租户1的管理员尝试删除租户2的配置
	url := fmt.Sprintf("/api/v1/model-configurations/%s", test.config2.ID.String())
	resp := test.makeRequest(t, "DELETE", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestListAvailable_OnlyEnabledAndNotDeleted 测试可用模型列表（仅返回已启用且未删除的配置）
func TestListAvailable_OnlyEnabledAndNotDeleted(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 创建多个配置：启用、禁用、已删除
	enabledConfig := createTestModelConfiguration(t, test.db, test.tenant1.ID, test.tenantAdmin1.ID, "启用配置", "openai")
	
	disabledConfig := createTestModelConfiguration(t, test.db, test.tenant1.ID, test.tenantAdmin1.ID, "禁用配置", "anthropic")
	test.db.Model(&model.ModelConfiguration{}).Where("id = ?", disabledConfig.ID).Update("is_enabled", false)
	
	deletedConfig := createTestModelConfiguration(t, test.db, test.tenant1.ID, test.tenantAdmin1.ID, "已删除配置", "googlegenai")
	test.db.Model(&model.ModelConfiguration{}).Where("id = ?", deletedConfig.ID).Updates(map[string]interface{}{
		"is_deleted": true,
		"deleted_at": time.Now(),
		"deleted_by": test.tenantAdmin1.ID,
	})

	// 查询可用配置列表
	resp := test.makeRequest(t, "GET", "/api/v1/model-configurations/available", nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].([]interface{})
	
	// 应该只返回启用且未删除的配置
	foundEnabled := false
	for _, item := range data {
		config := item.(map[string]interface{})
		if config["id"] == enabledConfig.ID.String() {
			foundEnabled = true
		}
		// 不应该包含禁用或已删除的配置
		assert.NotEqual(t, disabledConfig.ID.String(), config["id"], "不应该包含禁用的配置")
		assert.NotEqual(t, deletedConfig.ID.String(), config["id"], "不应该包含已删除的配置")
		
		// 验证不包含敏感信息
		_, hasAPIKey := config["apiKey"]
		assert.False(t, hasAPIKey, "可用列表不应该包含API密钥")
	}
	
	assert.True(t, foundEnabled, "应该包含启用的配置")
}

// TestValidateModelConfiguration_Success 测试验证模型配置（模拟成功）
func TestValidateModelConfiguration_Success(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 注意：实际验证需要真实的API密钥，这里只测试接口调用
	url := fmt.Sprintf("/api/v1/model-configurations/%s/validate", test.config1.ID.String())
	resp := test.makeRequest(t, "POST", url, nil, test.tenantAdmin1.Token)

	// 由于没有真实的API密钥，验证会失败，但接口应该正常响应
	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["valid"], "应该包含valid字段")
	assert.NotNil(t, data["message"], "应该包含message字段")
}

// TestValidateModelConfiguration_CrossTenantAccess 测试租户管理员尝试验证其他租户配置（应失败）
func TestValidateModelConfiguration_CrossTenantAccess(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 租户1的管理员尝试验证租户2的配置
	url := fmt.Sprintf("/api/v1/model-configurations/%s/validate", test.config2.ID.String())
	resp := test.makeRequest(t, "POST", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestAPIKeyMasking 测试API密钥脱敏
func TestAPIKeyMasking(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	// 创建配置时使用完整的API密钥
	reqBody := map[string]interface{}{
		"name":          "密钥脱敏测试",
		"model":         "gpt-4",
		"modelProvider": "openai",
		"apiKey":        "sk-1234567890abcdefghijklmnopqrstuvwxyz",
	}

	resp := test.makeRequest(t, "POST", "/api/v1/model-configurations", reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusCreated, resp.Code, "应该返回201状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	maskedKey := data["apiKey"].(string)
	
	// 验证密钥已脱敏
	assert.NotEqual(t, "sk-1234567890abcdefghijklmnopqrstuvwxyz", maskedKey, "API密钥不应该是原始值")
	assert.Contains(t, maskedKey, "****", "脱敏后的密钥应该包含****")
	assert.True(t, len(maskedKey) < len("sk-1234567890abcdefghijklmnopqrstuvwxyz"), "脱敏后的密钥应该更短")
}

// TestUnauthorizedAccess 测试未认证访问
func TestUnauthorizedAccess(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	testCases := []struct {
		name   string
		method string
		path   string
		body   map[string]interface{}
	}{
		{
			name:   "创建配置",
			method: "POST",
			path:   "/api/v1/model-configurations",
			body: map[string]interface{}{
				"name":          "测试",
				"model":         "gpt-4",
				"modelProvider": "openai",
				"apiKey":        "sk-test",
			},
		},
		{
			name:   "获取配置列表",
			method: "GET",
			path:   "/api/v1/model-configurations",
			body:   nil,
		},
		{
			name:   "获取配置详情",
			method: "GET",
			path:   fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String()),
			body:   nil,
		},
		{
			name:   "更新配置",
			method: "PUT",
			path:   fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String()),
			body:   map[string]interface{}{"name": "更新"},
		},
		{
			name:   "删除配置",
			method: "DELETE",
			path:   fmt.Sprintf("/api/v1/model-configurations/%s", test.config1.ID.String()),
			body:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 不提供Token
			resp := test.makeRequest(t, tc.method, tc.path, tc.body, "")
			assert.Equal(t, http.StatusUnauthorized, resp.Code, "应该返回401状态码")
		})
	}
}

// TestValidationErrors 测试参数验证错误
func TestValidationErrors(t *testing.T) {
	test := setupModelConfigurationAPITest(t)
	defer teardownModelConfigurationAPITest(t, test)

	testCases := []struct {
		name     string
		reqBody  map[string]interface{}
		expected int
	}{
		{
			name: "缺少name字段",
			reqBody: map[string]interface{}{
				"model":         "gpt-4",
				"modelProvider": "openai",
				"apiKey":        "sk-test",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "缺少model字段",
			reqBody: map[string]interface{}{
				"name":          "测试配置",
				"modelProvider": "openai",
				"apiKey":        "sk-test",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "缺少modelProvider字段",
			reqBody: map[string]interface{}{
				"name":   "测试配置",
				"model":  "gpt-4",
				"apiKey": "sk-test",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "缺少apiKey字段",
			reqBody: map[string]interface{}{
				"name":          "测试配置",
				"model":         "gpt-4",
				"modelProvider": "openai",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "无效的modelProvider",
			reqBody: map[string]interface{}{
				"name":          "测试配置",
				"model":         "gpt-4",
				"modelProvider": "invalid_provider",
				"apiKey":        "sk-test",
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.makeRequest(t, "POST", "/api/v1/model-configurations", tc.reqBody, test.tenantAdmin1.Token)
			assert.Equal(t, tc.expected, resp.Code, tc.name)
		})
	}
}

// makeRequest 发送HTTP请求
func (test *TestModelConfigurationAPI) makeRequest(t *testing.T, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err, "序列化请求体失败")
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		
		// 添加上下文信息（模拟中间件）
		ctx := req.Context()
		
		// 根据token确定用户信息
		var userID, tenantID, role string
		switch token {
		case test.systemAdmin.Token:
			userID = test.systemAdmin.ID.String()
			tenantID = test.systemAdmin.TenantID.String()
			role = test.systemAdmin.Role
		case test.tenantAdmin1.Token:
			userID = test.tenantAdmin1.ID.String()
			tenantID = test.tenantAdmin1.TenantID.String()
			role = test.tenantAdmin1.Role
		case test.tenantAdmin2.Token:
			userID = test.tenantAdmin2.ID.String()
			tenantID = test.tenantAdmin2.TenantID.String()
			role = test.tenantAdmin2.Role
		}
		
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, middleware.ContextKeyTenantID, tenantID)
		ctx = context.WithValue(ctx, "traceId", uuid.New().String())
		req = req.WithContext(ctx)
	}

	recorder := httptest.NewRecorder()
	test.router.ServeHTTP(recorder, req)

	return recorder
}

// 辅助函数

func cleanupModelConfigTestData(t *testing.T, db *gorm.DB) {
	// 清理测试数据（按依赖顺序）
	db.Exec("DELETE FROM model_configurations WHERE 1=1")
	db.Exec("DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@test.com')")
	db.Exec("DELETE FROM users WHERE email LIKE '%@test.com'")
	db.Exec("DELETE FROM tenants WHERE name LIKE '测试租户%'")
}

func createTestModelConfiguration(t *testing.T, db *gorm.DB, tenantID, createdBy uuid.UUID, name, provider string) *model.ModelConfiguration {
	config := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          name,
		Model:         "test-model",
		ModelProvider: provider,
		APIKey:        "encrypted_test_key_" + uuid.New().String(),
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
	err := db.Create(config).Error
	require.NoError(t, err, "创建测试模型配置失败")
	return config
}
