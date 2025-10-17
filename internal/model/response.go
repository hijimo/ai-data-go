package model

// ResponseData 通用响应数据结构
// 用于所有非分页接口的标准响应格式
type ResponseData[T any] struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"success"`
	// 响应数据
	Data *T `json:"data,omitempty"`
}

// PaginationData 分页数据结构
type PaginationData[T any] struct {
	// 数据
	Data T `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// ResponsePaginationData 分页响应数据结构
// 用于所有分页列表接口的标准响应格式
type ResponsePaginationData[T any] struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"success"`
	// 分页数据
	Data PaginationData[T] `json:"data"`
}

// ErrorResponse 错误响应结构（用于 Swagger 文档）
type ErrorResponse struct {
	// 响应代码
	Code int `json:"code" example:"400"`
	// 响应信息
	Message string `json:"message" example:"请求参数错误"`
}

// EmptyData 空数据结构（用于无数据返回的成功响应）
type EmptyData struct{}

// SuccessResponse 成功响应结构（无数据）
type SuccessResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
}
