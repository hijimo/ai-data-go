package service_test

import (
	"fmt"
	"log"
	"os"

	"genkit-ai-service/internal/service"
)

// ExampleEncryptionService_basic 演示加密服务的基本使用
func ExampleEncryptionService_basic() {
	// 设置环境变量（实际使用中应在.env文件中配置）
	os.Setenv("ENCRYPTION_SECRET_KEY", "12345678901234567890123456789012")
	defer os.Unsetenv("ENCRYPTION_SECRET_KEY")

	// 创建加密服务
	encryptionService, err := service.NewEncryptionServiceFromEnv()
	if err != nil {
		log.Fatalf("创建加密服务失败: %v", err)
	}

	// 原始API密钥
	originalKey := "sk-test-api-key-12345"

	// 加密
	encrypted, err := encryptionService.EncryptAPIKey(originalKey)
	if err != nil {
		log.Fatalf("加密失败: %v", err)
	}

	// 解密
	decrypted, err := encryptionService.DecryptAPIKey(encrypted)
	if err != nil {
		log.Fatalf("解密失败: %v", err)
	}

	// 验证
	if decrypted == originalKey {
		fmt.Println("加密解密成功")
	}

	// Output: 加密解密成功
}

// ExampleEncryptionService_mask 演示密钥脱敏功能
func ExampleEncryptionService_mask() {
	secretKey := []byte("12345678901234567890123456789012")
	encryptionService, _ := service.NewEncryptionService(secretKey)

	// 脱敏不同长度的密钥
	fmt.Println(encryptionService.MaskAPIKey("sk-test-api-key-12345"))
	fmt.Println(encryptionService.MaskAPIKey("short"))
	fmt.Println(encryptionService.MaskAPIKey(""))

	// Output:
	// sk-t****2345
	// ****
	//
}

// ExampleEncryptionService_workflow 演示完整的工作流程
func ExampleEncryptionService_workflow() {
	// 创建加密服务
	secretKey := []byte("12345678901234567890123456789012")
	encryptionService, err := service.NewEncryptionService(secretKey)
	if err != nil {
		log.Fatalf("创建加密服务失败: %v", err)
	}

	// 步骤1: 用户提供API密钥
	userAPIKey := "sk-openai-key-abc123"

	// 步骤2: 加密后存储到数据库
	encryptedKey, err := encryptionService.EncryptAPIKey(userAPIKey)
	if err != nil {
		log.Fatalf("加密失败: %v", err)
	}
	fmt.Println("已加密并存储到数据库")

	// 步骤3: 从数据库读取时脱敏显示
	maskedKey := encryptionService.MaskAPIKey(userAPIKey)
	fmt.Printf("显示给用户: %s\n", maskedKey)

	// 步骤4: 需要使用时解密
	decryptedKey, err := encryptionService.DecryptAPIKey(encryptedKey)
	if err != nil {
		log.Fatalf("解密失败: %v", err)
	}

	// 步骤5: 使用解密后的密钥调用API
	if decryptedKey == userAPIKey {
		fmt.Println("使用解密后的密钥调用API成功")
	}

	// Output:
	// 已加密并存储到数据库
	// 显示给用户: sk-o****c123
	// 使用解密后的密钥调用API成功
}
