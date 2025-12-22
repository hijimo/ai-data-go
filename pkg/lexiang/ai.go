// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AIQA AI问答（非流式）
// 实现 AI 问答功能，向乐享 API 发送问答请求并返回完整答案
// 官方文档: POST /ai/qa
// req: AI问答请求参数
// 返回: AI问答响应，包含完整答案内容和参考来源
func (c *lexiangClientImpl) AIQA(ctx context.Context, req *AIQARequest) (*AIQAResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query 不能为空")
	}

	// 确保非流式模式
	req.Stream = false

	// 发送请求（x-staff-id 由 doRequest 自动添加）
	resp, err := c.doAIRequest(ctx, "/ai/qa", req)
	if err != nil {
		return nil, fmt.Errorf("AI问答请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result AIQAResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析AI问答响应失败: %w", err)
	}

	// 检查业务状态码
	if result.Code != 0 {
		return nil, fmt.Errorf("AI问答失败: code=%d, message=%s", result.Code, result.Message)
	}

	return &result, nil
}

// AIQAStream AI问答（流式）
// 实现 AI 问答流式输出功能，返回一个事件通道
// 官方文档: POST /ai/qa (stream=true)
// req: AI问答请求参数
// 返回: 事件通道和错误通道
func (c *lexiangClientImpl) AIQAStream(ctx context.Context, req *AIQARequest) (<-chan *AIQAStreamEvent, <-chan error) {
	eventChan := make(chan *AIQAStreamEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		if req == nil {
			errChan <- fmt.Errorf("请求参数不能为空")
			return
		}
		if req.Query == "" {
			errChan <- fmt.Errorf("query 不能为空")
			return
		}

		// 确保流式模式
		req.Stream = true

		// 发送请求
		resp, err := c.doAIRequest(ctx, "/ai/qa", req)
		if err != nil {
			errChan <- fmt.Errorf("AI问答流式请求失败: %w", err)
			return
		}
		defer resp.Body.Close()

		// 检查响应状态码
		if resp.StatusCode != http.StatusOK {
			errChan <- handleAPIError(resp)
			return
		}

		// 解析 SSE 流
		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				errChan <- fmt.Errorf("读取流式响应失败: %w", err)
				return
			}

			// 去除换行符
			line = strings.TrimSpace(line)

			// 跳过空行
			if line == "" {
				continue
			}

			// 解析 SSE 数据行
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data)

				// 跳过 [DONE] 标记
				if data == "[DONE]" {
					return
				}

				// 解析 JSON 事件
				var event AIQAStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					// 忽略解析错误，继续处理下一行
					continue
				}

				// 发送事件
				select {
				case eventChan <- &event:
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}

				// 如果是结束事件，退出
				if event.IsStop {
					return
				}
			}
		}
	}()

	return eventChan, errChan
}

// AISearch AI搜索
// 实现 AI 搜索功能，向乐享 API 发送搜索请求并返回排序后的文档列表
// 官方文档: POST /ai/search
// req: AI搜索请求参数
// 返回: AI搜索响应，包含排序后的文档列表
func (c *lexiangClientImpl) AISearch(ctx context.Context, req *AISearchRequest) (*AISearchResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query 不能为空")
	}
	if req.TopN <= 0 || req.TopN > 50 {
		return nil, fmt.Errorf("top_n 必须在 1-50 之间")
	}

	// 发送请求（x-staff-id 由 doRequest 自动添加）
	resp, err := c.doAIRequest(ctx, "/ai/search", req)
	if err != nil {
		return nil, fmt.Errorf("AI搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	// 解析响应
	var result AISearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析AI搜索响应失败: %w", err)
	}

	// 检查业务状态码
	if result.Code != 0 {
		return nil, fmt.Errorf("AI搜索失败: code=%d, message=%s", result.Code, result.Message)
	}

	return &result, nil
}

// doAIRequest 执行 AI 相关的 HTTP 请求
// AI 接口需要 x-staff-id 请求头（即使是 GET 请求）
func (c *lexiangClientImpl) doAIRequest(ctx context.Context, path string, body any) (*http.Response, error) {
	// AI 接口需要 x-staff-id 请求头
	headers := map[string]string{
		"x-staff-id": c.staffID,
	}
	return c.DoWithHeader(ctx, http.MethodPost, path, body, headers)
}
