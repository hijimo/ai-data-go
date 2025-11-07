package handler

import (
	"testing"
)

// 测试extractSessionID函数的基本功能

func TestExtractSessionID(t *testing.T) {
	handler := &ContextHandler{}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "valid path",
			path:     "/api/v1/contexts/550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "valid path with trailing slash",
			path:     "/api/v1/contexts/550e8400-e29b-41d4-a716-446655440000/",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "invalid path - no session id",
			path:     "/api/v1/contexts/",
			expected: "",
		},
		{
			name:     "invalid path - wrong endpoint",
			path:     "/api/v1/sessions/550e8400-e29b-41d4-a716-446655440000",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractSessionID(tt.path)
			if result != tt.expected {
				t.Errorf("extractSessionID(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
