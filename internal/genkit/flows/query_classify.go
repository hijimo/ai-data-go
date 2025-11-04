// Package flows 实现查询分类相关的 Genkit Flow
package flows

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/genkit"

	"genkit-ai-service/internal/service"
)

// RegisterQueryClassifyFlows 注册查询分类相关的 Flow
func RegisterQueryClassifyFlows(g *genkit.Genkit, classifySvc service.QueryClassifyService) {
	genkit.DefineFlow(
		g,
		"queryClassifyFlow",
		func(ctx context.Context, input QueryClassifyInput) (QueryClassifyOutput, error) {
			// 1. 参数验证
			if err := validateQueryClassifyInput(input); err != nil {
				return QueryClassifyOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 调用服务层进行分类
			result, err := classifySvc.Classify(ctx, &service.ClassifyRequest{
				Query:          input.Query,
				SessionID:      input.SessionID,
				RecentMessages: input.RecentMessages,
			})
			if err != nil {
				return QueryClassifyOutput{}, fmt.Errorf("查询分类失败: %w", err)
			}

			// 3. 转换为输出格式
			output := QueryClassifyOutput{
				QueryType:           result.QueryType,
				Intent:              result.Intent,
				NeedsHistory:        result.NeedsHistory,
				NeedsLongTerm:       result.NeedsLongTerm,
				RecommendedStrategy: result.RecommendedStrategy,
				Confidence:          result.Confidence,
				Entities:            result.Entities,
			}

			return output, nil
		},
	)
}

// validateQueryClassifyInput 验证输入参数
func validateQueryClassifyInput(input QueryClassifyInput) error {
	if err := validate.Struct(input); err != nil {
		return err
	}

	// 额外的业务验证
	if len(input.Query) == 0 {
		return fmt.Errorf("query 不能为空")
	}

	if len(input.Query) > 2000 {
		return fmt.Errorf("query 长度不能超过 2000 字符")
	}

	if len(input.RecentMessages) > 5 {
		return fmt.Errorf("recentMessages 数量不能超过 5 条")
	}

	return nil
}
