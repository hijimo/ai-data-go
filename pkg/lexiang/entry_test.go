// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateFolder_Success 测试创建文件夹成功
func TestCreateFolder_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path == "/token" {
			// 返回 token
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		// 验证创建文件夹请求
		if r.URL.Path == "/v1/entries" && r.Method == http.MethodPost {
			// 验证请求头
			if r.Header.Get("x-staff-id") == "" {
				t.Error("缺少 x-staff-id 请求头")
			}
			if r.Header.Get("Authorization") != "Bearer test_token" {
				t.Errorf("Authorization 头不正确: %s", r.Header.Get("Authorization"))
			}

			// 解析请求体
			var req CreateEntryRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("解析请求体失败: %v", err)
			}

			// 验证请求参数
			if req.Name != "测试文件夹" {
				t.Errorf("文件夹名称不正确: %s", req.Name)
			}
			if req.EntryType != EntryTypeFolder {
				t.Errorf("节点类型不正确: %s", req.EntryType)
			}
			if req.Relationships.ParentEntry.Data.ID != "parent_123" {
				t.Errorf("父节点 ID 不正确: %s", req.Relationships.ParentEntry.Data.ID)
			}

			// 返回成功响应
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(EntryResponse{
				Data: struct {
					Type       string `json:"type"`
					ID         string `json:"id"`
					Attributes struct {
						Name              string `json:"name"`
						EntryType         string `json:"entry_type"`
						HasChildren       bool   `json:"has_children"`
						CreatedAt         string `json:"created_at"`
						UpdatedAt         string `json:"updated_at"`
						MemberInheritType string `json:"member_inherit_type"`
					} `json:"attributes"`
					Links struct {
						Download string `json:"download"`
					} `json:"links"`
				}{
					Type: "entry",
					ID:   "entry_456",
					Attributes: struct {
						Name              string `json:"name"`
						EntryType         string `json:"entry_type"`
						HasChildren       bool   `json:"has_children"`
						CreatedAt         string `json:"created_at"`
						UpdatedAt         string `json:"updated_at"`
						MemberInheritType string `json:"member_inherit_type"`
					}{
						Name:        "测试文件夹",
						EntryType:   "folder",
						HasChildren: false,
					},
				},
			})
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	result, err := client.CreateFolder(ctx, "staff_123", "parent_123", "测试文件夹")
	if err != nil {
		t.Fatalf("创建文件夹失败: %v", err)
	}

	// 验证结果
	if result.Data.ID != "entry_456" {
		t.Errorf("返回的节点 ID 不正确: %s", result.Data.ID)
	}
	if result.Data.Attributes.Name != "测试文件夹" {
		t.Errorf("返回的文件夹名称不正确: %s", result.Data.Attributes.Name)
	}
	if result.Data.Attributes.EntryType != "folder" {
		t.Errorf("返回的节点类型不正确: %s", result.Data.Attributes.EntryType)
	}
}

// TestCreateFileEntry_Success 测试创建文件知识节点成功
func TestCreateFileEntry_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if r.URL.Path == "/v1/entries" && r.Method == http.MethodPost {
			// 解析请求体
			var req CreateEntryRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("解析请求体失败: %v", err)
			}

			// 验证请求参数
			if req.State != "upload_state_123" {
				t.Errorf("state 不正确: %s", req.State)
			}
			if req.EntryType != EntryTypeFile {
				t.Errorf("节点类型不正确: %s", req.EntryType)
			}

			// 返回成功响应
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(EntryResponse{
				Data: struct {
					Type       string `json:"type"`
					ID         string `json:"id"`
					Attributes struct {
						Name              string `json:"name"`
						EntryType         string `json:"entry_type"`
						HasChildren       bool   `json:"has_children"`
						CreatedAt         string `json:"created_at"`
						UpdatedAt         string `json:"updated_at"`
						MemberInheritType string `json:"member_inherit_type"`
					} `json:"attributes"`
					Links struct {
						Download string `json:"download"`
					} `json:"links"`
				}{
					Type: "entry",
					ID:   "file_entry_789",
					Attributes: struct {
						Name              string `json:"name"`
						EntryType         string `json:"entry_type"`
						HasChildren       bool   `json:"has_children"`
						CreatedAt         string `json:"created_at"`
						UpdatedAt         string `json:"updated_at"`
						MemberInheritType string `json:"member_inherit_type"`
					}{
						Name:      "document.pdf",
						EntryType: "file",
					},
					Links: struct {
						Download string `json:"download"`
					}{
						Download: "https://example.com/download/document.pdf",
					},
				},
			})
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	result, err := client.CreateFileEntry(ctx, "staff_123", "parent_123", "upload_state_123", EntryTypeFile, "")
	if err != nil {
		t.Fatalf("创建文件知识节点失败: %v", err)
	}

	// 验证结果
	if result.Data.ID != "file_entry_789" {
		t.Errorf("返回的节点 ID 不正确: %s", result.Data.ID)
	}
	if result.Data.Attributes.EntryType != "file" {
		t.Errorf("返回的节点类型不正确: %s", result.Data.Attributes.EntryType)
	}
}

