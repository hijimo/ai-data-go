package flows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMemorySearchInput_Validation 测试记忆检索输入验证
func TestMemorySearchInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   MemorySearchInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: false,
		},
		{
			name: "缺少会话ID",
			input: MemorySearchInput{
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "缺少查询文本",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "TopK为0",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				Query:         "测试查询",
				TopK:          0,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "相似度超出范围",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemorySearchInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMemoryStoreInput_Validation 测试记忆存储输入验证
func TestMemoryStoreInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   MemoryStoreInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				MemoryType: "long_term",
				Content:    "测试内容",
				Importance: 0.8,
			},
			wantErr: false,
		},
		{
			name: "缺少会话ID",
			input: MemoryStoreInput{
				MemoryType: "long_term",
				Content:    "测试内容",
				Importance: 0.8,
			},
			wantErr: true,
		},
		{
			name: "缺少记忆类型",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				Content:    "测试内容",
				Importance: 0.8,
			},
			wantErr: true,
		},
		{
			name: "缺少内容",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				MemoryType: "long_term",
				Importance: 0.8,
			},
			wantErr: true,
		},
		{
			name: "重要性超出范围",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				MemoryType: "long_term",
				Content:    "测试内容",
				Importance: 1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemoryStoreInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMemoryCleanupInput_Validation 测试记忆清理输入验证
func TestMemoryCleanupInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   MemoryCleanupInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemoryCleanupInput{
				Strategy:  "expired",
				Mode:      "soft",
				BatchSize: 100,
			},
			wantErr: false,
		},
		{
			name: "缺少策略",
			input: MemoryCleanupInput{
				Mode:      "soft",
				BatchSize: 100,
			},
			wantErr: true,
		},
		{
			name: "缺少模式",
			input: MemoryCleanupInput{
				Strategy:  "expired",
				BatchSize: 100,
			},
			wantErr: true,
		},
		{
			name: "批量大小为0",
			input: MemoryCleanupInput{
				Strategy:  "expired",
				Mode:      "soft",
				BatchSize: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemoryCleanupInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFormatTimePtr 测试时间指针格式化
func TestFormatTimePtr(t *testing.T) {
	tests := []struct {
		name string
		time *time.Time
		want string
	}{
		{
			name: "nil时间",
			time: nil,
			want: "",
		},
		{
			name: "有效时间",
			time: func() *time.Time {
				t := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
				return &t
			}(),
			want: "2024-01-01T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimePtr(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}
