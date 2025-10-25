package response

import (
	"context"
	"testing"

	"genkit-ai-service/pkg/errors"
)

// TestGetTraceID 测试从 Context 提取 TraceID
func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "正常获取 TraceID",
			ctx:      context.WithValue(context.Background(), traceIDKey, "trace-1704067200-a3f9k2"),
			expected: "trace-1704067200-a3f9k2",
		},
		{
			name:     "Context 中无 TraceID",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "Context 为 nil",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "TraceID 类型错误",
			ctx:      context.WithValue(context.Background(), traceIDKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTraceID(tt.ctx)
			if result != tt.expected {
				t.Errorf("期望 %s，实际 %s", tt.expected, result)
			}
		})
	}
}

// TestSuccessWithContext 测试带 Context 的成功响应
func TestSuccessWithContext(t *testing.T) {
	type TestData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	data := &TestData{ID: "123", Name: "test"}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-test-123")

	resp := SuccessWithContext(ctx, data)

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeSuccess, resp.Code)
	}
	if resp.Message != errors.MsgSuccess {
		t.Errorf("期望 Message 为 %s，实际为 %s", errors.MsgSuccess, resp.Message)
	}
	if resp.TraceID != "trace-test-123" {
		t.Errorf("期望 TraceID 为 trace-test-123，实际为 %s", resp.TraceID)
	}
	if resp.Data == nil || resp.Data.ID != "123" {
		t.Error("Data 不正确")
	}
}

// TestSuccessWithContextNoTraceID 测试无 TraceID 的情况
func TestSuccessWithContextNoTraceID(t *testing.T) {
	type TestData struct {
		ID string `json:"id"`
	}

	data := &TestData{ID: "123"}
	ctx := context.Background()

	resp := SuccessWithContext(ctx, data)

	if resp.TraceID != "" {
		t.Errorf("期望 TraceID 为空，实际为 %s", resp.TraceID)
	}
	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeSuccess, resp.Code)
	}
}

// TestSuccessWithMessageContext 测试带自定义消息和 Context 的成功响应
func TestSuccessWithMessageContext(t *testing.T) {
	type TestData struct {
		Count int `json:"count"`
	}

	data := &TestData{Count: 42}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-msg-456")
	customMsg := "操作成功完成"

	resp := SuccessWithMessageContext(ctx, customMsg, data)

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeSuccess, resp.Code)
	}
	if resp.Message != customMsg {
		t.Errorf("期望 Message 为 %s，实际为 %s", customMsg, resp.Message)
	}
	if resp.TraceID != "trace-msg-456" {
		t.Errorf("期望 TraceID 为 trace-msg-456，实际为 %s", resp.TraceID)
	}
	if resp.Data == nil || resp.Data.Count != 42 {
		t.Error("Data 不正确")
	}
}

// TestErrorWithContext 测试带 Context 的错误响应
func TestErrorWithContext(t *testing.T) {
	type TestData struct {
		ID string `json:"id"`
	}

	ctx := context.WithValue(context.Background(), traceIDKey, "trace-error-789")

	resp := ErrorWithContext[TestData](ctx, errors.CodeBadRequest, "请求参数错误")

	if resp.Code != errors.CodeBadRequest {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeBadRequest, resp.Code)
	}
	if resp.Message != "请求参数错误" {
		t.Errorf("期望 Message 为 '请求参数错误'，实际为 %s", resp.Message)
	}
	if resp.TraceID != "trace-error-789" {
		t.Errorf("期望 TraceID 为 trace-error-789，实际为 %s", resp.TraceID)
	}
	if resp.Data != nil {
		t.Error("错误响应的 Data 应该为 nil")
	}
}

// TestErrorWithDataContext 测试带数据和 Context 的错误响应
func TestErrorWithDataContext(t *testing.T) {
	type ValidationError struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	}

	data := &ValidationError{Field: "email", Message: "邮箱格式不正确"}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-validation-001")

	resp := ErrorWithDataContext(ctx, errors.CodeBadRequest, "验证失败", data)

	if resp.Code != errors.CodeBadRequest {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeBadRequest, resp.Code)
	}
	if resp.TraceID != "trace-validation-001" {
		t.Errorf("期望 TraceID 为 trace-validation-001，实际为 %s", resp.TraceID)
	}
	if resp.Data == nil || resp.Data.Field != "email" {
		t.Error("Data 不正确")
	}
}

