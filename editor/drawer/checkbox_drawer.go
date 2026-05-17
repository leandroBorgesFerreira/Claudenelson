package drawer

import (
	"github.com/charmbracelet/lipgloss"

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
			return RenderFormattedContentFull(content, spans, styles.TextStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
		}
		if ctx.LineSelected {
			return RenderFormattedContentLineSelected(content, spans, styles.TextStyle)
		}
		return RenderFormattedContentWithSelection(content, spans, styles.TextStyle, ctx.SelectionStart, ctx.SelectionEnd)
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

	return prefix + styledContent
}

// SupportedType returns the block type this drawer supports
func (d *CheckboxDrawer) SupportedType() block.BlockType {
	return block.TypeCheckboxItem
}
