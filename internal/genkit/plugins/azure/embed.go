// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/firebase/genkit/go/ai"
)

// generateEmbeddings 生成文本嵌入
// 使用 Azure OpenAI 的 /openai/embeddings 端点
func generateEmbeddings(ctx context.Context, client *http.Client, baseURL, apiKey, apiVersion, modelName string, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	// 验证输入
	if len(req.Input) == 0 {
		return nil, NewRequestError("至少需要一个文档进行嵌入", nil)
	}

	// 提取所有文档的文本内容
	texts := make([]string, len(req.Input))
	for i, doc := range req.Input {
		texts[i] = concatenateDocumentText(doc)
	}

	// 构建嵌入请求
	embedReq := EmbeddingRequest{
		Model: modelName,
		Input: texts, // 批量处理所有文本
	}

	// 序列化请求
	reqBody, err := json.Marshal(embedReq)
	if err != nil {
		return nil, NewRequestError("序列化嵌入请求失败", err)
	}

	// 构建请求 URL - 使用 /openai/embeddings 端点
	url := fmt.Sprintf("%s/openai/embeddings?api-version=%s", baseURL, apiVersion)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, NewRequestError("创建 HTTP 请求失败", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", apiKey) // Azure OpenAI 使用 api-key 认证头

	// 发送请求
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, NewNetworkError("发送嵌入请求失败", err)
	}
	defer httpResp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, NewNetworkError("读取响应体失败", err)
	}

	// 检查 HTTP 状态码
	if httpResp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, NewAPIError(
				fmt.Sprintf("%d", httpResp.StatusCode),
				errResp.Error.Message,
				errResp.Error,
			)
		}
		return nil, NewAPIError(
			fmt.Sprintf("%d", httpResp.StatusCode),
			fmt.Sprintf("嵌入请求失败: %s", string(respBody)),
			nil,
		)
	}

	// 解析嵌入响应
	var embedResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, NewParseError("解析嵌入响应失败", err)
	}

	// 验证响应数据
	if len(embedResp.Data) != len(texts) {
		return nil, NewParseError(
			fmt.Sprintf("嵌入响应数量不匹配: 期望 %d, 实际 %d", len(texts), len(embedResp.Data)),
			nil,
		)
	}

	// 转换为 Genkit 嵌入响应格式
	return convertToEmbedResponse(&embedResp), nil
}

// concatenateDocumentText 连接文档中所有部分的文本
func concatenateDocumentText(doc *ai.Document) string {
	var builder strings.Builder
	for _, part := range doc.Content {
		if part.IsText() {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

// convertToEmbedResponse 将 Azure OpenAI 嵌入响应转换为 Genkit 格式
func convertToEmbedResponse(azureResp *EmbeddingResponse) *ai.EmbedResponse {
	embeddings := make([]*ai.Embedding, len(azureResp.Data))
	for i, data := range azureResp.Data {
		// 将 float64 转换为 float32
		embedding := make([]float32, len(data.Embedding))
		for j, val := range data.Embedding {
			embedding[j] = float32(val)
		}

		embeddings[i] = &ai.Embedding{
			Embedding: embedding,
		}
	}

	return &ai.EmbedResponse{
		Embeddings: embeddings,
	}
}
