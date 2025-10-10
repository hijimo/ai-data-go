package processor

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// DocumentProcessor 文档处理器接口
type DocumentProcessor interface {
	// SupportedTypes 支持的文件类型
	SupportedTypes() []string
	// Parse 解析文档
	Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error)
}

// FileMetadata 文件元数据
type FileMetadata struct {
	Filename    string            `json:"filename"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	SHA256      string            `json:"sha256"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// Document 解析后的文档
type Document struct {
	Title       string                 `json:"title"`        // 文档标题
	Content     string                 `json:"content"`      // 文档内容
	Metadata    map[string]interface{} `json:"metadata"`     // 文档元数据
	Structure   *DocumentStructure     `json:"structure"`    // 文档结构
	Images      []ImageInfo            `json:"images"`       // 图片信息
	Tables      []TableInfo            `json:"tables"`       // 表格信息
	Links       []LinkInfo             `json:"links"`        // 链接信息
	Language    string                 `json:"language"`     // 文档语言
	WordCount   int                    `json:"word_count"`   // 字数统计
	PageCount   int                    `json:"page_count"`   // 页数（如果适用）
}

// DocumentStructure 文档结构
type DocumentStructure struct {
	Headings []HeadingInfo `json:"headings"` // 标题层级
	Sections []Section     `json:"sections"` // 章节信息
}

// HeadingInfo 标题信息
type HeadingInfo struct {
	Level   int    `json:"level"`   // 标题级别 (1-6)
	Text    string `json:"text"`    // 标题文本
	Offset  int    `json:"offset"`  // 在文档中的位置
	ID      string `json:"id"`      // 标题ID
}

// Section 章节信息
type Section struct {
	Title   string `json:"title"`   // 章节标题
	Content string `json:"content"` // 章节内容
	Level   int    `json:"level"`   // 章节级别
	Start   int    `json:"start"`   // 开始位置
	End     int    `json:"end"`     // 结束位置
}

// ImageInfo 图片信息
type ImageInfo struct {
	Alt         string `json:"alt"`          // 替代文本
	Title       string `json:"title"`        // 图片标题
	URL         string `json:"url"`          // 图片URL
	Width       int    `json:"width"`        // 宽度
	Height      int    `json:"height"`       // 高度
	Size        int64  `json:"size"`         // 文件大小
	Format      string `json:"format"`       // 图片格式
	Description string `json:"description"`  // 图片描述
}

// TableInfo 表格信息
type TableInfo struct {
	Caption string     `json:"caption"` // 表格标题
	Headers []string   `json:"headers"` // 表头
	Rows    [][]string `json:"rows"`    // 表格数据
	Summary string     `json:"summary"` // 表格摘要
}

// LinkInfo 链接信息
type LinkInfo struct {
	Text   string `json:"text"`   // 链接文本
	URL    string `json:"url"`    // 链接地址
	Title  string `json:"title"`  // 链接标题
	Type   string `json:"type"`   // 链接类型 (internal/external)
}

// ProcessorManager 处理器管理器
type ProcessorManager struct {
	processors map[string]DocumentProcessor
}

// NewProcessorManager 创建处理器管理器
func NewProcessorManager() *ProcessorManager {
	manager := &ProcessorManager{
		processors: make(map[string]DocumentProcessor),
	}
	
	// 注册默认处理器
	manager.RegisterProcessor(NewPDFProcessor())
	manager.RegisterProcessor(NewDOCXProcessor())
	manager.RegisterProcessor(NewMarkdownProcessor())
	manager.RegisterProcessor(NewTextProcessor())
	manager.RegisterProcessor(NewHTMLProcessor())
	
	return manager
}

// RegisterProcessor 注册处理器
func (m *ProcessorManager) RegisterProcessor(processor DocumentProcessor) {
	for _, fileType := range processor.SupportedTypes() {
		m.processors[strings.ToLower(fileType)] = processor
	}
}

// GetProcessor 获取处理器
func (m *ProcessorManager) GetProcessor(fileType string) (DocumentProcessor, error) {
	processor, exists := m.processors[strings.ToLower(fileType)]
	if !exists {
		return nil, fmt.Errorf("不支持的文件类型: %s", fileType)
	}
	return processor, nil
}

// ProcessDocument 处理文档
func (m *ProcessorManager) ProcessDocument(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error) {
	// 根据文件扩展名获取处理器
	ext := getFileExtension(metadata.Filename)
	processor, err := m.GetProcessor(ext)
	if err != nil {
		return nil, err
	}
	
	// 解析文档
	doc, err := processor.Parse(ctx, reader, metadata)
	if err != nil {
		return nil, fmt.Errorf("解析文档失败: %w", err)
	}
	
	// 后处理：清理和标准化内容
	doc = m.postProcess(doc)
	
	return doc, nil
}

// postProcess 后处理文档
func (m *ProcessorManager) postProcess(doc *Document) *Document {
	// 清理内容
	doc.Content = cleanText(doc.Content)
	
	// 计算字数
	doc.WordCount = countWords(doc.Content)
	
	// 检测语言（简单实现）
	if doc.Language == "" {
		doc.Language = detectLanguage(doc.Content)
	}
	
	return doc
}

// getFileExtension 获取文件扩展名
func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return "." + strings.ToLower(parts[len(parts)-1])
}

// cleanText 清理文本
func cleanText(text string) string {
	// 移除多余的空白字符
	text = strings.TrimSpace(text)
	
	// 将多个连续的换行符替换为两个
	text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	
	// 移除行首行尾的空格
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	
	return strings.Join(lines, "\n")
}

// countWords 计算字数
func countWords(text string) int {
	if text == "" {
		return 0
	}
	
	// 简单的字数统计
	words := strings.Fields(text)
	return len(words)
}

// detectLanguage 检测语言（简单实现）
func detectLanguage(text string) string {
	// 简单的语言检测逻辑
	if text == "" {
		return "unknown"
	}
	
	// 检测中文字符
	chineseCount := 0
	englishCount := 0
	
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishCount++
		}
	}
	
	if chineseCount > englishCount {
		return "zh"
	} else if englishCount > 0 {
		return "en"
	}
	
	return "unknown"
}