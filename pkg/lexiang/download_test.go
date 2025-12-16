// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// GetDocFile 单元测试
// ============================================================================

// TestGetDocFile_ResponseParsing 测试 GetDocFile 响应解析
// 验证能正确解析附件详情响应，包含名称和下载链接
// _Requirements: 6.1_
func TestGetDocFile_ResponseParsing(t *testing.T) {
	// 预期的响应数据
	expectedFileID := "file_abc123"
	expectedFileName := "测试文档.pdf"
	expectedDownloadURL := "https://cos.example.com/download/file_abc123"

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		expectedPath := "/doc-files/" + expectedFileID
		if r.URL.Path != expectedPath {
			t.Errorf("请求路径错误: 期望 %s, 实际 %s", expectedPath, r.URL.Path)
		}

		// 验证请求方法
		if r.Method != http.MethodGet {
			t.Errorf("请求方法错误: 期望 %s, 实际 %s", http.MethodGet, r.Method)
		}

		// 构造响应
		resp := DocFileResponse{}
		resp.Data.Type = "doc_file"
		resp.Data.ID = expectedFileID
		resp.Data.Attributes.Name = expectedFileName
		resp.Data.Links.Download = expectedDownloadURL

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 GetDocFile
	ctx := context.Background()
	result, err := client.GetDocFile(ctx, expectedFileID)
	if err != nil {
		t.Fatalf("GetDocFile 调用失败: %v", err)
	}

	// 验证响应解析
	if result.Data.ID != expectedFileID {
		t.Errorf("附件 ID 解析错误: 期望 %s, 实际 %s", expectedFileID, result.Data.ID)
	}

	if result.Data.Attributes.Name != expectedFileName {
		t.Errorf("附件名称解析错误: 期望 %s, 实际 %s", expectedFileName, result.Data.Attributes.Name)
	}

	if result.Data.Links.Download != expectedDownloadURL {
		t.Errorf("下载链接解析错误: 期望 %s, 实际 %s", expectedDownloadURL, result.Data.Links.Download)
	}
}

// TestGetDocFile_EmptyFileID 测试 GetDocFile 空 fileID 参数
// 验证当 fileID 为空时返回错误
// _Requirements: 6.1_
func TestGetDocFile_EmptyFileID(t *testing.T) {
	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端（不需要真实服务器，因为应该在参数验证时就失败）
	client := NewClientWithTokenManager(mockTM, "http://localhost:9999")

	// 调用 GetDocFile 使用空 fileID
	ctx := context.Background()
	_, err := client.GetDocFile(ctx, "")
	if err == nil {
		t.Error("期望返回错误，但实际没有返回错误")
	}
}

// TestGetDocFile_NotFound 测试 GetDocFile 404 响应处理
// 验证当附件不存在时返回正确的错误
// _Requirements: 6.1_
func TestGetDocFile_NotFound(t *testing.T) {
	// 创建模拟服务器返回 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not_found", "message": "附件不存在"}`))
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, server.URL)

	// 调用 GetDocFile
	ctx := context.Background()
	_, err := client.GetDocFile(ctx, "nonexistent_file")
	if err == nil {
		t.Error("期望返回错误，但实际没有返回错误")
	}

	// 验证是 NotFound 错误
	if !IsNotFoundError(err) {
		t.Errorf("期望 NotFound 错误，实际错误类型: %T, 错误信息: %v", err, err)
	}
}

// ============================================================================
// DownloadDocFile 单元测试
// ============================================================================

// TestDownloadDocFile_CompleteFlow 测试 DownloadDocFile 完整流程
// 验证能正确获取附件详情并下载文件内容
// _Requirements: 6.2_
func TestDownloadDocFile_CompleteFlow(t *testing.T) {
	// 预期的数据
	expectedFileID := "file_xyz789"
	expectedFileName := "报告.docx"
	expectedFileContent := []byte("这是测试文件内容")

	// 用于跟踪请求的变量
	docFileRequestCount := 0
	downloadRequestCount := 0

	// 创建模拟下载服务器（模拟 COS）
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadRequestCount++
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedFileContent)
	}))
	defer downloadServer.Close()

	// 创建模拟 API 服务器
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		docFileRequestCount++

		// 验证请求路径
		expectedPath := "/doc-files/" + expectedFileID
		if r.URL.Path != expectedPath {
			t.Errorf("请求路径错误: 期望 %s, 实际 %s", expectedPath, r.URL.Path)
		}

		// 构造响应，下载链接指向模拟下载服务器
		resp := DocFileResponse{}
		resp.Data.Type = "doc_file"
		resp.Data.ID = expectedFileID
		resp.Data.Attributes.Name = expectedFileName
		resp.Data.Links.Download = downloadServer.URL + "/download/" + expectedFileID

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, apiServer.URL)

	// 调用 DownloadDocFile
	ctx := context.Background()
	data, fileName, err := client.DownloadDocFile(ctx, expectedFileID)
	if err != nil {
		t.Fatalf("DownloadDocFile 调用失败: %v", err)
	}

	// 验证文件名
	if fileName != expectedFileName {
		t.Errorf("文件名错误: 期望 %s, 实际 %s", expectedFileName, fileName)
	}

	// 验证文件内容
	if string(data) != string(expectedFileContent) {
		t.Errorf("文件内容错误: 期望 %s, 实际 %s", string(expectedFileContent), string(data))
	}

	// 验证请求次数
	if docFileRequestCount != 1 {
		t.Errorf("GetDocFile 请求次数错误: 期望 1, 实际 %d", docFileRequestCount)
	}

	if downloadRequestCount != 1 {
		t.Errorf("下载请求次数错误: 期望 1, 实际 %d", downloadRequestCount)
	}
}

