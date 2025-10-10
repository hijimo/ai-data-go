package processor

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ChunkStrategy 分块策略
type ChunkStrategy string

const (
	ChunkByFixedSize ChunkStrategy = "fixed_size" // 固定长度分块
	ChunkBySemantic  ChunkStrategy = "semantic"   // 语义分块
	ChunkByStructure ChunkStrategy = "structure"  // 结构化分块
	ChunkByCode      ChunkStrategy = "code"       // 代码分块
)

// ChunkConfig 分块配置
type ChunkConfig struct {
	Strategy        ChunkStrategy `json:"strategy"`         // 分块策略
	MaxSize         int           `json:"max_size"`         // 最大字符数
	Overlap         int           `json:"overlap"`          // 重叠字符数
	Separators      []string      `json:"separators"`       // 分隔符
	PreserveContext bool          `json:"preserve_context"` // 保持上下文完整性
}

// Chunk 文档块
type Chunk struct {
	Content     string                 `json:"content"`      // 块内容
	StartOffset int                    `json:"start_offset"` // 开始位置
	EndOffset   int                    `json:"end_offset"`   // 结束位置
	TokenCount  int                    `json:"token_count"`  // 词元数量
	Type        string                 `json:"type"`         // 块类型
	Metadata    map[string]interface{} `json:"metadata"`     // 元数据
}

// Chunker 分块器接口
type Chunker interface {
	Chunk(doc *Document, config *ChunkConfig) ([]Chunk, error)
}

// ChunkerManager 分块器管理器
type ChunkerManager struct {
	chunkers map[ChunkStrategy]Chunker
}

// NewChunkerManager 创建分块器管理器
func NewChunkerManager() *ChunkerManager {
	manager := &ChunkerManager{
		chunkers: make(map[ChunkStrategy]Chunker),
	}
	
	// 注册默认分块器
	manager.RegisterChunker(ChunkByFixedSize, NewFixedSizeChunker())
	manager.RegisterChunker(ChunkBySemantic, NewSemanticChunker())
	manager.RegisterChunker(ChunkByStructure, NewStructureChunker())
	manager.RegisterChunker(ChunkByCode, NewCodeChunker())
	
	return manager
}

// RegisterChunker 注册分块器
func (m *ChunkerManager) RegisterChunker(strategy ChunkStrategy, chunker Chunker) {
	m.chunkers[strategy] = chunker
}

// ChunkDocument 分块文档
func (m *ChunkerManager) ChunkDocument(doc *Document, config *ChunkConfig) ([]Chunk, error) {
	chunker, exists := m.chunkers[config.Strategy]
	if !exists {
		return nil, fmt.Errorf("不支持的分块策略: %s", config.Strategy)
	}
	
	return chunker.Chunk(doc, config)
}

// FixedSizeChunker 固定长度分块器
type FixedSizeChunker struct{}

// NewFixedSizeChunker 创建固定长度分块器
func NewFixedSizeChunker() Chunker {
	return &FixedSizeChunker{}
}

// Chunk 固定长度分块
func (c *FixedSizeChunker) Chunk(doc *Document, config *ChunkConfig) ([]Chunk, error) {
	var chunks []Chunk
	content := doc.Content
	
	if len(content) == 0 {
		return chunks, nil
	}
	
	maxSize := config.MaxSize
	if maxSize <= 0 {
		maxSize = 1500 // 默认1500字符
	}
	
	overlap := config.Overlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxSize {
		overlap = maxSize / 4 // 重叠不超过1/4
	}
	
	offset := 0
	chunkIndex := 0
	
	for offset < len(content) {
		end := offset + maxSize
		if end > len(content) {
			end = len(content)
		}
		
		// 如果需要保持上下文完整性，尝试在句子边界分割
		if config.PreserveContext && end < len(content) {
			end = c.findSentenceBoundary(content, offset, end)
		}
		
		chunkContent := content[offset:end]
		chunkContent = strings.TrimSpace(chunkContent)
		
		if len(chunkContent) > 0 {
			chunk := Chunk{
				Content:     chunkContent,
				StartOffset: offset,
				EndOffset:   end,
				TokenCount:  c.estimateTokenCount(chunkContent),
				Type:        "fixed_size",
				Metadata: map[string]interface{}{
					"chunk_index": chunkIndex,
					"strategy":    "fixed_size",
				},
			}
			chunks = append(chunks, chunk)
			chunkIndex++
		}
		
		// 计算下一个块的起始位置
		if end >= len(content) {
			break
		}
		
		offset = end - overlap
		if offset <= 0 {
			offset = end
		}
	}
	
	return chunks, nil
}

