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
		if ctx.ShowCursor && ctx.IsFocused {
			content = InsertCursor(content, ctx.CursorPos, CursorChar)
		}
		return styles.TextStyle.Render(content)
	}

	content := cb.Content()
	if ctx.ShowCursor && ctx.IsFocused {
		content = InsertCursor(content, ctx.CursorPos, CursorChar)
	}

	var prefix, styledContent string

	if cb.IsChecked() {
		prefix = styles.CheckboxCheckedStyle.Render("[x] ")
		styledContent = styles.CheckboxCheckedContentStyle.Render(content)
	} else {
		prefix = styles.CheckboxUncheckedStyle.Render("[ ] ")
		styledContent = styles.CheckboxContentStyle.Render(content)
	}

	return prefix + styledContent
}

// SupportedType returns the block type this drawer supports
func (d *CheckboxDrawer) SupportedType() block.BlockType {
	return block.TypeCheckboxItem
}
