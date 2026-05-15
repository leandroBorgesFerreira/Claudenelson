package drawer

import (
	"strings"

	"claudenelson/editor/block"
	"claudenelson/editor/styles"
)

// HeadingDrawer renders heading blocks (H1-H4)
type HeadingDrawer struct {
	Level int
}

// Draw renders a heading block with styled prefix and content
func (d *HeadingDrawer) Draw(b block.Block, ctx DrawContext) string {
	level := d.Level

	// If the block is a HeadingBlock, use its level
	if hb, ok := b.(*block.HeadingBlock); ok {
		level = hb.Level()
	}

	prefix := strings.Repeat("#", level) + " "
	content := b.Content()

	if ctx.ShowCursor && ctx.IsFocused {
		content = InsertCursor(content, ctx.CursorPos, CursorChar)
	}

	var contentStyle = styles.H1Style
	switch level {
	case 2:
		contentStyle = styles.H2Style
	case 3:
		contentStyle = styles.H3Style
	case 4:
		contentStyle = styles.H4Style
	}

	styledPrefix := styles.HeadingPrefixStyle.Render(prefix)
	styledContent := contentStyle.Render(content)

	return styledPrefix + styledContent
}

// SupportedType returns the block type this drawer supports
func (d *HeadingDrawer) SupportedType() block.BlockType {
	switch d.Level {
	case 1:
		return block.TypeH1
	case 2:
		return block.TypeH2
	case 3:
		return block.TypeH3
	case 4:
		return block.TypeH4
	default:
		return block.TypeH1
	}
}