// TestListEntries_WithPagination 测试获取知识节点列表（带分页参数）
func TestListEntries_WithPagination(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v1/entries") && r.Method == http.MethodGet {
			// 验证查询参数
			query := r.URL.Query()
			if query.Get("space_id") != "space_123" {
				t.Errorf("space_id 参数不正确: %s", query.Get("space_id"))
			}
			if query.Get("parent_id") != "parent_456" {
				t.Errorf("parent_id 参数不正确: %s", query.Get("parent_id"))
			}
			if query.Get("limit") != "10" {
				t.Errorf("limit 参数不正确: %s", query.Get("limit"))
			}
			if query.Get("page_token") != "token_abc" {
				t.Errorf("page_token 参数不正确: %s", query.Get("page_token"))
			}

			// 返回成功响应
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(EntryListResponse{
				Data: []struct {
					Type       string `json:"type"`
					ID         string `json:"id"`
					Attributes struct {
						Name        string `json:"name"`
						EntryType   string `json:"entry_type"`
						HasChildren bool   `json:"has_children"`
					} `json:"attributes"`
				}{
					{
						Type: "entry",
						ID:   "entry_1",
						Attributes: struct {
							Name        string `json:"name"`
							EntryType   string `json:"entry_type"`
							HasChildren bool   `json:"has_children"`
						}{
							Name:        "文件夹1",
							EntryType:   "folder",
							HasChildren: true,
						},
					},
					{
						Type: "entry",
						ID:   "entry_2",
						Attributes: struct {
							Name        string `json:"name"`
							EntryType   string `json:"entry_type"`
							HasChildren bool   `json:"has_children"`
						}{
							Name:        "文档.pdf",
							EntryType:   "file",
							HasChildren: false,
						},
					},
				},
				Meta: struct {
					PageToken string `json:"page_token"`
				}{
					PageToken: "next_page_token",
				},
			})
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	result, err := client.ListEntries(ctx, "space_123", "parent_456", 10, "token_abc")
	if err != nil {
		t.Fatalf("获取知识节点列表失败: %v", err)
	}

	// 验证结果
	if len(result.Data) != 2 {
		t.Errorf("返回的节点数量不正确: %d", len(result.Data))
	}
	if result.Meta.PageToken != "next_page_token" {
		t.Errorf("返回的 page_token 不正确: %s", result.Meta.PageToken)
	}
}

// TestGetEntry_Success 测试获取知识节点详情成功
func TestGetEntry_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if r.URL.Path == "/v1/entries/entry_123" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(EntryResponse{
				Data: struct {
					Type       string `json:"type"`
					ID         string `json:"id"`
					Attributes struct {
						Name              string `json:"name"`
						EntryType         string `json:"entry_type"`
						HasChildren       bool   `json:"has_children"`
						CreatedAt         string `json:"created_at"`
						UpdatedAt         string `json:"updated_at"`
						MemberInheritType string `json:"member_inherit_type"`
					} `json:"attributes"`
					Links struct {
						Download string `json:"download"`
					} `json:"links"`
				}{
					Type: "entry",
					ID:   "entry_123",
					Attributes: struct {
						Name              string `json:"name"`
						EntryType         string `json:"entry_type"`
						HasChildren       bool   `json:"has_children"`
						CreatedAt         string `json:"created_at"`
						UpdatedAt         string `json:"updated_at"`
						MemberInheritType string `json:"member_inherit_type"`
					}{
						Name:              "测试文档.pdf",
						EntryType:         "file",
						HasChildren:       false,
						CreatedAt:         "2024-01-01T00:00:00Z",
						UpdatedAt:         "2024-01-02T00:00:00Z",
						MemberInheritType: "default",
					},
					Links: struct {
						Download string `json:"download"`
					}{
						Download: "https://example.com/download/test.pdf",
					},
				},
			})
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	result, err := client.GetEntry(ctx, "entry_123")
	if err != nil {
		t.Fatalf("获取知识节点详情失败: %v", err)
	}

	// 验证结果
	if result.Data.ID != "entry_123" {
		t.Errorf("返回的节点 ID 不正确: %s", result.Data.ID)
	}
	if result.Data.Attributes.Name != "测试文档.pdf" {
		t.Errorf("返回的节点名称不正确: %s", result.Data.Attributes.Name)
	}
	if result.Data.Links.Download == "" {
		t.Error("返回的下载链接为空")
	}
}

