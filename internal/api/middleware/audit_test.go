package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShouldExcludePath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		excludedPaths []string
		expected      bool
	}{
		{
			name:          "精确匹配排除路径",
			path:          "/health",
			excludedPaths: []string{"/health", "/metrics"},
			expected:      true,
		},
		{
			name:          "前缀匹配排除路径",
			path:          "/api/v1/public/docs",
			excludedPaths: []string{"/api/v1/public/*"},
			expected:      true,
		},
		{
			name:          "不在排除列表中",
			path:          "/api/v1/sessions",
			excludedPaths: []string{"/health", "/metrics"},
			expected:      false,
		},
		{
			name:          "空排除列表",
			path:          "/api/v1/sessions",
			excludedPaths: []string{},
			expected:      false,
		},
		{
			name:          "前缀不匹配",
			path:          "/api/v1/sessions",
			excludedPaths: []string{"/api/v2/*"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExcludePath(tt.path, tt.excludedPaths)
			if result != tt.expected {
				t.Errorf("shouldExcludePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetermineEventType(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		method     string
		statusCode int
		expected   string
	}{
		{
			name:       "创建会话",
			path:       "/api/v1/sessions",
			method:     http.MethodPost,
			statusCode: http.StatusCreated,
			expected:   "session_create",
		},
		{
			name:       "读取会话",
			path:       "/api/v1/sessions/123",
			method:     http.MethodGet,
			statusCode: http.StatusOK,
			expected:   "session_read",
		},
		{
			name:       "更新会话",
			path:       "/api/v1/sessions/123",
			method:     http.MethodPut,
			statusCode: http.StatusOK,
			expected:   "session_update",
		},
		{
			name:       "删除会话",
			path:       "/api/v1/sessions/123",
			method:     http.MethodDelete,
			statusCode: http.StatusNoContent,
			expected:   "session_delete",
		},
		{
			name:       "创建记忆",
			path:       "/api/v1/memories",
			method:     http.MethodPost,
			statusCode: http.StatusCreated,
			expected:   "memory_create",
		},
		{
			name:       "读取记忆",
			path:       "/api/v1/memories/123",
			method:     http.MethodGet,
			statusCode: http.StatusOK,
			expected:   "memory_read",
		},
		{
			name:       "生成摘要",
			path:       "/api/v1/summaries",
			method:     http.MethodPost,
			statusCode: http.StatusOK,
			expected:   "summary_create",
		},
		{
			name:       "构建上下文",
			path:       "/api/v1/context",
			method:     http.MethodPost,
			statusCode: http.StatusOK,
			expected:   "context_build",
		},
		{
			name:       "对话生成",
			path:       "/api/v1/chat",
			method:     http.MethodPost,
			statusCode: http.StatusOK,
			expected:   "chat_generate",
		},
		{
			name:       "权限拒绝",
			path:       "/api/v1/sessions/123",
			method:     http.MethodGet,
			statusCode: http.StatusForbidden,
			expected:   "permission_denied",
		},
		{
			name:       "认证失败",
			path:       "/api/v1/sessions",
			method:     http.MethodGet,
			statusCode: http.StatusUnauthorized,
			expected:   "authentication_failed",
		},
		{
			name:       "默认请求",
			path:       "/api/v1/unknown",
			method:     http.MethodGet,
			statusCode: http.StatusOK,
			expected:   "api_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := determineEventType(req, tt.statusCode)
			if result != tt.expected {
				t.Errorf("determineEventType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldLogEvent(t *testing.T) {
	tests := []struct {
		name          string
		event         string
		enabledEvents []string
		expected      bool
	}{
		{
			name:          "空启用列表记录所有事件",
			event:         "session_create",
			enabledEvents: []string{},
			expected:      true,
		},
		{
			name:          "事件在启用列表中",
			event:         "session_create",
			enabledEvents: []string{"session_create", "session_delete"},
			expected:      true,
		},
		{
			name:          "事件不在启用列表中",
			event:         "session_read",
			enabledEvents: []string{"session_create", "session_delete"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldLogEvent(tt.event, tt.enabledEvents)
			if result != tt.expected {
				t.Errorf("shouldLogEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "包含子串",
			s:        "/api/v1/sessions/123",
			substr:   "/sessions",
			expected: true,
		},
		{
			name:     "不包含子串",
			s:        "/api/v1/memories/123",
			substr:   "/sessions",
			expected: false,
		},
		{
			name:     "空子串",
			s:        "/api/v1/sessions",
			substr:   "",
			expected: true,
		},
		{
			name:     "子串在开头",
			s:        "/api/v1/sessions",
			substr:   "/api",
			expected: true,
		},
		{
			name:     "子串在结尾",
			s:        "/api/v1/sessions",
			substr:   "sessions",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	// 创建一个测试响应记录器
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// 测试默认状态码
	if rw.statusCode != http.StatusOK {
		t.Errorf("默认状态码 = %v, want %v", rw.statusCode, http.StatusOK)
	}

	// 测试写入状态码
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("写入后状态码 = %v, want %v", rw.statusCode, http.StatusCreated)
	}

	// 验证底层 ResponseWriter 也收到了状态码
	if recorder.Code != http.StatusCreated {
		t.Errorf("底层 ResponseWriter 状态码 = %v, want %v", recorder.Code, http.StatusCreated)
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "找到子串",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "未找到子串",
			s:        "hello world",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "空子串",
			s:        "hello",
			substr:   "",
			expected: true,
		},
		{
			name:     "子串比字符串长",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "完全匹配",
			s:        "hello",
			substr:   "hello",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSubstring(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("findSubstring() = %v, want %v", result, tt.expected)
			}
		})
	}
}
