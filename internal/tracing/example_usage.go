// Package tracing 提供分布式追踪功能
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// 示例：在 Flow 中使用追踪

// ExampleContextBuildFlow 示例：追踪上下文构建 Flow
func ExampleContextBuildFlow(ctx context.Context, tracer Tracer, sessionID string) error {
	return tracer.TraceFlow(ctx, "contextBuildFlow", func(ctx context.Context) error {
		// 1. 追踪数据库查询
		err := TraceDBQuery(ctx, "get_recent_messages",
			"SELECT * FROM conversation_messages WHERE session_id = $1 ORDER BY created_at DESC LIMIT 10",
			func(ctx context.Context) error {
				// 执行数据库查询
				return nil
			},
		)
		if err != nil {
			return err
		}

		// 2. 追踪向量检索
		err = TraceVectorSearch(ctx, sessionID, 5, func(ctx context.Context) error {
			// 执行向量检索
			return nil
		})
		if err != nil {
			return err
		}

		// 3. 追踪缓存操作
		err = TraceCacheOperation(ctx, "get", fmt.Sprintf("context:%s", sessionID), func(ctx context.Context) error {
			// 从缓存获取数据
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
}

// ExampleChatGenerateFlow 示例：追踪对话生成 Flow
func ExampleChatGenerateFlow(ctx context.Context, tracer Tracer, model string, promptTokens int) error {
	return tracer.TraceFlow(ctx, "chatGenerateFlow", func(ctx context.Context) error {
		// 1. 追踪上下文构建
		err := tracer.TraceFlow(ctx, "buildContext", func(ctx context.Context) error {
			// 构建上下文
			return nil
		})
		if err != nil {
			return err
		}

		// 2. 追踪 AI 生成
		err = TraceAIGeneration(ctx, model, promptTokens, func(ctx context.Context) error {
			// 调用 AI 生成
			return nil
		})
		if err != nil {
			return err
		}

		// 3. 追踪消息保存
		err = TraceDBQuery(ctx, "save_message",
			"INSERT INTO conversation_messages (session_id, role, content) VALUES ($1, $2, $3)",
			func(ctx context.Context) error {
				// 保存消息
				return nil
			},
		)
		if err != nil {
			return err
		}

		return nil
	})
}

// ExampleMemorySearchFlow 示例：追踪记忆检索 Flow
func ExampleMemorySearchFlow(ctx context.Context, tracer Tracer, sessionID string, query string) error {
	return tracer.TraceFlow(ctx, "memorySearchFlow", func(ctx context.Context) error {
		// 1. 追踪向量生成
		err := TraceOperation(ctx, "vector.generate", func(ctx context.Context) error {
			// 生成查询向量
			return nil
		}, attribute.String("query", query))
		if err != nil {
			return err
		}

		// 2. 追踪向量检索
		err = TraceVectorSearch(ctx, sessionID, 10, func(ctx context.Context) error {
			// 执行向量检索
			return nil
		})
		if err != nil {
			return err
		}

		// 3. 追踪访问统计更新
		err = TraceDBQuery(ctx, "update_access_stats",
			"UPDATE conversation_memories SET access_count = access_count + 1 WHERE id = ANY($1)",
			func(ctx context.Context) error {
				// 更新访问统计
				return nil
			},
		)
		if err != nil {
			return err
		}

		return nil
	})
}

// ExampleSummaryGenerateFlow 示例：追踪摘要生成 Flow
func ExampleSummaryGenerateFlow(ctx context.Context, tracer Tracer, sessionID string) error {
	return tracer.TraceFlow(ctx, "summaryGenerateFlow", func(ctx context.Context) error {
		// 1. 追踪消息查询
		err := TraceDBQuery(ctx, "get_messages_for_summary",
			"SELECT * FROM conversation_messages WHERE session_id = $1 AND created_at > $2",
			func(ctx context.Context) error {
				// 查询消息
				return nil
			},
		)
		if err != nil {
			return err
		}

		// 2. 追踪 AI 摘要生成
		err = TraceAIGeneration(ctx, "gemini-2.5-flash", 1000, func(ctx context.Context) error {
			// 生成摘要
			return nil
		})
		if err != nil {
			return err
		}

		// 3. 追踪摘要保存
		err = TraceDBQuery(ctx, "save_summary",
			"INSERT INTO conversation_summaries (session_id, content, token_count) VALUES ($1, $2, $3)",
			func(ctx context.Context) error {
				// 保存摘要
				return nil
			},
		)
		if err != nil {
			return err
		}

		// 4. 追踪缓存失效
		err = TraceCacheOperation(ctx, "delete", fmt.Sprintf("summary:%s:*", sessionID), func(ctx context.Context) error {
			// 删除相关缓存
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
}

// ExampleNestedSpans 示例：嵌套 span
func ExampleNestedSpans(ctx context.Context, tracer Tracer) error {
	return tracer.TraceFlow(ctx, "parentFlow", func(ctx context.Context) error {
		// 创建子 span
		ctx, span1 := tracer.StartSpan(ctx, "step1")
		// 执行步骤 1
		span1.End()

		// 创建另一个子 span
		ctx, span2 := tracer.StartSpan(ctx, "step2")
		// 执行步骤 2
		span2.End()

		// 创建嵌套的子 span
		err := TraceOperation(ctx, "step3", func(ctx context.Context) error {
			// 在 step3 中创建更深层的 span
			return TraceOperation(ctx, "step3.substep", func(ctx context.Context) error {
				// 执行子步骤
				return nil
			})
		})

		return err
	})
}
