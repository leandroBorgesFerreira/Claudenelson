package drawer

import (
	"claudenelson/editor/block"
	"claudenelson/editor/styles"
)

// TextDrawer renders plain text blocks
type TextDrawer struct{}

// Draw renders a text block
func (d *TextDrawer) Draw(b block.Block, ctx DrawContext) string {
	content := b.Content()
	spans := b.Spans()

	// Render handle first
	handle := RenderHandle(ctx)

	var textContent string
	if ctx.ShowCursor && ctx.IsFocused {
		textContent = RenderFormattedContentFull(content, spans, styles.TextStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
	} else if ctx.LineSelected {
		textContent = RenderFormattedContentLineSelected(content, spans, styles.TextStyle)
	} else {
		textContent = RenderFormattedContentWithSelection(content, spans, styles.TextStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	return handle + textContent
}

// PrefixWidth returns the width of the text block prefix (none)
func (d *TextDrawer) PrefixWidth() int {
	return 0
}

// SupportedType returns the block type this drawer supports
func (d *TextDrawer) SupportedType() block.BlockType {
	return block.TypeText
}
