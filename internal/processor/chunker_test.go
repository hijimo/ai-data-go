package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixedSizeChunker_Chunk(t *testing.T) {
	chunker := NewFixedSizeChunker()
	
	doc := &Document{
		Content: "这是第一段内容。这是第二段内容。这是第三段内容。这是第四段内容。这是第五段内容。这是第六段内容。这是第七段内容。这是第八段内容。",
	}
	
	config := &ChunkConfig{
		Strategy:        ChunkByFixedSize,
		MaxSize:         50,
		Overlap:         10,
		PreserveContext: true,
	}
	
	chunks, err := chunker.Chunk(doc, config)
	
	assert.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
	
	// 检查每个块的大小
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk.Content), config.MaxSize+20) // 允许一些容差
		assert.Greater(t, chunk.TokenCount, 0)
		assert.Equal(t, "fixed_size", chunk.Type)
	}
	
	// 检查重叠
	if len(chunks) > 1 {
		// 第二个块应该与第一个块有重叠
		assert.Less(t, chunks[1].StartOffset, chunks[0].EndOffset)
	}
}

func TestSemanticChunker_Chunk(t *testing.T) {
	chunker := NewSemanticChunker()
	
	doc := &Document{
		Content: `第一段内容。
这是第一段的详细描述。

第二段内容。
这是第二段的详细描述。
包含更多信息。

第三段内容。
这是第三段的简短描述。`,
	}
	
	config := &ChunkConfig{
		Strategy: ChunkBySemantic,
		MaxSize:  100,
		Overlap:  0,
	}
	
	chunks, err := chunker.Chunk(doc, config)
	
	assert.NoError(t, err)
	assert.Greater(t, len(chunks), 0)
	
	// 检查块类型
	for _, chunk := range chunks {
		assert.Equal(t, "semantic", chunk.Type)
		assert.Greater(t, chunk.TokenCount, 0)
	}
}

func TestStructureChunker_Chunk(t *testing.T) {
	chunker := NewStructureChunker()
	
	doc := &Document{
		Content: "标题1内容\n\n标题2内容\n\n标题3内容",
		Structure: &DocumentStructure{
			Headings: []HeadingInfo{
				{Level: 1, Text: "标题1", Offset: 0},
				{Level: 2, Text: "标题2", Offset: 10},
				{Level: 2, Text: "标题3", Offset: 20},
			},
			Sections: []Section{
				{Title: "标题1", Content: "标题1内容", Level: 1, Start: 0, End: 10},
				{Title: "标题2", Content: "标题2内容", Level: 2, Start: 10, End: 20},
				{Title: "标题3", Content: "标题3内容", Level: 2, Start: 20, End: 30},
			},
		},
	}
	
	config := &ChunkConfig{
		Strategy: ChunkByStructure,
		MaxSize:  1000,
		Overlap:  0,
	}
	
	chunks, err := chunker.Chunk(doc, config)
	
	assert.NoError(t, err)
	assert.Equal(t, 3, len(chunks))
	
	// 检查块内容和元数据
	for i, chunk := range chunks {
		assert.Equal(t, "structure", chunk.Type)
		assert.Contains(t, chunk.Metadata, "section_title")
		assert.Contains(t, chunk.Metadata, "section_level")
		assert.Equal(t, doc.Structure.Sections[i].Title, chunk.Metadata["section_title"])
	}
}

func TestCodeChunker_Chunk(t *testing.T) {
	chunker := NewCodeChunker()
	
	doc := &Document{
		Content: `function hello() {
    console.log("Hello, World!");
}

function goodbye() {
    console.log("Goodbye!");
    return true;
}

class MyClass {
    constructor() {
        this.name = "test";
    }
    
    method() {
        return this.name;
    }
}`,
	}
	
	config := &ChunkConfig{
		Strategy: ChunkByCode,
		MaxSize:  200,
		Overlap:  0,
	}
	
	chunks, err := chunker.Chunk(doc, config)
	
	assert.NoError(t, err)
	assert.Greater(t, len(chunks), 0)
	
	// 检查是否识别为代码内容
	for _, chunk := range chunks {
		assert.Contains(t, []string{"code", "code_split"}, chunk.Type)
		assert.Greater(t, chunk.TokenCount, 0)
	}
}

