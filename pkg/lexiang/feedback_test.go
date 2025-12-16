// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// ============================================================================
// ListFeedbacks 单元测试
// ============================================================================

// TestListFeedbacks_ResponseParsing 测试 ListFeedbacks 响应解析
// 验证能正确解析反馈列表响应，包含反馈数据和分页信息
// _Requirements: 7.1_
func TestListFeedbacks_ResponseParsing(t *testing.T) {
	// 预期的响应数据
	expectedSpaceID := "space_abc123"
	expectedNextPageToken := "next_page_token_xyz"

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != http.MethodGet {
			t.Errorf("请求方法错误: 期望 %s, 实际 %s", http.MethodGet, r.Method)
		}

		// 验证请求路径
		if r.URL.Path != "/feedbacks" {
			t.Errorf("请求路径错误: 期望 /feedbacks, 实际 %s", r.URL.Path)
		}

		// 验证 space_id 参数
		if r.URL.Query().Get("space_id") != expectedSpaceID {
			t.Errorf("space_id 参数错误: 期望 %s, 实际 %s", expectedSpaceID, r.URL.Query().Get("space_id"))
		}

		// 构造包含多个反馈的响应
		respJSON := `{
			"data": [
				{
					"type": "feedback",
					"id": "feedback_001",
					"attributes": {
						"status": "unprocessed",
						"type": "kb_content_incomplete",
						"content": "这篇文档缺少示例代码",
						"created_at": "2024-01-15T10:30:00Z",
						"reviewed_at": ""
					},
					"relationships": {
						"owner": {
							"data": {
								"id": "user_001"
							}
						},
						"entry": {
							"data": {
								"id": "entry_001"
							}
						}
					}
				},
				{
					"type": "feedback",
					"id": "feedback_002",
					"attributes": {
						"status": "processed",
						"type": "kb_content_mistake",
						"content": "文档中有错别字",
						"created_at": "2024-01-14T09:00:00Z",
						"reviewed_at": "2024-01-14T15:00:00Z"
					},
					"relationships": {
						"owner": {
							"data": {
								"id": "user_002"
							}
						},
						"entry": {
							"data": {
								"id": "entry_002"
							}
						}
					}
				}
			],
			"meta": {
				"page_token": "` + expectedNextPageToken + `"
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respJSON))
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 ListFeedbacks
	ctx := context.Background()
	result, err := client.ListFeedbacks(ctx, expectedSpaceID, 10, "")
	if err != nil {
		t.Fatalf("ListFeedbacks 调用失败: %v", err)
	}

	// 验证列表长度
	if len(result.Data) != 2 {
		t.Errorf("反馈列表长度错误: 期望 2, 实际 %d", len(result.Data))
	}

	// 验证第一个反馈
	if result.Data[0].ID != "feedback_001" {
		t.Errorf("第一个反馈 ID 错误: 期望 feedback_001, 实际 %s", result.Data[0].ID)
	}
	if result.Data[0].Type != "feedback" {
		t.Errorf("第一个反馈 Type 错误: 期望 feedback, 实际 %s", result.Data[0].Type)
	}
	if result.Data[0].Attributes.Status != "unprocessed" {
		t.Errorf("第一个反馈状态错误: 期望 unprocessed, 实际 %s", result.Data[0].Attributes.Status)
	}
	if result.Data[0].Attributes.Type != "kb_content_incomplete" {
		t.Errorf("第一个反馈类型错误: 期望 kb_content_incomplete, 实际 %s", result.Data[0].Attributes.Type)
	}
	if result.Data[0].Attributes.Content != "这篇文档缺少示例代码" {
		t.Errorf("第一个反馈内容错误: 期望 '这篇文档缺少示例代码', 实际 '%s'", result.Data[0].Attributes.Content)
	}
	if result.Data[0].Relationships.Owner.Data.ID != "user_001" {
		t.Errorf("第一个反馈所有者 ID 错误: 期望 user_001, 实际 %s", result.Data[0].Relationships.Owner.Data.ID)
	}
	if result.Data[0].Relationships.Entry.Data.ID != "entry_001" {
		t.Errorf("第一个反馈关联节点 ID 错误: 期望 entry_001, 实际 %s", result.Data[0].Relationships.Entry.Data.ID)
	}

	// 验证第二个反馈
	if result.Data[1].ID != "feedback_002" {
		t.Errorf("第二个反馈 ID 错误: 期望 feedback_002, 实际 %s", result.Data[1].ID)
	}
	if result.Data[1].Attributes.Status != "processed" {
		t.Errorf("第二个反馈状态错误: 期望 processed, 实际 %s", result.Data[1].Attributes.Status)
	}
	if result.Data[1].Attributes.ReviewedAt != "2024-01-14T15:00:00Z" {
		t.Errorf("第二个反馈审核时间错误: 期望 2024-01-14T15:00:00Z, 实际 %s", result.Data[1].Attributes.ReviewedAt)
	}

	// 验证分页 token
	if result.Meta.PageToken != expectedNextPageToken {
		t.Errorf("分页 token 错误: 期望 %s, 实际 %s", expectedNextPageToken, result.Meta.PageToken)
	}
}

// TestListFeedbacks_IncludedParsing 测试 ListFeedbacks 响应中 included 字段解析
// 验证能正确解析关联的用户和节点信息
// _Requirements: 7.2_
func TestListFeedbacks_IncludedParsing(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造包含 included 字段的响应
		respJSON := `{
			"data": [
				{
					"type": "feedback",
					"id": "feedback_001",
					"attributes": {
						"status": "unprocessed",
						"type": "kb_content_suggestion",
						"content": "建议增加更多示例",
						"created_at": "2024-01-15T10:30:00Z",
						"reviewed_at": ""
					},
					"relationships": {
						"owner": {
							"data": {
								"id": "user_001"
							}
						},
						"entry": {
							"data": {
								"id": "entry_001"
							}
						}
					}
				}
			],
			"included": [
				{
					"type": "staff",
					"id": "user_001",
					"attributes": {
						"name": "张三",
						"email": "zhangsan@example.com",
						"avatar": "https://example.com/avatar/user_001.png"
					}
				},
				{
					"type": "entry",
					"id": "entry_001",
					"attributes": {
						"name": "API 使用指南",
						"entry_type": "page"
					}
				}
			],
			"meta": {
				"page_token": ""
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respJSON))
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 ListFeedbacks
	ctx := context.Background()
	result, err := client.ListFeedbacks(ctx, "space_001", 10, "")
	if err != nil {
		t.Fatalf("ListFeedbacks 调用失败: %v", err)
	}

	// 验证 included 字段长度
	if len(result.Included) != 2 {
		t.Errorf("included 列表长度错误: 期望 2, 实际 %d", len(result.Included))
	}

	// 验证第一个 included 项（用户信息）
	var staffIncluded, entryIncluded *struct {
		Type       string         `json:"type"`
		ID         string         `json:"id"`
		Attributes map[string]any `json:"attributes"`
	}

	for i := range result.Included {
		if result.Included[i].Type == "staff" {
			staffIncluded = &result.Included[i]
		} else if result.Included[i].Type == "entry" {
			entryIncluded = &result.Included[i]
		}
	}

	// 验证用户信息
	if staffIncluded == nil {
		t.Fatal("未找到 staff 类型的 included 项")
	}
	if staffIncluded.ID != "user_001" {
		t.Errorf("用户 ID 错误: 期望 user_001, 实际 %s", staffIncluded.ID)
	}
	if name, ok := staffIncluded.Attributes["name"].(string); !ok || name != "张三" {
		t.Errorf("用户名称错误: 期望 张三, 实际 %v", staffIncluded.Attributes["name"])
	}
	if email, ok := staffIncluded.Attributes["email"].(string); !ok || email != "zhangsan@example.com" {
		t.Errorf("用户邮箱错误: 期望 zhangsan@example.com, 实际 %v", staffIncluded.Attributes["email"])
	}

	// 验证节点信息
	if entryIncluded == nil {
		t.Fatal("未找到 entry 类型的 included 项")
	}
	if entryIncluded.ID != "entry_001" {
		t.Errorf("节点 ID 错误: 期望 entry_001, 实际 %s", entryIncluded.ID)
	}
	if name, ok := entryIncluded.Attributes["name"].(string); !ok || name != "API 使用指南" {
		t.Errorf("节点名称错误: 期望 API 使用指南, 实际 %v", entryIncluded.Attributes["name"])
	}
	if entryType, ok := entryIncluded.Attributes["entry_type"].(string); !ok || entryType != "page" {
		t.Errorf("节点类型错误: 期望 page, 实际 %v", entryIncluded.Attributes["entry_type"])
	}
}