// findSentenceBoundary 查找句子边界
func (c *FixedSizeChunker) findSentenceBoundary(content string, start, maxEnd int) int {
	// 在最后200个字符中查找句子结束符
	searchStart := maxEnd - 200
	if searchStart < start {
		searchStart = start
	}
	
	searchContent := content[searchStart:maxEnd]
	
	// 查找句子结束符
	sentenceEnders := []string{"。", "！", "？", ".", "!", "?", "\n\n"}
	
	bestEnd := maxEnd
	for _, ender := range sentenceEnders {
		if pos := strings.LastIndex(searchContent, ender); pos != -1 {
			actualPos := searchStart + pos + len(ender)
			if actualPos > start && actualPos < bestEnd {
				bestEnd = actualPos
			}
		}
	}
	
	return bestEnd
}

// estimateTokenCount 估算词元数量
func (c *FixedSizeChunker) estimateTokenCount(content string) int {
	// 简单的词元估算：中文按字符数，英文按单词数
	chineseCount := 0
	englishWords := 0
	
	for _, r := range content {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		}
	}
	
	englishWords = len(strings.Fields(content))
	
	// 中文字符 + 英文单词数
	return chineseCount + englishWords
}

// SemanticChunker 语义分块器
type SemanticChunker struct{}

// NewSemanticChunker 创建语义分块器
func NewSemanticChunker() Chunker {
	return &SemanticChunker{}
}

