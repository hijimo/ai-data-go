package processor

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// HTMLProcessor HTML文档处理器
type HTMLProcessor struct{}

// NewHTMLProcessor 创建HTML处理器
func NewHTMLProcessor() DocumentProcessor {
	return &HTMLProcessor{}
}

// SupportedTypes 支持的文件类型
func (p *HTMLProcessor) SupportedTypes() []string {
	return []string{".html", ".htm"}
}

// Parse 解析HTML文档
func (p *HTMLProcessor) Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error) {
	// 读取文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取HTML文件失败: %w", err)
	}

	html := string(content)

	doc := &Document{
		Title:    p.extractTitle(html),
		Content:  p.extractTextContent(html),
		Metadata: map[string]interface{}{
			"file_type":    "html",
			"file_size":    metadata.Size,
			"sha256":       metadata.SHA256,
			"content_type": metadata.ContentType,
		},
		Structure: &DocumentStructure{
			Headings: []HeadingInfo{},
			Sections: []Section{},
		},
		Images:    p.extractImages(html),
		Tables:    p.extractTables(html),
		Links:     p.extractLinks(html),
		Language:  p.extractLanguage(html),
		WordCount: 0,
		PageCount: 1,
	}

	// 提取文档结构
	doc.Structure = p.extractStructure(html)

	return doc, nil
}

