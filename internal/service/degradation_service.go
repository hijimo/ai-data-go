package service

import (
	"context"

	"genkit-ai-service/internal/model"
)

// DegradationService 降级服务接口
// 当外部服务（AI服务、向量数据库等）不可用时，提供降级策略
type DegradationService interface {
	// DegradeAIService AI服务降级
	// 当AI服务不可用时，尝试从缓存获取响应或返回默认响应
	// 参数:
	//   - ctx: 上下文
	//   - sessionID: 会话ID
	//   - userQuery: 用户查询
	// 返回:
	//   - 降级响应内容
	//   - 错误信息
	DegradeAIService(ctx context.Context, sessionID, userQuery string) (string, error)

	// DegradeVectorSearch 向量检索降级
	// 当向量数据库不可用时，使用全文搜索或返回空结果
	// 参数:
	//   - ctx: 上下文
	//   - sessionID: 会话ID
	//   - query: 查询文本
	// 返回:
	//   - 记忆列表
	//   - 错误信息
	DegradeVectorSearch(ctx context.Context, sessionID, query string) ([]*model.ConversationMemory, error)

	// DegradeSummaryGeneration 摘要生成降级
	// 当AI服务不可用时，使用简单截断策略生成摘要
	// 参数:
	//   - ctx: 上下文
	//   - messages: 消息列表
	//   - targetLength: 目标长度（字符数）
	// 返回:
	//   - 摘要内容
	//   - 错误信息
	DegradeSummaryGeneration(ctx context.Context, messages []*model.ChatMessage, targetLength int) (string, error)
}
