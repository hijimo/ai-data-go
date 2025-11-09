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

// TestContextAPI 上下文管理API端到端测试套件
type TestContextAPI struct {
	db            *gorm.DB
	router        http.Handler
	log           logger.Logger
	systemAdmin   *testUser
	tenantAdmin1  *testUser
	tenantAdmin2  *testUser
	tenant1       *model.Tenant
	tenant2       *model.Tenant
	session1      *model.ConversationSession
	session2      *model.ConversationSession
}

// testUser 测试用户
type testUser struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Email    string
	Role     string
	Token    string
}

// setupContextAPITest 设置测试环境
func setupContextAPITest(t *testing.T) *TestContextAPI {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "加载配置失败")

	// 初始化日志
	log := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 连接测试数据库
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	require.NoError(t, err, "连接数据库失败")

	// 清理测试数据
	cleanupTestData(t, db)

	// 创建测试数据
	test := &TestContextAPI{
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

	// 创建会话
	test.session1 = createTestSession(t, db, test.tenant1.ID, test.tenantAdmin1.ID, "测试会话1")
	test.session2 = createTestSession(t, db, test.tenant2.ID, test.tenantAdmin2.ID, "测试会话2")

	// 创建上下文配置
	createTestContextConfig(t, db, test.tenant1.ID, test.session1.ID)
	createTestContextConfig(t, db, test.tenant2.ID, test.session2.ID)

	// 初始化路由
	test.router = api.SetupRouter(cfg, db, log)

	return test
}

// teardownContextAPITest 清理测试环境
func teardownContextAPITest(t *testing.T, test *TestContextAPI) {
	cleanupTestData(t, test.db)
}

// TestBuildContext_Success 测试构建上下文成功
func TestBuildContext_Success(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	// 准备请求
	reqBody := map[string]interface{}{
		"sessionId":       test.session1.ID.String(),
		"userQuery":       "这是一个测试查询",
		"maxTokens":       4000,
		"strategy":        "auto",
		"includeSummary":  true,
		"includeLongTerm": true,
		"shortTermWindow": 10,
	}

	// 发送请求
	resp := test.makeRequest(t, "POST", "/api/v1/contexts/build", reqBody, test.tenantAdmin1.Token)

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	assert.NotNil(t, result["data"], "应该包含数据")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, test.session1.ID.String(), data["sessionId"], "会话ID应该匹配")
	assert.NotNil(t, data["shortTermMessages"], "应该包含短期消息")
	assert.NotNil(t, data["totalTokens"], "应该包含总Token数")
}

// TestBuildContext_Unauthorized 测试未认证访问
func TestBuildContext_Unauthorized(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId": test.session1.ID.String(),
		"userQuery": "测试查询",
	}

	// 不提供Token
	resp := test.makeRequest(t, "POST", "/api/v1/contexts/build", reqBody, "")

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "应该返回401状态码")
}

