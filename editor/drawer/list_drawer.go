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
	spans := b.Spans()
	prefix := styles.ListPrefixStyle.Render("• ")

	var styledContent string
	if ctx.ShowCursor && ctx.IsFocused {
		styledContent = RenderFormattedContentWithCursorAndSelection(content, spans, styles.ListContentStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd)
	} else {
		styledContent = RenderFormattedContentWithSelection(content, spans, styles.ListContentStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	return prefix + styledContent
}

// SupportedType returns the block type this drawer supports
func (d *ListDrawer) SupportedType() block.BlockType {
	return block.TypeListItem
}
