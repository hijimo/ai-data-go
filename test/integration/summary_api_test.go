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

// TestSummaryAPI 摘要管理API端到端测试套件
type TestSummaryAPI struct {
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
	summary1     *model.ConversationSummary
	summary2     *model.ConversationSummary
}

// setupSummaryAPITest 设置测试环境
func setupSummaryAPITest(t *testing.T) *TestSummaryAPI {
	// 加载配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "加载配置失败")

	// 初始化日志
	log := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 连接测试数据库
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	require.NoError(t, err, "连接数据库失败")

	// 清理测试数据
	cleanupSummaryTestData(t, db)

	// 创建测试数据
	test := &TestSummaryAPI{
		db:  db,
		log: log,
	}

	// 创建租户
	test.tenant1 = createTestTenant(t, db, "摘要测试租户1")
	test.tenant2 = createTestTenant(t, db, "摘要测试租户2")

	// 创建用户
	test.systemAdmin = createTestUser(t, db, test.tenant1.ID, "system@summary.test", model.RoleSystemAdmin)
	test.tenantAdmin1 = createTestUser(t, db, test.tenant1.ID, "admin1@summary.test", model.RoleTenantAdmin)
	test.tenantAdmin2 = createTestUser(t, db, test.tenant2.ID, "admin2@summary.test", model.RoleTenantAdmin)

	// 创建会话
	test.session1 = createTestSession(t, db, test.tenant1.ID, test.tenantAdmin1.ID, "摘要测试会话1")
	test.session2 = createTestSession(t, db, test.tenant2.ID, test.tenantAdmin2.ID, "摘要测试会话2")

	// 创建摘要
	test.summary1 = createTestSummary(t, db, test.tenant1.ID, test.session1.ID, "测试摘要1")
	test.summary2 = createTestSummary(t, db, test.tenant2.ID, test.session2.ID, "测试摘要2")

	// 初始化路由
	test.router = api.SetupRouter(cfg, db, log)

	return test
}

// teardownSummaryAPITest 清理测试环境
func teardownSummaryAPITest(t *testing.T, test *TestSummaryAPI) {
	cleanupSummaryTestData(t, test.db)
}

// TestGenerateSummary_Success 测试生成摘要成功
func TestGenerateSummary_Success(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId":       test.session1.ID.String(),
		"messageIds":      []string{},
		"summaryType":     "incremental",
		"targetLength":    500,
		"previousSummary": "",
	}

	resp := test.makeSummaryRequest(t, "POST", "/api/v1/summaries", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	assert.NotNil(t, result["data"], "应该包含数据")

	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["id"], "应该返回摘要ID")
	assert.Equal(t, test.session1.ID.String(), data["sessionId"], "会话ID应该匹配")
	assert.Equal(t, "incremental", data["summaryType"], "摘要类型应该匹配")
	assert.NotNil(t, data["content"], "应该包含摘要内容")
}

// TestGenerateSummary_Unauthorized 测试未认证访问
func TestGenerateSummary_Unauthorized(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	reqBody := map[string]interface{}{
		"sessionId":   test.session1.ID.String(),
		"summaryType": "full",
	}

	resp := test.makeSummaryRequest(t, "POST", "/api/v1/summaries", reqBody, nil)

	assert.Equal(t, http.StatusUnauthorized, resp.Code, "应该返回401状态码")
}

