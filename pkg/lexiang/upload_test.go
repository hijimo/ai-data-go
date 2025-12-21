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

// TestGetUploadSign_RequestFormat 测试获取上传签名的请求格式
func TestGetUploadSign_RequestFormat(t *testing.T) {
	var capturedReq struct {
		Method  string
		Path    string
		Headers http.Header
		Body    UploadSignRequest
	}

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq.Method = r.Method
		capturedReq.Path = r.URL.Path
		capturedReq.Headers = r.Header

		// 解析请求体
		json.NewDecoder(r.Body).Decode(&capturedReq.Body)

		// 返回模拟响应（新的数据结构）
		resp := UploadSignResponse{}
		resp.Object.State = "test_state_123"
		resp.Options.Bucket = "lexiang-10029162"
		resp.Options.Region = "ap-shanghai"
		resp.Object.Key = "company_xx/files/2024/01/uuid.pdf"
		resp.Object.Auth.Authorization = "test_auth"
		resp.Object.Auth.XCosSecurityToken = "test_token"
		resp.Object.Headers.ContentDisposition = `attachment; filename="test.pdf"`

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建模拟 TokenManager
	mockTM := &mockTokenManager{token: "test_token"}
	client := NewClientWithTokenManager(mockTM, server.URL, "staff123")

	// 调用方法
	ctx := context.Background()
	result, err := client.GetUploadSign(ctx, "test.pdf", "file")

	// 验证
	if err != nil {
		t.Fatalf("GetUploadSign 失败: %v", err)
	}

	// 验证请求方法
	if capturedReq.Method != http.MethodPost {
		t.Errorf("期望 POST 方法，实际: %s", capturedReq.Method)
	}

	// 验证 x-staff-id 头
	if capturedReq.Headers.Get("x-staff-id") != "staff123" {
		t.Errorf("期望 x-staff-id=staff123，实际: %s", capturedReq.Headers.Get("x-staff-id"))
	}

	// 验证请求体
	if capturedReq.Body.Name != "test.pdf" {
		t.Errorf("期望 name=test.pdf，实际: %s", capturedReq.Body.Name)
	}
	if capturedReq.Body.MediaType != "file" {
		t.Errorf("期望 media_type=file，实际: %s", capturedReq.Body.MediaType)
	}

	// 验证响应解析
	if result.Object.State != "test_state_123" {
		t.Errorf("期望 state=test_state_123，实际: %s", result.Object.State)
	}
	if result.Options.Bucket != "lexiang-10029162" {
		t.Errorf("期望 bucket=lexiang-10029162，实际: %s", result.Options.Bucket)
	}
}

