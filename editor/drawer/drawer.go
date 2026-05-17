package drawer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudenelson/editor/block"
	"claudenelson/editor/format"
)

// DrawContext contains context information for drawing a block
type DrawContext struct {
	Width          int
	IsFocused      bool
	CursorPos      int
	LineNumber     int
	ShowCursor     bool
	SelectionStart int  // -1 if no selection (within line)
	SelectionEnd   int  // -1 if no selection (within line)
	LineSelected   bool // True if entire line is selected (multi-line selection)
}

// Drawer is the interface for rendering blocks
type Drawer interface {
	Draw(b block.Block, ctx DrawContext) string
	SupportedType() block.BlockType
}

// DrawerRegistry maps BlockType to Drawer implementations
type DrawerRegistry struct {
	drawers map[block.BlockType]Drawer
}

// NewDrawerRegistry creates a new DrawerRegistry
func NewDrawerRegistry() *DrawerRegistry {
	return &DrawerRegistry{
		drawers: make(map[block.BlockType]Drawer),
	}
}

// Register adds a drawer to the registry
func (r *DrawerRegistry) Register(d Drawer) {
	r.drawers[d.SupportedType()] = d
}

// Draw renders a block using the appropriate drawer
func (r *DrawerRegistry) Draw(b block.Block, ctx DrawContext) string {
	if d, ok := r.drawers[b.Type()]; ok {
		return d.Draw(b, ctx)
	}
	// Fallback to text drawer if no specific drawer found
	if d, ok := r.drawers[block.TypeText]; ok {
		return d.Draw(b, ctx)
	}
	return b.Content()
}

// RegisterAll registers all standard drawers
func (r *DrawerRegistry) RegisterAll() {
	r.Register(&TextDrawer{})
	r.Register(&HeadingDrawer{Level: 1})
	r.Register(&HeadingDrawer{Level: 2})
	r.Register(&HeadingDrawer{Level: 3})
	r.Register(&HeadingDrawer{Level: 4})
	r.Register(&ListDrawer{})
	r.Register(&CheckboxDrawer{})
}

// RenderFormattedContent renders content with formatting spans applied
// baseStyle is the default style for unformatted text
func RenderFormattedContent(content string, spans format.Spans, baseStyle lipgloss.Style) string {
	return RenderFormattedContentWithSelection(content, spans, baseStyle, -1, -1)
}

// RenderFormattedContentWithSelection renders content with formatting and optional selection highlight
func RenderFormattedContentWithSelection(content string, spans format.Spans, baseStyle lipgloss.Style, selStart, selEnd int) string {
	return renderFormattedContentInternal(content, spans, baseStyle, -1, selStart, selEnd, false)
}

// RenderFormattedContentLineSelected renders content with entire line selected
func RenderFormattedContentLineSelected(content string, spans format.Spans, baseStyle lipgloss.Style) string {
	return renderFormattedContentInternal(content, spans, baseStyle, -1, -1, -1, true)
}

// renderFormattedContentInternal is the internal renderer with all options
func renderFormattedContentInternal(content string, spans format.Spans, baseStyle lipgloss.Style, cursorPos int, selStart, selEnd int, lineSelected bool) string {
	runes := []rune(content)

	// Handle empty content
	if len(runes) == 0 {
		if lineSelected {
			// Show at least one space for empty selected lines
			return lipgloss.NewStyle().Background(lipgloss.Color("62")).Render(" ")
		}
		if len(spans) == 0 {
			return baseStyle.Render(content)
		}
		return ""
	}

	var result strings.Builder

	// Build a style map for each character position
	styleMap := make([]format.Style, len(runes))
	selectionMap := make([]bool, len(runes))

	for _, span := range spans {
		for i := span.Start; i < span.End && i < len(runes); i++ {
			if i >= 0 {
				if span.Style.Bold {
					styleMap[i].Bold = true
				}
				if span.Style.Italic {
					styleMap[i].Italic = true
				}
				if span.Style.Underline {
					styleMap[i].Underline = true
				}
				if span.Style.Highlight {
					styleMap[i].Highlight = true
				}
			}
		}
	}

	// Mark selection range (within line)
	if selStart >= 0 && selEnd >= 0 && selStart != selEnd {
		start, end := selStart, selEnd
		if start > end {
			start, end = end, start
		}
		for i := start; i < end && i < len(runes); i++ {
			if i >= 0 {
				selectionMap[i] = true
			}
		}
	}

	// Group consecutive characters with the same style and selection state
	i := 0
	for i < len(runes) {
		currentStyle := styleMap[i]
		currentSelection := selectionMap[i]
		j := i + 1
		for j < len(runes) && styleMap[j] == currentStyle && selectionMap[j] == currentSelection {
			j++
		}

		// Build the style for this segment
		segmentStyle := baseStyle
		if currentStyle.Bold {
			segmentStyle = segmentStyle.Bold(true)
		}
		if currentStyle.Italic {
			segmentStyle = segmentStyle.Italic(true)
		}
		if currentStyle.Underline {
			segmentStyle = segmentStyle.Underline(true)
		}
		if currentStyle.Highlight {
			// Yellow highlight background
			segmentStyle = segmentStyle.Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))
		}
		if currentSelection {
			// Selection highlight (blue background)
			segmentStyle = segmentStyle.Background(lipgloss.Color("39")).Foreground(lipgloss.Color("0"))
		}
		if lineSelected {
			// Line selection (purple background)
			segmentStyle = segmentStyle.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
		}

		segment := string(runes[i:j])
		result.WriteString(segmentStyle.Render(segment))

		i = j
	}

	return result.String()
}