// TestListFeedbacks_PaginationParams 测试 ListFeedbacks 分页参数传递
// 验证 limit 和 page_token 参数被正确拼接到查询字符串中
// _Requirements: 7.1_
func TestListFeedbacks_PaginationParams(t *testing.T) {
	testCases := []struct {
		name              string
		spaceID           string
		limit             int
		pageToken         string
		expectedSpaceID   string
		expectedLimit     string
		expectedPageToken string
	}{
		{
			name:              "基本分页参数",
			spaceID:           "space_001",
			limit:             10,
			pageToken:         "",
			expectedSpaceID:   "space_001",
			expectedLimit:     "10",
			expectedPageToken: "",
		},
		{
			name:              "带 page_token 的分页",
			spaceID:           "space_002",
			limit:             20,
			pageToken:         "next_page_token_abc",
			expectedSpaceID:   "space_002",
			expectedLimit:     "20",
			expectedPageToken: "next_page_token_abc",
		},
		{
			name:              "limit 为 0 时不传递",
			spaceID:           "space_003",
			limit:             0,
			pageToken:         "",
			expectedSpaceID:   "space_003",
			expectedLimit:     "",
			expectedPageToken: "",
		},
		{
			name:              "大 limit 值",
			spaceID:           "space_004",
			limit:             100,
			pageToken:         "page_token_xyz",
			expectedSpaceID:   "space_004",
			expectedLimit:     "100",
			expectedPageToken: "page_token_xyz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 用于捕获请求的变量
			var capturedQuery url.Values

			// 创建模拟服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 捕获查询参数
				capturedQuery = r.URL.Query()

				// 返回空列表响应
				resp := FeedbackListResponse{}
				resp.Meta.PageToken = "next_token"

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// 创建模拟 TokenManager
			mockTM := &mockTokenManager{token: "test_token"}

			// 创建客户端
			client := NewClientWithTokenManager(mockTM, server.URL)

			// 调用 ListFeedbacks
			ctx := context.Background()
			_, err := client.ListFeedbacks(ctx, tc.spaceID, tc.limit, tc.pageToken)
			if err != nil {
				t.Fatalf("ListFeedbacks 调用失败: %v", err)
			}

			// 验证 space_id 参数
			if capturedQuery.Get("space_id") != tc.expectedSpaceID {
				t.Errorf("space_id 参数错误: 期望 %s, 实际 %s", tc.expectedSpaceID, capturedQuery.Get("space_id"))
			}

			// 验证 limit 参数
			if tc.expectedLimit != "" {
				if capturedQuery.Get("limit") != tc.expectedLimit {
					t.Errorf("limit 参数错误: 期望 %s, 实际 %s", tc.expectedLimit, capturedQuery.Get("limit"))
				}
			} else {
				// limit 为 0 时不应该传递该参数
				if capturedQuery.Get("limit") != "" {
					t.Errorf("limit 为 0 时不应传递该参数, 实际传递了: %s", capturedQuery.Get("limit"))
				}
			}

			// 验证 page_token 参数
			if tc.expectedPageToken != "" {
				if capturedQuery.Get("page_token") != tc.expectedPageToken {
					t.Errorf("page_token 参数错误: 期望 %s, 实际 %s", tc.expectedPageToken, capturedQuery.Get("page_token"))
				}
			} else {
				// page_token 为空时不应该传递该参数
				if capturedQuery.Get("page_token") != "" {
					t.Errorf("page_token 为空时不应传递该参数, 实际传递了: %s", capturedQuery.Get("page_token"))
				}
			}
		})
	}
}