// TestFromAppErrorContext 测试从 AppError 构建响应（带 Context）
func TestFromAppErrorContext(t *testing.T) {
	type TestData struct {
		ID string `json:"id"`
	}

	appErr := errors.NewBadRequestError("无效的请求参数")
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-apperr-002")

	resp := FromAppErrorContext[TestData](ctx, appErr)

	if resp.Code != appErr.Code {
		t.Errorf("期望 Code 为 %d，实际为 %d", appErr.Code, resp.Code)
	}
	if resp.Message != appErr.Message {
		t.Errorf("期望 Message 为 %s，实际为 %s", appErr.Message, resp.Message)
	}
	if resp.TraceID != "trace-apperr-002" {
		t.Errorf("期望 TraceID 为 trace-apperr-002，实际为 %s", resp.TraceID)
	}
	if resp.Data != nil {
		t.Error("Data 应该为 nil")
	}
}

// TestFromAppErrorWithDataContext 测试从 AppError 构建带数据的响应（带 Context）
func TestFromAppErrorWithDataContext(t *testing.T) {
	type ErrorDetail struct {
		Code    string `json:"code"`
		Details string `json:"details"`
	}

	appErr := errors.NewInternalError(nil)
	data := &ErrorDetail{Code: "DB_ERROR", Details: "数据库连接失败"}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-apperr-003")

	resp := FromAppErrorWithDataContext(ctx, appErr, data)

	if resp.Code != appErr.Code {
		t.Errorf("期望 Code 为 %d，实际为 %d", appErr.Code, resp.Code)
	}
	if resp.TraceID != "trace-apperr-003" {
		t.Errorf("期望 TraceID 为 trace-apperr-003，实际为 %s", resp.TraceID)
	}
	if resp.Data == nil || resp.Data.Code != "DB_ERROR" {
		t.Error("Data 不正确")
	}
}

// TestPaginationWithContext 测试带 Context 的分页响应
func TestPaginationWithContext(t *testing.T) {
	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	users := []User{
		{ID: "1", Name: "User 1"},
		{ID: "2", Name: "User 2"},
	}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-page-001")

	resp := PaginationWithContext(ctx, users, 1, 10, 25)

	if resp.Code != errors.CodeSuccess {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeSuccess, resp.Code)
	}
	if resp.TraceID != "trace-page-001" {
		t.Errorf("期望 TraceID 为 trace-page-001，实际为 %s", resp.TraceID)
	}
	if resp.Data.PageNo != 1 {
		t.Errorf("期望 PageNo 为 1，实际为 %d", resp.Data.PageNo)
	}
	if resp.Data.PageSize != 10 {
		t.Errorf("期望 PageSize 为 10，实际为 %d", resp.Data.PageSize)
	}
	if resp.Data.TotalCount != 25 {
		t.Errorf("期望 TotalCount 为 25，实际为 %d", resp.Data.TotalCount)
	}
	if resp.Data.TotalPage != 3 {
		t.Errorf("期望 TotalPage 为 3，实际为 %d", resp.Data.TotalPage)
	}
	if len(resp.Data.Data) != 2 {
		t.Errorf("期望 Data 长度为 2，实际为 %d", len(resp.Data.Data))
	}
}

// TestPaginationWithMessageContext 测试带自定义消息和 Context 的分页响应
func TestPaginationWithMessageContext(t *testing.T) {
	type Item struct {
		ID string `json:"id"`
	}

	items := []Item{{ID: "1"}}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-page-002")
	customMsg := "查询成功"

	resp := PaginationWithMessageContext(ctx, customMsg, items, 2, 5, 12)

	if resp.Message != customMsg {
		t.Errorf("期望 Message 为 %s，实际为 %s", customMsg, resp.Message)
	}
	if resp.TraceID != "trace-page-002" {
		t.Errorf("期望 TraceID 为 trace-page-002，实际为 %s", resp.TraceID)
	}
	if resp.Data.TotalPage != 3 {
		t.Errorf("期望 TotalPage 为 3，实际为 %d", resp.Data.TotalPage)
	}
}