func TestChunkerManager_ChunkDocument(t *testing.T) {
	manager := NewChunkerManager()
	
	doc := &Document{
		Content: "这是一个测试文档。包含多个段落。每个段落都有不同的内容。用于测试分块功能。",
	}
	
	tests := []struct {
		name     string
		strategy ChunkStrategy
		maxSize  int
	}{
		{
			name:     "固定长度分块",
			strategy: ChunkByFixedSize,
			maxSize:  30,
		},
		{
			name:     "语义分块",
			strategy: ChunkBySemantic,
			maxSize:  50,
		},
		{
			name:     "结构化分块",
			strategy: ChunkByStructure,
			maxSize:  100,
		},
		{
			name:     "代码分块",
			strategy: ChunkByCode,
			maxSize:  100,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ChunkConfig{
				Strategy: tt.strategy,
				MaxSize:  tt.maxSize,
				Overlap:  5,
			}
			
			chunks, err := manager.ChunkDocument(doc, config)
			
			assert.NoError(t, err)
			assert.Greater(t, len(chunks), 0)
			
			// 验证块的基本属性
			for _, chunk := range chunks {
				assert.NotEmpty(t, chunk.Content)
				assert.GreaterOrEqual(t, chunk.EndOffset, chunk.StartOffset)
				assert.Greater(t, chunk.TokenCount, 0)
				assert.NotEmpty(t, chunk.Type)
				assert.NotNil(t, chunk.Metadata)
			}
		})
	}
}

func TestChunkConfig_Validation(t *testing.T) {
	chunker := NewFixedSizeChunker()
	doc := &Document{Content: "测试内容"}
	
	tests := []struct {
		name        string
		config      *ChunkConfig
		expectError bool
	}{
		{
			name: "有效配置",
			config: &ChunkConfig{
				Strategy: ChunkByFixedSize,
				MaxSize:  100,
				Overlap:  10,
			},
			expectError: false,
		},
		{
			name: "最大长度为0",
			config: &ChunkConfig{
				Strategy: ChunkByFixedSize,
				MaxSize:  0,
				Overlap:  10,
			},
			expectError: false, // 应该使用默认值
		},
		{
			name: "重叠大于最大长度",
			config: &ChunkConfig{
				Strategy: ChunkByFixedSize,
				MaxSize:  50,
				Overlap:  100,
			},
			expectError: false, // 应该自动调整
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := chunker.Chunk(doc, tt.config)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Greater(t, len(chunks), 0)
			}
		})
	}
}

func TestTokenCountEstimation(t *testing.T) {
	chunker := &FixedSizeChunker{}
	
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "纯中文",
			content:  "这是中文内容",
			expected: 6, // 6个中文字符
		},
		{
			name:     "纯英文",
			content:  "This is English content",
			expected: 4, // 4个英文单词
		},
		{
			name:     "中英混合",
			content:  "这是 mixed 内容",
			expected: 5, // 3个中文字符 + 2个英文单词
		},
		{
			name:     "空内容",
			content:  "",
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.estimateTokenCount(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSentenceBoundaryDetection(t *testing.T) {
	chunker := &FixedSizeChunker{}
	
	content := "这是第一句话。这是第二句话！这是第三句话？这是第四句话。"
	
	tests := []struct {
		name     string
		start    int
		maxEnd   int
		expected int
	}{
		{
			name:     "在句号处分割",
			start:    0,
			maxEnd:   15,
			expected: 8, // 应该在第一个句号后分割
		},
		{
			name:     "在感叹号处分割",
			start:    0,
			maxEnd:   25,
			expected: 16, // 应该在感叹号后分割
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.findSentenceBoundary(content, tt.start, tt.maxEnd)
			assert.LessOrEqual(t, result, tt.maxEnd)
			assert.GreaterOrEqual(t, result, tt.start)
		})
	}
}

func TestCodeDetection(t *testing.T) {
	chunker := &CodeChunker{}
	
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "JavaScript代码",
			content: `function test() {
    console.log("hello");
    return true;
}`,
			expected: true,
		},
		{
			name: "Python代码",
			content: `def test():
    print("hello")
    return True`,
			expected: true,
		},
		{
			name: "HTML代码",
			content: `<html>
<head><title>Test</title></head>
<body><p>Hello</p></body>
</html>`,
			expected: true,
		},
		{
			name:     "普通文本",
			content:  "这是一个普通的文本文档，不包含代码。",
			expected: false,
		},
		{
			name:     "空内容",
			content:  "",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.isCodeContent(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}