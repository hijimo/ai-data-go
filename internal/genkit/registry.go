// Package genkit 提供 Genkit Flow 注册和管理功能
package genkit

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/genkit"

	"genkit-ai-service/internal/genkit/flows"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/internal/service/session"
)

// Registry Flow 注册器
// 负责管理和注册所有 Genkit Flow
type Registry struct {
	genkit   *genkit.Genkit
	services *Services
	logger   logger.Logger
}

// Services 服务集合
// 包含所有需要注册到 Flow 中的服务
type Services struct {
	// 上下文服务
	ContextService service.ContextService

	// 对话服务
	ChatService service.ChatService

	// 记忆服务
	MemoryService service.MemoryService

	// 摘要服务
	SummaryService session.SummaryService

	// Token 管理服务
	TokenManager service.TokenManager

	// 查询分类服务
	QueryClassifyService service.QueryClassifyService

	// 会话健康检查服务
	SessionHealthService service.SessionHealthService

	// 向量服务
	VectorService service.VectorService

	// 缓存服务
	CacheService service.CacheService

	// Repository
	GenkitMemoryRepo  repository.GenkitMemoryRepository
	GenkitContextRepo repository.GenkitContextRepository
	GenkitSummaryRepo repository.GenkitSummaryRepository
	SessionRepo       repository.SessionRepository
	MessageRepo       repository.MessageRepository
}

// NewRegistry 创建新的 Flow 注册器
// 参数:
//   - g: Genkit 实例
//   - services: 服务集合
//   - log: 日志记录器
//
// 返回:
//   - *Registry: Flow 注册器实例
func NewRegistry(g *genkit.Genkit, services *Services, log logger.Logger) *Registry {
	return &Registry{
		genkit:   g,
		services: services,
		logger:   log,
	}
}

// RegisterAllFlows 注册所有 Flow
// 按照功能模块依次注册各类 Flow
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - error: 注册过程中的错误
func (r *Registry) RegisterAllFlows(ctx context.Context) error {
	r.logger.InfoContext(ctx, "开始注册所有 Genkit Flow")

	// 1. 注册查询相关 Flow（不依赖服务，基于规则）
	r.logger.InfoContext(ctx, "注册查询相关 Flow")
	flows.RegisterQueryFlows(r.genkit)

	// 2. 注册上下文相关 Flow
	if r.services.ContextService != nil {
		r.logger.InfoContext(ctx, "注册上下文相关 Flow")
		flows.RegisterContextFlows(r.genkit, r.services.ContextService)
	} else {
		r.logger.WarnContext(ctx, "ContextService 未提供，跳过上下文 Flow 注册")
	}

	// 3. 注册查询分类 Flow
	if r.services.QueryClassifyService != nil {
		r.logger.InfoContext(ctx, "注册查询分类 Flow")
		flows.RegisterQueryClassifyFlows(r.genkit, r.services.QueryClassifyService)
	} else {
		r.logger.WarnContext(ctx, "QueryClassifyService 未提供，跳过查询分类 Flow 注册")
	}

	// 4. 注册对话相关 Flow
	if r.canRegisterChatFlows() {
		r.logger.InfoContext(ctx, "注册对话相关 Flow")
		chatServices := &flows.ChatFlowServices{
			ChatService:      r.services.ChatService,
			ContextService:   r.services.ContextService,
			MemoryService:    r.services.MemoryService,
			SessionRepo:      r.services.SessionRepo,
			MessageRepo:      r.services.MessageRepo,
			VectorService:    r.services.VectorService,
			CacheService:     r.services.CacheService,
			TokenManager:     r.services.TokenManager,
			SessionHealthSvc: r.services.SessionHealthService,
		}
		flows.RegisterChatFlows(r.genkit, chatServices)
		flows.RegisterChatStreamFlows(r.genkit, chatServices)
	} else {
		r.logger.WarnContext(ctx, "对话相关服务不完整，跳过对话 Flow 注册")
	}

	// 5. 注册记忆相关 Flow
	if r.canRegisterMemoryFlows() {
		r.logger.InfoContext(ctx, "注册记忆相关 Flow")
		flows.RegisterMemoryFlows(
			r.genkit,
			r.services.GenkitMemoryRepo,
			r.services.VectorService,
			r.services.CacheService,
			r.logger,
		)
	} else {
		r.logger.WarnContext(ctx, "记忆相关服务不完整，跳过记忆 Flow 注册")
	}

	// 6. 注册摘要相关 Flow
	if r.canRegisterSummaryFlows() {
		r.logger.InfoContext(ctx, "注册摘要相关 Flow")
		summaryServices := &flows.SummaryFlowServices{
			SummaryService:    r.services.SummaryService,
			MessageRepo:       r.services.MessageRepo,
			GenkitSummaryRepo: r.services.GenkitSummaryRepo,
			GenkitContextRepo: r.services.GenkitContextRepo,
			TokenManager:      r.services.TokenManager,
		}
		flows.RegisterSummaryFlows(r.genkit, summaryServices, r.logger)
	} else {
		r.logger.WarnContext(ctx, "摘要相关服务不完整，跳过摘要 Flow 注册")
	}

	// 7. 注册 Token 管理相关 Flow
	if r.services.TokenManager != nil {
		r.logger.InfoContext(ctx, "注册 Token 管理相关 Flow")
		flows.RegisterTokenFlows(r.genkit, r.services.TokenManager)
	} else {
		r.logger.WarnContext(ctx, "TokenManager 未提供，跳过 Token Flow 注册")
	}

	// 8. 注册健康检查相关 Flow
	if r.services.SessionHealthService != nil {
		r.logger.InfoContext(ctx, "注册健康检查相关 Flow")
		flows.RegisterHealthFlows(r.genkit, r.services.SessionHealthService)
	} else {
		r.logger.WarnContext(ctx, "SessionHealthService 未提供，跳过健康检查 Flow 注册")
	}

	r.logger.InfoContext(ctx, "所有 Genkit Flow 注册完成")
	return nil
}

