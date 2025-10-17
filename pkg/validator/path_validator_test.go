package validator

import (
	"testing"
)

func TestValidateProviderID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{
			name:      "有效的提供商ID - 小写字母",
			id:        "tongyi",
			wantError: false,
		},
		{
			name:      "有效的提供商ID - 带连字符",
			id:        "google-ai",
			wantError: false,
		},
		{
			name:      "有效的提供商ID - 带下划线",
			id:        "open_ai",
			wantError: false,
		},
		{
			name:      "有效的提供商ID - 带数字",
			id:        "gemini2",
			wantError: false,
		},
		{
			name:      "有效的提供商ID - 带点号",
			id:        "model.v1",
			wantError: false,
		},
		{
			name:      "无效的提供商ID - 空字符串",
			id:        "",
			wantError: true,
		},
		{
			name:      "无效的提供商ID - 包含路径遍历",
			id:        "../etc",
			wantError: true,
		},
		{
			name:      "无效的提供商ID - 包含斜杠",
			id:        "provider/test",
			wantError: true,
		},
		{
			name:      "无效的提供商ID - 包含反斜杠",
			id:        "provider\\test",
			wantError: true,
		},
		{
			name:      "无效的提供商ID - 包含空格",
			id:        "provider test",
			wantError: true,
		},
		{
			name:      "无效的提供商ID - 超长",
			id:        string(make([]byte, MaxIDLength+1)),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderID(tt.id)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateProviderID() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateModelID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{
			name:      "有效的模型ID - 简单名称",
			id:        "qwen-max",
			wantError: false,
		},
		{
			name:      "有效的模型ID - 带版本号",
			id:        "gemini-1.5-pro",
			wantError: false,
		},
		{
			name:      "有效的模型ID - 带日期",
			id:        "qwen-plus-0919",
			wantError: false,
		},
		{
			name:      "有效的模型ID - 复杂名称",
			id:        "qwen2.5-72b-instruct",
			wantError: false,
		},
		{
			name:      "无效的模型ID - 空字符串",
			id:        "",
			wantError: true,
		},
		{
			name:      "无效的模型ID - 包含路径遍历",
			id:        "../model",
			wantError: true,
		},
		{
			name:      "无效的模型ID - 包含斜杠",
			id:        "model/test",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelID(tt.id)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateModelID() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "正常路径",
			path: "provider/model",
			want: false,
		},
		{
			name: "包含父目录引用 ..",
			path: "../etc",
			want: true,
		},
		{
			name: "包含当前目录引用 ./",
			path: "./config",
			want: true,
		},
		{
			name: "包含父目录引用 ../",
			path: "test/../config",
			want: true,
		},
		{
			name: "包含 /..",
			path: "test/..",
			want: true,
		},
		{
			name: "包含反斜杠",
			path: "test\\config",
			want: true,
		},
		{
			name: "正常的点号",
			path: "model.v1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsPathTraversal(tt.path); got != tt.want {
				t.Errorf("containsPathTraversal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePathSafety(t *testing.T) {
	tests := []struct {
		name       string
		basePath   string
		targetPath string
		wantError  bool
	}{
		{
			name:       "安全的相对路径",
			basePath:   "/app/models",
			targetPath: "tongyi/provider",
			wantError:  false,
		},
		{
			name:       "包含父目录引用",
			basePath:   "/app/models",
			targetPath: "../etc/passwd",
			wantError:  true,
		},
		{
			name:       "包含路径遍历",
			basePath:   "/app/models",
			targetPath: "tongyi/../../../etc",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePathSafety(tt.basePath, tt.targetPath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePathSafety() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
