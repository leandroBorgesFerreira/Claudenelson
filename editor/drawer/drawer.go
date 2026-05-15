package drawer

import "claudenelson/editor/block"

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
