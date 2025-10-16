package validator

import (
	"testing"
)

// TestValidationError 测试结构体
type TestValidationError struct {
	Name  string   `json:"name" validate:"required"`
	Email string   `json:"email" validate:"required,email"`
	Age   int      `json:"age" validate:"gte=0,lte=150"`
	Score *float64 `json:"score" validate:"omitempty,gte=0,lte=100"`
}

func TestNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("期望创建验证器实例，但得到 nil")
	}
	if v.validate == nil {
		t.Fatal("期望验证器内部实例不为 nil")
	}
}

func TestValidator_Validate_Success(t *testing.T) {
	v := New()
	score := 85.5
	
	data := TestValidationError{
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   25,
		Score: &score,
	}
	
	err := v.Validate(data)
	if err != nil {
		t.Errorf("期望验证通过，但得到错误: %v", err)
	}
}

func TestValidator_Validate_RequiredField(t *testing.T) {
	v := New()
	
	data := TestValidationError{
		Email: "test@example.com",
		Age:   25,
	}
	
	err := v.Validate(data)
	if err == nil {
		t.Error("期望验证失败（缺少必填字段），但验证通过")
	}
}

func TestValidator_Validate_EmailFormat(t *testing.T) {
	v := New()
	
	data := TestValidationError{
		Name:  "测试",
		Email: "invalid-email",
		Age:   25,
	}
	
	err := v.Validate(data)
	if err == nil {
		t.Error("期望验证失败（邮箱格式错误），但验证通过")
	}
}

func TestValidator_Validate_RangeValidation(t *testing.T) {
	v := New()
	
	tests := []struct {
		name    string
		data    TestValidationError
		wantErr bool
	}{
		{
			name: "年龄超出范围（负数）",
			data: TestValidationError{
				Name:  "测试",
				Email: "test@example.com",
				Age:   -1,
			},
			wantErr: true,
		},
		{
			name: "年龄超出范围（过大）",
			data: TestValidationError{
				Name:  "测试",
				Email: "test@example.com",
				Age:   151,
			},
			wantErr: true,
		},
		{
			name: "年龄在有效范围内",
			data: TestValidationError{
				Name:  "测试",
				Email: "test@example.com",
				Age:   30,
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_FormatErrors(t *testing.T) {
	v := New()
	
	data := TestValidationError{
		Email: "invalid-email",
		Age:   -1,
	}
	
	err := v.Validate(data)
	if err == nil {
		t.Fatal("期望验证失败")
	}
	
	errors := v.FormatErrors(err)
	if len(errors) == 0 {
		t.Error("期望返回格式化的错误列表，但列表为空")
	}
	
	// 检查错误是否包含字段名和消息
	for _, e := range errors {
		if e.Field == "" {
			t.Error("期望错误包含字段名，但字段名为空")
		}
		if e.Message == "" {
			t.Error("期望错误包含消息，但消息为空")
		}
	}
}

func TestValidator_ValidateStruct(t *testing.T) {
	v := New()
	
	t.Run("验证成功返回 nil", func(t *testing.T) {
		data := TestValidationError{
			Name:  "测试",
			Email: "test@example.com",
			Age:   25,
		}
		
		errors := v.ValidateStruct(data)
		if errors != nil {
			t.Errorf("期望返回 nil，但得到错误: %v", errors)
		}
	})
	
	t.Run("验证失败返回错误列表", func(t *testing.T) {
		data := TestValidationError{
			Email: "invalid",
			Age:   -1,
		}
		
		errors := v.ValidateStruct(data)
		if errors == nil {
			t.Error("期望返回错误列表，但得到 nil")
		}
		if len(errors) == 0 {
			t.Error("期望错误列表不为空")
		}
	})
}

func TestDefaultValidator(t *testing.T) {
	if DefaultValidator == nil {
		t.Fatal("期望默认验证器不为 nil")
	}
}

func TestValidate(t *testing.T) {
	data := TestValidationError{
		Name:  "测试",
		Email: "test@example.com",
		Age:   25,
	}
	
	err := Validate(data)
	if err != nil {
		t.Errorf("期望验证通过，但得到错误: %v", err)
	}
}

func TestValidateStruct_Function(t *testing.T) {
	data := TestValidationError{
		Email: "invalid",
	}
	
	errors := ValidateStruct(data)
	if errors == nil {
		t.Error("期望返回错误列表，但得到 nil")
	}
}

func TestFormatErrors_Function(t *testing.T) {
	data := TestValidationError{}
	err := Validate(data)
	
	if err == nil {
		t.Fatal("期望验证失败")
	}
	
	errors := FormatErrors(err)
	if len(errors) == 0 {
		t.Error("期望返回格式化的错误列表，但列表为空")
	}
}

func TestGetErrorMessage(t *testing.T) {
	v := New()
	
	tests := []struct {
		name      string
		data      interface{}
		wantField string
	}{
		{
			name: "required 错误消息",
			data: TestValidationError{
				Email: "test@example.com",
				Age:   25,
			},
			wantField: "name",
		},
		{
			name: "email 错误消息",
			data: TestValidationError{
				Name:  "测试",
				Email: "invalid",
				Age:   25,
			},
			wantField: "email",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.data)
			if err == nil {
				t.Fatal("期望验证失败")
			}
			
			errors := v.FormatErrors(err)
			found := false
			for _, e := range errors {
				if e.Field == tt.wantField {
					found = true
					if e.Message == "" {
						t.Errorf("字段 %s 的错误消息为空", tt.wantField)
					}
					break
				}
			}
			
			if !found {
				t.Errorf("期望找到字段 %s 的错误，但未找到", tt.wantField)
			}
		})
	}
}