// TestListFeedbacks_EmptyResponse 测试 ListFeedbacks 空响应处理
// 验证当没有反馈时能正确处理空列表
// _Requirements: 7.1_
func TestListFeedbacks_EmptyResponse(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回空列表响应
		respJSON := `{
			"data": [],
			"included": [],
			"meta": {
				"page_token": ""
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respJSON))
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 ListFeedbacks
	ctx := context.Background()
	result, err := client.ListFeedbacks(ctx, "space_001", 10, "")
	if err != nil {
		t.Fatalf("ListFeedbacks 调用失败: %v", err)
	}

	// 验证空列表
	if len(result.Data) != 0 {
		t.Errorf("反馈列表应为空: 期望 0, 实际 %d", len(result.Data))
	}

	// 验证空 included
	if len(result.Included) != 0 {
		t.Errorf("included 列表应为空: 期望 0, 实际 %d", len(result.Included))
	}

	// 验证空分页 token
	if result.Meta.PageToken != "" {
		t.Errorf("分页 token 应为空: 期望空字符串, 实际 %s", result.Meta.PageToken)
	}
}

// TestListFeedbacks_AllFeedbackTypes 测试 ListFeedbacks 所有反馈类型解析
// 验证能正确解析所有类型的反馈
// _Requirements: 7.1_
func TestListFeedbacks_AllFeedbackTypes(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造包含所有反馈类型的响应
		respJSON := `{
			"data": [
				{
					"type": "feedback",
					"id": "feedback_001",
					"attributes": {
						"status": "unprocessed",
						"type": "kb_content_incomplete",
						"content": "内容缺失",
						"created_at": "2024-01-15T10:00:00Z",
						"reviewed_at": ""
					},
					"relationships": {
						"owner": {"data": {"id": "user_001"}},
						"entry": {"data": {"id": "entry_001"}}
					}
				},
				{
					"type": "feedback",
					"id": "feedback_002",
					"attributes": {
						"status": "processing",
						"type": "kb_content_mistake",
						"content": "内容有误",
						"created_at": "2024-01-15T11:00:00Z",
						"reviewed_at": ""
					},
					"relationships": {
						"owner": {"data": {"id": "user_002"}},
						"entry": {"data": {"id": "entry_002"}}
					}
				},
				{
					"type": "feedback",
					"id": "feedback_003",
					"attributes": {
						"status": "processed",
						"type": "kb_content_suggestion",
						"content": "内容建议",
						"created_at": "2024-01-15T12:00:00Z",
						"reviewed_at": "2024-01-15T14:00:00Z"
					},
					"relationships": {
						"owner": {"data": {"id": "user_003"}},
						"entry": {"data": {"id": "entry_003"}}
					}
				},
				{
					"type": "feedback",
					"id": "feedback_004",
					"attributes": {
						"status": "not_process",
						"type": "kb_content_too_old",
						"content": "内容陈旧",
						"created_at": "2024-01-15T13:00:00Z",
						"reviewed_at": "2024-01-15T15:00:00Z"
					},
					"relationships": {
						"owner": {"data": {"id": "user_004"}},
						"entry": {"data": {"id": "entry_004"}}
					}
				},
				{
					"type": "feedback",
					"id": "feedback_005",
					"attributes": {
						"status": "unprocessed",
						"type": "kb_content_other",
						"content": "其他问题",
						"created_at": "2024-01-15T14:00:00Z",
						"reviewed_at": ""
					},
					"relationships": {
						"owner": {"data": {"id": "user_005"}},
						"entry": {"data": {"id": "entry_005"}}
					}
				}
			],
			"meta": {
				"page_token": ""
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(respJSON))
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 ListFeedbacks
	ctx := context.Background()
	result, err := client.ListFeedbacks(ctx, "space_001", 10, "")
	if err != nil {
		t.Fatalf("ListFeedbacks 调用失败: %v", err)
	}

	// 验证列表长度
	if len(result.Data) != 5 {
		t.Errorf("反馈列表长度错误: 期望 5, 实际 %d", len(result.Data))
	}

	// 验证所有反馈类型
	expectedTypes := []string{
		"kb_content_incomplete",
		"kb_content_mistake",
		"kb_content_suggestion",
		"kb_content_too_old",
		"kb_content_other",
	}

	for i, expectedType := range expectedTypes {
		if result.Data[i].Attributes.Type != expectedType {
			t.Errorf("反馈 %d 类型错误: 期望 %s, 实际 %s", i+1, expectedType, result.Data[i].Attributes.Type)
		}
	}

	// 验证所有反馈状态
	expectedStatuses := []string{
		"unprocessed",
		"processing",
		"processed",
		"not_process",
		"unprocessed",
	}

	for i, expectedStatus := range expectedStatuses {
		if result.Data[i].Attributes.Status != expectedStatus {
			t.Errorf("反馈 %d 状态错误: 期望 %s, 实际 %s", i+1, expectedStatus, result.Data[i].Attributes.Status)
		}
	}
}

// TestListFeedbacks_ErrorHandling 测试 ListFeedbacks 错误处理
// 验证当 API 返回错误时能正确处理
// _Requirements: 7.1_
func TestListFeedbacks_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrFn  func(error) bool
		expectedErrMsg string
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "not_found", "message": "知识库不存在"}`,
			expectedErrFn:  IsNotFoundError,
			expectedErrMsg: "NotFound",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"error": "forbidden", "message": "无权访问该知识库"}`,
			expectedErrFn:  IsForbiddenError,
			expectedErrMsg: "Forbidden",
		},
		{
			name:           "500 Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "internal_error", "message": "服务器内部错误"}`,
			expectedErrFn:  IsServerError,
			expectedErrMsg: "ServerError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建模拟服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// 创建模拟 TokenManager
			mockTM := &mockTokenManager{token: "test_token"}

			// 创建客户端
			client := NewClientWithTokenManager(mockTM, server.URL)

			// 调用 ListFeedbacks
			ctx := context.Background()
			_, err := client.ListFeedbacks(ctx, "space_001", 10, "")
			if err == nil {
				t.Error("期望返回错误，但实际没有返回错误")
				return
			}

			// 验证错误类型
			if !tc.expectedErrFn(err) {
				t.Errorf("错误类型不匹配: 期望 %s 错误, 实际错误: %v", tc.expectedErrMsg, err)
			}
		})
	}
}
