package drawer

import (
	"claudenelson/editor/block"
	"claudenelson/editor/styles"
)

// ListDrawer renders list item blocks
type ListDrawer struct{}

// Draw renders a list item block with a bullet prefix
func (d *ListDrawer) Draw(b block.Block, ctx DrawContext) string {
	content := b.Content()
	if ctx.ShowCursor && ctx.IsFocused {
		content = InsertCursor(content, ctx.CursorPos, CursorChar)
	}
	prefix := styles.ListPrefixStyle.Render("• ")
	styledContent := styles.ListContentStyle.Render(content)

	return prefix + styledContent
}

// SupportedType returns the block type this drawer supports
func (d *ListDrawer) SupportedType() block.BlockType {
	return block.TypeListItem
}
