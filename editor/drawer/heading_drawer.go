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

// Draw renders a heading block with styled content (no prefix)
func (d *HeadingDrawer) Draw(b block.Block, ctx DrawContext) string {
	// Render handle first
	handle := RenderHandle(ctx)

	level := d.Level

	// If the block is a HeadingBlock, use its level
	if hb, ok := b.(*block.HeadingBlock); ok {
		level = hb.Level()
	}

	content := b.Content()
	spans := b.Spans()

	// H1 is rendered in uppercase for emphasis
	if level == 1 {
		content = strings.ToUpper(content)
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

	var textContent string
	if ctx.ShowCursor && ctx.IsFocused {
		textContent = RenderFormattedContentFull(content, spans, contentStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
	} else if ctx.LineSelected {
		textContent = RenderFormattedContentLineSelected(content, spans, contentStyle)
	} else {
		textContent = RenderFormattedContentWithSelection(content, spans, contentStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	return handle + textContent
}

// PrefixWidth returns the width of the heading prefix (none)
func (d *HeadingDrawer) PrefixWidth() int {
	return 0
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