// canRegisterChatFlows 检查是否可以注册对话相关 Flow
// 需要的服务: ChatService, ContextService, SessionRepo, MessageRepo
func (r *Registry) canRegisterChatFlows() bool {
	return r.services.ChatService != nil &&
		r.services.ContextService != nil &&
		r.services.SessionRepo != nil &&
		r.services.MessageRepo != nil
}

// canRegisterMemoryFlows 检查是否可以注册记忆相关 Flow
// 需要的服务: GenkitMemoryRepo, VectorService
func (r *Registry) canRegisterMemoryFlows() bool {
	return r.services.GenkitMemoryRepo != nil &&
		r.services.VectorService != nil
}

// canRegisterSummaryFlows 检查是否可以注册摘要相关 Flow
// 需要的服务: SummaryService, MessageRepo, GenkitSummaryRepo
func (r *Registry) canRegisterSummaryFlows() bool {
	return r.services.SummaryService != nil &&
		r.services.MessageRepo != nil &&
		r.services.GenkitSummaryRepo != nil
}

// LookupFlow 查找并返回指定名称的 Flow
// 这是一个泛型辅助方法，用于类型安全地查找 Flow
// 参数:
//   - flowName: Flow 名称
//
// 返回:
//   - *genkit.Flow[I, O]: 类型安全的 Flow 实例
//   - error: 查找失败时的错误
func (r *Registry) LookupFlow(flowName string) (*genkit.Flow[any, any], error) {
	flow := genkit.LookupFlow[any, any](r.genkit, flowName)
	if flow == nil {
		return nil, fmt.Errorf("Flow 不存在: %s", flowName)
	}
	return flow, nil
}

