package storage

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestOSSClientIntegration 集成测试（需要真实的OSS配置）
func TestOSSClientIntegration(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=true 来运行")
	}

	// 检查必要的环境变量
	accessKeyID := os.Getenv("OSS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("OSS_BUCKET_NAME")
	region := os.Getenv("OSS_REGION")

	if accessKeyID == "" || accessKeySecret == "" || bucketName == "" || region == "" {
		t.Skip("跳过集成测试，缺少必要的环境变量")
	}

	// 创建OSS客户端
	config := &OSSConfig{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		BucketName:      bucketName,
		Region:          region,
	}

	client, err := NewOSSClient(config)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()
	projectID := uuid.New()
	filename := "test-file.txt"
	content := "这是一个测试文件内容"
	
	// 创建模拟的multipart.File
	reader := strings.NewReader(content)
	
	// 注意：这里需要创建一个真实的multipart.File用于测试
	// 在实际集成测试中，你需要使用真实的文件上传
	t.Log("集成测试需要真实的multipart.File，这里仅作为示例")
}

// TestOSSClientFromEnv 测试从环境变量创建客户端
func TestOSSClientFromEnv(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=true 来运行")
	}

	bucketName := os.Getenv("OSS_BUCKET_NAME")
	region := os.Getenv("OSS_REGION")

	if bucketName == "" || region == "" {
		t.Skip("跳过集成测试，缺少必要的环境变量")
	}

	client, err := NewOSSClientFromEnv(bucketName, region)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

// TestMockOSSClient 测试Mock OSS客户端
func TestMockOSSClient(t *testing.T) {
	client := NewMockOSSClient()
	assert.NotNil(t, client)

	ctx := context.Background()
	projectID := uuid.New()
	filename := "test-file.txt"
	content := "这是一个测试文件内容"
	
	// 创建模拟的multipart.File
	reader := strings.NewReader(content)
	
	// 注意：在实际测试中需要创建真实的multipart.File
	// 这里仅测试Mock客户端的基本功能
	
	// 测试文件不存在
	exists, err := client.CheckFileExists(ctx, "non-existent-file")
	assert.NoError(t, err)
	assert.False(t, exists)
	
	// 测试获取不存在文件的URL
	_, err = client.GetFileURL(ctx, "non-existent-file", time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "文件不存在")
	
	// 测试删除不存在的文件（应该成功）
	err = client.DeleteFile(ctx, "non-existent-file")
	assert.NoError(t, err)
}