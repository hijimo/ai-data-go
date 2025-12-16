// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ListFeedbacks 获取知识反馈列表
// 实现 Requirements 7.1, 7.2: 返回指定知识库的反馈列表并支持分页，正确解析关联的用户和节点信息
// spaceID: 知识库 ID
// limit: 拉取条数（0 表示使用默认值）
// pageToken: 分页游标（空字符串表示第一页）
func (c *lexiangClientImpl) ListFeedbacks(ctx context.Context, spaceID string, limit int, pageToken string) (*FeedbackListResponse, error) {
	// 构建查询参数
	params := url.Values{}
	params.Set("space_id", spaceID)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pageToken != "" {
		params.Set("page_token", pageToken)
	}

	// 构建请求路径
	path := "/feedbacks?" + params.Encode()

	// 发送请求
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("获取知识反馈列表请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result FeedbackListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析知识反馈列表响应失败: %w", err)
	}

	return &result, nil
}