// extractTitle 提取HTML标题
func (p *HTMLProcessor) extractTitle(html string) string {
	// 查找<title>标签
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	matches := titleRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		title := p.cleanHTMLText(matches[1])
		if strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
	}

	// 查找第一个h1标签
	h1Regex := regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`)
	matches = h1Regex.FindStringSubmatch(html)
	if len(matches) > 1 {
		title := p.cleanHTMLText(matches[1])
		if strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
	}

	return "HTML文档"
}

// extractTextContent 提取HTML文本内容
func (p *HTMLProcessor) extractTextContent(html string) string {
	// 移除脚本和样式标签
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	html = scriptRegex.ReplaceAllString(html, "")
	
	styleRegex := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	html = styleRegex.ReplaceAllString(html, "")

	// 移除注释
	commentRegex := regexp.MustCompile(`<!--.*?-->`)
	html = commentRegex.ReplaceAllString(html, "")

	// 将块级元素转换为换行
	blockElements := []string{"div", "p", "h1", "h2", "h3", "h4", "h5", "h6", "li", "br"}
	for _, element := range blockElements {
		regex := regexp.MustCompile(`(?i)<` + element + `[^>]*>`)
		html = regex.ReplaceAllString(html, "\n")
		regex = regexp.MustCompile(`(?i)</` + element + `>`)
		html = regex.ReplaceAllString(html, "\n")
	}

	// 移除所有HTML标签
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text := tagRegex.ReplaceAllString(html, "")

	// 解码HTML实体
	text = p.decodeHTMLEntities(text)

	// 清理文本
	return p.cleanText(text)
}

// extractStructure 提取HTML文档结构
func (p *HTMLProcessor) extractStructure(html string) *DocumentStructure {
	structure := &DocumentStructure{
		Headings: []HeadingInfo{},
		Sections: []Section{},
	}

	// 提取标题标签 h1-h6
	for level := 1; level <= 6; level++ {
		headingRegex := regexp.MustCompile(fmt.Sprintf(`(?i)<h%d[^>]*>(.*?)</h%d>`, level, level))
		matches := headingRegex.FindAllStringSubmatch(html, -1)

		for _, match := range matches {
			if len(match) > 1 {
				text := p.cleanHTMLText(match[1])
				if strings.TrimSpace(text) != "" {
					heading := HeadingInfo{
						Level:  level,
						Text:   strings.TrimSpace(text),
						Offset: 0, // HTML中位置计算较复杂，这里简化
						ID:     p.generateHeadingID(text),
					}
					structure.Headings = append(structure.Headings, heading)
				}
			}
		}
	}

	// 基于标题创建章节
	structure.Sections = p.createSectionsFromHeadings(structure.Headings, html)

	return structure
}

// createSectionsFromHeadings 基于标题创建章节
func (p *HTMLProcessor) createSectionsFromHeadings(headings []HeadingInfo, html string) []Section {
	var sections []Section

	for i, heading := range headings {
		section := Section{
			Title:   heading.Text,
			Content: "",
			Level:   heading.Level,
			Start:   0,
			End:     0,
		}

		// 提取章节内容（简化实现）
		// 在实际项目中，应该更精确地定位章节内容
		if i < len(headings)-1 {
			// 不是最后一个标题，内容到下一个同级或更高级标题
			section.Content = fmt.Sprintf("章节内容：%s", heading.Text)
		} else {
			// 最后一个标题，内容到文档结束
			section.Content = fmt.Sprintf("章节内容：%s", heading.Text)
		}

		sections = append(sections, section)
	}

	return sections
}

// extractImages 提取HTML图片信息
func (p *HTMLProcessor) extractImages(html string) []ImageInfo {
	var images []ImageInfo

	// 匹配img标签
	imgRegex := regexp.MustCompile(`(?i)<img[^>]*>`)
	matches := imgRegex.FindAllString(html, -1)

	for _, match := range matches {
		image := ImageInfo{
			Alt:         p.extractAttribute(match, "alt"),
			Title:       p.extractAttribute(match, "title"),
			URL:         p.extractAttribute(match, "src"),
			Width:       p.parseIntAttribute(match, "width"),
			Height:      p.parseIntAttribute(match, "height"),
			Size:        0,
			Format:      p.getImageFormatFromURL(p.extractAttribute(match, "src")),
			Description: p.extractAttribute(match, "alt"),
		}

		if image.URL != "" {
			images = append(images, image)
		}
	}

	return images
}

// extractTables 提取HTML表格信息
func (p *HTMLProcessor) extractTables(html string) []TableInfo {
	var tables []TableInfo

	// 匹配table标签
	tableRegex := regexp.MustCompile(`(?i)<table[^>]*>(.*?)</table>`)
	matches := tableRegex.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 {
			table := p.parseHTMLTable(match[1])
			if len(table.Rows) > 0 || len(table.Headers) > 0 {
				tables = append(tables, table)
			}
		}
	}

	return tables
}

// parseHTMLTable 解析HTML表格
func (p *HTMLProcessor) parseHTMLTable(tableHTML string) TableInfo {
	table := TableInfo{
		Caption: "",
		Headers: []string{},
		Rows:    [][]string{},
		Summary: "",
	}

	// 提取表格标题
	captionRegex := regexp.MustCompile(`(?i)<caption[^>]*>(.*?)</caption>`)
	if matches := captionRegex.FindStringSubmatch(tableHTML); len(matches) > 1 {
		table.Caption = p.cleanHTMLText(matches[1])
	}

	// 提取表头
	theadRegex := regexp.MustCompile(`(?i)<thead[^>]*>(.*?)</thead>`)
	if matches := theadRegex.FindStringSubmatch(tableHTML); len(matches) > 1 {
		table.Headers = p.parseTableRow(matches[1], "th")
	}

	// 如果没有thead，尝试第一行tr作为表头
	if len(table.Headers) == 0 {
		trRegex := regexp.MustCompile(`(?i)<tr[^>]*>(.*?)</tr>`)
		if matches := trRegex.FindStringSubmatch(tableHTML); len(matches) > 1 {
			if strings.Contains(matches[1], "<th") {
				table.Headers = p.parseTableRow(matches[1], "th")
			}
		}
	}

	// 提取表格数据
	tbodyRegex := regexp.MustCompile(`(?i)<tbody[^>]*>(.*?)</tbody>`)
	tbody := tableHTML
	if matches := tbodyRegex.FindStringSubmatch(tableHTML); len(matches) > 1 {
		tbody = matches[1]
	}

	trRegex := regexp.MustCompile(`(?i)<tr[^>]*>(.*?)</tr>`)
	trMatches := trRegex.FindAllStringSubmatch(tbody, -1)

	for _, trMatch := range trMatches {
		if len(trMatch) > 1 {
			// 跳过表头行
			if len(table.Headers) > 0 && strings.Contains(trMatch[1], "<th") {
				continue
			}
			row := p.parseTableRow(trMatch[1], "td")
			if len(row) > 0 {
				table.Rows = append(table.Rows, row)
			}
		}
	}

	return table
}

// parseTableRow 解析表格行
func (p *HTMLProcessor) parseTableRow(rowHTML, cellTag string) []string {
	var cells []string

	cellRegex := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>(.*?)</%s>`, cellTag, cellTag))
	matches := cellRegex.FindAllStringSubmatch(rowHTML, -1)

	for _, match := range matches {
		if len(match) > 1 {
			cellText := p.cleanHTMLText(match[1])
			cells = append(cells, strings.TrimSpace(cellText))
		}
	}

	return cells
}

