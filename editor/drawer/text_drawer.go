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

	if ctx.ShowCursor && ctx.IsFocused {
		return RenderFormattedContentWithCursorAndSelection(content, spans, styles.TextStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd)
	}
	return RenderFormattedContentWithSelection(content, spans, styles.TextStyle, ctx.SelectionStart, ctx.SelectionEnd)
}

// SupportedType returns the block type this drawer supports
func (d *TextDrawer) SupportedType() block.BlockType {
	return block.TypeText
}
