package factory

import (
	"fmt"
	"strings"
	"sync/atomic"

	"claudenelson/editor/block"
	"claudenelson/editor/format"
)

var idCounter uint64

// generateID creates a unique ID for a block
func generateID() string {
	id := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("block-%d", id)
}

// BlockFactory creates blocks from content or markdown-style lines
type BlockFactory struct{}

// NewBlockFactory creates a new BlockFactory
func NewBlockFactory() *BlockFactory {
	return &BlockFactory{}
}

// CreateFromLine parses a line of text and creates the appropriate block type
// Parsing rules:
// - "# " → H1
// - "## " → H2
// - "### " → H3
// - "#### " → H4
// - "- " → LIST_ITEM
// - "[] " → CHECKBOX (unchecked)
// - "[x] " → CHECKBOX (checked)
// - No prefix → TEXT (default)
func (f *BlockFactory) CreateFromLine(line string) block.Block {
	// Check for headings (must check longer prefixes first)
	if strings.HasPrefix(line, "#### ") {
		return block.NewHeadingBlock(generateID(), strings.TrimPrefix(line, "#### "), 4)
	}
	if strings.HasPrefix(line, "### ") {
		return block.NewHeadingBlock(generateID(), strings.TrimPrefix(line, "### "), 3)
	}
	if strings.HasPrefix(line, "## ") {
		return block.NewHeadingBlock(generateID(), strings.TrimPrefix(line, "## "), 2)
	}
	if strings.HasPrefix(line, "# ") {
		return block.NewHeadingBlock(generateID(), strings.TrimPrefix(line, "# "), 1)
	}

	// Check for checkbox items
	if strings.HasPrefix(line, "[x] ") {
		return block.NewCheckboxBlock(generateID(), strings.TrimPrefix(line, "[x] "), true)
	}
	if strings.HasPrefix(line, "[] ") {
		return block.NewCheckboxBlock(generateID(), strings.TrimPrefix(line, "[] "), false)
	}

	// Check for list items
	if strings.HasPrefix(line, "- ") {
		return block.NewListItemBlock(generateID(), strings.TrimPrefix(line, "- "))
	}

	// Default to text block
	return block.NewTextBlock(generateID(), line)
}

// Create creates a block of the specified type with the given content
func (f *BlockFactory) Create(blockType block.BlockType, content string) block.Block {
	id := generateID()

	switch blockType {
	case block.TypeH1:
		return block.NewHeadingBlock(id, content, 1)
	case block.TypeH2:
		return block.NewHeadingBlock(id, content, 2)
	case block.TypeH3:
		return block.NewHeadingBlock(id, content, 3)
	case block.TypeH4:
		return block.NewHeadingBlock(id, content, 4)
	case block.TypeListItem:
		return block.NewListItemBlock(id, content)
	case block.TypeCheckboxItem:
		return block.NewCheckboxBlock(id, content, false)
	default:
		return block.NewTextBlock(id, content)
	}
}

// CreateText creates a new text block
func (f *BlockFactory) CreateText(content string) *block.TextBlock {
	return block.NewTextBlock(generateID(), content)
}

// CreateHeading creates a new heading block
func (f *BlockFactory) CreateHeading(content string, level int) *block.HeadingBlock {
	return block.NewHeadingBlock(generateID(), content, level)
}

// CreateListItem creates a new list item block
func (f *BlockFactory) CreateListItem(content string) *block.ListItemBlock {
	return block.NewListItemBlock(generateID(), content)
}

// CreateCheckbox creates a new checkbox block
func (f *BlockFactory) CreateCheckbox(content string, checked bool) *block.CheckboxBlock {
	return block.NewCheckboxBlock(generateID(), content, checked)
}

// CreateTextWithSpans creates a new text block with formatting spans
func (f *BlockFactory) CreateTextWithSpans(content string, spans format.Spans) *block.TextBlock {
	b := block.NewTextBlock(generateID(), content)
	b.SetSpans(spans)
	return b
}

// CreateListItemWithSpans creates a new list item block with formatting spans
func (f *BlockFactory) CreateListItemWithSpans(content string, spans format.Spans) *block.ListItemBlock {
	b := block.NewListItemBlock(generateID(), content)
	b.SetSpans(spans)
	return b
}

// CreateCheckboxWithSpans creates a new checkbox block with formatting spans
func (f *BlockFactory) CreateCheckboxWithSpans(content string, checked bool, spans format.Spans) *block.CheckboxBlock {
	b := block.NewCheckboxBlock(generateID(), content, checked)
	b.SetSpans(spans)
	return b
}