// Chunk 语义分块
func (c *SemanticChunker) Chunk(doc *Document, config *ChunkConfig) ([]Chunk, error) {
	var chunks []Chunk
	
	// 基于段落进行语义分块
	paragraphs := strings.Split(doc.Content, "\n\n")
	
	currentChunk := strings.Builder{}
	startOffset := 0
	chunkIndex := 0
	
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		
		// 检查添加当前段落是否会超过最大长度
		if currentChunk.Len() > 0 && currentChunk.Len()+len(paragraph) > config.MaxSize {
			// 保存当前块
			content := strings.TrimSpace(currentChunk.String())
			if content != "" {
				chunk := Chunk{
					Content:     content,
					StartOffset: startOffset,
					EndOffset:   startOffset + len(content),
					TokenCount:  c.estimateTokenCount(content),
					Type:        "semantic",
					Metadata: map[string]interface{}{
						"chunk_index": chunkIndex,
						"strategy":    "semantic",
					},
				}
				chunks = append(chunks, chunk)
				chunkIndex++
			}
			
			// 开始新块
			currentChunk.Reset()
			startOffset = startOffset + len(content)
		}
		
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(paragraph)
	}
	
	// 保存最后一个块
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		chunk := Chunk{
			Content:     content,
			StartOffset: startOffset,
			EndOffset:   startOffset + len(content),
			TokenCount:  c.estimateTokenCount(content),
			Type:        "semantic",
			Metadata: map[string]interface{}{
				"chunk_index": chunkIndex,
				"strategy":    "semantic",
			},
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// estimateTokenCount 估算词元数量
func (c *SemanticChunker) estimateTokenCount(content string) int {
	return len(strings.Fields(content)) + utf8.RuneCountInString(content)/2
}

// StructureChunker 结构化分块器
type StructureChunker struct{}

// NewStructureChunker 创建结构化分块器
func NewStructureChunker() Chunker {
	return &StructureChunker{}
}

// Chunk 结构化分块
func (c *StructureChunker) Chunk(doc *Document, config *ChunkConfig) ([]Chunk, error) {
	var chunks []Chunk
	
	// 如果文档有结构信息，基于章节分块
	if doc.Structure != nil && len(doc.Structure.Sections) > 0 {
		for i, section := range doc.Structure.Sections {
			// 如果章节内容太长，进一步分割
			if len(section.Content) > config.MaxSize {
				subChunks := c.splitLongSection(section.Content, config, i)
				chunks = append(chunks, subChunks...)
			} else {
				chunk := Chunk{
					Content:     section.Content,
					StartOffset: section.Start,
					EndOffset:   section.End,
					TokenCount:  c.estimateTokenCount(section.Content),
					Type:        "structure",
					Metadata: map[string]interface{}{
						"section_title": section.Title,
						"section_level": section.Level,
						"chunk_index":   i,
						"strategy":      "structure",
					},
				}
				chunks = append(chunks, chunk)
			}
		}
	} else {
		// 如果没有结构信息，回退到语义分块
		semanticChunker := NewSemanticChunker()
		return semanticChunker.Chunk(doc, config)
	}
	
	return chunks, nil
}

// splitLongSection 分割长章节
func (c *StructureChunker) splitLongSection(content string, config *ChunkConfig, sectionIndex int) []Chunk {
	var chunks []Chunk
	
	// 使用固定长度分块器分割长章节
	fixedChunker := NewFixedSizeChunker()
	tempDoc := &Document{Content: content}
	tempChunks, _ := fixedChunker.Chunk(tempDoc, config)
	
	for i, tempChunk := range tempChunks {
		chunk := Chunk{
			Content:     tempChunk.Content,
			StartOffset: tempChunk.StartOffset,
			EndOffset:   tempChunk.EndOffset,
			TokenCount:  tempChunk.TokenCount,
			Type:        "structure_split",
			Metadata: map[string]interface{}{
				"section_index": sectionIndex,
				"sub_chunk":     i,
				"strategy":      "structure",
			},
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// estimateTokenCount 估算词元数量
func (c *StructureChunker) estimateTokenCount(content string) int {
	return len(strings.Fields(content)) + utf8.RuneCountInString(content)/2
}

// CodeChunker 代码分块器
type CodeChunker struct{}

// NewCodeChunker 创建代码分块器
func NewCodeChunker() Chunker {
	return &CodeChunker{}
}

// Chunk 代码分块
func (c *CodeChunker) Chunk(doc *Document, config *ChunkConfig) ([]Chunk, error) {
	var chunks []Chunk
	
	content := doc.Content
	
	// 检测是否为代码文件
	if !c.isCodeContent(content) {
		// 不是代码内容，使用结构化分块
		structureChunker := NewStructureChunker()
		return structureChunker.Chunk(doc, config)
	}
	
	// 基于函数/类/方法分块
	codeBlocks := c.extractCodeBlocks(content)
	
	chunkIndex := 0
	for _, block := range codeBlocks {
		if len(block.Content) > config.MaxSize {
			// 如果代码块太长，按行分割
			subChunks := c.splitLongCodeBlock(block, config, chunkIndex)
			chunks = append(chunks, subChunks...)
			chunkIndex += len(subChunks)
		} else {
			chunk := Chunk{
				Content:     block.Content,
				StartOffset: block.StartOffset,
				EndOffset:   block.EndOffset,
				TokenCount:  c.estimateTokenCount(block.Content),
				Type:        "code",
				Metadata: map[string]interface{}{
					"block_type":  block.Type,
					"block_name":  block.Name,
					"chunk_index": chunkIndex,
					"strategy":    "code",
				},
			}
			chunks = append(chunks, chunk)
			chunkIndex++
		}
	}
	
	return chunks, nil
}

// CodeBlock 代码块
type CodeBlock struct {
	Content     string `json:"content"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Type        string `json:"type"` // function, class, method, etc.
	Name        string `json:"name"`
}

// isCodeContent 检测是否为代码内容
func (c *CodeChunker) isCodeContent(content string) bool {
	// 简单的代码检测逻辑
	codeIndicators := []string{
		"function", "class", "def ", "public ", "private ", "protected ",
		"import ", "from ", "#include", "package ", "namespace ",
		"{", "}", "(", ")", ";", "//", "/*", "*/", "<!--", "-->",
	}
	
	indicatorCount := 0
	for _, indicator := range codeIndicators {
		if strings.Contains(content, indicator) {
			indicatorCount++
		}
	}
	
	// 如果包含多个代码指示符，认为是代码内容
	return indicatorCount >= 3
}

// extractCodeBlocks 提取代码块
func (c *CodeChunker) extractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	
	lines := strings.Split(content, "\n")
	currentBlock := strings.Builder{}
	blockStart := 0
	blockType := "code"
	blockName := ""
	
	for i, line := range lines {
		// 检测函数/类定义
		if c.isFunctionStart(line) {
			// 保存前一个块
			if currentBlock.Len() > 0 {
				blocks = append(blocks, CodeBlock{
					Content:     strings.TrimSpace(currentBlock.String()),
					StartOffset: blockStart,
					EndOffset:   i,
					Type:        blockType,
					Name:        blockName,
				})
			}
			
			// 开始新块
			currentBlock.Reset()
			blockStart = i
			blockType = "function"
			blockName = c.extractFunctionName(line)
		}
		
		currentBlock.WriteString(line)
		currentBlock.WriteString("\n")
	}
	
	// 保存最后一个块
	if currentBlock.Len() > 0 {
		blocks = append(blocks, CodeBlock{
			Content:     strings.TrimSpace(currentBlock.String()),
			StartOffset: blockStart,
			EndOffset:   len(lines),
			Type:        blockType,
			Name:        blockName,
		})
	}
	
	return blocks
}

// isFunctionStart 检测是否为函数开始
func (c *CodeChunker) isFunctionStart(line string) bool {
	line = strings.TrimSpace(line)
	
	// 常见的函数定义模式
	patterns := []string{
		`^function\s+\w+`,
		`^def\s+\w+`,
		`^public\s+.*\s+\w+\s*\(`,
		`^private\s+.*\s+\w+\s*\(`,
		`^protected\s+.*\s+\w+\s*\(`,
		`^\w+\s+\w+\s*\([^)]*\)\s*{`,
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}
	
	return false
}

// extractFunctionName 提取函数名
func (c *CodeChunker) extractFunctionName(line string) string {
	// 简单的函数名提取
	patterns := []string{
		`function\s+(\w+)`,
		`def\s+(\w+)`,
		`(\w+)\s*\([^)]*\)\s*{`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return "unknown"
}

// splitLongCodeBlock 分割长代码块
func (c *CodeChunker) splitLongCodeBlock(block CodeBlock, config *ChunkConfig, baseIndex int) []Chunk {
	var chunks []Chunk
	
	lines := strings.Split(block.Content, "\n")
	currentChunk := strings.Builder{}
	chunkStart := 0
	subIndex := 0
	
	for i, line := range lines {
		if currentChunk.Len()+len(line) > config.MaxSize && currentChunk.Len() > 0 {
			// 保存当前块
			content := strings.TrimSpace(currentChunk.String())
			chunk := Chunk{
				Content:     content,
				StartOffset: block.StartOffset + chunkStart,
				EndOffset:   block.StartOffset + i,
				TokenCount:  c.estimateTokenCount(content),
				Type:        "code_split",
				Metadata: map[string]interface{}{
					"parent_block": block.Name,
					"sub_index":    subIndex,
					"chunk_index":  baseIndex + subIndex,
					"strategy":     "code",
				},
			}
			chunks = append(chunks, chunk)
			
			// 开始新块
			currentChunk.Reset()
			chunkStart = i
			subIndex++
		}
		
		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
	}
	
	// 保存最后一个块
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		chunk := Chunk{
			Content:     content,
			StartOffset: block.StartOffset + chunkStart,
			EndOffset:   block.EndOffset,
			TokenCount:  c.estimateTokenCount(content),
			Type:        "code_split",
			Metadata: map[string]interface{}{
				"parent_block": block.Name,
				"sub_index":    subIndex,
				"chunk_index":  baseIndex + subIndex,
				"strategy":     "code",
			},
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// estimateTokenCount 估算词元数量
func (c *CodeChunker) estimateTokenCount(content string) int {
	// 代码的词元估算：按单词和符号计算
	tokens := regexp.MustCompile(`\w+|[^\w\s]`).FindAllString(content, -1)
	return len(tokens)
}