package processor

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// DOCXProcessor DOCX文档处理器
type DOCXProcessor struct{}

// NewDOCXProcessor 创建DOCX处理器
func NewDOCXProcessor() DocumentProcessor {
	return &DOCXProcessor{}
}

// SupportedTypes 支持的文件类型
func (p *DOCXProcessor) SupportedTypes() []string {
	return []string{".docx", ".doc"}
}

// Parse 解析DOCX文档
func (p *DOCXProcessor) Parse(ctx context.Context, reader io.Reader, metadata *FileMetadata) (*Document, error) {
	// 读取文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取DOCX文件失败: %w", err)
	}

	// 检查文件格式
	if !p.isValidDOCX(content) {
		return nil, fmt.Errorf("无效的DOCX文件格式")
	}

	// 解析DOCX文件
	doc, err := p.parseDOCX(content, metadata)
	if err != nil {
		return nil, fmt.Errorf("解析DOCX文件失败: %w", err)
	}

	return doc, nil
}

// isValidDOCX 检查是否为有效的DOCX文件
func (p *DOCXProcessor) isValidDOCX(content []byte) bool {
	// DOCX文件是ZIP格式，检查ZIP文件头
	if len(content) < 4 {
		return false
	}
	
	// ZIP文件头: PK\x03\x04
	return content[0] == 0x50 && content[1] == 0x4B && 
		   (content[2] == 0x03 || content[2] == 0x05 || content[2] == 0x07) &&
		   (content[3] == 0x04 || content[3] == 0x06 || content[3] == 0x08)
}

// parseDOCX 解析DOCX文件
func (p *DOCXProcessor) parseDOCX(content []byte, metadata *FileMetadata) (*Document, error) {
	// 创建ZIP读取器
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("打开DOCX文件失败: %w", err)
	}

	doc := &Document{
		Title:    extractTitleFromFilename(metadata.Filename),
		Content:  "",
		Metadata: map[string]interface{}{
			"file_type":    "docx",
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
		PageCount: 1,
	}

	// 解析文档内容
	for _, file := range zipReader.File {
		switch file.Name {
		case "word/document.xml":
			// 主文档内容
			if err := p.parseDocumentXML(file, doc); err != nil {
				return nil, err
			}
		case "docProps/core.xml":
			// 文档属性
			if err := p.parseCoreProperties(file, doc); err != nil {
				// 属性解析失败不影响主要内容
				continue
			}
		case "word/styles.xml":
			// 样式信息（用于识别标题）
			if err := p.parseStyles(file, doc); err != nil {
				// 样式解析失败不影响主要内容
				continue
			}
		}
	}

	// 提取文档结构
	doc.Structure = p.extractStructureFromContent(doc.Content)

	return doc, nil
}

// parseDocumentXML 解析主文档XML
func (p *DOCXProcessor) parseDocumentXML(file *zip.File, doc *Document) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	// 解析XML并提取文本
	text, tables, links := p.extractTextFromXML(content)
	doc.Content = text
	doc.Tables = tables
	doc.Links = links

	return nil
}

// parseCoreProperties 解析文档核心属性
func (p *DOCXProcessor) parseCoreProperties(file *zip.File, doc *Document) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	// 简单的XML解析提取标题和作者
	var coreProps struct {
		Title   string `xml:"title"`
		Creator string `xml:"creator"`
		Subject string `xml:"subject"`
	}

	if err := xml.Unmarshal(content, &coreProps); err == nil {
		if coreProps.Title != "" {
			doc.Title = coreProps.Title
		}
		if coreProps.Creator != "" {
			doc.Metadata["author"] = coreProps.Creator
		}
		if coreProps.Subject != "" {
			doc.Metadata["subject"] = coreProps.Subject
		}
	}

	return nil
}

// parseStyles 解析样式信息
func (p *DOCXProcessor) parseStyles(file *zip.File, doc *Document) error {
	// 样式解析用于更好地识别标题
	// 这里是简化实现
	return nil
}

// extractTextFromXML 从XML中提取文本、表格和链接
func (p *DOCXProcessor) extractTextFromXML(xmlContent []byte) (string, []TableInfo, []LinkInfo) {
	var textBuilder strings.Builder
	var tables []TableInfo
	var links []LinkInfo

	// 简化的XML解析
	content := string(xmlContent)
	
	// 提取段落文本 <w:t>...</w:t>
	textStart := 0
	for {
		start := strings.Index(content[textStart:], "<w:t")
		if start == -1 {
			break
		}
		start += textStart
		
		// 找到标签结束
		tagEnd := strings.Index(content[start:], ">")
		if tagEnd == -1 {
			break
		}
		tagEnd += start + 1
		
		// 找到文本结束
		end := strings.Index(content[tagEnd:], "</w:t>")
		if end == -1 {
			break
		}
		end += tagEnd
		
		// 提取文本内容
		text := content[tagEnd:end]
		text = p.decodeXMLText(text)
		
		if strings.TrimSpace(text) != "" {
			textBuilder.WriteString(text)
			textBuilder.WriteString(" ")
		}
		
		textStart = end + 6 // 跳过 </w:t>
	}

	// 提取表格（简化实现）
	tables = p.extractTables(content)
	
	// 提取链接（简化实现）
	links = p.extractLinks(content)

	// 清理文本
	finalText := textBuilder.String()
	finalText = strings.ReplaceAll(finalText, "\r", "\n")
	finalText = strings.ReplaceAll(finalText, "\n\n", "\n")
	
	return strings.TrimSpace(finalText), tables, links
}

