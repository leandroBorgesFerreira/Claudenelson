package drawer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudenelson/editor/block"
	"claudenelson/editor/format"
)

// CursorChar is the character used to display the cursor
const CursorChar = "│"

// DrawContext contains context information for drawing a block
type DrawContext struct {
	Width      int
	IsFocused  bool
	CursorPos  int
	LineNumber int
	ShowCursor bool
}

// InsertCursor inserts a cursor character at the specified position in content
func InsertCursor(content string, pos int, cursorChar string) string {
	runes := []rune(content)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	result := make([]rune, 0, len(runes)+len([]rune(cursorChar)))
	result = append(result, runes[:pos]...)
	result = append(result, []rune(cursorChar)...)
	result = append(result, runes[pos:]...)
	return string(result)
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
	if len(spans) == 0 {
		return baseStyle.Render(content)
	}

	runes := []rune(content)
	if len(runes) == 0 {
		return ""
	}

	var result strings.Builder

	// Build a style map for each character position
	styleMap := make([]format.Style, len(runes))
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
			}
		}
	}

	// Group consecutive characters with the same style
	i := 0
	for i < len(runes) {
		currentStyle := styleMap[i]
		j := i + 1
		for j < len(runes) && styleMap[j] == currentStyle {
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

		segment := string(runes[i:j])
		result.WriteString(segmentStyle.Render(segment))

		i = j
	}

	return result.String()
}

// RenderFormattedContentWithCursor renders content with formatting and cursor
func RenderFormattedContentWithCursor(content string, spans format.Spans, baseStyle lipgloss.Style, cursorPos int) string {
	if cursorPos < 0 {
		cursorPos = 0
	}
	runes := []rune(content)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}

	// Insert cursor into content
	contentWithCursor := InsertCursor(content, cursorPos, CursorChar)

	// Adjust spans for the cursor insertion
	adjustedSpans := make(format.Spans, len(spans))
	for i, span := range spans {
		adjustedSpan := span
		if span.Start >= cursorPos {
			adjustedSpan.Start++
		}
		if span.End > cursorPos {
			adjustedSpan.End++
		}
		adjustedSpans[i] = adjustedSpan
	}

	return RenderFormattedContent(contentWithCursor, adjustedSpans, baseStyle)
}
