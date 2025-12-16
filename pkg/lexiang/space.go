// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// UpdateSpaceRequest 更新知识库请求
type UpdateSpaceRequest struct {
	Data struct {
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

// CreateSpace 创建知识库
// 实现 Requirements 3.1: 向乐享 API 发送创建知识库请求并返回包含知识库 ID 和根目录 ID 的响应
func (c *lexiangClientImpl) CreateSpace(ctx context.Context, staffID, teamID, name string) (*SpaceResponse, error) {
	// 构建请求体
	req := CreateSpaceRequest{}
	req.Data.Attributes.Name = name
	req.Data.Relationships.Team.Data.ID = teamID

	// 发送请求，需要设置 x-staff-id 请求头
	resp, err := c.DoWithHeader(ctx, http.MethodPost, "/spaces", req, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return nil, fmt.Errorf("创建知识库请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result SpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析创建知识库响应失败: %w", err)
	}

	return &result, nil
}

// UpdateSpace 更新知识库
// 实现 Requirements 3.2: 向乐享 API 发送更新知识库请求
func (c *lexiangClientImpl) UpdateSpace(ctx context.Context, staffID, spaceID, name string) (*SpaceResponse, error) {
	// 构建请求体
	req := UpdateSpaceRequest{}
	req.Data.Attributes.Name = name

	// 发送请求
	resp, err := c.DoWithHeader(ctx, http.MethodPut, "/spaces/"+spaceID, req, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return nil, fmt.Errorf("更新知识库请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result SpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析更新知识库响应失败: %w", err)
	}

	return &result, nil
}

// DeleteSpace 删除知识库
// 实现 Requirements 3.3: 向乐享 API 发送删除知识库请求
func (c *lexiangClientImpl) DeleteSpace(ctx context.Context, staffID, spaceID string) error {
	// 发送请求
	resp, err := c.DoWithHeader(ctx, http.MethodDelete, "/spaces/"+spaceID, nil, map[string]string{
		"x-staff-id": staffID,
	})
	if err != nil {
		return fmt.Errorf("删除知识库请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码（删除成功可能返回 200 或 204）
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return handleAPIError(resp)
	}

	return nil
}

// ListSpaces 获取知识库列表
// 实现 Requirements 3.4: 返回指定团队下的知识库列表并支持分页
func (c *lexiangClientImpl) ListSpaces(ctx context.Context, teamID string, limit int, pageToken string) (*SpaceListResponse, error) {
	// 构建查询参数
	params := url.Values{}
	params.Set("team_id", teamID)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}

	// 构建请求路径
	path := "/spaces?" + params.Encode()

	// 发送请求
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("获取知识库列表请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result SpaceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析知识库列表响应失败: %w", err)
	}

	return &result, nil
}

// GetSpace 获取知识库详情
// 实现 Requirements 3.5: 返回指定知识库的详细信息
func (c *lexiangClientImpl) GetSpace(ctx context.Context, spaceID string) (*SpaceResponse, error) {
	// 发送请求
	resp, err := c.Get(ctx, "/spaces/"+spaceID)
	if err != nil {
		return nil, fmt.Errorf("获取知识库详情请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result SpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析知识库详情响应失败: %w", err)
	}

	return &result, nil
}