// extractLinks 提取HTML链接信息
func (p *HTMLProcessor) extractLinks(html string) []LinkInfo {
	var links []LinkInfo

	// 匹配a标签
	linkRegex := regexp.MustCompile(`(?i)<a[^>]*>(.*?)</a>`)
	matches := linkRegex.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 {
			link := LinkInfo{
				Text:  p.cleanHTMLText(match[1]),
				URL:   p.extractAttribute(match[0], "href"),
				Title: p.extractAttribute(match[0], "title"),
				Type:  "",
			}

			if link.URL != "" {
				link.Type = p.getLinkType(link.URL)
				links = append(links, link)
			}
		}
	}

	return links
}

// extractLanguage 提取HTML语言信息
func (p *HTMLProcessor) extractLanguage(html string) string {
	// 查找html标签的lang属性
	langRegex := regexp.MustCompile(`(?i)<html[^>]*lang\s*=\s*["']([^"']+)["']`)
	matches := langRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}

	// 查找meta标签的语言信息
	metaRegex := regexp.MustCompile(`(?i)<meta[^>]*http-equiv\s*=\s*["']content-language["'][^>]*content\s*=\s*["']([^"']+)["']`)
	matches = metaRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// 辅助方法

// cleanHTMLText 清理HTML文本
func (p *HTMLProcessor) cleanHTMLText(text string) string {
	// 移除HTML标签
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text = tagRegex.ReplaceAllString(text, "")

	// 解码HTML实体
	text = p.decodeHTMLEntities(text)

	return text
}

// decodeHTMLEntities 解码HTML实体
func (p *HTMLProcessor) decodeHTMLEntities(text string) string {
	entities := map[string]string{
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&apos;":   "'",
		"&nbsp;":   " ",
		"&copy;":   "©",
		"&reg;":    "®",
		"&trade;":  "™",
		"&hellip;": "…",
		"&mdash;":  "—",
		"&ndash;":  "–",
		"&lsquo;":  "'",
		"&rsquo;":  "'",
		"&ldquo;":  """,
		"&rdquo;":  """,
	}

	for entity, replacement := range entities {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// 处理数字实体 &#123; 和 &#x1A;
	numericRegex := regexp.MustCompile(`&#(\d+);`)
	text = numericRegex.ReplaceAllStringFunc(text, func(match string) string {
		// 简化处理，实际应该转换为对应的Unicode字符
		return ""
	})

	hexRegex := regexp.MustCompile(`&#x([0-9A-Fa-f]+);`)
	text = hexRegex.ReplaceAllStringFunc(text, func(match string) string {
		// 简化处理，实际应该转换为对应的Unicode字符
		return ""
	})

	return text
}

// extractAttribute 提取HTML属性值
func (p *HTMLProcessor) extractAttribute(tag, attr string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`(?i)%s\s*=\s*["']([^"']*)["']`, attr))
	matches := regex.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseIntAttribute 解析整数属性
func (p *HTMLProcessor) parseIntAttribute(tag, attr string) int {
	value := p.extractAttribute(tag, attr)
	if value == "" {
		return 0
	}
	
	// 简单的整数解析
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}

// getImageFormatFromURL 从URL获取图片格式
func (p *HTMLProcessor) getImageFormatFromURL(url string) string {
	url = strings.ToLower(url)
	if strings.Contains(url, ".jpg") || strings.Contains(url, ".jpeg") {
		return "jpeg"
	} else if strings.Contains(url, ".png") {
		return "png"
	} else if strings.Contains(url, ".gif") {
		return "gif"
	} else if strings.Contains(url, ".svg") {
		return "svg"
	} else if strings.Contains(url, ".webp") {
		return "webp"
	}
	return "unknown"
}

// getLinkType 获取链接类型
func (p *HTMLProcessor) getLinkType(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return "external"
	} else if strings.HasPrefix(url, "#") {
		return "anchor"
	} else if strings.HasPrefix(url, "mailto:") {
		return "email"
	} else if strings.HasPrefix(url, "tel:") {
		return "phone"
	} else {
		return "internal"
	}
}

// generateHeadingID 生成标题ID
func (p *HTMLProcessor) generateHeadingID(title string) string {
	// 转换为小写并替换空格为连字符
	id := strings.ToLower(title)
	id = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(id, "")
	id = regexp.MustCompile(`\s+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
	return id
}

// cleanText 清理文本
func (p *HTMLProcessor) cleanText(text string) string {
	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	// 将多个连续的换行符替换为两个
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	
	return text
}