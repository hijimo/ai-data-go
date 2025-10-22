package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// .env 中配置的密码
	password := "Admin123456"
	
	// 生成密码哈希（模拟系统初始化时的操作）
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Printf("生成密码哈希失败: %v\n", err)
		return
	}
	
	fmt.Printf("密码: %s\n", password)
	fmt.Printf("哈希值: %s\n", string(hash))
	fmt.Println()
	
	// 验证密码（模拟登录时的操作）
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		fmt.Printf("❌ 密码验证失败: %v\n", err)
	} else {
		fmt.Printf("✅ 密码验证成功\n")
	}
	
	// 测试一些常见的错误情况
	fmt.Println("\n测试其他密码:")
	testPasswords := []string{
		"Admin123456",  // 正确密码
		"admin123456",  // 小写
		"Admin123456 ", // 带空格
		" Admin123456", // 前面有空格
		"Admin12345",   // 少一个字符
	}
	
	for _, testPwd := range testPasswords {
		err = bcrypt.CompareHashAndPassword(hash, []byte(testPwd))
		if err != nil {
			fmt.Printf("❌ '%s' - 验证失败\n", testPwd)
		} else {
			fmt.Printf("✅ '%s' - 验证成功\n", testPwd)
		}
	}
}
