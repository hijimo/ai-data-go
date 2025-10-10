package processor

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// PDFProcessor PDF文档处理器
type PDFProcessor struct{}

// NewPDFProcessor 创建PDF处理器
func NewPDFProcessor() DocumentProcessor {
	return &PDFProcessor{}
}

// SupportedTypes 支持的文件类型
func (p *PDFProcessor) SupportedTypes() []string {
	return []string{".pdf"}
}

// Parse 解析PDF文档
func (p *PDFProcessor) Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error) {
	// 读取PDF内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取PDF文件失败: %w", err)
	}

	// 检查PDF文件头
	if len(content) < 4 || string(content[:4]) != "%PDF" {
		return nil, fmt.Errorf("无效的PDF文件格式")
	}

	// 由于这是MVP实现，我们使用简化的PDF解析
	// 在生产环境中，应该使用专业的PDF解析库如 github.com/ledongthuc/pdf
	doc := &Document{
		Title:    extractTitleFromFilename(metadata.Filename),
		Content:  p.extractTextFromPDF(content),
		Metadata: map[string]interface{}{
			"file_type":    "pdf",
			"file_size":    metadata.Size,
			"sha256":       metadata.SHA256,
			"content_type": metadata.ContentType,
		},
		Structure: &DocumentStructure{
			Headings: []HeadingInfo{},
			Sections: []Section{},
		},
		Images:    []ImageInfo{},
		Tables:    []TableInfo{},
		Links:     []LinkInfo{},
		Language:  "",
		WordCount: 0,
		PageCount: p.estimatePageCount(content),
	}

	// 提取结构信息
	doc.Structure = p.extractStructure(doc.Content)

	return doc, nil
}

// extractTextFromPDF 从PDF中提取文本（简化实现）
func (p *PDFProcessor) extractTextFromPDF(content []byte) string {
	// 这是一个非常简化的PDF文本提取实现
	// 在实际项目中，应该使用专业的PDF解析库
	
	text := string(content)
	
	// 查找文本对象
	var extractedText strings.Builder
	
	// 简单的文本提取逻辑（仅用于演示）
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// 查找可能的文本内容
		if strings.Contains(line, "Tj") || strings.Contains(line, "TJ") {
			// 提取括号内的文本
			start := strings.Index(line, "(")
			end := strings.LastIndex(line, ")")
			if start != -1 && end != -1 && end > start {
				textContent := line[start+1 : end]
				// 清理转义字符
				textContent = strings.ReplaceAll(textContent, "\\n", "\n")
				textContent = strings.ReplaceAll(textContent, "\\r", "\r")
				textContent = strings.ReplaceAll(textContent, "\\t", "\t")
				textContent = strings.ReplaceAll(textContent, "\\(", "(")
				textContent = strings.ReplaceAll(textContent, "\\)", ")")
				textContent = strings.ReplaceAll(textContent, "\\\\", "\\")
				
				if strings.TrimSpace(textContent) != "" {
					extractedText.WriteString(textContent)
					extractedText.WriteString(" ")
				}
			}
		}
	}
	
	result := extractedText.String()
	if strings.TrimSpace(result) == "" {
		// 如果无法提取文本，返回提示信息
		return fmt.Sprintf("PDF文档 (%s) - 需要专业PDF解析库来提取文本内容", extractTitleFromFilename(""))
	}
	
	return strings.TrimSpace(result)
}

// extractStructure 提取文档结构
func (p *PDFProcessor) extractStructure(content string) *DocumentStructure {
	structure := &DocumentStructure{
		Headings: []HeadingInfo{},
		Sections: []Section{},
	}
	
	lines := strings.Split(content, "\n")
	currentSection := ""
	sectionContent := strings.Builder{}
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// 简单的标题检测逻辑
		if p.isHeading(line) {
			// 保存前一个章节
			if currentSection != "" {
				structure.Sections = append(structure.Sections, Section{
					Title:   currentSection,
					Content: strings.TrimSpace(sectionContent.String()),
					Level:   1,
					Start:   0,
					End:     len(sectionContent.String()),
				})
			}
			
			// 开始新章节
			currentSection = line
			sectionContent.Reset()
			
			// 添加标题信息
			structure.Headings = append(structure.Headings, HeadingInfo{
				Level:  1,
				Text:   line,
				Offset: i,
				ID:     fmt.Sprintf("heading-%d", len(structure.Headings)),
			})
		} else {
			sectionContent.WriteString(line)
			sectionContent.WriteString("\n")
		}
	}
	
	// 保存最后一个章节
	if currentSection != "" {
		structure.Sections = append(structure.Sections, Section{
			Title:   currentSection,
			Content: strings.TrimSpace(sectionContent.String()),
			Level:   1,
			Start:   0,
			End:     len(sectionContent.String()),
		})
	}
	
	return structure
}

// isHeading 判断是否为标题
func (p *PDFProcessor) isHeading(line string) bool {
	// 简单的标题判断逻辑
	if len(line) < 3 || len(line) > 100 {
		return false
	}
	
	// 检查是否以数字开头（如 "1. 标题"）
	if len(line) > 2 && line[1] == '.' && line[0] >= '0' && line[0] <= '9' {
		return true
	}
	
	// 检查是否全大写（可能是标题）
	if strings.ToUpper(line) == line && len(strings.Fields(line)) <= 10 {
		return true
	}
	
	// 检查是否包含常见的标题关键词
	titleKeywords := []string{"第", "章", "节", "部分", "Chapter", "Section", "Part"}
	for _, keyword := range titleKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	
	return false
}

// estimatePageCount 估算页数
func (p *PDFProcessor) estimatePageCount(content []byte) int {
	// 查找PDF页面对象
	pageCount := strings.Count(string(content), "/Type /Page")
	if pageCount == 0 {
		// 如果找不到页面对象，根据内容长度估算
		pageCount = len(content) / 3000 // 假设每页约3KB
		if pageCount == 0 {
			pageCount = 1
		}
	}
	return pageCount
}

// extractTitleFromFilename 从文件名提取标题
func extractTitleFromFilename(filename string) string {
	if filename == "" {
		return "未知文档"
	}
	
	// 移除扩展名
	name := filename
	if lastDot := strings.LastIndex(filename, "."); lastDot != -1 {
		name = filename[:lastDot]
	}
	
	// 替换下划线和连字符为空格
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	
	return strings.TrimSpace(name)
}