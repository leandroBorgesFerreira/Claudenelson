package drawer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudenelson/editor/block"
	"claudenelson/editor/format"
)

// ActionType represents what action the drawer wants the editor to take
type ActionType int

const (
	ActionNone         ActionType = iota
	ActionSetCursor               // Set cursor to position
	ActionSelectWord              // Select the word at position
	ActionSelectLine              // Select the entire line
	ActionToggleCheck             // Toggle checkbox state
	ActionStartDrag               // Start drag selection
	ActionExtendDrag              // Extend drag selection
	ActionEndDrag                 // End drag selection
)

// Action represents an action returned by a drawer's mouse handler
type Action struct {
	Type      ActionType
	CursorCol int // Cursor column position
	SelStart  int // Selection start (for word/line selection)
	SelEnd    int // Selection end (for word/line selection)
}

// MouseEventType represents the type of mouse event
type MouseEventType int

const (
	MousePress MouseEventType = iota
	MouseRelease
	MouseMotion
)

// MouseContext contains mouse event information passed to a drawer
type MouseContext struct {
	X          int            // X position relative to block content start (after prefix)
	RawX       int            // Raw X position (for prefix detection)
	EventType  MouseEventType // Press, Release, or Motion
	ClickCount int            // 1=single, 2=double, 3=triple click
	IsDragging bool           // Whether currently dragging
}

// DrawContext contains context information for drawing a block
type DrawContext struct {
	Width            int
	IsFocused        bool
	CursorPos        int
	LineNumber       int
	ShowCursor       bool
	SelectionStart   int  // -1 if no selection (within line)
	SelectionEnd     int  // -1 if no selection (within line)
	LineSelected     bool // True if entire line is selected (multi-line selection)
	IsHovered        bool // True if mouse is hovering over this line
	IsHandleSelected bool // True if this line is selected via handle click
}

// RenderHandle renders the selection handle "|| " based on context
func RenderHandle(ctx DrawContext) string {
	showHandle := ctx.IsHovered || ctx.LineSelected || ctx.IsHandleSelected
	if showHandle {
		if ctx.LineSelected || ctx.IsHandleSelected {
			// Selected style (bright)
			return HandleSelectedStyle.Render("||") + " "
		}
		// Hover style (dim)
		return HandleStyle.Render("||") + " "
	}
	// Invisible but takes space
	return "   "
}

// Handle styles (defined here to avoid import cycle with styles package)
var (
	HandleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	HandleSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)
)

// Drawer is the interface for rendering blocks
type Drawer interface {
	Draw(b block.Block, ctx DrawContext) string
	HandleMouse(b block.Block, ctx MouseContext) Action
	PrefixWidth() int // Returns the width of the block's prefix (bullet, checkbox, etc.)
	SupportedType() block.BlockType
}

// BlockDrawer represents a drawable block with its position
type BlockDrawer struct {
	Block      block.Block
	Drawer     Drawer
	Y          int  // Screen Y position
	BlockIndex int  // Index in document
}

// Draw renders the block
func (bd *BlockDrawer) Draw(ctx DrawContext) string {
	return bd.Drawer.Draw(bd.Block, ctx)
}

// ContainsY returns true if the given screen Y is on this drawer
func (bd *BlockDrawer) ContainsY(y int) bool {
	return y == bd.Y
}

// HandleMouse handles mouse event if it's on this drawer, returns action and whether it handled it
func (bd *BlockDrawer) HandleMouse(screenY int, ctx MouseContext) (Action, bool) {
	if !bd.ContainsY(screenY) {
		return Action{Type: ActionNone}, false
	}
	action := bd.Drawer.HandleMouse(bd.Block, ctx)
	return action, true
}

// PrefixWidth returns the prefix width of the underlying drawer
func (bd *BlockDrawer) PrefixWidth() int {
	return bd.Drawer.PrefixWidth()
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

// HandleMouse delegates mouse handling to the appropriate drawer
func (r *DrawerRegistry) HandleMouse(b block.Block, ctx MouseContext) Action {
	if d, ok := r.drawers[b.Type()]; ok {
		return d.HandleMouse(b, ctx)
	}
	// Fallback to text drawer if no specific drawer found
	if d, ok := r.drawers[block.TypeText]; ok {
		return d.HandleMouse(b, ctx)
	}
	return Action{Type: ActionNone}
}

// PrefixWidth returns the prefix width for a block type
func (r *DrawerRegistry) PrefixWidth(b block.Block) int {
	if d, ok := r.drawers[b.Type()]; ok {
		return d.PrefixWidth()
	}
	return 0
}

// CreateBlockDrawer creates a BlockDrawer for the given block at the specified position
func (r *DrawerRegistry) CreateBlockDrawer(b block.Block, y int, blockIndex int) *BlockDrawer {
	var drw Drawer
	if d, ok := r.drawers[b.Type()]; ok {
		drw = d
	} else if d, ok := r.drawers[block.TypeText]; ok {
		drw = d
	} else {
		return nil
	}

	return &BlockDrawer{
		Block:      b,
		Drawer:     drw,
		Y:          y,
		BlockIndex: blockIndex,
	}
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
