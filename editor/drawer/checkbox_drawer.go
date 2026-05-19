package drawer

import (
	"github.com/charmbracelet/lipgloss"

	"claudenelson/editor/block"
	"claudenelson/editor/styles"
)

// CheckboxDrawer renders checkbox/todo item blocks
type CheckboxDrawer struct{}

const checkboxPrefixWidth = 4 // "[x] " or "[ ] " = 4 characters

// Draw renders a checkbox block with a checkbox prefix
func (d *CheckboxDrawer) Draw(b block.Block, ctx DrawContext) string {
	// Render handle first
	handle := RenderHandle(ctx)

	cb, ok := b.(*block.CheckboxBlock)
	if !ok {
		// Fallback rendering if not a CheckboxBlock
		content := b.Content()
		spans := b.Spans()
		if ctx.ShowCursor && ctx.IsFocused {
			return handle + RenderFormattedContentFull(content, spans, styles.TextStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
		}
		if ctx.LineSelected {
			return handle + RenderFormattedContentLineSelected(content, spans, styles.TextStyle)
		}
		return handle + RenderFormattedContentWithSelection(content, spans, styles.TextStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	content := cb.Content()
	spans := cb.Spans()

	var prefix string
	var contentStyle = styles.CheckboxContentStyle

	lineSelectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))

	if cb.IsChecked() {
		if ctx.LineSelected {
			prefix = lineSelectedStyle.Render("[x] ")
		} else {
			prefix = styles.CheckboxCheckedStyle.Render("[x] ")
		}
		contentStyle = styles.CheckboxCheckedContentStyle
	} else {
		if ctx.LineSelected {
			prefix = lineSelectedStyle.Render("[ ] ")
		} else {
			prefix = styles.CheckboxUncheckedStyle.Render("[ ] ")
		}
	}

	var styledContent string
	if ctx.ShowCursor && ctx.IsFocused {
		styledContent = RenderFormattedContentFull(content, spans, contentStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
	} else if ctx.LineSelected {
		styledContent = RenderFormattedContentLineSelected(content, spans, contentStyle)
	} else {
		styledContent = RenderFormattedContentWithSelection(content, spans, contentStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	return handle + prefix + styledContent
}

// PrefixWidth returns the width of the checkbox prefix
func (d *CheckboxDrawer) PrefixWidth() int {
	return checkboxPrefixWidth
}

// SupportedType returns the block type this drawer supports
func (d *CheckboxDrawer) SupportedType() block.BlockType {
	return block.TypeCheckboxItem
}
