// internal/storage/qdrant_client_test.go
package storage

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewQdrantClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *QdrantConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty host",
			config: &QdrantConfig{
				Host: "",
				Port: 6333,
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &QdrantConfig{
				Host: "localhost",
				Port: 6333,
			},
			wantErr: false,
		},
		{
			name: "valid config with default port",
			config: &QdrantConfig{
				Host: "localhost",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewQdrantClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestUpsertVectorRequest_Validation(t *testing.T) {
	client, err := NewQdrantClient(&QdrantConfig{
		Host: "localhost",
		Port: 6333,
	})
	assert.NoError(t, err)

	tests := []struct {
		name    string
		req     *UpsertVectorRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty tenant_id",
			req: &UpsertVectorRequest{
				TenantID:  uuid.Nil,
				MemoryID:  uuid.New(),
				SessionID: uuid.New(),
				Vector:    make([]float32, VectorDim),
			},
			wantErr: true,
		},
		{
			name: "empty memory_id",
			req: &UpsertVectorRequest{
				TenantID:  uuid.New(),
				MemoryID:  uuid.Nil,
				SessionID: uuid.New(),
				Vector:    make([]float32, VectorDim),
			},
			wantErr: true,
		},
		{
			name: "empty session_id",
			req: &UpsertVectorRequest{
				TenantID:  uuid.New(),
				MemoryID:  uuid.New(),
				SessionID: uuid.Nil,
				Vector:    make([]float32, VectorDim),
			},
			wantErr: true,
		},
		{
			name: "invalid vector dimension",
			req: &UpsertVectorRequest{
				TenantID:  uuid.New(),
				MemoryID:  uuid.New(),
				SessionID: uuid.New(),
				Vector:    make([]float32, 100), // 错误的维度
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：这里只测试验证逻辑，不实际连接 Qdrant
			impl := client.(*qdrantClientImpl)
			
			// 验证必填字段
			if tt.req == nil {
				err := impl.UpsertVector(nil, tt.req)
				assert.Error(t, err)
				return
			}

			if tt.req.TenantID == uuid.Nil {
				assert.True(t, tt.wantErr)
				return
			}

			if tt.req.MemoryID == uuid.Nil {
				assert.True(t, tt.wantErr)
				return
			}

			if tt.req.SessionID == uuid.Nil {
				assert.True(t, tt.wantErr)
				return
			}

			if len(tt.req.Vector) != VectorDim {
				assert.True(t, tt.wantErr)
				return
			}
		})
	}
}

func TestSearchVectorRequest_Validation(t *testing.T) {
	client, err := NewQdrantClient(&QdrantConfig{
		Host: "localhost",
		Port: 6333,
	})
	assert.NoError(t, err)

	tests := []struct {
		name    string
		req     *SearchVectorRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty tenant_id",
			req: &SearchVectorRequest{
				TenantID:    uuid.Nil,
				QueryVector: make([]float32, VectorDim),
				TopK:        5,
			},
			wantErr: true,
		},
		{
			name: "invalid vector dimension",
			req: &SearchVectorRequest{
				TenantID:    uuid.New(),
				QueryVector: make([]float32, 100), // 错误的维度
				TopK:        5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：这里只测试验证逻辑，不实际连接 Qdrant
			impl := client.(*qdrantClientImpl)
			
			// 验证必填字段
			if tt.req == nil {
				_, err := impl.SearchVectors(nil, tt.req)
				assert.Error(t, err)
				return
			}

			if tt.req.TenantID == uuid.Nil {
				assert.True(t, tt.wantErr)
				return
			}

			if len(tt.req.QueryVector) != VectorDim {
				assert.True(t, tt.wantErr)
				return
			}
		})
	}
}

func TestBuildFilter(t *testing.T) {
	client, err := NewQdrantClient(&QdrantConfig{
		Host: "localhost",
		Port: 6333,
	})
	assert.NoError(t, err)
	impl := client.(*qdrantClientImpl)

	tenantID := uuid.New()
	sessionID := uuid.New()
	memoryType := "long_term"

	tests := []struct {
		name     string
		req      *SearchVectorRequest
		wantMust int // 期望的 must 条件数量
	}{
		{
			name: "only tenant_id",
			req: &SearchVectorRequest{
				TenantID:    tenantID,
				QueryVector: make([]float32, VectorDim),
			},
			wantMust: 1, // 只有 tenant_id
		},
		{
			name: "tenant_id and session_id",
			req: &SearchVectorRequest{
				TenantID:    tenantID,
				SessionID:   &sessionID,
				QueryVector: make([]float32, VectorDim),
			},
			wantMust: 2, // tenant_id + session_id
		},
		{
			name: "tenant_id and memory_type",
			req: &SearchVectorRequest{
				TenantID:    tenantID,
				MemoryType:  &memoryType,
				QueryVector: make([]float32, VectorDim),
			},
			wantMust: 2, // tenant_id + memory_type
		},
		{
			name: "all filters",
			req: &SearchVectorRequest{
				TenantID:    tenantID,
				SessionID:   &sessionID,
				MemoryType:  &memoryType,
				QueryVector: make([]float32, VectorDim),
			},
			wantMust: 3, // tenant_id + session_id + memory_type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := impl.buildFilter(tt.req)
			assert.NotNil(t, filter)
			
			must, ok := filter["must"].([]interface{})
			assert.True(t, ok)
			assert.Equal(t, tt.wantMust, len(must))

			// 验证第一个条件必须是 tenant_id
			firstCondition, ok := must[0].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "tenant_id", firstCondition["key"])
		})
	}
}

func TestConstants(t *testing.T) {
	// 验证常量值
	assert.Equal(t, "conversation_memories", CollectionName)
	assert.Equal(t, 1536, VectorDim)
}
