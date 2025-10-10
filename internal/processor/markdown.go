package processor

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// MarkdownProcessor Markdown文档处理器
type MarkdownProcessor struct{}

// NewMarkdownProcessor 创建Markdown处理器
func NewMarkdownProcessor() DocumentProcessor {
	return &MarkdownProcessor{}
}

// SupportedTypes 支持的文件类型
func (p *MarkdownProcessor) SupportedTypes() []string {
	return []string{".md", ".markdown"}
}

// Parse 解析Markdown文档
func (p *MarkdownProcessor) Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error) {
	// 读取文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取Markdown文件失败: %w", err)
	}

	text := string(content)

	doc := &Document{
		Title:    p.extractTitle(text, metadata.Filename),
		Content:  text,
		Metadata: map[string]interface{}{
			"file_type":    "markdown",
			"file_size":    metadata.Size,
			"sha256":       metadata.SHA256,
			"content_type": metadata.ContentType,
		},
		Structure: p.extractStructure(text),
		Images:    p.extractImages(text),
		Tables:    p.extractTables(text),
		Links:     p.extractLinks(text),
		Language:  "",
		WordCount: 0,
		PageCount: 1,
	}

	return doc, nil
}

// extractTitle 提取文档标题
func (p *MarkdownProcessor) extractTitle(content, filename string) string {
	lines := strings.Split(content, "\n")
	
	// 查找第一个一级标题
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimSpace(line[2:])
			if title != "" {
				return title
			}
		}
	}
	
	// 查找YAML front matter中的标题
	if strings.HasPrefix(content, "---") {
		endIndex := strings.Index(content[3:], "---")
		if endIndex != -1 {
			frontMatter := content[3 : endIndex+3]
			lines := strings.Split(frontMatter, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "title:") {
					title := strings.TrimSpace(line[6:])
					title = strings.Trim(title, "\"'")
					if title != "" {
						return title
					}
				}
			}
		}
	}
	
	// 使用文件名作为标题
	return extractTitleFromFilename(filename)
}

// extractStructure 提取文档结构
func (p *MarkdownProcessor) extractStructure(content string) *DocumentStructure {
	structure := &DocumentStructure{
		Headings: []HeadingInfo{},
		Sections: []Section{},
	}
	
	lines := strings.Split(content, "\n")
	var currentSections []*Section
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// 检查是否为标题
		if level, title := p.parseHeading(line); level > 0 {
			// 添加标题信息
			heading := HeadingInfo{
				Level:  level,
				Text:   title,
				Offset: i,
				ID:     p.generateHeadingID(title),
			}
			structure.Headings = append(structure.Headings, heading)
			
			// 结束之前的章节
			p.finalizeSections(currentSections, level)
			
			// 开始新章节
			section := &Section{
				Title:   title,
				Content: "",
				Level:   level,
				Start:   i,
				End:     i,
			}
			
			// 调整章节层级
			if level <= len(currentSections) {
				currentSections = currentSections[:level-1]
			}
			currentSections = append(currentSections, section)
			structure.Sections = append(structure.Sections, *section)
		} else if len(currentSections) > 0 {
			// 添加内容到当前章节
			lastSection := currentSections[len(currentSections)-1]
			if lastSection.Content != "" {
				lastSection.Content += "\n"
			}
			lastSection.Content += line
			lastSection.End = i
		}
	}
	
	return structure
}

// parseHeading 解析标题
func (p *MarkdownProcessor) parseHeading(line string) (int, string) {
	// ATX风格标题 (# ## ### ...)
	if strings.HasPrefix(line, "#") {
		level := 0
		for i, char := range line {
			if char == '#' {
				level++
			} else if char == ' ' {
				title := strings.TrimSpace(line[i:])
				if title != "" && level <= 6 {
					return level, title
				}
				break
			} else {
				break
			}
		}
	}
	
	return 0, ""
}

// generateHeadingID 生成标题ID
func (p *MarkdownProcessor) generateHeadingID(title string) string {
	// 转换为小写并替换空格为连字符
	id := strings.ToLower(title)
	id = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(id, "")
	id = regexp.MustCompile(`\s+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
	
	return id
}

// finalizeSections 完成章节内容
func (p *MarkdownProcessor) finalizeSections(sections []*Section, newLevel int) {
	for _, section := range sections {
		section.Content = strings.TrimSpace(section.Content)
	}
}

// extractImages 提取图片信息
func (p *MarkdownProcessor) extractImages(content string) []ImageInfo {
	var images []ImageInfo
	
	// 匹配Markdown图片语法: ![alt](url "title")
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)(?:\s+"([^"]*)")?\)`)
	matches := imageRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		image := ImageInfo{
			Alt:         match[1],
			URL:         match[2],
			Title:       "",
			Width:       0,
			Height:      0,
			Size:        0,
			Format:      p.getImageFormat(match[2]),
			Description: match[1],
		}
		
		if len(match) > 3 && match[3] != "" {
			image.Title = match[3]
		}
		
		images = append(images, image)
	}
	
	return images
}

