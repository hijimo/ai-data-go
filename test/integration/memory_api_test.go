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

// TestMemoryAPI 记忆管理API端到端测试套件
type TestMemoryAPI struct {
	db           *gorm.DB
	router       http.Handler
	log          logger.Logger
	systemAdmin  *testUser
	tenantAdmin1 *testUser
	tenantAdmin2 *testUser
	tenant1      *model.Tenant
	tenant2      *model.Tenant
	session1     *model.ConversationSession
	session2     *model.ConversationSession
	memory1      *model.ConversationMemory
	memory2      *model.ConversationMemory
}

// setupMemoryAPITest 设置测试环境
func setupMemoryAPITest(t *testing.T) *TestMemoryAPI {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "加载配置失败")

	// 初始化日志
	log := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 连接测试数据库
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	require.NoError(t, err, "连接数据库失败")

	// 清理测试数据
	cleanupMemoryTestData(t, db)

	// 创建测试数据
	test := &TestMemoryAPI{
		db:  db,
		log: log,
	}

	// 创建租户
	test.tenant1 = createTestTenant(t, db, "记忆测试租户1")
	test.tenant2 = createTestTenant(t, db, "记忆测试租户2")

	// 创建用户
	test.systemAdmin = createTestUser(t, db, test.tenant1.ID, "system@memory.test", model.RoleSystemAdmin)
	test.tenantAdmin1 = createTestUser(t, db, test.tenant1.ID, "admin1@memory.test", model.RoleTenantAdmin)
	test.tenantAdmin2 = createTestUser(t, db, test.tenant2.ID, "admin2@memory.test", model.RoleTenantAdmin)

	// 创建会话
	test.session1 = createTestSession(t, db, test.tenant1.ID, test.tenantAdmin1.ID, "记忆测试会话1")
	test.session2 = createTestSession(t, db, test.tenant2.ID, test.tenantAdmin2.ID, "记忆测试会话2")

	// 创建记忆
	test.memory1 = createTestMemory(t, db, test.tenant1.ID, test.session1.ID, "测试记忆1")
	test.memory2 = createTestMemory(t, db, test.tenant2.ID, test.session2.ID, "测试记忆2")

	// 初始化路由
	test.router = api.SetupRouter(cfg, db, log)

	return test
}

// teardownMemoryAPITest 清理测试环境
func teardownMemoryAPITest(t *testing.T, test *TestMemoryAPI) {
	cleanupMemoryTestData(t, test.db)
}

// TestSearchMemories_Success 测试检索记忆成功
func TestSearchMemories_Success(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId":            test.session1.ID.String(),
		"query":                "测试查询",
		"topK":                 5,
		"minSimilarity":        0.7,
		"timeRangeDays":        30,
		"memoryTypes":          []string{"fact", "context"},
		"includeCrossSessions": false,
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/search", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	assert.NotNil(t, result["data"], "应该包含数据")

	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["results"], "应该包含检索结果")
	assert.NotNil(t, data["totalCount"], "应该包含总数")
	assert.NotNil(t, data["searchTime"], "应该包含搜索时间")
}

// TestSearchMemories_Unauthorized 测试未认证访问
func TestSearchMemories_Unauthorized(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId": test.session1.ID.String(),
		"query":     "测试查询",
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/search", reqBody, nil)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "应该返回401状态码")
}

