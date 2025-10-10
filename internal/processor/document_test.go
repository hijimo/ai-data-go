package processor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessorManager_ProcessDocument(t *testing.T) {
	manager := NewProcessorManager()

	tests := []struct {
		name        string
		metadata    *FileMetadata
		content     string
		expectError bool
	}{
		{
			name: "处理Markdown文档",
			metadata: &FileMetadata{
				Filename:    "test.md",
				ContentType: "text/markdown",
				Size:        100,
				SHA256:      "test-hash",
			},
			content:     "# 标题\n\n这是一个测试文档。\n\n## 子标题\n\n这是子章节内容。",
			expectError: false,
		},
		{
			name: "处理文本文档",
			metadata: &FileMetadata{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Size:        50,
				SHA256:      "test-hash",
			},
			content:     "这是一个简单的文本文档。\n包含多行内容。",
			expectError: false,
		},
		{
			name: "处理HTML文档",
			metadata: &FileMetadata{
				Filename:    "test.html",
				ContentType: "text/html",
				Size:        200,
				SHA256:      "test-hash",
			},
			content:     "<html><head><title>测试</title></head><body><h1>标题</h1><p>内容</p></body></html>",
			expectError: false,
		},
		{
			name: "不支持的文件类型",
			metadata: &FileMetadata{
				Filename:    "test.xyz",
				ContentType: "application/unknown",
				Size:        50,
				SHA256:      "test-hash",
			},
			content:     "未知格式内容",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			doc, err := manager.ProcessDocument(context.Background(), reader, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, doc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, doc)
				assert.NotEmpty(t, doc.Content)
				assert.Greater(t, doc.WordCount, 0)
			}
		})
	}
}

func TestMarkdownProcessor_Parse(t *testing.T) {
	processor := NewMarkdownProcessor()
	
	content := `# 主标题

这是介绍段落。

## 子标题1

这是第一个子章节的内容。

### 三级标题

这是三级标题的内容。

## 子标题2

这是第二个子章节的内容。

| 列1 | 列2 | 列3 |
|-----|-----|-----|
| 数据1 | 数据2 | 数据3 |
| 数据4 | 数据5 | 数据6 |

这里有一个链接：[测试链接](http://example.com)

还有一个图片：![测试图片](http://example.com/image.png "图片标题")
`

	metadata := &FileMetadata{
		Filename:    "test.md",
		ContentType: "text/markdown",
		Size:        int64(len(content)),
		SHA256:      "test-hash",
	}

	reader := strings.NewReader(content)
	doc, err := processor.Parse(context.Background(), reader, metadata)

	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "主标题", doc.Title)
	assert.Contains(t, doc.Content, "介绍段落")
	
	// 检查结构
	assert.NotNil(t, doc.Structure)
	assert.Len(t, doc.Structure.Headings, 4) // 4个标题
	assert.Equal(t, 1, doc.Structure.Headings[0].Level)
	assert.Equal(t, "主标题", doc.Structure.Headings[0].Text)
	
	// 检查表格
	assert.Len(t, doc.Tables, 1)
	assert.Equal(t, []string{"列1", "列2", "列3"}, doc.Tables[0].Headers)
	assert.Len(t, doc.Tables[0].Rows, 2)
	
	// 检查链接
	assert.Len(t, doc.Links, 1)
	assert.Equal(t, "测试链接", doc.Links[0].Text)
	assert.Equal(t, "http://example.com", doc.Links[0].URL)
	
	// 检查图片
	assert.Len(t, doc.Images, 1)
	assert.Equal(t, "测试图片", doc.Images[0].Alt)
	assert.Equal(t, "http://example.com/image.png", doc.Images[0].URL)
}

