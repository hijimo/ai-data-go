package response

import (
	"context"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

const (
	// traceIDKey TraceID 上下文键
	// 使用字符串类型以确保与 middleware 包兼容
	traceIDKey = "traceId"
)

// getTraceID 从 Context 提取 TraceID
// 如果 TraceID 不存在，返回空字符串
func getTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	// 使用字符串类型的键
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	
	return ""
}

// Success 构建成功响应
func Success[T any](data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    errors.CodeSuccess,
		Message: errors.MsgSuccess,
		Data:    data,
	}
}

// SuccessWithMessage 构建带自定义消息的成功响应
func SuccessWithMessage[T any](message string, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    errors.CodeSuccess,
		Message: message,
		Data:    data,
	}
}

// Error 构建错误响应
func Error[T any](code int, message string) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// ErrorWithData 构建带数据的错误响应（如验证错误详情）
func ErrorWithData[T any](code int, message string, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// FromAppError 从 AppError 构建响应
func FromAppError[T any](err *errors.AppError) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    err.Code,
		Message: err.Message,
		Data:    nil,
	}
}

// FromAppErrorWithData 从 AppError 构建带数据的响应
func FromAppErrorWithData[T any](err *errors.AppError, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    err.Code,
		Message: err.Message,
		Data:    data,
	}
}

// Pagination 构建分页响应
func Pagination[T any](data T, pageNo, pageSize, totalCount int) model.ResponsePaginationData[T] {
	totalPage := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPage++
	}

	return model.ResponsePaginationData[T]{
		Code:    errors.CodeSuccess,
		Message: errors.MsgSuccess,
		Data: model.PaginationData[T]{
			Data:       data,
			PageNo:     pageNo,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPage:  totalPage,
		},
	}
}

// PaginationWithMessage 构建带自定义消息的分页响应
func PaginationWithMessage[T any](message string, data T, pageNo, pageSize, totalCount int) model.ResponsePaginationData[T] {
	totalPage := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPage++
	}

	return model.ResponsePaginationData[T]{
		Code:    errors.CodeSuccess,
		Message: message,
		Data: model.PaginationData[T]{
			Data:       data,
			PageNo:     pageNo,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPage:  totalPage,
		},
	}
}

// PaginationError 构建分页错误响应
func PaginationError[T any](code int, message string) model.ResponsePaginationData[T] {
	return model.ResponsePaginationData[T]{
		Code:    code,
		Message: message,
		Data: model.PaginationData[T]{
			Data:       *new(T),
			PageNo:     0,
			PageSize:   0,
			TotalCount: 0,
			TotalPage:  0,
		},
	}
}

// SuccessWithContext 构建成功响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func SuccessWithContext[T any](ctx context.Context, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    errors.CodeSuccess,
		Message: errors.MsgSuccess,
		Data:    data,
		TraceID: getTraceID(ctx),
	}
}

// SuccessWithMessageContext 构建带自定义消息的成功响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func SuccessWithMessageContext[T any](ctx context.Context, message string, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    errors.CodeSuccess,
		Message: message,
		Data:    data,
		TraceID: getTraceID(ctx),
	}
}

// ErrorWithContext 构建错误响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func ErrorWithContext[T any](ctx context.Context, code int, message string) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    code,
		Message: message,
		Data:    nil,
		TraceID: getTraceID(ctx),
	}
}

// ErrorWithDataContext 构建带数据的错误响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func ErrorWithDataContext[T any](ctx context.Context, code int, message string, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: getTraceID(ctx),
	}
}

// FromAppErrorContext 从 AppError 构建响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func FromAppErrorContext[T any](ctx context.Context, err *errors.AppError) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    err.Code,
		Message: err.Message,
		Data:    nil,
		TraceID: getTraceID(ctx),
	}
}

// FromAppErrorWithDataContext 从 AppError 构建带数据的响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func FromAppErrorWithDataContext[T any](ctx context.Context, err *errors.AppError, data *T) model.ResponseData[T] {
	return model.ResponseData[T]{
		Code:    err.Code,
		Message: err.Message,
		Data:    data,
		TraceID: getTraceID(ctx),
	}
}

// PaginationWithContext 构建分页响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func PaginationWithContext[T any](ctx context.Context, data T, pageNo, pageSize, totalCount int) model.ResponsePaginationData[T] {
	totalPage := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPage++
	}

	return model.ResponsePaginationData[T]{
		Code:    errors.CodeSuccess,
		Message: errors.MsgSuccess,
		Data: model.PaginationData[T]{
			Data:       data,
			PageNo:     pageNo,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPage:  totalPage,
		},
		TraceID: getTraceID(ctx),
	}
}

// PaginationWithMessageContext 构建带自定义消息的分页响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func PaginationWithMessageContext[T any](ctx context.Context, message string, data T, pageNo, pageSize, totalCount int) model.ResponsePaginationData[T] {
	totalPage := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPage++
	}

	return model.ResponsePaginationData[T]{
		Code:    errors.CodeSuccess,
		Message: message,
		Data: model.PaginationData[T]{
			Data:       data,
			PageNo:     pageNo,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPage:  totalPage,
		},
		TraceID: getTraceID(ctx),
	}
}

// PaginationErrorContext 构建分页错误响应（带 Context）
// 自动从 Context 提取 TraceID 并注入到响应结构
func PaginationErrorContext[T any](ctx context.Context, code int, message string) model.ResponsePaginationData[T] {
	return model.ResponsePaginationData[T]{
		Code:    code,
		Message: message,
		Data: model.PaginationData[T]{
			Data:       *new(T),
			PageNo:     0,
			PageSize:   0,
			TotalCount: 0,
			TotalPage:  0,
		},
		TraceID: getTraceID(ctx),
	}
}