// TestPaginationErrorContext 测试带 Context 的分页错误响应
func TestPaginationErrorContext(t *testing.T) {
	type User struct {
		ID string `json:"id"`
	}

	ctx := context.WithValue(context.Background(), traceIDKey, "trace-page-err-001")

	resp := PaginationErrorContext[[]User](ctx, errors.CodeBadRequest, "无效的分页参数")

	if resp.Code != errors.CodeBadRequest {
		t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeBadRequest, resp.Code)
	}
	if resp.Message != "无效的分页参数" {
		t.Errorf("期望 Message 为 '无效的分页参数'，实际为 %s", resp.Message)
	}
	if resp.TraceID != "trace-page-err-001" {
		t.Errorf("期望 TraceID 为 trace-page-err-001，实际为 %s", resp.TraceID)
	}
	if resp.Data.TotalCount != 0 {
		t.Errorf("期望 TotalCount 为 0，实际为 %d", resp.Data.TotalCount)
	}
}

// TestBackwardCompatibility 测试向后兼容性（不带 Context 的函数）
func TestBackwardCompatibility(t *testing.T) {
	type TestData struct {
		Value string `json:"value"`
	}

	t.Run("Success 不包含 TraceID", func(t *testing.T) {
		data := &TestData{Value: "test"}
		resp := Success(data)

		if resp.TraceID != "" {
			t.Errorf("期望 TraceID 为空，实际为 %s", resp.TraceID)
		}
		if resp.Code != errors.CodeSuccess {
			t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeSuccess, resp.Code)
		}
	})

	t.Run("Error 不包含 TraceID", func(t *testing.T) {
		resp := Error[TestData](errors.CodeBadRequest, "错误")

		if resp.TraceID != "" {
			t.Errorf("期望 TraceID 为空，实际为 %s", resp.TraceID)
		}
		if resp.Code != errors.CodeBadRequest {
			t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeBadRequest, resp.Code)
		}
	})

	t.Run("Pagination 不包含 TraceID", func(t *testing.T) {
		data := []TestData{{Value: "test"}}
		resp := Pagination(data, 1, 10, 15)

		if resp.TraceID != "" {
			t.Errorf("期望 TraceID 为空，实际为 %s", resp.TraceID)
		}
		if resp.Code != errors.CodeSuccess {
			t.Errorf("期望 Code 为 %d，实际为 %d", errors.CodeSuccess, resp.Code)
		}
	})
}

// TestPaginationTotalPageCalculation 测试分页总页数计算
func TestPaginationTotalPageCalculation(t *testing.T) {
	type Item struct {
		ID string `json:"id"`
	}

	tests := []struct {
		name          string
		totalCount    int
		pageSize      int
		expectedPages int
	}{
		{"整除情况", 20, 10, 2},
		{"有余数情况", 25, 10, 3},
		{"单页情况", 5, 10, 1},
		{"空数据情况", 0, 10, 0},
		{"刚好一页", 10, 10, 1},
	}

	ctx := context.WithValue(context.Background(), traceIDKey, "trace-calc-test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []Item{}
			resp := PaginationWithContext(ctx, items, 1, tt.pageSize, tt.totalCount)

			if resp.Data.TotalPage != tt.expectedPages {
				t.Errorf("期望 TotalPage 为 %d，实际为 %d", tt.expectedPages, resp.Data.TotalPage)
			}
		})
	}
}

// BenchmarkSuccessWithContext 性能测试：带 Context 的成功响应
func BenchmarkSuccessWithContext(b *testing.B) {
	type TestData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	data := &TestData{ID: "123", Name: "test"}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-bench-001")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SuccessWithContext(ctx, data)
	}
}

// BenchmarkGetTraceID 性能测试：TraceID 提取
func BenchmarkGetTraceID(b *testing.B) {
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-bench-002")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getTraceID(ctx)
	}
}

// BenchmarkPaginationWithContext 性能测试：带 Context 的分页响应
func BenchmarkPaginationWithContext(b *testing.B) {
	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	users := []User{
		{ID: "1", Name: "User 1"},
		{ID: "2", Name: "User 2"},
	}
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-bench-003")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PaginationWithContext(ctx, users, 1, 10, 25)
	}
}