// TestGetUploadSign_ResponseParsing 测试上传签名响应解析
func TestGetUploadSign_ResponseParsing(t *testing.T) {
	// 模拟乐享 API 的真实响应格式（新结构）
	mockResponse := `{
		"state": "abc123xyz",
		"options": {
			"bucket": "lexiang-10029162",
			"region": "ap-shanghai"
		},
		"object": {
			"key": "company_xx/files/2024/01/uuid.pdf",
			"auth": {
				"Authorization": "q-sign-algorithm=sha1&q-ak=xxx",
				"XCosSecurityToken": "security_token_xxx"
			},
			"headers": {
				"Content-Disposition": "attachment; filename=\"document.pdf\""
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	mockTM := &mockTokenManager{token: "test_token"}
	client := NewClientWithTokenManager(mockTM, server.URL, "staff123")

	ctx := context.Background()
	result, err := client.GetUploadSign(ctx, "document.pdf", "file")

	if err != nil {
		t.Fatalf("GetUploadSign 失败: %v", err)
	}

	// 验证所有字段正确解析
	if result.Object.State != "abc123xyz" {
		t.Errorf("state 解析错误，期望: abc123xyz，实际: %s", result.Object.State)
	}
	if result.Object.Key != "company_xx/files/2024/01/uuid.pdf" {
		t.Errorf("key 解析错误，期望: company_xx/files/2024/01/uuid.pdf，实际: %s", result.Object.Key)
	}
	if result.Options.Bucket != "lexiang-10029162" {
		t.Errorf("bucket 解析错误，期望: lexiang-10029162，实际: %s", result.Options.Bucket)
	}
	if result.Options.Region != "ap-shanghai" {
		t.Errorf("region 解析错误，期望: ap-shanghai，实际: %s", result.Options.Region)
	}
	if result.Object.Auth.Authorization != "q-sign-algorithm=sha1&q-ak=xxx" {
		t.Errorf("Authorization 解析错误")
	}
	if result.Object.Auth.XCosSecurityToken != "security_token_xxx" {
		t.Errorf("XCosSecurityToken 解析错误")
	}
	if result.Object.Headers.ContentDisposition != `attachment; filename="document.pdf"` {
		t.Errorf("Content-Disposition 解析错误")
	}
}

// TestUploadFileToCOS_NilSign 测试空签名参数
func TestUploadFileToCOS_NilSign(t *testing.T) {
	mockTM := &mockTokenManager{token: "test_token"}
	client := NewClientWithTokenManager(mockTM, "http://localhost", "staff123")

	ctx := context.Background()
	err := client.UploadFileToCOS(ctx, nil, []byte("test data"))

	if err == nil {
		t.Error("期望返回错误，但没有")
	}
	if !strings.Contains(err.Error(), "上传签名不能为空") {
		t.Errorf("错误信息不正确: %v", err)
	}
}

// TestUploadFile_GetSignStep 测试完整上传流程的签名获取步骤
func TestUploadFile_GetSignStep(t *testing.T) {
	// 创建乐享 API 模拟服务器
	lexiangServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := UploadSignResponse{}
		resp.Object.State = "complete_flow_state"
		resp.Options.Bucket = "test-bucket"
		resp.Options.Region = "test-region"
		resp.Object.Key = "test/file.pdf"
		resp.Object.Auth.Authorization = "test_auth"
		resp.Object.Auth.XCosSecurityToken = "test_token"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer lexiangServer.Close()

	mockTM := &mockTokenManager{token: "test_token"}
	client := NewClientWithTokenManager(mockTM, lexiangServer.URL, "staff123")

	ctx := context.Background()

	// 测试获取签名部分
	sign, err := client.GetUploadSign(ctx, "test.pdf", "file")
	if err != nil {
		t.Fatalf("GetUploadSign 失败: %v", err)
	}

	if sign.Object.State != "complete_flow_state" {
		t.Errorf("state 不正确，期望: complete_flow_state，实际: %s", sign.Object.State)
	}
}

// TestGetUploadSign_ErrorHandling 测试错误处理
func TestGetUploadSign_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "400 错误",
			statusCode: 400,
			response:   `{"errors":[{"code":"invalid_param","title":"参数错误","detail":"文件名不能为空"}]}`,
			wantErr:    true,
		},
		{
			name:       "403 错误",
			statusCode: 403,
			response:   `{"errors":[{"code":"forbidden","title":"权限不足","detail":"无权限上传文件"}]}`,
			wantErr:    true,
		},
		{
			name:       "500 错误",
			statusCode: 500,
			response:   `{"errors":[{"code":"server_error","title":"服务器错误","detail":"内部错误"}]}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			mockTM := &mockTokenManager{token: "test_token"}
			client := NewClientWithTokenManager(mockTM, server.URL, "staff123")

			ctx := context.Background()
			_, err := client.GetUploadSign(ctx, "test.pdf", "file")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUploadSign() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetUploadSign_MediaTypes 测试不同媒体类型
func TestGetUploadSign_MediaTypes(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
	}{
		{"文件类型", "file"},
		{"视频类型", "video"},
		{"音频类型", "audio"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedMediaType string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req UploadSignRequest
				json.NewDecoder(r.Body).Decode(&req)
				capturedMediaType = req.MediaType

				resp := UploadSignResponse{}
				resp.Object.State = "test_state"
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			mockTM := &mockTokenManager{token: "test_token"}
			client := NewClientWithTokenManager(mockTM, server.URL, "staff123")

			ctx := context.Background()
			_, err := client.GetUploadSign(ctx, "test.mp4", tt.mediaType)

			if err != nil {
				t.Fatalf("GetUploadSign 失败: %v", err)
			}

			if capturedMediaType != tt.mediaType {
				t.Errorf("media_type 不正确，期望: %s，实际: %s", tt.mediaType, capturedMediaType)
			}
		})
	}
}