// TestBuildContext_CrossTenantAccess 测试跨租户访问
func TestBuildContext_CrossTenantAccess(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	// 租户1的管理员尝试访问租户2的会话
	reqBody := map[string]interface{}{
		"sessionId": test.session2.ID.String(),
		"userQuery": "测试查询",
	}

	resp := test.makeRequest(t, "POST", "/api/v1/contexts/build", reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestBuildContext_SystemAdminAccess 测试平台管理员访问所有租户
func TestBuildContext_SystemAdminAccess(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	// 平台管理员访问租户2的会话
	reqBody := map[string]interface{}{
		"sessionId": test.session2.ID.String(),
		"userQuery": "测试查询",
	}

	resp := test.makeRequest(t, "POST", "/api/v1/contexts/build", reqBody, test.systemAdmin.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "平台管理员应该可以访问所有租户的会话")
}

// TestBuildContext_ValidationError 测试参数验证错误
func TestBuildContext_ValidationError(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	testCases := []struct {
		name     string
		reqBody  map[string]interface{}
		expected int
	}{
		{
			name: "缺少会话ID",
			reqBody: map[string]interface{}{
				"userQuery": "测试查询",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "会话ID格式无效",
			reqBody: map[string]interface{}{
				"sessionId": "invalid-uuid",
				"userQuery": "测试查询",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "查询内容为空",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
				"userQuery": "",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "MaxTokens超出范围",
			reqBody: map[string]interface{}{
				"sessionId": test.session1.ID.String(),
				"userQuery": "测试查询",
				"maxTokens": 50000,
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.makeRequest(t, "POST", "/api/v1/contexts/build", tc.reqBody, test.tenantAdmin1.Token)
			assert.Equal(t, tc.expected, resp.Code, tc.name)
		})
	}
}

// TestGetContextConfig_Success 测试获取上下文配置成功
func TestGetContextConfig_Success(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	url := fmt.Sprintf("/api/v1/contexts/%s", test.session1.ID.String())
	resp := test.makeRequest(t, "GET", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	data := result["data"].(map[string]interface{})
	assert.Equal(t, test.session1.ID.String(), data["sessionId"], "会话ID应该匹配")
}

// TestGetContextConfig_NotFound 测试获取不存在的配置
func TestGetContextConfig_NotFound(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	nonExistentID := uuid.New().String()
	url := fmt.Sprintf("/api/v1/contexts/%s", nonExistentID)
	resp := test.makeRequest(t, "GET", url, nil, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusNotFound, resp.Code, "应该返回404状态码")
}

// TestUpdateContextConfig_Success 测试更新上下文配置成功
func TestUpdateContextConfig_Success(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	reqBody := map[string]interface{}{
		"maxTokens":       5000,
		"strategy":        "full",
		"includeSummary":  false,
		"includeLongTerm": true,
		"shortTermWindow": 15,
	}

	url := fmt.Sprintf("/api/v1/contexts/%s", test.session1.ID.String())
	resp := test.makeRequest(t, "PUT", url, reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	data := result["data"].(map[string]interface{})
	assert.Equal(t, float64(5000), data["maxTokens"], "MaxTokens应该已更新")
	assert.Equal(t, "full", data["strategy"], "Strategy应该已更新")
	assert.Equal(t, false, data["includeSummary"], "IncludeSummary应该已更新")
}

// TestUpdateContextConfig_CrossTenantAccess 测试跨租户更新配置
func TestUpdateContextConfig_CrossTenantAccess(t *testing.T) {
	test := setupContextAPITest(t)
	defer teardownContextAPITest(t, test)

	reqBody := map[string]interface{}{
		"maxTokens": 5000,
	}

	// 租户1的管理员尝试更新租户2的配置
	url := fmt.Sprintf("/api/v1/contexts/%s", test.session2.ID.String())
	resp := test.makeRequest(t, "PUT", url, reqBody, test.tenantAdmin1.Token)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// makeRequest 发送HTTP请求
func (test *TestContextAPI) makeRequest(t *testing.T, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
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
	}

	// 添加上下文信息（模拟中间件）
	if token != "" {
		ctx := req.Context()
		ctx = context.WithValue(ctx, middleware.ContextKeyUserID, test.tenantAdmin1.ID.String())
		ctx = context.WithValue(ctx, middleware.ContextKeyTenantID, test.tenantAdmin1.TenantID.String())
		ctx = context.WithValue(ctx, "traceId", uuid.New().String())
		req = req.WithContext(ctx)
	}

	recorder := httptest.NewRecorder()
	test.router.ServeHTTP(recorder, req)

	return recorder
}

// 辅助函数

func cleanupTestData(t *testing.T, db *gorm.DB) {
	// 清理测试数据（按依赖顺序）
	db.Exec("DELETE FROM conversation_contexts WHERE 1=1")
	db.Exec("DELETE FROM conversation_sessions WHERE 1=1")
	db.Exec("DELETE FROM users WHERE email LIKE '%@test.com'")
	db.Exec("DELETE FROM tenants WHERE name LIKE '测试租户%'")
}

func createTestTenant(t *testing.T, db *gorm.DB, name string) *model.Tenant {
	tenant := &model.Tenant{
		ID:     uuid.New(),
		Name:   name,
		Domain: fmt.Sprintf("%s.test.com", name),
		Status: "active",
	}
	err := db.Create(tenant).Error
	require.NoError(t, err, "创建测试租户失败")
	return tenant
}

func createTestUser(t *testing.T, db *gorm.DB, tenantID uuid.UUID, email, role string) *testUser {
	user := &model.User{
		ID:       uuid.New(),
		TenantID: tenantID,
		Email:    email,
		Username: email,
		Password: "hashed_password",
		Status:   "active",
	}
	err := db.Create(user).Error
	require.NoError(t, err, "创建测试用户失败")

	// 创建用户角色
	userRole := &model.UserRole{
		UserID: user.ID,
		Role:   role,
	}
	err = db.Create(userRole).Error
	require.NoError(t, err, "创建用户角色失败")

	// 生成测试Token（简化版）
	token := fmt.Sprintf("test_token_%s", user.ID.String())

	return &testUser{
		ID:       user.ID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		Token:    token,
	}
}

func createTestSession(t *testing.T, db *gorm.DB, tenantID, userID uuid.UUID, title string) *model.ConversationSession {
	session := &model.ConversationSession{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Title:     title,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := db.Create(session).Error
	require.NoError(t, err, "创建测试会话失败")
	return session
}

func createTestContextConfig(t *testing.T, db *gorm.DB, tenantID, sessionID uuid.UUID) *model.ConversationContext {
	config := &model.ConversationContext{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
		TotalMessages:   0,
		TotalTokensUsed: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err := db.Create(config).Error
	require.NoError(t, err, "创建测试上下文配置失败")
	return config
}