// TestGenerateSummary_CrossTenantAccess 测试跨租户访问
func TestGenerateSummary_CrossTenantAccess(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	// 租户1的管理员尝试为租户2的会话生成摘要
	reqBody := map[string]interface{}{
		"sessionId":   test.session2.ID.String(),
		"summaryType": "full",
	}

	resp := test.makeSummaryRequest(t, "POST", "/api/v1/summaries", reqBody, test.tenantAdmin1)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestGenerateSummary_SystemAdminAccess 测试平台管理员访问所有租户
func TestGenerateSummary_SystemAdminAccess(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	// 平台管理员为租户2的会话生成摘要
	reqBody := map[string]interface{}{
		"sessionId":   test.session2.ID.String(),
		"summaryType": "full",
	}

	resp := test.makeSummaryRequest(t, "POST", "/api/v1/summaries", reqBody, test.systemAdmin)

	assert.Equal(t, http.StatusOK, resp.Code, "平台管理员应该可以访问所有租户的会话")
}

// TestGenerateSummary_ValidationError 测试参数验证错误
func TestGenerateSummary_ValidationError(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	testCases := []struct {
		name     string
		reqBody  map[string]interface{}
		expected int
	}{
		{
			name: "缺少会话ID",
			reqBody: map[string]interface{}{
				"summaryType": "full",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "会话ID格式无效",
			reqBody: map[string]interface{}{
				"sessionId":   "invalid-uuid",
				"summaryType": "full",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "无效的摘要类型",
			reqBody: map[string]interface{}{
				"sessionId":   test.session1.ID.String(),
				"summaryType": "invalid",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "目标长度超出范围",
			reqBody: map[string]interface{}{
				"sessionId":    test.session1.ID.String(),
				"summaryType":  "full",
				"targetLength": 5000,
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.makeSummaryRequest(t, "POST", "/api/v1/summaries", tc.reqBody, test.tenantAdmin1)
			assert.Equal(t, tc.expected, resp.Code, tc.name)
		})
	}
}

// TestGetSummary_Success 测试获取摘要详情成功
func TestGetSummary_Success(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	url := fmt.Sprintf("/api/v1/summaries/%s", test.summary1.ID.String())
	resp := test.makeSummaryRequest(t, "GET", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	data := result["data"].(map[string]interface{})
	assert.Equal(t, test.summary1.ID.String(), data["id"], "摘要ID应该匹配")
	assert.Equal(t, test.session1.ID.String(), data["sessionId"], "会话ID应该匹配")
}

// TestGetSummary_NotFound 测试获取不存在的摘要
func TestGetSummary_NotFound(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	nonExistentID := uuid.New().String()
	url := fmt.Sprintf("/api/v1/summaries/%s", nonExistentID)
	resp := test.makeSummaryRequest(t, "GET", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusNotFound, resp.Code, "应该返回404状态码")
}

// TestGetSummary_CrossTenantAccess 测试跨租户获取摘要
func TestGetSummary_CrossTenantAccess(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	// 租户1的管理员尝试获取租户2的摘要
	url := fmt.Sprintf("/api/v1/summaries/%s", test.summary2.ID.String())
	resp := test.makeSummaryRequest(t, "GET", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestListSummaries_Success 测试获取摘要列表成功
func TestListSummaries_Success(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	url := fmt.Sprintf("/api/v1/sessions/%s/summaries", test.session1.ID.String())
	resp := test.makeSummaryRequest(t, "GET", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["summaries"], "应该包含摘要列表")
	assert.Equal(t, test.session1.ID.String(), data["sessionId"], "会话ID应该匹配")
}

// TestListSummaries_CrossTenantAccess 测试跨租户获取摘要列表
func TestListSummaries_CrossTenantAccess(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	// 租户1的管理员尝试获取租户2的摘要列表
	url := fmt.Sprintf("/api/v1/sessions/%s/summaries", test.session2.ID.String())
	resp := test.makeSummaryRequest(t, "GET", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestCheckTrigger_Success 测试检查摘要触发条件成功
func TestCheckTrigger_Success(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	url := fmt.Sprintf("/api/v1/sessions/%s/summaries/check-trigger", test.session1.ID.String())
	resp := test.makeSummaryRequest(t, "POST", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusOK, resp.Code, "应该返回200状态码")

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err, "解析响应失败")

	assert.Equal(t, float64(errors.CodeSuccess), result["code"], "应该返回成功代码")
	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["shouldSummarize"], "应该包含是否需要摘要的标志")
	assert.NotNil(t, data["triggerReason"], "应该包含触发原因")
	assert.NotNil(t, data["messageCount"], "应该包含消息数量")
}

// TestCheckTrigger_CrossTenantAccess 测试跨租户检查触发条件
func TestCheckTrigger_CrossTenantAccess(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	// 租户1的管理员尝试检查租户2的触发条件
	url := fmt.Sprintf("/api/v1/sessions/%s/summaries/check-trigger", test.session2.ID.String())
	resp := test.makeSummaryRequest(t, "POST", url, nil, test.tenantAdmin1)

	assert.Equal(t, http.StatusForbidden, resp.Code, "应该返回403状态码")
}

// TestSummaryAPI_ErrorResponses 测试错误响应格式
func TestSummaryAPI_ErrorResponses(t *testing.T) {
	test := setupSummaryAPITest(t)
	defer teardownSummaryAPITest(t, test)

	testCases := []struct {
		name         string
		method       string
		url          string
		body         interface{}
		user         *testUser
		expectedCode int
	}{
		{
			name:         "未认证访问",
			method:       "POST",
			url:          "/api/v1/summaries",
			body:         map[string]interface{}{"sessionId": test.session1.ID.String()},
			user:         nil,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "无效的UUID格式",
			method:       "GET",
			url:          "/api/v1/summaries/invalid-uuid",
			body:         nil,
			user:         test.tenantAdmin1,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "资源不存在",
			method:       "GET",
			url:          fmt.Sprintf("/api/v1/summaries/%s", uuid.New().String()),
			body:         nil,
			user:         test.tenantAdmin1,
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.makeSummaryRequest(t, tc.method, tc.url, tc.body, tc.user)
			assert.Equal(t, tc.expectedCode, resp.Code, tc.name)

			// 验证错误响应格式
			var result map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &result)
			require.NoError(t, err, "解析错误响应失败")

			assert.NotEqual(t, float64(errors.CodeSuccess), result["code"], "错误响应不应该返回成功代码")
			assert.NotEmpty(t, result["message"], "错误响应应该包含错误消息")
		})
	}
}

// makeSummaryRequest 发送HTTP请求
func (test *TestSummaryAPI) makeSummaryRequest(t *testing.T, method, path string, body interface{}, user *testUser) *httptest.ResponseRecorder {
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

func cleanupSummaryTestData(t *testing.T, db *gorm.DB) {
	// 清理测试数据（按依赖顺序）
	db.Exec("DELETE FROM conversation_summaries WHERE 1=1")
	db.Exec("DELETE FROM conversation_sessions WHERE 1=1")
	db.Exec("DELETE FROM users WHERE email LIKE '%@summary.test'")
	db.Exec("DELETE FROM tenants WHERE name LIKE '摘要测试租户%'")
}

func createTestSummary(t *testing.T, db *gorm.DB, tenantID, sessionID uuid.UUID, content string) *model.ConversationSummary {
	qualityScore := 0.85
	compressionRate := 0.6

	summary := &model.ConversationSummary{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "incremental",
		Content:         content,
		TokenCount:      100,
		MessageCount:    10,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
		KeyTopics:       []string{"测试", "摘要"},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err := db.Create(summary).Error
	require.NoError(t, err, "创建测试摘要失败")
	return summary
}