// extractTables 提取表格信息
func (p *MarkdownProcessor) extractTables(content string) []TableInfo {
	var tables []TableInfo
	
	lines := strings.Split(content, "\n")
	i := 0
	
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		
		// 检查是否为表格行（包含 | 分隔符）
		if strings.Contains(line, "|") && p.isTableRow(line) {
			table := p.parseMarkdownTable(lines, &i)
			if len(table.Rows) > 0 || len(table.Headers) > 0 {
				tables = append(tables, table)
			}
		} else {
			i++
		}
	}
	
	return tables
}

// isTableRow 检查是否为表格行
func (p *MarkdownProcessor) isTableRow(line string) bool {
	// 简单检查：包含至少两个 | 字符
	return strings.Count(line, "|") >= 2
}

// parseMarkdownTable 解析Markdown表格
func (p *MarkdownProcessor) parseMarkdownTable(lines []string, index *int) TableInfo {
	table := TableInfo{
		Caption: "",
		Headers: []string{},
		Rows:    [][]string{},
		Summary: "",
	}
	
	i := *index
	
	// 解析表头
	if i < len(lines) {
		headerLine := strings.TrimSpace(lines[i])
		if p.isTableRow(headerLine) {
			table.Headers = p.parseTableRow(headerLine)
			i++
		}
	}
	
	// 跳过分隔行（如果存在）
	if i < len(lines) {
		separatorLine := strings.TrimSpace(lines[i])
		if p.isTableSeparator(separatorLine) {
			i++
		}
	}
	
	// 解析表格数据行
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if !p.isTableRow(line) {
			break
		}
		
		row := p.parseTableRow(line)
		if len(row) > 0 {
			table.Rows = append(table.Rows, row)
		}
		i++
	}
	
	*index = i
	return table
}

// isTableSeparator 检查是否为表格分隔行
func (p *MarkdownProcessor) isTableSeparator(line string) bool {
	// 表格分隔行包含 - 和 | 字符
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	
	return cleaned == ""
}

// parseTableRow 解析表格行
func (p *MarkdownProcessor) parseTableRow(line string) []string {
	// 移除首尾的 | 字符
	line = strings.Trim(line, "|")
	
	// 按 | 分割
	cells := strings.Split(line, "|")
	
	// 清理每个单元格
	for i, cell := range cells {
		cells[i] = strings.TrimSpace(cell)
	}
	
	return cells
}

// extractLinks 提取链接信息
func (p *MarkdownProcessor) extractLinks(content string) []LinkInfo {
	var links []LinkInfo
	
	// 匹配Markdown链接语法: [text](url "title")
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)(?:\s+"([^"]*)")?\)`)
	matches := linkRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		link := LinkInfo{
			Text:  match[1],
			URL:   match[2],
			Title: "",
			Type:  p.getLinkType(match[2]),
		}
		
		if len(match) > 3 && match[3] != "" {
			link.Title = match[3]
		}
		
		links = append(links, link)
	}
	
	// 匹配引用式链接: [text][ref]
	refLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\[([^\]]*)\]`)
	refMatches := refLinkRegex.FindAllStringSubmatch(content, -1)
	
	// 提取引用定义: [ref]: url "title"
	refDefRegex := regexp.MustCompile(`^\[([^\]]+)\]:\s*([^\s]+)(?:\s+"([^"]*)")?`)
	refDefs := make(map[string]LinkInfo)
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := refDefRegex.FindStringSubmatch(line); matches != nil {
			refDefs[matches[1]] = LinkInfo{
				URL:   matches[2],
				Title: matches[3],
				Type:  p.getLinkType(matches[2]),
			}
		}
	}
	
	// 解析引用式链接
	for _, match := range refMatches {
		text := match[1]
		ref := match[2]
		if ref == "" {
			ref = text // 如果没有指定引用，使用文本作为引用
		}
		
		if refDef, exists := refDefs[ref]; exists {
			link := LinkInfo{
				Text:  text,
				URL:   refDef.URL,
				Title: refDef.Title,
				Type:  refDef.Type,
			}
			links = append(links, link)
		}
	}
	
	return links
}

// getImageFormat 获取图片格式
func (p *MarkdownProcessor) getImageFormat(url string) string {
	url = strings.ToLower(url)
	if strings.HasSuffix(url, ".jpg") || strings.HasSuffix(url, ".jpeg") {
		return "jpeg"
	} else if strings.HasSuffix(url, ".png") {
		return "png"
	} else if strings.HasSuffix(url, ".gif") {
		return "gif"
	} else if strings.HasSuffix(url, ".svg") {
		return "svg"
	} else if strings.HasSuffix(url, ".webp") {
		return "webp"
	}
	return "unknown"
}

// getLinkType 获取链接类型
func (p *MarkdownProcessor) getLinkType(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return "external"
	} else if strings.HasPrefix(url, "#") {
		return "anchor"
	} else if strings.HasPrefix(url, "mailto:") {
		return "email"
	} else {
		return "internal"
	}
}