// TestDownloadDocFile_EmptyDownloadURL 测试 DownloadDocFile 空下载链接处理
// 验证当附件下载链接为空时返回错误
// _Requirements: 6.2_
func TestDownloadDocFile_EmptyDownloadURL(t *testing.T) {
	// 创建模拟 API 服务器，返回空下载链接
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造响应，下载链接为空
		resp := DocFileResponse{}
		resp.Data.Type = "doc_file"
		resp.Data.ID = "file_001"
		resp.Data.Attributes.Name = "test.pdf"
		resp.Data.Links.Download = "" // 空下载链接

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, apiServer.URL)

	// 调用 DownloadDocFile
	ctx := context.Background()
	_, _, err := client.DownloadDocFile(ctx, "file_001")
	if err == nil {
		t.Error("期望返回错误，但实际没有返回错误")
	}
}

// TestDownloadDocFile_DownloadFailed 测试 DownloadDocFile 下载失败处理
// 验证当下载请求失败时返回正确的错误
// _Requirements: 6.2_
func TestDownloadDocFile_DownloadFailed(t *testing.T) {
	// 创建模拟下载服务器，返回错误
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer downloadServer.Close()

	// 创建模拟 API 服务器
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造响应，下载链接指向会返回错误的服务器
		resp := DocFileResponse{}
		resp.Data.Type = "doc_file"
		resp.Data.ID = "file_001"
		resp.Data.Attributes.Name = "test.pdf"
		resp.Data.Links.Download = downloadServer.URL + "/download/file_001"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, apiServer.URL)

	// 调用 DownloadDocFile
	ctx := context.Background()
	_, _, err := client.DownloadDocFile(ctx, "file_001")
	if err == nil {
		t.Error("期望返回错误，但实际没有返回错误")
	}
}

// TestDownloadDocFile_LargeFile 测试 DownloadDocFile 大文件下载
// 验证能正确下载较大的文件
// _Requirements: 6.2_
func TestDownloadDocFile_LargeFile(t *testing.T) {
	// 生成较大的测试数据（1MB）
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	expectedFileName := "large_file.bin"

	// 创建模拟下载服务器
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(largeContent)
	}))
	defer downloadServer.Close()

	// 创建模拟 API 服务器
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := DocFileResponse{}
		resp.Data.Type = "doc_file"
		resp.Data.ID = "file_large"
		resp.Data.Attributes.Name = expectedFileName
		resp.Data.Links.Download = downloadServer.URL + "/download/file_large"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}

	// 创建客户端
	client := NewClientWithTokenManager(mockTM, apiServer.URL)

	// 调用 DownloadDocFile
	ctx := context.Background()
	data, fileName, err := client.DownloadDocFile(ctx, "file_large")
	if err != nil {
		t.Fatalf("DownloadDocFile 调用失败: %v", err)
	}

	// 验证文件名
	if fileName != expectedFileName {
		t.Errorf("文件名错误: 期望 %s, 实际 %s", expectedFileName, fileName)
	}

	// 验证文件大小
	if len(data) != len(largeContent) {
		t.Errorf("文件大小错误: 期望 %d, 实际 %d", len(largeContent), len(data))
	}

	// 验证文件内容（抽样检查）
	for i := 0; i < 100; i++ {
		idx := i * 10000
		if idx < len(data) && data[idx] != largeContent[idx] {
			t.Errorf("文件内容在位置 %d 不匹配: 期望 %d, 实际 %d", idx, largeContent[idx], data[idx])
		}
	}
}