// TestGetEntryContent_Success 测试获取线上文档内容成功
func TestGetEntryContent_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if r.URL.Path == "/v1/entries/page_entry_123/content" && r.Method == http.MethodGet {
			// 验证查询参数
			if r.URL.Query().Get("content_type") != "html" {
				t.Errorf("content_type 参数不正确: %s", r.URL.Query().Get("content_type"))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(EntryContentResponse{
				Name:        "线上文档标题",
				HTMLContent: "<h1>文档内容</h1><p>这是一段测试内容</p>",
			})
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	result, err := client.GetEntryContent(ctx, "page_entry_123")
	if err != nil {
		t.Fatalf("获取线上文档内容失败: %v", err)
	}

	// 验证结果
	if result.Name != "线上文档标题" {
		t.Errorf("返回的文档标题不正确: %s", result.Name)
	}
	if !strings.Contains(result.HTMLContent, "<h1>文档内容</h1>") {
		t.Errorf("返回的 HTML 内容不正确: %s", result.HTMLContent)
	}
}

// TestDeleteEntry_Success 测试删除知识节点成功
func TestDeleteEntry_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if r.URL.Path == "/v1/entries/entry_to_delete" && r.Method == http.MethodDelete {
			// 验证请求头
			if r.Header.Get("x-staff-id") != "staff_123" {
				t.Errorf("x-staff-id 请求头不正确: %s", r.Header.Get("x-staff-id"))
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	err := client.DeleteEntry(ctx, "staff_123", "entry_to_delete")
	if err != nil {
		t.Fatalf("删除知识节点失败: %v", err)
	}
}

// TestReuploadFile_Success 测试重新上传文件成功
func TestReuploadFile_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if r.URL.Path == "/v1/entries/entry_123/file" && r.Method == http.MethodPut {
			// 验证请求头
			if r.Header.Get("x-staff-id") != "staff_123" {
				t.Errorf("x-staff-id 请求头不正确: %s", r.Header.Get("x-staff-id"))
			}

			// 解析请求体
			var req ReuploadFileRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("解析请求体失败: %v", err)
			}

			// 验证请求参数
			if req.State != "new_upload_state" {
				t.Errorf("state 不正确: %s", req.State)
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	err := client.ReuploadFile(ctx, "staff_123", "entry_123", "new_upload_state")
	if err != nil {
		t.Fatalf("重新上传文件失败: %v", err)
	}
}

// TestCreateFolder_Error 测试创建文件夹失败（权限不足）
func TestCreateFolder_Error(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse{
				TokenType:   "Bearer",
				ExpiresIn:   7200,
				AccessToken: "test_token",
			})
			return
		}

		if r.URL.Path == "/v1/entries" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":{"code":"forbidden","message":"权限不足"}}`))
			return
		}

		t.Errorf("未预期的请求: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	// 创建客户端
	tm, _ := NewTokenManager(&Config{
		AppKey:    "test_key",
		AppSecret: "test_secret",
		BaseURL:   server.URL,
	})
	client := NewClientWithTokenManager(tm, server.URL+"/v1")

	// 执行测试
	ctx := context.Background()
	_, err := client.CreateFolder(ctx, "staff_123", "parent_123", "测试文件夹")
	if err == nil {
		t.Fatal("预期应该返回错误")
	}

	// 验证错误类型
	if !IsForbiddenError(err) {
		t.Errorf("预期返回 Forbidden 错误，实际: %v", err)
	}
}