// ListRegisteredFlows 列出所有已注册的 Flow 名称
// 返回:
//   - []string: Flow 名称列表
func (r *Registry) ListRegisteredFlows() []string {
	// 根据已注册的服务返回 Flow 列表
	var flowNames []string

	// 查询相关 Flow（总是注册）
	flowNames = append(flowNames, "queryClassifyFlow")

	// 上下文相关 Flow
	if r.services.ContextService != nil {
		flowNames = append(flowNames,
			"contextBuildFlow",
			"contextOptimizeFlow",
		)
	}

	// 查询分类 Flow
	if r.services.QueryClassifyService != nil {
		flowNames = append(flowNames, "queryClassifyAIFlow")
	}

	// 对话相关 Flow
	if r.canRegisterChatFlows() {
		flowNames = append(flowNames,
			"chatGenerateFlow",
			"chatStreamFlow",
			"multiTurnChatFlow",
			"chatRetryFlow",
			"completeConversationFlow",
			"batchConversationFlow",
		)
	}

	// 记忆相关 Flow
	if r.canRegisterMemoryFlows() {
		flowNames = append(flowNames,
			"memorySearchFlow",
			"memoryStoreFlow",
			"memoryCleanupFlow",
		)
	}

	// 摘要相关 Flow
	if r.canRegisterSummaryFlows() {
		flowNames = append(flowNames,
			"summaryGenerateFlow",
			"summaryTriggerFlow",
			"summaryQualityFlow",
		)
	}

	// Token 管理相关 Flow
	if r.services.TokenManager != nil {
		flowNames = append(flowNames,
			"tokenBudgetFlow",
			"tokenOptimizeFlow",
			"tokenAnalysisFlow",
		)
	}

	// 健康检查相关 Flow
	if r.services.SessionHealthService != nil {
		flowNames = append(flowNames, "sessionHealthCheckFlow")
	}

	return flowNames
}

