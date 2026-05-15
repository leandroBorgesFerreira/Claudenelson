package document

import "claudenelson/editor/block"

// Document represents a collection of blocks with cursor state
type Document struct {
	Blocks     []block.Block
	CursorLine int
	CursorCol  int
}

// NewDocument creates a new empty document
func NewDocument() *Document {
	return &Document{
		Blocks:     make([]block.Block, 0),
		CursorLine: 0,
	}
}

// AddBlock adds a block to the end of the document
func (d *Document) AddBlock(b block.Block) {
	d.Blocks = append(d.Blocks, b)
}

// InsertBlock inserts a block at the specified index
func (d *Document) InsertBlock(index int, b block.Block) {
	if index < 0 {
		index = 0
	}
	if index > len(d.Blocks) {
		index = len(d.Blocks)
	}

	d.Blocks = append(d.Blocks[:index], append([]block.Block{b}, d.Blocks[index:]...)...)
}

// RemoveBlock removes a block at the specified index
func (d *Document) RemoveBlock(index int) {
	if index < 0 || index >= len(d.Blocks) {
		return
	}

	d.Blocks = append(d.Blocks[:index], d.Blocks[index+1:]...)

	// Adjust cursor if needed
	if d.CursorLine >= len(d.Blocks) && len(d.Blocks) > 0 {
		d.CursorLine = len(d.Blocks) - 1
	}
}

// MoveUp moves the cursor up one block
func (d *Document) MoveUp() bool {
	if d.CursorLine > 0 {
		d.CursorLine--
		d.clampCursorCol()
		return true
	}
	return false
}

// MoveDown moves the cursor down one block
func (d *Document) MoveDown() bool {
	if d.CursorLine < len(d.Blocks)-1 {
		d.CursorLine++
		d.clampCursorCol()
		return true
	}
	return false
}

// MoveLeft moves the cursor left one character
func (d *Document) MoveLeft() bool {
	if d.CursorCol > 0 {
		d.CursorCol--
		return true
	}
	return false
}

// MoveRight moves the cursor right one character
func (d *Document) MoveRight() bool {
	b := d.CurrentBlock()
	if b == nil {
		return false
	}
	content := []rune(b.Content())
	if d.CursorCol < len(content) {
		d.CursorCol++
		return true
	}
	return false
}

// MoveToLineStart moves the cursor to the start of the line
func (d *Document) MoveToLineStart() {
	d.CursorCol = 0
}

// MoveToLineEnd moves the cursor to the end of the line
func (d *Document) MoveToLineEnd() {
	b := d.CurrentBlock()
	if b == nil {
		return
	}
	d.CursorCol = len([]rune(b.Content()))
}

// InsertChar inserts a character at the cursor position
func (d *Document) InsertChar(ch rune) {
	b := d.CurrentBlock()
	if b == nil {
		return
	}
	content := []rune(b.Content())
	if d.CursorCol > len(content) {
		d.CursorCol = len(content)
	}
	newContent := make([]rune, 0, len(content)+1)
	newContent = append(newContent, content[:d.CursorCol]...)
	newContent = append(newContent, ch)
	newContent = append(newContent, content[d.CursorCol:]...)
	b.SetContent(string(newContent))
	d.CursorCol++
}

// DeleteCharBackward deletes the character before the cursor (backspace)
func (d *Document) DeleteCharBackward() bool {
	if d.CursorCol == 0 {
		return false
	}
	b := d.CurrentBlock()
	if b == nil {
		return false
	}
	content := []rune(b.Content())
	if d.CursorCol > len(content) {
		d.CursorCol = len(content)
	}
	newContent := make([]rune, 0, len(content)-1)
	newContent = append(newContent, content[:d.CursorCol-1]...)
	newContent = append(newContent, content[d.CursorCol:]...)
	b.SetContent(string(newContent))
	d.CursorCol--
	return true
}

// DeleteCharForward deletes the character at the cursor (delete key)
func (d *Document) DeleteCharForward() bool {
	b := d.CurrentBlock()
	if b == nil {
		return false
	}
	content := []rune(b.Content())
	if d.CursorCol >= len(content) {
		return false
	}
	newContent := make([]rune, 0, len(content)-1)
	newContent = append(newContent, content[:d.CursorCol]...)
	newContent = append(newContent, content[d.CursorCol+1:]...)
	b.SetContent(string(newContent))
	return true
}

// SplitBlockAtCursor splits the current block at the cursor position
// Returns the text after the cursor (which should go to the new block)
func (d *Document) SplitBlockAtCursor() string {
	b := d.CurrentBlock()
	if b == nil {
		return ""
	}
	content := []rune(b.Content())
	if d.CursorCol > len(content) {
		d.CursorCol = len(content)
	}
	left := string(content[:d.CursorCol])
	right := string(content[d.CursorCol:])
	b.SetContent(left)
	return right
}

// clampCursorCol ensures CursorCol is within valid bounds for current block
func (d *Document) clampCursorCol() {
	b := d.CurrentBlock()
	if b == nil {
		d.CursorCol = 0
		return
	}
	maxCol := len([]rune(b.Content()))
	if d.CursorCol > maxCol {
		d.CursorCol = maxCol
	}
}

// CurrentBlock returns the block at the current cursor position
func (d *Document) CurrentBlock() block.Block {
	if len(d.Blocks) == 0 || d.CursorLine < 0 || d.CursorLine >= len(d.Blocks) {
		return nil
	}
	return d.Blocks[d.CursorLine]
}

// BlockAt returns the block at the specified index
func (d *Document) BlockAt(index int) block.Block {
	if index < 0 || index >= len(d.Blocks) {
		return nil
	}
	return d.Blocks[index]
}

// BlockCount returns the number of blocks in the document
func (d *Document) BlockCount() int {
	return len(d.Blocks)
}

// SetCursor sets the cursor position
func (d *Document) SetCursor(line int) {
	if line < 0 {
		line = 0
	}
	if line >= len(d.Blocks) && len(d.Blocks) > 0 {
		line = len(d.Blocks) - 1
	}
	d.CursorLine = line
}