// RenderFormattedContentWithCursor renders content with formatting and block cursor
func RenderFormattedContentWithCursor(content string, spans format.Spans, baseStyle lipgloss.Style, cursorPos int) string {
	return RenderFormattedContentWithCursorAndSelection(content, spans, baseStyle, cursorPos, -1, -1)
}

// RenderContentWithBlockCursor renders plain content with a block cursor (no spans)
func RenderContentWithBlockCursor(content string, baseStyle lipgloss.Style, cursorPos int) string {
	return RenderFormattedContentWithCursorAndSelection(content, nil, baseStyle, cursorPos, -1, -1)
}

// RenderFormattedContentWithCursorAndSelection renders content with formatting, block cursor, and selection
func RenderFormattedContentWithCursorAndSelection(content string, spans format.Spans, baseStyle lipgloss.Style, cursorPos int, selStart, selEnd int) string {
	return RenderFormattedContentFull(content, spans, baseStyle, cursorPos, selStart, selEnd, false)
}

// RenderFormattedContentFull renders content with all options including line selection
func RenderFormattedContentFull(content string, spans format.Spans, baseStyle lipgloss.Style, cursorPos int, selStart, selEnd int, lineSelected bool) string {
	if cursorPos < 0 {
		cursorPos = 0
	}
	runes := []rune(content)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}

	// If cursor is at end of content, append a space for the block cursor
	atEnd := cursorPos >= len(runes)
	if atEnd {
		runes = append(runes, ' ')
	}

	if len(runes) == 0 {
		// Empty content with cursor - show block cursor on space
		cursorStyle := lipgloss.NewStyle().Reverse(true)
		if lineSelected {
			cursorStyle = cursorStyle.Background(lipgloss.Color("62"))
		}
		return cursorStyle.Render(" ")
	}

	var result strings.Builder

	// Build a style map for each character position
	styleMap := make([]format.Style, len(runes))
	selectionMap := make([]bool, len(runes))

	for _, span := range spans {
		for i := span.Start; i < span.End && i < len(runes); i++ {
			if i >= 0 {
				if span.Style.Bold {
					styleMap[i].Bold = true
				}
				if span.Style.Italic {
					styleMap[i].Italic = true
				}
				if span.Style.Underline {
					styleMap[i].Underline = true
				}
				if span.Style.Highlight {
					styleMap[i].Highlight = true
				}
			}
		}
	}

	// Mark selection range
	if selStart >= 0 && selEnd >= 0 && selStart != selEnd {
		start, end := selStart, selEnd
		if start > end {
			start, end = end, start
		}
		for i := start; i < end && i < len(runes); i++ {
			if i >= 0 {
				selectionMap[i] = true
			}
		}
	}

	// Render each character, applying block cursor at cursorPos
	for i, r := range runes {
		currentStyle := styleMap[i]
		isSelected := selectionMap[i]
		isCursor := i == cursorPos

		// Build the style for this character
		charStyle := baseStyle
		if currentStyle.Bold {
			charStyle = charStyle.Bold(true)
		}
		if currentStyle.Italic {
			charStyle = charStyle.Italic(true)
		}
		if currentStyle.Underline {
			charStyle = charStyle.Underline(true)
		}
		if currentStyle.Highlight {
			charStyle = charStyle.Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))
		}
		if isSelected {
			charStyle = charStyle.Background(lipgloss.Color("39")).Foreground(lipgloss.Color("0"))
		}
		if lineSelected && !isCursor {
			// Line selection (purple background)
			charStyle = charStyle.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
		}
		if isCursor {
			// Block cursor - invert colors
			charStyle = charStyle.Reverse(true)
		}

		result.WriteString(charStyle.Render(string(r)))
	}

	return result.String()
}
