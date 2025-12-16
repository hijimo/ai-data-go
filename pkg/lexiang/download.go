// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetDocFile 获取附件详情
// fileID: 附件 ID
// 返回附件详情，包含名称和下载链接
func (c *lexiangClientImpl) GetDocFile(ctx context.Context, fileID string) (*DocFileResponse, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID 不能为空")
	}

	resp, err := c.Get(ctx, "/doc-files/"+fileID)
	if err != nil {
		return nil, fmt.Errorf("获取附件详情失败: %w", err)
	}
	defer resp.Body.Close()

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	var result DocFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// DownloadDocFile 下载附件
// fileID: 附件 ID
// 返回文件数据、文件名和错误
func (c *lexiangClientImpl) DownloadDocFile(ctx context.Context, fileID string) ([]byte, string, error) {
	// 1. 获取附件详情
	docFile, err := c.GetDocFile(ctx, fileID)
	if err != nil {
		return nil, "", fmt.Errorf("获取附件详情失败: %w", err)
	}

	// 检查下载链接是否存在
	downloadURL := docFile.Data.Links.Download
	if downloadURL == "" {
		return nil, "", fmt.Errorf("附件下载链接为空")
	}

	// 2. 下载文件
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("创建下载请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	return data, docFile.Data.Attributes.Name, nil
}
