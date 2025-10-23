package auth

import (
	"context"

	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
)

// GetCreatorInfoFromContext 从 Context 中获取创建者信息
// 该函数从 JWT Claims 中提取用户ID和显示名称，用于在创建数据时自动填充创建者信息
//
// 返回值：
//   - userIDPtr: 用户ID指针，如果无法获取则返回 nil
//   - displayNamePtr: 显示名称指针，如果无法获取或为空则返回 nil
//
// 使用场景：
//   - 创建租户时自动设置 created_by 和 created_by_name
//   - 创建用户时自动设置 created_by 和 created_by_name
//   - 创建会话时自动设置 created_by 和 created_by_name
func GetCreatorInfoFromContext(ctx context.Context) (*uuid.UUID, *string) {
	// 从上下文中获取 JWT Claims
	claims, ok := GetJWTClaimsFromContext(ctx)
	if !ok {
		// JWT Claims 不存在，记录警告日志
		logger.WarnContext(ctx, "无法从上下文中获取 JWT Claims",
			logger.Fields{
				"reason": "JWT Claims 不存在于上下文中",
			})
		return nil, nil
	}

	// 解析用户ID（从 Subject 字段）
	var userIDPtr *uuid.UUID
	if claims.Subject != "" {
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			// 用户ID解析失败，记录警告日志
			logger.WarnContext(ctx, "无法解析用户ID",
				logger.Fields{
					"subject": claims.Subject,
					"error":   err.Error(),
				})
		} else {
			userIDPtr = &userID
		}
	} else {
		// Subject 字段为空
		logger.WarnContext(ctx, "JWT Claims 中的 Subject 字段为空")
	}

	// 提取显示名称（从 DisplayName 字段）
	var displayNamePtr *string
	if claims.DisplayName != "" {
		displayNamePtr = &claims.DisplayName
	} else {
		// DisplayName 为空，记录调试日志（这是正常情况，用户可能未设置显示名称）
		logger.DebugContext(ctx, "JWT Claims 中的 DisplayName 字段为空")
	}

	// 记录成功获取创建者信息
	if userIDPtr != nil || displayNamePtr != nil {
		logger.DebugContext(ctx, "成功从上下文中获取创建者信息",
			logger.Fields{
				"has_user_id":      userIDPtr != nil,
				"has_display_name": displayNamePtr != nil,
			})
	}

	return userIDPtr, displayNamePtr
}
