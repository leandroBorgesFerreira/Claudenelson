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
		return RenderFormattedContentFull(content, spans, styles.TextStyle, ctx.CursorPos, ctx.SelectionStart, ctx.SelectionEnd, ctx.LineSelected)
	}
	if ctx.LineSelected {
		return RenderFormattedContentLineSelected(content, spans, styles.TextStyle)
	}
	return RenderFormattedContentWithSelection(content, spans, styles.TextStyle, ctx.SelectionStart, ctx.SelectionEnd)
}

// HandleMouse handles mouse events for text blocks
func (d *TextDrawer) HandleMouse(b block.Block, ctx MouseContext) Action {
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
			// Single click: set cursor and start drag
			return Action{
				Type:      ActionStartDrag,
				CursorCol: col,
				SelStart:  col,
				SelEnd:    col,
			}
		case 2:
			// Double click: select word
			start, end := getWordBoundsAt(content, col)
			return Action{
				Type:      ActionSelectWord,
				CursorCol: end,
				SelStart:  start,
				SelEnd:    end,
			}
		default:
			// Triple click: select whole line
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

// PrefixWidth returns the width of the text block prefix (none)
func (d *TextDrawer) PrefixWidth() int {
	return 0
}

// SupportedType returns the block type this drawer supports
func (d *TextDrawer) SupportedType() block.BlockType {
	return block.TypeText
}

// getWordBoundsAt returns the start and end of the word at the given position
func getWordBoundsAt(content []rune, col int) (start, end int) {
	if len(content) == 0 {
		return 0, 0
	}

	// Clamp column to valid range
	if col < 0 {
		col = 0
	}
	if col >= len(content) {
		col = len(content) - 1
	}
	if col < 0 {
		return 0, 0
	}

	// Check if we're on a word character
	isWordChar := func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_'
	}

	// If not on a word character, just select the single character
	if !isWordChar(content[col]) {
		return col, col + 1
	}

	// Find start of word
	start = col
	for start > 0 && isWordChar(content[start-1]) {
		start--
	}

	// Find end of word
	end = col
	for end < len(content) && isWordChar(content[end]) {
		end++
	}

	return start, end
}
