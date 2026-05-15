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
	if ctx.ShowCursor && ctx.IsFocused {
		content = InsertCursor(content, ctx.CursorPos, CursorChar)
	}
	return styles.TextStyle.Render(content)
}

// SupportedType returns the block type this drawer supports
func (d *TextDrawer) SupportedType() block.BlockType {
	return block.TypeText
}
