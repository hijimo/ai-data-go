// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ============================================================================
// CreateSpace 单元测试
// ============================================================================

// TestCreateSpace_RequestFormat 测试 CreateSpace 请求格式
// 验证请求体包含正确的 JSON 结构和字段
// _Requirements: 3.1_
func TestCreateSpace_RequestFormat(t *testing.T) {
	// 测试参数
	staffID := "staff_123"
	teamID := "team_456"
	spaceName := "测试知识库"

	// 用于捕获请求的变量
	var capturedRequest CreateSpaceRequest
	var capturedHeaders http.Header
	var capturedMethod string
	var capturedPath string

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 捕获请求信息
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedHeaders = r.Header.Clone()

		// 解析请求体
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Errorf("解析请求体失败: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 返回成功响应
		resp := SpaceResponse{}
		resp.Data.Type = "space"
		resp.Data.ID = "space_789"
		resp.Data.Attributes.Name = spaceName
		resp.Data.Relationships.Team.Data.ID = teamID
		resp.Data.Relationships.RootEntry.Data.ID = "entry_root_001"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 CreateSpace
	ctx := context.Background()
	_, err := client.CreateSpace(ctx, staffID, teamID, spaceName)
	if err != nil {
		t.Fatalf("CreateSpace 调用失败: %v", err)
	}

	// 验证请求方法
	if capturedMethod != http.MethodPost {
		t.Errorf("请求方法错误: 期望 %s, 实际 %s", http.MethodPost, capturedMethod)
	}

	// 验证请求路径
	if capturedPath != "/spaces" {
		t.Errorf("请求路径错误: 期望 /spaces, 实际 %s", capturedPath)
	}

	// 验证 x-staff-id 请求头
	if capturedHeaders.Get("x-staff-id") != staffID {
		t.Errorf("x-staff-id 请求头错误: 期望 %s, 实际 %s", staffID, capturedHeaders.Get("x-staff-id"))
	}

	// 验证 Authorization 请求头
	authHeader := capturedHeaders.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Errorf("Authorization 请求头格式错误: %s", authHeader)
	}

	// 验证 Content-Type 请求头
	contentType := capturedHeaders.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type 请求头错误: 期望包含 application/json, 实际 %s", contentType)
	}

	// 验证请求体中的知识库名称
	if capturedRequest.Data.Attributes.Name != spaceName {
		t.Errorf("请求体中的 name 错误: 期望 %s, 实际 %s", spaceName, capturedRequest.Data.Attributes.Name)
	}

	// 验证请求体中的团队 ID
	if capturedRequest.Data.Relationships.Team.Data.ID != teamID {
		t.Errorf("请求体中的 team_id 错误: 期望 %s, 实际 %s", teamID, capturedRequest.Data.Relationships.Team.Data.ID)
	}
}

// TestCreateSpace_ResponseParsing 测试 CreateSpace 响应解析
// 验证能正确解析包含知识库 ID 和根目录 ID 的响应
// _Requirements: 3.1_
func TestCreateSpace_ResponseParsing(t *testing.T) {
	// 预期的响应数据
	expectedSpaceID := "space_abc123"
	expectedSpaceName := "我的知识库"
	expectedTeamID := "team_xyz789"
	expectedRootEntryID := "entry_root_456"
	expectedLogo := "https://example.com/logo.png"
	expectedVisibleType := 1
	expectedManagerInheritType := "manager"
	expectedMemberInheritType := "viewer"

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造完整的响应
		resp := SpaceResponse{}
		resp.Data.Type = "space"
		resp.Data.ID = expectedSpaceID
		resp.Data.Attributes.Name = expectedSpaceName
		resp.Data.Attributes.Logo = expectedLogo
		resp.Data.Attributes.VisibleType = expectedVisibleType
		resp.Data.Attributes.ManagerInheritType = expectedManagerInheritType
		resp.Data.Attributes.MemberInheritType = expectedMemberInheritType
		resp.Data.Relationships.Team.Data.ID = expectedTeamID
		resp.Data.Relationships.RootEntry.Data.ID = expectedRootEntryID

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 CreateSpace
	ctx := context.Background()
	result, err := client.CreateSpace(ctx, "staff_001", expectedTeamID, expectedSpaceName)
	if err != nil {
		t.Fatalf("CreateSpace 调用失败: %v", err)
	}

	// 验证响应解析
	if result.Data.ID != expectedSpaceID {
		t.Errorf("知识库 ID 解析错误: 期望 %s, 实际 %s", expectedSpaceID, result.Data.ID)
	}

	if result.Data.Attributes.Name != expectedSpaceName {
		t.Errorf("知识库名称解析错误: 期望 %s, 实际 %s", expectedSpaceName, result.Data.Attributes.Name)
	}

	if result.Data.Attributes.Logo != expectedLogo {
		t.Errorf("知识库 Logo 解析错误: 期望 %s, 实际 %s", expectedLogo, result.Data.Attributes.Logo)
	}

	if result.Data.Attributes.VisibleType != expectedVisibleType {
		t.Errorf("知识库可见类型解析错误: 期望 %d, 实际 %d", expectedVisibleType, result.Data.Attributes.VisibleType)
	}

	if result.Data.Relationships.Team.Data.ID != expectedTeamID {
		t.Errorf("团队 ID 解析错误: 期望 %s, 实际 %s", expectedTeamID, result.Data.Relationships.Team.Data.ID)
	}

	if result.Data.Relationships.RootEntry.Data.ID != expectedRootEntryID {
		t.Errorf("根目录 ID 解析错误: 期望 %s, 实际 %s", expectedRootEntryID, result.Data.Relationships.RootEntry.Data.ID)
	}
}

