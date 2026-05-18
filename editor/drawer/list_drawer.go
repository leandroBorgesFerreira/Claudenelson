package drawer

import (
	"github.com/charmbracelet/lipgloss"

	"claudenelson/editor/block"
	"claudenelson/editor/styles"
)

// ListDrawer renders list item blocks
type ListDrawer struct{}

const listPrefixWidth = 2 // "• " = 2 characters (bullet + space)

// Draw renders a list item block with a bullet prefix
func (d *ListDrawer) Draw(b block.Block, ctx DrawContext) string {
	content := b.Content()
	spans := b.Spans()

	var prefix string
	if ctx.LineSelected {
		prefix = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Render("• ")
	} else {
		prefix = styles.ListPrefixStyle.Render("• ")
	}

	var styledContent string
	if ctx.ShowCursor && ctx.IsFocused {
		styledContent = RenderFormattedContentFull(content, spans, styles.ListContentStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
	} else if ctx.LineSelected {
		styledContent = RenderFormattedContentLineSelected(content, spans, styles.ListContentStyle)
	} else {
		styledContent = RenderFormattedContentWithSelection(content, spans, styles.ListContentStyle, ctx.SelectionStart, ctx.SelectionEnd)
	}

	return prefix + styledContent
}

// HandleMouse handles mouse events for list blocks
func (d *ListDrawer) HandleMouse(b block.Block, ctx MouseContext) Action {
	content := []rune(b.Content())
	contentLen := len(content)

	// Clamp X to valid range
	col := ctx.X
	if col < 0 {
		col = 0
	}
	if col > contentLen {
		col = contentLen
	}

	switch ctx.EventType {
	case MousePress:
		switch ctx.ClickCount {
		case 1:
			return Action{
				Type:      ActionStartDrag,
				CursorCol: col,
				SelStart:  col,
				SelEnd:    col,
			}
		case 2:
			start, end := getWordBoundsAt(content, col)
			return Action{
				Type:      ActionSelectWord,
				CursorCol: end,
				SelStart:  start,
				SelEnd:    end,
			}
		default:
			return Action{
				Type:      ActionSelectLine,
				CursorCol: contentLen,
				SelStart:  0,
				SelEnd:    contentLen,
			}
		}

	case MouseMotion:
		if ctx.IsDragging {
			return Action{
				Type:      ActionExtendDrag,
				CursorCol: col,
			}
		}

	case MouseRelease:
		return Action{
			Type: ActionEndDrag,
		}
	}

	return Action{Type: ActionNone}
}

// PrefixWidth returns the width of the list item prefix
func (d *ListDrawer) PrefixWidth() int {
	return listPrefixWidth
}

// SupportedType returns the block type this drawer supports
func (d *ListDrawer) SupportedType() block.BlockType {
	return block.TypeListItem
}
