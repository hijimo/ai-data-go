package model

// ResponseData 通用响应数据结构
// 用于所有非分页接口的标准响应格式
type ResponseData[T any] struct {
	// 响应代码
	Code int `json:"code"`
	// 响应信息
	Message string `json:"message"`
	// 响应数据
	Data *T `json:"data,omitempty"`
}

// PaginationData 分页数据结构
type PaginationData[T any] struct {
	// 数据
	Data T `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo"`
	// 每页大小
	PageSize int `json:"pageSize"`
	// 总记录数
	TotalCount int `json:"totalCount"`
	// 总页数
	TotalPage int `json:"totalPage"`
}

// ResponsePaginationData 分页响应数据结构
// 用于所有分页列表接口的标准响应格式
type ResponsePaginationData[T any] struct {
	// 响应代码
	Code int `json:"code"`
	// 响应信息
	Message string `json:"message"`
	// 分页数据
	Data PaginationData[T] `json:"data"`
}
