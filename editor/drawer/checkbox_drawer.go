package drawer

import (
	"claudenelson/editor/block"
	"claudenelson/editor/styles"
)

// CheckboxDrawer renders checkbox/todo item blocks
type CheckboxDrawer struct{}

// Draw renders a checkbox block with a checkbox prefix
func (d *CheckboxDrawer) Draw(b block.Block, ctx DrawContext) string {
	cb, ok := b.(*block.CheckboxBlock)
	if !ok {
		// Fallback rendering if not a CheckboxBlock
		content := b.Content()
		spans := b.Spans()
		if ctx.ShowCursor && ctx.IsFocused {
			return RenderFormattedContentWithCursorAndSelection(content, spans, styles.TextStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd)
		}
		return RenderFormattedContentWithSelection(content, spans, styles.TextStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	content := cb.Content()
	spans := cb.Spans()

	var prefix string
	var contentStyle = styles.CheckboxContentStyle

	if cb.IsChecked() {
		prefix = styles.CheckboxCheckedStyle.Render("[x] ")
		contentStyle = styles.CheckboxCheckedContentStyle
	} else {
		prefix = styles.CheckboxUncheckedStyle.Render("[ ] ")
	}

	var styledContent string
	if ctx.ShowCursor && ctx.IsFocused {
		styledContent = RenderFormattedContentWithCursorAndSelection(content, spans, contentStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd)
	} else {
		styledContent = RenderFormattedContentWithSelection(content, spans, contentStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	return prefix + styledContent
}

// SupportedType returns the block type this drawer supports
func (d *CheckboxDrawer) SupportedType() block.BlockType {
	return block.TypeCheckboxItem
}