// decodeXMLText 解码XML文本
func (p *DOCXProcessor) decodeXMLText(text string) string {
	// 解码XML实体
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	
	return text
}

// extractTables 提取表格信息
func (p *DOCXProcessor) extractTables(content string) []TableInfo {
	var tables []TableInfo
	
	// 查找表格标签 <w:tbl>
	tableStart := 0
	for {
		start := strings.Index(content[tableStart:], "<w:tbl")
		if start == -1 {
			break
		}
		start += tableStart
		
		end := strings.Index(content[start:], "</w:tbl>")
		if end == -1 {
			break
		}
		end += start + 8 // 包含 </w:tbl>
		
		// 解析表格内容
		tableXML := content[start:end]
		table := p.parseTable(tableXML)
		if len(table.Rows) > 0 {
			tables = append(tables, table)
		}
		
		tableStart = end
	}
	
	return tables
}

// parseTable 解析单个表格
func (p *DOCXProcessor) parseTable(tableXML string) TableInfo {
	table := TableInfo{
		Caption: "",
		Headers: []string{},
		Rows:    [][]string{},
		Summary: "",
	}
	
	// 查找表格行 <w:tr>
	rowStart := 0
	isFirstRow := true
	
	for {
		start := strings.Index(tableXML[rowStart:], "<w:tr")
		if start == -1 {
			break
		}
		start += rowStart
		
		end := strings.Index(tableXML[start:], "</w:tr>")
		if end == -1 {
			break
		}
		end += start + 7 // 包含 </w:tr>
		
		// 解析行内容
		rowXML := tableXML[start:end]
		cells := p.parseTableRow(rowXML)
		
		if isFirstRow && len(cells) > 0 {
			table.Headers = cells
			isFirstRow = false
		} else if len(cells) > 0 {
			table.Rows = append(table.Rows, cells)
		}
		
		rowStart = end
	}
	
	return table
}

// parseTableRow 解析表格行
func (p *DOCXProcessor) parseTableRow(rowXML string) []string {
	var cells []string
	
	// 查找表格单元格 <w:tc>
	cellStart := 0
	for {
		start := strings.Index(rowXML[cellStart:], "<w:tc")
		if start == -1 {
			break
		}
		start += cellStart
		
		end := strings.Index(rowXML[start:], "</w:tc>")
		if end == -1 {
			break
		}
		end += start + 7 // 包含 </w:tc>
		
		// 提取单元格文本
		cellXML := rowXML[start:end]
		cellText := p.extractCellText(cellXML)
		cells = append(cells, cellText)
		
		cellStart = end
	}
	
	return cells
}

// extractCellText 提取单元格文本
func (p *DOCXProcessor) extractCellText(cellXML string) string {
	var textBuilder strings.Builder
	
	// 查找文本标签 <w:t>
	textStart := 0
	for {
		start := strings.Index(cellXML[textStart:], "<w:t")
		if start == -1 {
			break
		}
		start += textStart
		
		tagEnd := strings.Index(cellXML[start:], ">")
		if tagEnd == -1 {
			break
		}
		tagEnd += start + 1
		
		end := strings.Index(cellXML[tagEnd:], "</w:t>")
		if end == -1 {
			break
		}
		end += tagEnd
		
		text := cellXML[tagEnd:end]
		text = p.decodeXMLText(text)
		textBuilder.WriteString(text)
		textBuilder.WriteString(" ")
		
		textStart = end + 6
	}
	
	return strings.TrimSpace(textBuilder.String())
}

// extractLinks 提取链接信息
func (p *DOCXProcessor) extractLinks(content string) []LinkInfo {
	var links []LinkInfo
	
	// 查找超链接标签 <w:hyperlink>
	linkStart := 0
	for {
		start := strings.Index(content[linkStart:], "<w:hyperlink")
		if start == -1 {
			break
		}
		start += linkStart
		
		end := strings.Index(content[start:], "</w:hyperlink>")
		if end == -1 {
			break
		}
		end += start + 14 // 包含 </w:hyperlink>
		
		// 解析链接
		linkXML := content[start:end]
		link := p.parseLink(linkXML)
		if link.URL != "" || link.Text != "" {
			links = append(links, link)
		}
		
		linkStart = end
	}
	
	return links
}

// parseLink 解析链接
func (p *DOCXProcessor) parseLink(linkXML string) LinkInfo {
	link := LinkInfo{
		Text:  "",
		URL:   "",
		Title: "",
		Type:  "external",
	}
	
	// 提取链接文本
	text, _, _ := p.extractTextFromXML([]byte(linkXML))
	link.Text = text
	
	// 提取URL（简化实现）
	if strings.Contains(linkXML, "r:id=") {
		// 这里需要解析关系文件来获取实际URL
		// 简化实现中暂时跳过
		link.URL = "需要解析关系文件"
	}
	
	return link
}

// extractStructureFromContent 从内容中提取结构
func (p *DOCXProcessor) extractStructureFromContent(content string) *DocumentStructure {
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
		
		// 简单的标题检测
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
func (p *DOCXProcessor) isHeading(line string) bool {
	// 简单的标题判断逻辑
	if len(line) < 3 || len(line) > 100 {
		return false
	}
	
	// 检查是否以数字开头
	if len(line) > 2 && line[1] == '.' && line[0] >= '0' && line[0] <= '9' {
		return true
	}
	
	// 检查常见标题模式
	titlePatterns := []string{"第", "章", "节", "部分", "Chapter", "Section", "Part"}
	for _, pattern := range titlePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	
	return false
}