// ============================================================================
// ListSpaces 单元测试
// ============================================================================

// TestListSpaces_PaginationParams 测试 ListSpaces 分页参数传递
// 验证 limit 和 page_token 参数被正确拼接到查询字符串中
// _Requirements: 3.4_
func TestListSpaces_PaginationParams(t *testing.T) {
	testCases := []struct {
		name              string
		teamID            string
		limit             int
		pageToken         string
		expectedTeamID    string
		expectedLimit     string
		expectedPageToken string
	}{
		{
			name:              "基本分页参数",
			teamID:            "team_001",
			limit:             10,
			pageToken:         "",
			expectedTeamID:    "team_001",
			expectedLimit:     "10",
			expectedPageToken: "",
		},
		{
			name:              "带 page_token 的分页",
			teamID:            "team_002",
			limit:             20,
			pageToken:         "next_page_token_abc",
			expectedTeamID:    "team_002",
			expectedLimit:     "20",
			expectedPageToken: "next_page_token_abc",
		},
		{
			name:              "limit 为 0 时不传递",
			teamID:            "team_003",
			limit:             0,
			pageToken:         "",
			expectedTeamID:    "team_003",
			expectedLimit:     "",
			expectedPageToken: "",
		},
		{
			name:              "大 limit 值",
			teamID:            "team_004",
			limit:             100,
			pageToken:         "page_token_xyz",
			expectedTeamID:    "team_004",
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
				resp := SpaceListResponse{}
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

			// 调用 ListSpaces
			ctx := context.Background()
			_, err := client.ListSpaces(ctx, tc.teamID, tc.limit, tc.pageToken)
			if err != nil {
				t.Fatalf("ListSpaces 调用失败: %v", err)
			}

			// 验证 team_id 参数
			if capturedQuery.Get("team_id") != tc.expectedTeamID {
				t.Errorf("team_id 参数错误: 期望 %s, 实际 %s", tc.expectedTeamID, capturedQuery.Get("team_id"))
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

// TestListSpaces_ResponseParsing 测试 ListSpaces 响应解析
// 验证能正确解析知识库列表和分页信息
// _Requirements: 3.4_
func TestListSpaces_ResponseParsing(t *testing.T) {
	// 预期的响应数据
	expectedNextPageToken := "next_page_token_123"

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造包含多个知识库的响应
		respJSON := `{
			"data": [
				{
					"type": "space",
					"id": "space_001",
					"attributes": {
						"name": "知识库一",
						"logo": "https://example.com/logo1.png"
					},
					"relationships": {
						"root_entry": {
							"data": {
								"id": "entry_001"
							}
						}
					}
				},
				{
					"type": "space",
					"id": "space_002",
					"attributes": {
						"name": "知识库二",
						"logo": "https://example.com/logo2.png"
					},
					"relationships": {
						"root_entry": {
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

	// 调用 ListSpaces
	ctx := context.Background()
	result, err := client.ListSpaces(ctx, "team_001", 10, "")
	if err != nil {
		t.Fatalf("ListSpaces 调用失败: %v", err)
	}

	// 验证列表长度
	if len(result.Data) != 2 {
		t.Errorf("知识库列表长度错误: 期望 2, 实际 %d", len(result.Data))
	}

	// 验证第一个知识库
	if result.Data[0].ID != "space_001" {
		t.Errorf("第一个知识库 ID 错误: 期望 space_001, 实际 %s", result.Data[0].ID)
	}
	if result.Data[0].Attributes.Name != "知识库一" {
		t.Errorf("第一个知识库名称错误: 期望 知识库一, 实际 %s", result.Data[0].Attributes.Name)
	}
	if result.Data[0].Relationships.RootEntry.Data.ID != "entry_001" {
		t.Errorf("第一个知识库根目录 ID 错误: 期望 entry_001, 实际 %s", result.Data[0].Relationships.RootEntry.Data.ID)
	}

	// 验证第二个知识库
	if result.Data[1].ID != "space_002" {
		t.Errorf("第二个知识库 ID 错误: 期望 space_002, 实际 %s", result.Data[1].ID)
	}
	if result.Data[1].Attributes.Name != "知识库二" {
		t.Errorf("第二个知识库名称错误: 期望 知识库二, 实际 %s", result.Data[1].Attributes.Name)
	}

	// 验证分页 token
	if result.Meta.PageToken != expectedNextPageToken {
		t.Errorf("分页 token 错误: 期望 %s, 实际 %s", expectedNextPageToken, result.Meta.PageToken)
	}
}

// ============================================================================
// 辅助类型和函数
// ============================================================================

// mockTokenManager 模拟 TokenManager 用于测试
type mockTokenManager struct {
	token string
	err   error
}

func (m *mockTokenManager) GetToken(ctx context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func (m *mockTokenManager) InvalidateToken() {
	// 测试用，不做任何操作
}
