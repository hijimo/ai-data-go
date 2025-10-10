package handler

import (
	"context"
	"net/http"
	"time"

	"ai-knowledge-platform/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
	Time     string            `json:"time"`
}

// HealthCheck 健康检查处理器
// @Summary 健康检查
// @Description 检查服务和依赖组件的健康状态
// @Tags 系统
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} map[string]interface{}
// @Router /health [get]
func HealthCheck(db *gorm.DB, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		services := make(map[string]string)
		overallStatus := "healthy"

		// 检查数据库连接
		if err := database.HealthCheck(); err != nil {
			services["database"] = "unhealthy: " + err.Error()
			overallStatus = "unhealthy"
		} else {
			services["database"] = "healthy"
		}

		// 检查Redis连接
		if err := redisClient.Ping(ctx).Err(); err != nil {
			services["redis"] = "unhealthy: " + err.Error()
			overallStatus = "unhealthy"
		} else {
			services["redis"] = "healthy"
		}

		response := HealthResponse{
			Status:   overallStatus,
			Services: services,
			Time:     time.Now().Format(time.RFC3339),
		}

		if overallStatus == "healthy" {
			c.JSON(http.StatusOK, response)
		} else {
			c.JSON(http.StatusServiceUnavailable, response)
		}
	}
}