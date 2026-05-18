package editor

// getVisibleLineCount returns the number of lines available for content
func (m *Model) getVisibleLineCount() int {
	// Reserve lines for: title (1) + empty (1) + format indicators (2) + help (2) + padding
	reserved := 7
	visible := m.height - reserved
	if visible < 1 {
		visible = 1
	}
	return visible
}

// ensureCursorVisible adjusts scroll offset to keep cursor in view
func (m *Model) ensureCursorVisible() {
	visibleLines := m.getVisibleLineCount()

	// If cursor is above the viewport, scroll up
	if m.doc.CursorLine < m.scrollOffset {
		m.scrollOffset = m.doc.CursorLine
	}

	// If cursor is below the viewport, scroll down
	if m.doc.CursorLine >= m.scrollOffset+visibleLines {
		m.scrollOffset = m.doc.CursorLine - visibleLines + 1
	}

	// Ensure scroll offset is valid
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	maxOffset := m.doc.BlockCount() - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

// scrollUp scrolls the viewport up by n lines
func (m *Model) scrollUp(n int) {
	m.scrollOffset -= n
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// scrollDown scrolls the viewport down by n lines
func (m *Model) scrollDown(n int) {
	maxOffset := m.doc.BlockCount() - m.getVisibleLineCount()
	if maxOffset < 0 {
		maxOffset = 0
	}
	m.scrollOffset += n
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}
