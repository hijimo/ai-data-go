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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

func TestGenerateEmbeddings(t *testing.T) {
	tests := []struct {
		name           string
		documents      []*ai.Document
		mockResponse   EmbeddingResponse
		mockStatusCode int
		wantErr        bool
		errType        string
	}{
		{
			name: "单个文档嵌入成功",
			documents: []*ai.Document{
				{
					Content: []*ai.Part{
						ai.NewTextPart("Hello world"),
					},
				},
			},
			mockResponse: EmbeddingResponse{
				Object: "list",
				Model:  "text-embedding-ada-002",
				Data: []EmbeddingData{
					{
						Object:    "embedding",
						Index:     0,
						Embedding: []float64{0.1, 0.2, 0.3},
					},
				},
				Usage: EmbeddingUsage{
					PromptTokens: 2,
					TotalTokens:  2,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "批量文档嵌入成功",
			documents: []*ai.Document{
				{
					Content: []*ai.Part{
						ai.NewTextPart("First document"),
					},
				},
				{
					Content: []*ai.Part{
						ai.NewTextPart("Second document"),
					},
				},
				{
					Content: []*ai.Part{
						ai.NewTextPart("Third document"),
					},
				},
			},
			mockResponse: EmbeddingResponse{
				Object: "list",
				Model:  "text-embedding-ada-002",
				Data: []EmbeddingData{
					{
						Object:    "embedding",
						Index:     0,
						Embedding: []float64{0.1, 0.2, 0.3},
					},
					{
						Object:    "embedding",
						Index:     1,
						Embedding: []float64{0.4, 0.5, 0.6},
					},
					{
						Object:    "embedding",
						Index:     2,
						Embedding: []float64{0.7, 0.8, 0.9},
					},
				},
				Usage: EmbeddingUsage{
					PromptTokens: 6,
					TotalTokens:  6,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "多部分文档内容连接",
			documents: []*ai.Document{
				{
					Content: []*ai.Part{
						ai.NewTextPart("Hello "),
						ai.NewTextPart("world "),
						ai.NewTextPart("from "),
						ai.NewTextPart("Azure"),
					},
				},
			},
			mockResponse: EmbeddingResponse{
				Object: "list",
				Model:  "text-embedding-ada-002",
				Data: []EmbeddingData{
					{
						Object:    "embedding",
						Index:     0,
						Embedding: []float64{0.1, 0.2, 0.3},
					},
				},
				Usage: EmbeddingUsage{
					PromptTokens: 4,
					TotalTokens:  4,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "空文档列表错误",
			documents:      []*ai.Document{},
			mockStatusCode: http.StatusOK,
			wantErr:        true,
			errType:        "request",
		},
		{
			name: "API 错误响应",
			documents: []*ai.Document{
				{
					Content: []*ai.Part{
						ai.NewTextPart("Test"),
					},
				},
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			errType:        "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 验证请求方法
				if r.Method != "POST" {
					t.Errorf("期望 POST 请求, 实际: %s", r.Method)
				}

				// 验证请求路径
				if r.URL.Path != "/openai/embeddings" {
					t.Errorf("期望路径 /openai/embeddings, 实际: %s", r.URL.Path)
				}

				// 验证 API 版本参数
				apiVersion := r.URL.Query().Get("api-version")
				if apiVersion == "" {
					t.Error("缺少 api-version 查询参数")
				}

				// 验证认证头
				apiKey := r.Header.Get("api-key")
				if apiKey != "test-api-key" {
					t.Errorf("期望 api-key: test-api-key, 实际: %s", apiKey)
				}

				// 验证 Content-Type
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("期望 Content-Type: application/json, 实际: %s", contentType)
				}

				// 解析请求体
				var embedReq EmbeddingRequest
				if err := json.NewDecoder(r.Body).Decode(&embedReq); err != nil {
					t.Errorf("解析请求体失败: %v", err)
				}

				// 验证请求体中的文本数量
				if !tt.wantErr && tt.errType != "request" {
					texts, ok := embedReq.Input.([]interface{})
					if !ok {
						t.Error("Input 应该是数组")
					}
					if len(texts) != len(tt.documents) {
						t.Errorf("期望 %d 个文本, 实际: %d", len(tt.documents), len(texts))
					}
				}

				// 返回模拟响应
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.mockResponse)
				} else {
					json.NewEncoder(w).Encode(ErrorResponse{
						Error: ErrorDetail{
							Message: "Unauthorized",
							Type:    "invalid_request_error",
							Code:    "401",
						},
					})
				}
			}))
			defer server.Close()

			// 创建 HTTP 客户端
			client := &http.Client{}

			// 创建嵌入请求
			req := &ai.EmbedRequest{
				Input: tt.documents,
			}

			// 调用 generateEmbeddings
			resp, err := generateEmbeddings(
				context.Background(),
				client,
				server.URL,
				"test-api-key",
				"2025-04-01-preview",
				"text-embedding-ada-002",
				req,
			)

			// 验证错误
			if tt.wantErr {
				if err == nil {
					t.Error("期望错误，但没有返回错误")
				} else if tt.errType != "" {
					azErr, ok := err.(*AzureAIError)
					if !ok {
						t.Errorf("期望 AzureAIError, 实际: %T", err)
					} else if azErr.Type != tt.errType {
						t.Errorf("期望错误类型 %s, 实际: %s", tt.errType, azErr.Type)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误，但返回了错误: %v", err)
				return
			}

			// 验证响应
			if resp == nil {
				t.Error("响应不应为 nil")
				return
			}

			// 验证嵌入数量
			if len(resp.Embeddings) != len(tt.documents) {
				t.Errorf("期望 %d 个嵌入, 实际: %d", len(tt.documents), len(resp.Embeddings))
			}

			// 验证嵌入向量
			for i, embedding := range resp.Embeddings {
				if len(embedding.Embedding) != len(tt.mockResponse.Data[i].Embedding) {
					t.Errorf("嵌入 %d: 期望向量长度 %d, 实际: %d",
						i, len(tt.mockResponse.Data[i].Embedding), len(embedding.Embedding))
				}

				// 验证向量值（float64 转 float32）
				for j, val := range tt.mockResponse.Data[i].Embedding {
					expected := float32(val)
					if embedding.Embedding[j] != expected {
						t.Errorf("嵌入 %d, 位置 %d: 期望 %f, 实际: %f",
							i, j, expected, embedding.Embedding[j])
					}
				}
			}
		})
	}
}

func TestConcatenateDocumentText(t *testing.T) {
	tests := []struct {
		name     string
		document *ai.Document
		want     string
	}{
		{
			name: "单个文本部分",
			document: &ai.Document{
				Content: []*ai.Part{
					ai.NewTextPart("Hello world"),
				},
			},
			want: "Hello world",
		},
		{
			name: "多个文本部分",
			document: &ai.Document{
				Content: []*ai.Part{
					ai.NewTextPart("Hello "),
					ai.NewTextPart("world "),
					ai.NewTextPart("from "),
					ai.NewTextPart("Azure"),
				},
			},
			want: "Hello world from Azure",
		},
		{
			name: "空文档",
			document: &ai.Document{
				Content: []*ai.Part{},
			},
			want: "",
		},
		{
			name: "包含空文本的部分",
			document: &ai.Document{
				Content: []*ai.Part{
					ai.NewTextPart("Hello"),
					ai.NewTextPart(""),
					ai.NewTextPart("world"),
				},
			},
			want: "Helloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := concatenateDocumentText(tt.document)
			if got != tt.want {
				t.Errorf("concatenateDocumentText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertToEmbedResponse(t *testing.T) {
	tests := []struct {
		name       string
		azureResp  *EmbeddingResponse
		wantLength int
	}{
		{
			name: "单个嵌入",
			azureResp: &EmbeddingResponse{
				Data: []EmbeddingData{
					{
						Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
					},
				},
			},
			wantLength: 1,
		},
		{
			name: "多个嵌入",
			azureResp: &EmbeddingResponse{
				Data: []EmbeddingData{
					{
						Embedding: []float64{0.1, 0.2, 0.3},
					},
					{
						Embedding: []float64{0.4, 0.5, 0.6},
					},
					{
						Embedding: []float64{0.7, 0.8, 0.9},
					},
				},
			},
			wantLength: 3,
		},
		{
			name: "空嵌入列表",
			azureResp: &EmbeddingResponse{
				Data: []EmbeddingData{},
			},
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := convertToEmbedResponse(tt.azureResp)

			if len(resp.Embeddings) != tt.wantLength {
				t.Errorf("期望 %d 个嵌入, 实际: %d", tt.wantLength, len(resp.Embeddings))
			}

			// 验证每个嵌入的转换
			for i, embedding := range resp.Embeddings {
				azureData := tt.azureResp.Data[i]
				if len(embedding.Embedding) != len(azureData.Embedding) {
					t.Errorf("嵌入 %d: 期望向量长度 %d, 实际: %d",
						i, len(azureData.Embedding), len(embedding.Embedding))
				}

				// 验证 float64 到 float32 的转换
				for j, val := range azureData.Embedding {
					expected := float32(val)
					if embedding.Embedding[j] != expected {
						t.Errorf("嵌入 %d, 位置 %d: 期望 %f, 实际: %f",
							i, j, expected, embedding.Embedding[j])
					}
				}
			}
		})
	}
}