// GetFlowInfo 获取 Flow 的详细信息
// 参数:
//   - flowName: Flow 名称
//
// 返回:
//   - *FlowInfo: Flow 信息
//   - error: 获取失败时的错误
func (r *Registry) GetFlowInfo(flowName string) (*FlowInfo, error) {
	// 定义所有 Flow 的信息映射
	flowInfoMap := map[string]*FlowInfo{
		"queryClassifyFlow": {
			Name:        "queryClassifyFlow",
			Description: "查询分类 Flow - 基于规则分析用户查询类型",
			Category:    "query",
			InputType:   "QueryClassifyInput",
			OutputType:  "QueryClassifyOutput",
		},
		"contextBuildFlow": {
			Name:        "contextBuildFlow",
			Description: "上下文构建 Flow - 智能构建对话上下文",
			Category:    "context",
			InputType:   "ContextBuildInput",
			OutputType:  "ContextBuildOutput",
		},
		"contextOptimizeFlow": {
			Name:        "contextOptimizeFlow",
			Description: "上下文优化 Flow - 优化上下文以减少 Token 消耗",
			Category:    "context",
			InputType:   "ContextOptimizeInput",
			OutputType:  "ContextOptimizeOutput",
		},
		"queryClassifyAIFlow": {
			Name:        "queryClassifyAIFlow",
			Description: "AI 查询分类 Flow - 使用 AI 分析查询意图",
			Category:    "query",
			InputType:   "QueryClassifyInput",
			OutputType:  "QueryClassifyOutput",
		},
		"chatGenerateFlow": {
			Name:        "chatGenerateFlow",
			Description: "对话生成 Flow - 生成 AI 响应",
			Category:    "chat",
			InputType:   "ChatGenerateInput",
			OutputType:  "ChatGenerateOutput",
		},
		"chatStreamFlow": {
			Name:        "chatStreamFlow",
			Description: "流式对话 Flow - 实时流式返回 AI 响应",
			Category:    "chat",
			InputType:   "ChatStreamInput",
			OutputType:  "ChatStreamOutput",
		},
		"multiTurnChatFlow": {
			Name:        "multiTurnChatFlow",
			Description: "多轮对话管理 Flow - 管理多轮对话状态",
			Category:    "chat",
			InputType:   "MultiTurnChatInput",
			OutputType:  "MultiTurnChatOutput",
		},
		"chatRetryFlow": {
			Name:        "chatRetryFlow",
			Description: "对话重试 Flow - 处理 AI 生成失败的重试逻辑",
			Category:    "chat",
			InputType:   "ChatRetryInput",
			OutputType:  "ChatRetryOutput",
		},
		"completeConversationFlow": {
			Name:        "completeConversationFlow",
			Description: "完整对话流程 Flow - 编排完整的对话处理流程",
			Category:    "chat",
			InputType:   "CompleteConversationInput",
			OutputType:  "CompleteConversationOutput",
		},
		"batchConversationFlow": {
			Name:        "batchConversationFlow",
			Description: "批量对话处理 Flow - 并发处理多个对话请求",
			Category:    "chat",
			InputType:   "BatchConversationInput",
			OutputType:  "BatchConversationOutput",
		},
		"memorySearchFlow": {
			Name:        "memorySearchFlow",
			Description: "记忆检索 Flow - 基于向量相似度检索记忆",
			Category:    "memory",
			InputType:   "MemorySearchInput",
			OutputType:  "MemorySearchOutput",
		},
		"memoryStoreFlow": {
			Name:        "memoryStoreFlow",
			Description: "记忆存储 Flow - 存储对话记忆并生成向量",
			Category:    "memory",
			InputType:   "MemoryStoreInput",
			OutputType:  "MemoryStoreOutput",
		},
		"memoryCleanupFlow": {
			Name:        "memoryCleanupFlow",
			Description: "记忆清理 Flow - 清理过期或低质量的记忆",
			Category:    "memory",
			InputType:   "MemoryCleanupInput",
			OutputType:  "MemoryCleanupOutput",
		},
		"summaryGenerateFlow": {
			Name:        "summaryGenerateFlow",
			Description: "摘要生成 Flow - 生成对话摘要",
			Category:    "summary",
			InputType:   "SummaryGenerateInput",
			OutputType:  "SummaryGenerateOutput",
		},
		"summaryTriggerFlow": {
			Name:        "summaryTriggerFlow",
			Description: "摘要触发检查 Flow - 判断是否需要生成摘要",
			Category:    "summary",
			InputType:   "SummaryTriggerInput",
			OutputType:  "SummaryTriggerOutput",
		},
		"summaryQualityFlow": {
			Name:        "summaryQualityFlow",
			Description: "摘要质量评估 Flow - 评估摘要质量",
			Category:    "summary",
			InputType:   "SummaryQualityInput",
			OutputType:  "SummaryQualityOutput",
		},
		"tokenBudgetFlow": {
			Name:        "tokenBudgetFlow",
			Description: "Token 预算管理 Flow - 管理 Token 使用预算",
			Category:    "token",
			InputType:   "TokenBudgetInput",
			OutputType:  "TokenBudgetOutput",
		},
		"tokenOptimizeFlow": {
			Name:        "tokenOptimizeFlow",
			Description: "Token 优化 Flow - 优化内容以减少 Token 消耗",
			Category:    "token",
			InputType:   "TokenOptimizeInput",
			OutputType:  "TokenOptimizeOutput",
		},
		"tokenAnalysisFlow": {
			Name:        "tokenAnalysisFlow",
			Description: "Token 使用分析 Flow - 分析 Token 使用模式",
			Category:    "token",
			InputType:   "TokenAnalysisInput",
			OutputType:  "TokenAnalysisOutput",
		},
		"sessionHealthCheckFlow": {
			Name:        "sessionHealthCheckFlow",
			Description: "会话健康检查 Flow - 检查会话健康状态",
			Category:    "health",
			InputType:   "SessionHealthCheckInput",
			OutputType:  "SessionHealthCheckOutput",
		},
	}

	info, exists := flowInfoMap[flowName]
	if !exists {
		return nil, fmt.Errorf("Flow 信息不存在: %s", flowName)
	}

	return info, nil
}

// FlowInfo Flow 信息结构
type FlowInfo struct {
	Name        string // Flow 名称
	Description string // Flow 描述
	Category    string // Flow 分类
	InputType   string // 输入类型名称
	OutputType  string // 输出类型名称
}
