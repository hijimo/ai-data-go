package processor

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// TextProcessor 纯文本处理器
type TextProcessor struct{}

// NewTextProcessor 创建文本处理器
func NewTextProcessor() DocumentProcessor {
	return &TextProcessor{}
}

// SupportedTypes 支持的文件类型
func (p *TextProcessor) SupportedTypes() []string {
	return []string{".txt", ".text"}
}

// Parse 解析文本文档
func (p *TextProcessor) Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error) {
	// 读取文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取文本文件失败: %w", err)
	}

	// 检测编码并转换为UTF-8
	text, encoding := p.detectAndConvertEncoding(content)

	doc := &Document{
		Title:    p.extractTitle(text, metadata.Filename),
		Content:  text,
		Metadata: map[string]interface{}{
			"file_type":    "text",
			"file_size":    metadata.Size,
			"sha256":       metadata.SHA256,
			"content_type": metadata.ContentType,
			"encoding":     encoding,
		},
		Structure: p.extractStructure(text),
		Images:    []ImageInfo{}, // 纯文本没有图片
		Tables:    p.extractTables(text),
		Links:     p.extractLinks(text),
		Language:  "",
		WordCount: 0,
		PageCount: p.estimatePageCount(text),
	}

	return doc, nil
}

// detectAndConvertEncoding 检测并转换编码
func (p *TextProcessor) detectAndConvertEncoding(content []byte) (string, string) {
	// 检查是否为有效的UTF-8
	if utf8.Valid(content) {
		return string(content), "UTF-8"
	}

	// 尝试其他常见编码（简化实现）
	text := string(content)
	
	// 检查是否包含中文编码特征
	if p.containsChineseCharacters(text) {
		// 可能是GBK或GB2312编码，这里简化处理
		return text, "GBK"
	}

	// 默认返回原始内容
	return text, "Unknown"
}

// containsChineseCharacters 检查是否包含中文字符
func (p *TextProcessor) containsChineseCharacters(text string) bool {
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// extractTitle 提取文档标题
func (p *TextProcessor) extractTitle(content, filename string) string {
	lines := strings.Split(content, "\n")
	
	// 查找第一行非空内容作为标题
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// 如果第一行太长，截取前50个字符
			if len(line) > 50 {
				return line[:50] + "..."
			}
			return line
		}
	}
	
	// 使用文件名作为标题
	return extractTitleFromFilename(filename)
}

// extractStructure 提取文档结构
func (p *TextProcessor) extractStructure(content string) *DocumentStructure {
	structure := &DocumentStructure{
		Headings: []HeadingInfo{},
		Sections: []Section{},
	}
	
	lines := strings.Split(content, "\n")
	currentSection := ""
	sectionContent := strings.Builder{}
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// 检查是否为标题
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
		} else if line != "" {
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
	
	// 如果没有找到明显的章节，将整个文档作为一个章节
	if len(structure.Sections) == 0 {
		structure.Sections = append(structure.Sections, Section{
			Title:   "正文",
			Content: content,
			Level:   1,
			Start:   0,
			End:     len(content),
		})
	}
	
	return structure
}

// isHeading 判断是否为标题
func (p *TextProcessor) isHeading(line string) bool {
	if len(line) < 3 || len(line) > 100 {
		return false
	}
	
	// 检查数字编号标题 (如 "1. 标题", "一、标题")
	numberPatterns := []string{
		`^\d+\.\s+.+`,           // 1. 标题
		`^\d+\)\s+.+`,           // 1) 标题
		`^[一二三四五六七八九十]+[、．]\s*.+`, // 一、标题
		`^第[一二三四五六七八九十\d]+[章节部分]\s*.+`, // 第一章
	}
	
	for _, pattern := range numberPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}
	
	// 检查全大写标题
	if strings.ToUpper(line) == line && len(strings.Fields(line)) <= 10 {
		// 检查是否包含字母
		hasLetter := false
		for _, r := range line {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				hasLetter = true
				break
			}
		}
		if hasLetter {
			return true
		}
	}
	
	// 检查居中标题（前后有空格或特殊字符）
	trimmed := strings.Trim(line, " \t=*-_")
	if len(trimmed) < len(line) && len(trimmed) > 0 {
		return true
	}
	
	return false
}