func TestTextProcessor_Parse(t *testing.T) {
	processor := NewTextProcessor()
	
	content := `文档标题

这是第一段内容。
包含多行文本。

1. 这是一个标题
这是标题下的内容。

二、另一个标题
这是另一个章节的内容。

表格数据：
姓名	年龄	城市
张三	25	北京
李四	30	上海

联系方式：test@example.com
网站：https://example.com
`

	metadata := &FileMetadata{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Size:        int64(len(content)),
		SHA256:      "test-hash",
	}

	reader := strings.NewReader(content)
	doc, err := processor.Parse(context.Background(), reader, metadata)

	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "文档标题", doc.Title)
	assert.Contains(t, doc.Content, "第一段内容")
	
	// 检查结构
	assert.NotNil(t, doc.Structure)
	assert.Greater(t, len(doc.Structure.Headings), 0)
	
	// 检查表格
	assert.Greater(t, len(doc.Tables), 0)
	
	// 检查链接
	assert.Greater(t, len(doc.Links), 0)
	
	// 检查邮箱和URL是否被识别
	foundEmail := false
	foundURL := false
	for _, link := range doc.Links {
		if link.Type == "email" {
			foundEmail = true
		}
		if link.Type == "external" {
			foundURL = true
		}
	}
	assert.True(t, foundEmail)
	assert.True(t, foundURL)
}

func TestHTMLProcessor_Parse(t *testing.T) {
	processor := NewHTMLProcessor()
	
	content := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <title>测试HTML文档</title>
</head>
<body>
    <h1>主标题</h1>
    <p>这是介绍段落。</p>
    
    <h2>子标题</h2>
    <p>这是子章节内容。</p>
    
    <table>
        <caption>数据表格</caption>
        <thead>
            <tr>
                <th>列1</th>
                <th>列2</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td>数据1</td>
                <td>数据2</td>
            </tr>
        </tbody>
    </table>
    
    <p>这里有一个<a href="http://example.com" title="示例">链接</a>。</p>
    <img src="image.png" alt="测试图片" width="100" height="200">
    
    <script>console.log('这段脚本应该被移除');</script>
    <style>body { color: red; }</style>
</body>
</html>`

	metadata := &FileMetadata{
		Filename:    "test.html",
		ContentType: "text/html",
		Size:        int64(len(content)),
		SHA256:      "test-hash",
	}

	reader := strings.NewReader(content)
	doc, err := processor.Parse(context.Background(), reader, metadata)

	assert.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "测试HTML文档", doc.Title)
	assert.Contains(t, doc.Content, "介绍段落")
	assert.NotContains(t, doc.Content, "console.log") // 脚本应该被移除
	assert.NotContains(t, doc.Content, "color: red")  // 样式应该被移除
	
	// 检查语言
	assert.Equal(t, "zh-CN", doc.Language)
	
	// 检查结构
	assert.NotNil(t, doc.Structure)
	assert.Len(t, doc.Structure.Headings, 2)
	assert.Equal(t, "主标题", doc.Structure.Headings[0].Text)
	assert.Equal(t, 1, doc.Structure.Headings[0].Level)
	
	// 检查表格
	assert.Len(t, doc.Tables, 1)
	assert.Equal(t, "数据表格", doc.Tables[0].Caption)
	assert.Equal(t, []string{"列1", "列2"}, doc.Tables[0].Headers)
	
	// 检查链接
	assert.Len(t, doc.Links, 1)
	assert.Equal(t, "链接", doc.Links[0].Text)
	assert.Equal(t, "http://example.com", doc.Links[0].URL)
	assert.Equal(t, "示例", doc.Links[0].Title)
	
	// 检查图片
	assert.Len(t, doc.Images, 1)
	assert.Equal(t, "测试图片", doc.Images[0].Alt)
	assert.Equal(t, "image.png", doc.Images[0].URL)
	assert.Equal(t, 100, doc.Images[0].Width)
	assert.Equal(t, 200, doc.Images[0].Height)
}

func TestLanguageDetection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "中文内容",
			content:  "这是一个中文文档，包含中文字符。",
			expected: "zh",
		},
		{
			name:     "英文内容",
			content:  "This is an English document with English words.",
			expected: "en",
		},
		{
			name:     "混合内容（中文为主）",
			content:  "这是一个混合文档 with some English words.",
			expected: "zh",
		},
		{
			name:     "空内容",
			content:  "",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}