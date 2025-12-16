// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ReuploadFileRequest 重新上传文件请求
type ReuploadFileRequest struct {
	State string `json:"state"`
}

// CreateFolder 创建文件夹
// 实现 Requirements 4.1: 在指定父节点下创建文件夹并返回节点信息
// staffID: 成员帐号，作为文件夹创建者
// parentID: 父节点 ID（可使用知识库的 root_entry ID）
// name: 文件夹名称
func (c *lexiangClientImpl) CreateFolder(ctx context.Context, staffID, parentID, name string) (*EntryResponse, error) {
	// 构建请求体
	req := CreateEntryRequest{
		Name:      name,
		EntryType: EntryTypeFolder,
	}
	req.Relationships.ParentEntry.Data.ID = parentID

	// 发送请求，需要设置 x-staff-id 请求头
	resp, err := c.DoWithHeader(ctx, http.MethodPost, "/entries", req, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return nil, fmt.Errorf("创建文件夹请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result EntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析创建文件夹响应失败: %w", err)
	}

	return &result, nil
}

// CreateFileEntry 创建文件知识节点
// 实现 Requirements 4.2: 使用上传的 state 创建文件类型的知识节点
// staffID: 成员帐号，作为文件创建者
// parentID: 父节点 ID
// state: 文件上传临时标识（从上传签名接口获取）
// entryType: 节点类型（file/video/audio）
// name: 节点标题，缺省时使用上传的文件名
func (c *lexiangClientImpl) CreateFileEntry(ctx context.Context, staffID, parentID, state string, entryType EntryType, name string) (*EntryResponse, error) {
	// 构建请求体
	req := CreateEntryRequest{
		State:     state,
		EntryType: entryType,
	}
	if name != "" {
		req.Name = name
	}
	req.Relationships.ParentEntry.Data.ID = parentID

	// 发送请求
	resp, err := c.DoWithHeader(ctx, http.MethodPost, "/entries", req, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return nil, fmt.Errorf("创建文件知识节点请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result EntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析创建文件知识节点响应失败: %w", err)
	}

	return &result, nil
}

// ReuploadFile 重新上传文件
// 实现 Requirements 4.3: 更新指定节点的文件内容
// staffID: 成员帐号，需具有操作权限
// entryID: 知识节点 ID
// state: 新文件的上传临时标识
// 注意：新版本文件扩展名必须与原文件一致
func (c *lexiangClientImpl) ReuploadFile(ctx context.Context, staffID, entryID, state string) error {
	// 构建请求体
	req := ReuploadFileRequest{
		State: state,
	}

	// 发送请求
	resp, err := c.DoWithHeader(ctx, http.MethodPut, "/entries/"+entryID+"/file", req, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return fmt.Errorf("重新上传文件请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return handleAPIError(resp)
	}

	return nil
}

// DeleteEntry 删除知识节点
// 实现 Requirements 4.4: 删除指定的知识节点
// staffID: 成员帐号，需具有操作权限，或使用 "system-bot" 忽略权限校验
// entryID: 知识节点 ID
// 注意：删除时需保证节点下没有子节点
func (c *lexiangClientImpl) DeleteEntry(ctx context.Context, staffID, entryID string) error {
	// 发送请求
	resp, err := c.DoWithHeader(ctx, http.MethodDelete, "/entries/"+entryID, nil, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return fmt.Errorf("删除知识节点请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码（删除成功可能返回 200 或 204）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return handleAPIError(resp)
	}

	return nil
}

// ListEntries 获取知识节点列表
// 实现 Requirements 4.5: 返回指定知识库或父节点下的知识节点列表并支持分页
// spaceID: 知识库 ID
// parentID: 父节点 ID，空字符串表示查询根目录
// limit: 拉取条数（0 表示使用默认值）
// pageToken: 分页游标（空字符串表示第一页）
func (c *lexiangClientImpl) ListEntries(ctx context.Context, spaceID, parentID string, limit int, pageToken string) (*EntryListResponse, error) {
	// 构建查询参数
	params := url.Values{}
	params.Set("space_id", spaceID)
	if parentID != "" {
		params.Set("parent_id", parentID)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}

	// 构建请求路径
	path := "/entries?" + params.Encode()

	// 发送请求
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("获取知识节点列表请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result EntryListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析知识节点列表响应失败: %w", err)
	}

	return &result, nil
}

// GetEntry 获取知识节点详情
// 实现 Requirements 4.6: 返回指定知识节点的详细信息
// entryID: 知识节点 ID
func (c *lexiangClientImpl) GetEntry(ctx context.Context, entryID string) (*EntryResponse, error) {
	// 发送请求
	resp, err := c.Get(ctx, "/entries/"+entryID)
	if err != nil {
		return nil, fmt.Errorf("获取知识节点详情请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result EntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析知识节点详情响应失败: %w", err)
	}

	return &result, nil
}

// GetEntryContent 获取线上文档内容
// 实现 Requirements 4.7: 返回线上文档的 HTML 内容
// entryID: 知识节点 ID（仅支持 entry_type=page 的节点）
// 注意：返回的 html_content 可能包含附件链接 /kb_files/{file_id}，需使用附件详情接口下载
func (c *lexiangClientImpl) GetEntryContent(ctx context.Context, entryID string) (*EntryContentResponse, error) {
	// 发送请求，固定 content_type=html
	resp, err := c.Get(ctx, "/entries/"+entryID+"/content?content_type=html")
	if err != nil {
		return nil, fmt.Errorf("获取线上文档内容请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result EntryContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析线上文档内容响应失败: %w", err)
	}

	return &result, nil
}