// TestSearchMemories_CrossTenantAccess 测试跨租户访问
func TestSearchMemories_CrossTenantAccess(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	// 租户1的管理员尝试检索租户2的记忆
	reqBody := map[string]interface{}{
		"sessionId": test.session2.ID.String(),
		"query":     "测试查询",
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/search", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestSearchMemories_SystemAdminAccess 测试平台管理员访问所有租户
func TestSearchMemories_SystemAdminAccess(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	// 平台管理员检索租户2的记忆
	reqBody := map[string]interface{}{
		"sessionId": test.session2.ID.String(),
		"query":     "测试查询",
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/search", reqBody, test.systemAdmin)

	assert.Equal(t, http.StatusOK, resp.Code, "平台管理员应该可以访问所有租户的记忆")
}

// TestSearchMemories_ValidationError 测试参数验证错误
func TestSearchMemories_ValidationError(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	testCases := []struct {
		name     string
		reqBody  map[string]interface{}
		expected int
	}{
		{
			name: "缺少会话ID",
			reqBody: map[string]interface{}{
				"query": "测试查询",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "会话ID格式无效",
			reqBody: map[string]interface{}{
				"sessionId": "invalid-uuid",
				"query":     "测试查询",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "查询内容为空",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
				"query":     "",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "TopK超出范围",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
				"query":     "测试查询",
				"topK":      100,
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/search", tc.reqBody, test.tenantAdmin1)
			assert.Equal(t, tc.expected, resp.Code, tc.name)
		})
	}
}

// TestStoreMemory_Success 测试存储记忆成功
func TestStoreMemory_Success(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId":      test.session1.ID.String(),
		"messageIds":     []string{},
		"memoryType":     "fact",
		"content":        "这是一个新的测试记忆",
		"importance":     0.8,
		"expirationDays": 90,
		"metadata": map[string]interface{}{
			"source": "test",
		},
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"], "应该返回记忆ID")
	assert.Equal(t, "fact", data["memoryType"], "记忆类型应该匹配")
	assert.Equal(t, 0.8, data["importance"], "重要性应该匹配")
}

// TestStoreMemory_CrossTenantAccess 测试跨租户存储记忆
func TestStoreMemory_CrossTenantAccess(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	// 租户1的管理员尝试在租户2的会话中存储记忆
	reqBody := map[string]interface{}{
		"sessionId":  test.session2.ID.String(),
		"memoryType": "fact",
		"content":    "测试记忆",
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestCleanupMemories_Success 测试清理记忆成功
func TestCleanupMemories_Success(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId": test.session1.ID.String(),
		"strategy":  "expired",
		"mode":      "soft",
		"batchSize": 100,
		"execute":   false, // 预览模式
	}

	resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/cleanup", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["cleanedCount"], "应该包含清理数量")
	assert.Equal(t, true, data["preview"], "应该是预览模式")
	assert.Equal(t, "expired", data["strategy"], "策略应该匹配")
}

// TestCleanupMemories_ValidationError 测试清理记忆参数验证
func TestCleanupMemories_ValidationError(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	testCases := []struct {
		name     string
		reqBody  map[string]interface{}
		expected int
	}{
		{
			name: "缺少策略",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "无效的策略",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
				"strategy":  "invalid",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "无效的模式",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
				"strategy":  "expired",
				"mode":      "invalid",
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.makeMemoryRequest(t, "POST", "/api/v1/memories/cleanup", tc.reqBody, test.tenantAdmin1)
			assert.Equal(t, tc.expected, resp.Code, tc.name)
		})
	}
}

// TestGetMemory_NotImplemented 测试获取记忆详情（未实现）
func TestGetMemory_NotImplemented(t *testing.T) {
	test := setupMemoryAPITest(t)
	defer teardownMemoryAPITest(t, test)

	url := fmt.Sprintf("/api/v1/memories/%s", test.memory1.ID.String())
	resp := test.makeMemoryRequest(t, "GET", url, nil, test.tenantAdmin1)

	// 由于功能未完全实现，应该返回503
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code, "应该返回503状态码")
}

// makeMemoryRequest 发送HTTP请求
func (test *TestMemoryAPI) makeMemoryRequest(t *testing.T, method, path string, body interface{}, user *testUser) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err, "序列化请求体失败")
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	if user != nil {
		req.Header.Set("Authorization", "Bearer "+user.Token)

		// 添加上下文信息（模拟中间件）
		ctx := req.Context()
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, user.ID.String())
		ctx = context.WithValue(ctx, middleware.ContextKeyTenantID, user.TenantID.String())
		ctx = context.WithValue(ctx, "traceId", uuid.New().String())
		req = req.WithContext(ctx)
	}

	recorder := httptest.NewRecorder()
	test.router.ServeHTTP(recorder, req)

	return recorder
}

// 辅助函数

func cleanupMemoryTestData(t *testing.T, db *gorm.DB) {
	// 清理测试数据（按依赖顺序）
	db.Exec("DELETE FROM conversation_memories WHERE 1=1")
	db.Exec("DELETE FROM conversation_sessions WHERE 1=1")
	db.Exec("DELETE FROM users WHERE email LIKE '%@memory.test'")
	db.Exec("DELETE FROM tenants WHERE name LIKE '记忆测试租户%'")
}

func createTestMemory(t *testing.T, db *gorm.DB, tenantID, sessionID uuid.UUID, content string) *model.ConversationMemory {
	memory := &model.ConversationMemory{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SessionID:   sessionID,
		MemoryType:  "fact",
		Content:     content,
		TokenCount:  10,
		Importance:  0.5,
		AccessCount: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := db.Create(memory).Error
	require.NoError(t, err, "创建测试记忆失败")
	return memory
}