// extractTables 提取表格信息
func (p *TextProcessor) extractTables(content string) []TableInfo {
	var tables []TableInfo
	
	lines := strings.Split(content, "\n")
	
	// 查找表格模式（简单的制表符分隔或空格对齐）
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		
		// 检查是否为表格行
		if p.isTableLine(line) {
			table := p.parseTextTable(lines, &i)
			if len(table.Rows) > 0 || len(table.Headers) > 0 {
				tables = append(tables, table)
			}
		}
	}
	
	return tables
}

// isTableLine 检查是否为表格行
func (p *TextProcessor) isTableLine(line string) bool {
	// 检查制表符分隔
	if strings.Count(line, "\t") >= 2 {
		return true
	}
	
	// 检查多个空格分隔（可能是对齐的表格）
	if regexp.MustCompile(`\s{3,}`).MatchString(line) {
		fields := regexp.MustCompile(`\s{3,}`).Split(line, -1)
		return len(fields) >= 3
	}
	
	// 检查竖线分隔
	if strings.Count(line, "|") >= 2 {
		return true
	}
	
	return false
}

// parseTextTable 解析文本表格
func (p *TextProcessor) parseTextTable(lines []string, index *int) TableInfo {
	table := TableInfo{
		Caption: "",
		Headers: []string{},
		Rows:    [][]string{},
		Summary: "",
	}
	
	i := *index
	isFirstRow := true
	
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		
		if !p.isTableLine(line) {
			break
		}
		
		row := p.parseTextTableRow(line)
		if len(row) > 0 {
			if isFirstRow {
				table.Headers = row
				isFirstRow = false
			} else {
				table.Rows = append(table.Rows, row)
			}
		}
		
		i++
	}
	
	*index = i
	return table
}

// parseTextTableRow 解析文本表格行
func (p *TextProcessor) parseTextTableRow(line string) []string {
	var cells []string
	
	// 尝试制表符分隔
	if strings.Contains(line, "\t") {
		cells = strings.Split(line, "\t")
	} else if strings.Contains(line, "|") {
		// 尝试竖线分隔
		line = strings.Trim(line, "|")
		cells = strings.Split(line, "|")
	} else {
		// 尝试多空格分隔
		cells = regexp.MustCompile(`\s{3,}`).Split(line, -1)
	}
	
	// 清理每个单元格
	for i, cell := range cells {
		cells[i] = strings.TrimSpace(cell)
	}
	
	// 过滤空单元格
	var filteredCells []string
	for _, cell := range cells {
		if cell != "" {
			filteredCells = append(filteredCells, cell)
		}
	}
	
	return filteredCells
}

// extractLinks 提取链接信息
func (p *TextProcessor) extractLinks(content string) []LinkInfo {
	var links []LinkInfo
	
	// 匹配URL模式
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := urlRegex.FindAllString(content, -1)
	
	for _, match := range matches {
		link := LinkInfo{
			Text:  match,
			URL:   match,
			Title: "",
			Type:  "external",
		}
		links = append(links, link)
	}
	
	// 匹配邮箱地址
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emailMatches := emailRegex.FindAllString(content, -1)
	
	for _, match := range emailMatches {
		link := LinkInfo{
			Text:  match,
			URL:   "mailto:" + match,
			Title: "",
			Type:  "email",
		}
		links = append(links, link)
	}
	
	return links
}

// estimatePageCount 估算页数
func (p *TextProcessor) estimatePageCount(content string) int {
	// 按字符数估算页数（假设每页约2000字符）
	charCount := len(content)
	pageCount := charCount / 2000
	if pageCount == 0 {
		pageCount = 1
	}
	return pageCount
}