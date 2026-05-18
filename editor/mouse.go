package editor

import "claudenelson/editor/block"

// getBlockAtY returns the block index at the given Y position, or -1 if none
func (m Model) getBlockAtY(y int) int {
	// View structure:
	// Line 0: Title
	// Line 1: Empty
	// Line 2: (optional) "↑ N more lines above" if scrollOffset > 0
	// Line 2 or 3: First visible block
	headerLines := 1
	if m.scrollOffset > 0 {
		headerLines = 2 // Extra line for scroll indicator
	}

	screenBlockIndex := y - headerLines
	if screenBlockIndex < 0 {
		return -1
	}

	blockIndex := screenBlockIndex + m.scrollOffset
	if blockIndex >= 0 && blockIndex < m.doc.BlockCount() {
		return blockIndex
	}
	return -1
}

// getColumnAtX returns the column index at the given X position for a block
func (m Model) getColumnAtX(blockIndex, x int) int {
	if blockIndex < 0 || blockIndex >= m.doc.BlockCount() {
		return 0
	}
	blk := m.doc.BlockAt(blockIndex)
	if blk == nil {
		return 0
	}

	// Account for selection handle "|| " (3 chars) and block prefix
	prefix := m.getBlockPrefix(blk)
	prefixLen := len([]rune(prefix)) // Use rune length for display width
	handleLen := 3                   // "|| " = 3 characters

	// Calculate column from x position
	col := x - handleLen - prefixLen
	if col < 0 {
		col = 0
	}

	// Clamp to content length
	content := []rune(blk.Content())
	if col > len(content) {
		col = len(content)
	}

	return col
}

// isClickOnHandle returns true if the X position is on the "||" handle
func (m Model) isClickOnHandle(x int) bool {
	return x < 2 // "||" is at positions 0 and 1
}

// getBlockPrefix returns the prefix string for a block type (bullet, checkbox, etc.)
func (m Model) getBlockPrefix(blk block.Block) string {
	switch blk.Type() {
	case block.TypeListItem:
		return "• "
	case block.TypeCheckboxItem:
		if cb, ok := blk.(*block.CheckboxBlock); ok {
			if cb.IsChecked() {
				return "☑ "
			}
			return "☐ "
		}
		return "☐ "
	default:
		return ""
	}
}

// getWordBoundsAt returns the start and end column of the word at the given position
func (m Model) getWordBoundsAt(line, col int) (start, end int) {
	if line < 0 || line >= m.doc.BlockCount() {
		return 0, 0
	}
	blk := m.doc.BlockAt(line)
	if blk == nil {
		return 0, 0
	}

	content := []rune(blk.Content())
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
