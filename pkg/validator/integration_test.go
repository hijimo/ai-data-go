package validator

import (
	"genkit-ai-service/internal/model"
	"testing"
)

// TestChatRequest_Validation 测试 ChatRequest 的验证
func TestChatRequest_Validation(t *testing.T) {
	v := New()
	
	t.Run("有效的 ChatRequest", func(t *testing.T) {
		temp := 0.7
		maxTokens := 1000
		topP := 0.9
		topK := 40
		
		req := model.ChatRequest{
			Message:   "你好，请介绍一下 Firebase",
			SessionID: "test-session-123",
			Options: &model.ChatOptions{
				Temperature: &temp,
				MaxTokens:   &maxTokens,
				TopP:        &topP,
				TopK:        &topK,
			},
		}
		
		err := v.Validate(req)
		if err != nil {
			t.Errorf("期望验证通过，但得到错误: %v", err)
		}
	})
	
	t.Run("缺少必填字段 Message", func(t *testing.T) {
		req := model.ChatRequest{
			SessionID: "test-session-123",
		}
		
		errors := v.ValidateStruct(req)
		if errors == nil {
			t.Error("期望验证失败，但验证通过")
			return
		}
		
		// 检查是否包含 message 字段的错误
		found := false
		for _, e := range errors {
			if e.Field == "message" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("期望包含 message 字段的验证错误")
		}
	})
	
	t.Run("Temperature 超出范围", func(t *testing.T) {
		invalidTemp := 3.0 // 超出范围 (0-2)
		
		req := model.ChatRequest{
			Message: "测试",
			Options: &model.ChatOptions{
				Temperature: &invalidTemp,
			},
		}
		
		errors := v.ValidateStruct(req)
		if errors == nil {
			t.Error("期望验证失败（temperature 超出范围），但验证通过")
			return
		}
		
		// 检查是否包含 temperature 字段的错误
		found := false
		for _, e := range errors {
			if e.Field == "temperature" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("期望包含 temperature 字段的验证错误")
		}
	})
	
	t.Run("MaxTokens 为负数", func(t *testing.T) {
		invalidTokens := -100
		
		req := model.ChatRequest{
			Message: "测试",
			Options: &model.ChatOptions{
				MaxTokens: &invalidTokens,
			},
		}
		
		errors := v.ValidateStruct(req)
		if errors == nil {
			t.Error("期望验证失败（maxTokens 为负数），但验证通过")
			return
		}
		
		// 检查是否包含 maxTokens 字段的错误
		found := false
		for _, e := range errors {
			if e.Field == "maxTokens" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("期望包含 maxTokens 字段的验证错误")
		}
	})
	
	t.Run("TopP 超出范围", func(t *testing.T) {
		invalidTopP := 1.5 // 超出范围 (0-1)
		
		req := model.ChatRequest{
			Message: "测试",
			Options: &model.ChatOptions{
				TopP: &invalidTopP,
			},
		}
		
		errors := v.ValidateStruct(req)
		if errors == nil {
			t.Error("期望验证失败（topP 超出范围），但验证通过")
			return
		}
		
		// 检查是否包含 topP 字段的错误
		found := false
		for _, e := range errors {
			if e.Field == "topP" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("期望包含 topP 字段的验证错误")
		}
	})
	
	t.Run("可选字段为 nil 时验证通过", func(t *testing.T) {
		req := model.ChatRequest{
			Message: "测试",
			Options: &model.ChatOptions{
				// 所有可选字段都为 nil
			},
		}
		
		err := v.Validate(req)
		if err != nil {
			t.Errorf("期望验证通过（可选字段为 nil），但得到错误: %v", err)
		}
	})
}

// TestAbortRequest_Validation 测试 AbortRequest 的验证
func TestAbortRequest_Validation(t *testing.T) {
	v := New()
	
	t.Run("有效的 AbortRequest", func(t *testing.T) {
		req := model.AbortRequest{
			SessionID: "test-session-123",
		}
		
		err := v.Validate(req)
		if err != nil {
			t.Errorf("期望验证通过，但得到错误: %v", err)
		}
	})
	
	t.Run("缺少必填字段 SessionID", func(t *testing.T) {
		req := model.AbortRequest{}
		
		errors := v.ValidateStruct(req)
		if errors == nil {
			t.Error("期望验证失败，但验证通过")
			return
		}
		
		// 打印所有错误字段（用于调试）
		t.Logf("验证错误数量: %d", len(errors))
		for _, e := range errors {
			t.Logf("- 字段: %s, 消息: %s", e.Field, e.Message)
		}
		
		// 检查是否包含 sessionId 或 sessionID 字段的错误
		found := false
		for _, e := range errors {
			if e.Field == "sessionId" || e.Field == "sessionID" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("期望包含 sessionId 字段的验证错误")
		}
	})
}

// TestMultipleValidationErrors 测试多个验证错误
func TestMultipleValidationErrors(t *testing.T) {
	v := New()
	
	invalidTemp := 5.0
	invalidTokens := -1
	invalidTopP := 2.0
	
	req := model.ChatRequest{
		// Message 缺失
		Options: &model.ChatOptions{
			Temperature: &invalidTemp,
			MaxTokens:   &invalidTokens,
			TopP:        &invalidTopP,
		},
	}
	
	errors := v.ValidateStruct(req)
	if errors == nil {
		t.Fatal("期望验证失败，但验证通过")
	}
	
	// 应该有多个错误
	if len(errors) < 2 {
		t.Errorf("期望至少有 2 个验证错误，但只有 %d 个", len(errors))
	}
	
	// 打印所有错误（用于调试）
	t.Logf("验证错误数量: %d", len(errors))
	for _, e := range errors {
		t.Logf("- 字段: %s, 消息: %s", e.Field, e.Message)
	}
}
