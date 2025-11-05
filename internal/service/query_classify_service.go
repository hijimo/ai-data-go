package service

import (
	"context"
)

// QueryClassifyService 查询分类服务接口
type QueryClassifyService interface {
	// ClassifyQuery 分类查询
	ClassifyQuery(ctx context.Context, req ClassifyQueryRequest) (*ClassifyQueryResult, error)
}

// ClassifyQueryRequest 分类查询请求
type ClassifyQueryRequest struct {
	Query     string
	SessionID string
}

// ClassifyQueryResult 分类查询结果
type ClassifyQueryResult struct {
	QueryType           string
	NeedsHistory        bool
	KeyEntities         []string
	RecommendedStrategy string
	Confidence          float64
	Reasoning           string
}

// queryClassifyServiceImpl 查询分类服务实现（占位符）
type queryClassifyServiceImpl struct {
	// TODO: 添加依赖
}

// NewQueryClassifyService 创建查询分类服务实例
func NewQueryClassifyService() QueryClassifyService {
	return &queryClassifyServiceImpl{}
}

// ClassifyQuery 分类查询（占位符实现）
func (s *queryClassifyServiceImpl) ClassifyQuery(ctx context.Context, req ClassifyQueryRequest) (*ClassifyQueryResult, error) {
	// TODO: 实现实际的查询分类逻辑
	return &ClassifyQueryResult{
		QueryType:           "simple_question",
		NeedsHistory:        false,
		KeyEntities:         []string{},
		RecommendedStrategy: "auto",
		Confidence:          0.8,
		Reasoning:           "这是一个简单的问题",
	}, nil
}
