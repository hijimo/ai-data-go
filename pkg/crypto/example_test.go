package crypto_test

import (
	"fmt"
	"log"

	"genkit-ai-service/pkg/crypto"
)

// ExampleHashPassword 演示如何对密码进行哈希
func ExampleHashPassword() {
	password := "MySecurePassword123!"

	// 对密码进行哈希
	hash, err := crypto.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("密码哈希长度: %d\n", len(hash))
	// Output: 密码哈希长度: 60
}

// ExampleVerifyPassword 演示如何验证密码
func ExampleVerifyPassword() {
	password := "MySecurePassword123!"

	// 首先对密码进行哈希
	hash, _ := crypto.HashPassword(password)

	// 验证正确的密码
	err := crypto.VerifyPassword(hash, password)
	if err == nil {
		fmt.Println("密码验证成功")
	}

	// 验证错误的密码
	err = crypto.VerifyPassword(hash, "WrongPassword")
	if err == crypto.ErrPasswordMismatch {
		fmt.Println("密码不匹配")
	}

	// Output:
	// 密码验证成功
	// 密码不匹配
}

// ExampleValidatePasswordStrength 演示如何验证密码强度
func ExampleValidatePasswordStrength() {
	// 强密码
	err := crypto.ValidatePasswordStrength("StrongPass123!")
	if err == nil {
		fmt.Println("密码强度符合要求")
	}

	// 弱密码
	err = crypto.ValidatePasswordStrength("weak")
	if err != nil {
		fmt.Println("密码强度不足")
	}

	// Output:
	// 密码强度符合要求
	// 密码强度不足
}

// ExampleGetPasswordStrength 演示如何获取密码强度等级
func ExampleGetPasswordStrength() {
	passwords := []string{
		"VeryStrongPass123!@#",
		"Password123",
		"weakpass",
	}

	for _, pwd := range passwords {
		strength := crypto.GetPasswordStrength(pwd)
		switch strength {
		case crypto.PasswordStrengthStrong:
			fmt.Printf("%s: 强\n", pwd)
		case crypto.PasswordStrengthMedium:
			fmt.Printf("%s: 中等\n", pwd)
		case crypto.PasswordStrengthWeak:
			fmt.Printf("%s: 弱\n", pwd)
		}
	}

	// Output:
	// VeryStrongPass123!@#: 强
	// Password123: 强
	// weakpass: 弱
}

// ExampleIsCommonPassword 演示如何检测常见密码
func ExampleIsCommonPassword() {
	passwords := []string{
		"password",
		"MyUniquePass123!",
	}

	for _, pwd := range passwords {
		if crypto.IsCommonPassword(pwd) {
			fmt.Printf("%s: 是常见密码\n", pwd)
		} else {
			fmt.Printf("%s: 不是常见密码\n", pwd)
		}
	}

	// Output:
	// password: 是常见密码
	// MyUniquePass123!: 不是常见密码
}

// ExampleValidatePassword 演示综合密码验证
func ExampleValidatePassword() {
	// 有效密码
	err := crypto.ValidatePassword("StrongPass123!")
	if err == nil {
		fmt.Println("密码有效")
	}

	// 常见密码
	err = crypto.ValidatePassword("password")
	if err != nil {
		fmt.Println("密码无效: 常见密码")
	}

	// 强度不足
	err = crypto.ValidatePassword("weakpass")
	if err != nil {
		fmt.Println("密码无效: 强度不足")
	}

	// Output:
	// 密码有效
	// 密码无效: 常见密码
	// 密码无效: 强度不足
}
