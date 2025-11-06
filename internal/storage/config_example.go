// internal/storage/config_example.go
package storage

import (
	"fmt"
	"os"
	"strconv"
)

// LoadQdrantConfigFromEnv 从环境变量加载 Qdrant 配置
// 支持两种配置方式：
// 1. Qdrant Cloud: 使用 QDRANT_ENDPOINT + QDRANT_ACCESS_KEY
// 2. 自托管: 使用 QDRANT_HOST + QDRANT_PORT + QDRANT_API_KEY
func LoadQdrantConfigFromEnv() (*QdrantConfig, error) {
	config := &QdrantConfig{}

	// 优先使用 Qdrant Cloud 配置
	if endpoint := os.Getenv("QDRANT_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
		config.APIKey = os.Getenv("QDRANT_ACCESS_KEY")
		config.ClusterID = os.Getenv("QDRANT_CLUSTER_ID") // 可选

		if config.APIKey == "" {
			return nil, fmt.Errorf("QDRANT_ACCESS_KEY 环境变量未设置")
		}

		return config, nil
	}

	// 使用自托管配置
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		return nil, fmt.Errorf("必须设置 QDRANT_ENDPOINT 或 QDRANT_HOST 环境变量")
	}

	config.Host = host

	// 解析端口
	if portStr := os.Getenv("QDRANT_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("无效的 QDRANT_PORT: %w", err)
		}
		config.Port = port
	} else {
		config.Port = 6333 // 默认端口
	}

	// API Key（可选，取决于是否启用认证）
	config.APIKey = os.Getenv("QDRANT_API_KEY")
	if config.APIKey == "" {
		// 如果没有设置 API Key，尝试使用 ACCESS_KEY
		config.APIKey = os.Getenv("QDRANT_ACCESS_KEY")
	}

	// TLS 配置
	if useTLSStr := os.Getenv("QDRANT_USE_TLS"); useTLSStr != "" {
		config.UseTLS = useTLSStr == "true" || useTLSStr == "1"
	}

	return config, nil
}

// 使用示例：
//
// 在 main.go 或初始化代码中：
//
// ```go
// import "genkit-ai-service/internal/storage"
//
// func initQdrant() (storage.QdrantClient, error) {
//     // 从环境变量加载配置
//     config, err := storage.LoadQdrantConfigFromEnv()
//     if err != nil {
//         return nil, fmt.Errorf("加载 Qdrant 配置失败: %w", err)
//     }
//
//     // 创建客户端
//     client, err := storage.NewQdrantClient(config)
//     if err != nil {
//         return nil, fmt.Errorf("创建 Qdrant 客户端失败: %w", err)
//     }
//
//     // 初始化 Collection
//     ctx := context.Background()
//     if err := client.InitializeCollection(ctx); err != nil {
//         return nil, fmt.Errorf("初始化 Collection 失败: %w", err)
//     }
//
//     return client, nil
// }
// ```
//
// .env 文件配置示例（Qdrant Cloud）：
//
// ```bash
// # Qdrant Cloud 配置
// QDRANT_ENDPOINT=https://37612f1c-dafd-48ab-afe7-7852d81a0868.us-west-2-0.aws.cloud.qdrant.io
// QDRANT_ACCESS_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// QDRANT_CLUSTER_ID=37612f1c-dafd-48ab-afe7-7852d81a0868
// ```
//
// .env 文件配置示例（自托管）：
//
// ```bash
// # 自托管 Qdrant 配置
// QDRANT_HOST=localhost
// QDRANT_PORT=6333
// QDRANT_API_KEY=your-api-key
// QDRANT_USE_TLS=false
// ```
