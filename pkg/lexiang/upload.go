// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetUploadSign 获取上传签名
// fileName: 文件名称（需带扩展名）
// mediaType: 媒体类型（file/video/audio）
// 返回包含 COS 上传签名和 state 的响应
// API 路径：POST /kb/files/upload-params
func (c *lexiangClientImpl) GetUploadSign(ctx context.Context, fileName, mediaType string) (*UploadSignResponse, error) {
	req := UploadSignRequest{
		Name:      fileName,
		MediaType: mediaType,
	}

	// 发送请求（x-staff-id 由 doRequest 自动添加）
	resp, err := c.Post(ctx, "/files/upload-params", req)
	if err != nil {
		return nil, fmt.Errorf("获取上传签名请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码（文档说明返回 201 Created）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, handleAPIError(resp)
	}

	var result UploadSignResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析上传签名响应失败: %w", err)
	}

	return &result, nil
}

// UploadFileToCOS 上传文件到腾讯云 COS
// sign: 上传签名响应（从 GetUploadSign 获取）
// fileData: 文件二进制数据
// 上传成功时 COS 响应包含 ETag 头
func (c *lexiangClientImpl) UploadFileToCOS(ctx context.Context, sign *UploadSignResponse, fileData []byte) error {
	if sign == nil {
		return fmt.Errorf("上传签名不能为空")
	}

	// 构建 COS 上传 URL
	// 根据文档：options 在顶层包含 Bucket/Region，object 包含 key
	url := fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s",
		sign.Options.Bucket,
		sign.Options.Region,
		sign.Object.Key,
	)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(fileData))
	if err != nil {
		return fmt.Errorf("创建 COS 上传请求失败: %w", err)
	}

	// 设置必需的请求头
	req.Header.Set("Authorization", sign.Object.Auth.Authorization)
	req.Header.Set("x-cos-security-token", sign.Object.Auth.XCosSecurityToken)

	// 设置可选的 Content-Disposition 头
	if sign.Object.Headers.ContentDisposition != "" {
		req.Header.Set("Content-Disposition", sign.Object.Headers.ContentDisposition)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("COS 上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("COS 上传失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 检查 ETag 确认上传成功
	if resp.Header.Get("ETag") == "" {
		return fmt.Errorf("COS 上传可能失败：响应中没有 ETag")
	}

	return nil
}

// UploadFile 完整的文件上传流程
// fileName: 文件名称（需带扩展名）
// mediaType: 媒体类型（file/video/audio）
// fileData: 文件二进制数据
// 返回 state 用于后续创建知识节点
func (c *lexiangClientImpl) UploadFile(ctx context.Context, fileName, mediaType string, fileData []byte) (string, error) {
	// 1. 获取上传签名
	sign, err := c.GetUploadSign(ctx, fileName, mediaType)
	if err != nil {
		return "", fmt.Errorf("获取上传签名失败: %w", err)
	}

	// 2. 上传到腾讯云 COS
	if err := c.UploadFileToCOS(ctx, sign, fileData); err != nil {
		return "", fmt.Errorf("上传到 COS 失败: %w", err)
	}

	// 3. 返回 state 用于后续关联（state 在 object 内部）
	return sign.Object.State, nil
}
