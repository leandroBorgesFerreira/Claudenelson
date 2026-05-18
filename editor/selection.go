package editor

import (
	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/undo"
)

// clearMultiLineSelection clears the multi-line selection
func (m *Model) clearMultiLineSelection() {
	m.multiLineSelect = false
	m.lineSelectionStart = 0
	m.lineSelectionEnd = 0
}

// clearCharSelection clears the character selection
func (m *Model) clearCharSelection() {
	m.charSelect = false
	m.charSelStartLine = 0
	m.charSelStartCol = 0
	m.charSelEndLine = 0
	m.charSelEndCol = 0
}

// clearAllSelections clears both line and character selections
func (m *Model) clearAllSelections() {
	m.clearMultiLineSelection()
	m.clearCharSelection()
	m.clearHandleSelection()
}

// clearHandleSelection clears handle-based line selections
func (m *Model) clearHandleSelection() {
	m.selectedLines = make(map[int]bool)
}

// isLineHandleSelected returns true if the line is selected via handle
func (m *Model) isLineHandleSelected(line int) bool {
	return m.selectedLines[line]
}

// toggleLineHandleSelection toggles the handle selection for a line
func (m *Model) toggleLineHandleSelection(line int) {
	if m.selectedLines[line] {
		delete(m.selectedLines, line)
	} else {
		m.selectedLines[line] = true
	}
}

// hasHandleSelection returns true if any lines are handle-selected
func (m *Model) hasHandleSelection() bool {
	return len(m.selectedLines) > 0
}

// getCharSelectionForLine returns the selection range for a specific line
// Returns (-1, -1) if line is not part of selection
func (m *Model) getCharSelectionForLine(line int) (int, int) {
	if !m.charSelect {
		return -1, -1
	}

	// Normalize selection direction
	startLine, startCol := m.charSelStartLine, m.charSelStartCol
	endLine, endCol := m.charSelEndLine, m.charSelEndCol
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	// Check if this line is in selection range
	if line < startLine || line > endLine {
		return -1, -1
	}

	blk := m.doc.BlockAt(line)
	if blk == nil {
		return -1, -1
	}
	lineLen := len([]rune(blk.Content()))

	// Calculate selection range for this line
	selStart := 0
	selEnd := lineLen

	if line == startLine {
		selStart = startCol
	}
	if line == endLine {
		selEnd = endCol
	}

	if selStart >= selEnd && line != startLine && line != endLine {
		// Full line selection for middle lines
		selStart = 0
		selEnd = lineLen
	}

	return selStart, selEnd
}

// getSelectedLineRange returns the start and end lines of multi-line selection (ordered)
func (m *Model) getSelectedLineRange() (int, int) {
	start, end := m.lineSelectionStart, m.lineSelectionEnd
	if start > end {
		start, end = end, start
	}
	return start, end
}

// isLineSelected returns true if the given line is within multi-line selection
func (m *Model) isLineSelected(line int) bool {
	if !m.multiLineSelect {
		return false
	}
	start, end := m.getSelectedLineRange()
	return line >= start && line <= end
}

// deleteCharSelection deletes characters in the character selection
func (m *Model) deleteCharSelection() tea.Cmd {
	if !m.charSelect {
		return nil
	}

	// Normalize selection direction
	startLine, startCol := m.charSelStartLine, m.charSelStartCol
	endLine, endCol := m.charSelEndLine, m.charSelEndCol
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	// Capture states for undo
	var deletedStates []undo.BlockState
	for i := startLine; i <= endLine && i < m.doc.BlockCount(); i++ {
		blk := m.doc.BlockAt(i)
		if blk != nil {
			state := undo.CaptureBlockState(blk)
			if state != nil {
				deletedStates = append(deletedStates, *state)
			}
		}
	}

	cursorLine := m.doc.CursorLine
	cursorCol := m.doc.CursorCol

	if startLine == endLine {
		// Selection within single line - just delete the range
		blk := m.doc.BlockAt(startLine)
		if blk != nil {
			content := []rune(blk.Content())
			if startCol < len(content) && endCol <= len(content) {
				newContent := string(content[:startCol]) + string(content[endCol:])
				blk.SetContent(newContent)
				// Adjust spans for deletion
				spans := blk.Spans()
				for i := endCol - 1; i >= startCol; i-- {
					spans = spans.DeleteAt(i)
				}
				blk.SetSpans(spans)
			}
		}
		m.doc.CursorLine = startLine
		m.doc.CursorCol = startCol
	} else {
		// Multi-line selection
		// Keep content before selection on first line
		// Keep content after selection on last line
		// Delete lines in between

		firstBlock := m.doc.BlockAt(startLine)
		lastBlock := m.doc.BlockAt(endLine)

		if firstBlock != nil && lastBlock != nil {
			firstContent := []rune(firstBlock.Content())
			lastContent := []rune(lastBlock.Content())

			// New content: start of first line + end of last line
			var newContent string
			if startCol < len(firstContent) {
				newContent = string(firstContent[:startCol])
			}
			if endCol < len(lastContent) {
				newContent += string(lastContent[endCol:])
			}

			firstBlock.SetContent(newContent)

			// Delete lines from end to start+1
			for i := endLine; i > startLine; i-- {
				m.doc.RemoveBlock(i)
			}
		}

		m.doc.CursorLine = startLine
		m.doc.CursorCol = startCol
	}

	// Record for undo
	if len(deletedStates) > 0 {
		m.undoManager.RecordMultiDelete(startLine, deletedStates, cursorLine, cursorCol)
	}

	m.clearCharSelection()
	m.ensureCursorVisible()
	return m.markDirty()
}

// deleteSelectedLines deletes all lines in multi-line selection
func (m *Model) deleteSelectedLines() tea.Cmd {
	if !m.multiLineSelect {
		return nil
	}

	start, end := m.getSelectedLineRange()
	count := end - start + 1

	// Capture states of all blocks to be deleted for undo
	var deletedStates []undo.BlockState
	for i := start; i <= end && i < m.doc.BlockCount(); i++ {
		blk := m.doc.BlockAt(i)
		if blk != nil {
			state := undo.CaptureBlockState(blk)
			if state != nil {
				deletedStates = append(deletedStates, *state)
			}
		}
	}

	cursorLine := m.doc.CursorLine
	cursorCol := m.doc.CursorCol

	// Don't delete all blocks - keep at least one
	if count >= m.doc.BlockCount() {
		// Clear all content but keep one empty block
		for m.doc.BlockCount() > 1 {
			m.doc.RemoveBlock(m.doc.BlockCount() - 1)
		}
		m.doc.Blocks[0].SetContent("")
		m.doc.Blocks[0].SetSpans(nil)
		m.doc.CursorLine = 0
		m.doc.CursorCol = 0
	} else {
		// Delete selected blocks from end to start to maintain indices
		for i := end; i >= start; i-- {
			m.doc.RemoveBlock(i)
		}
		// Adjust cursor position
		if start >= m.doc.BlockCount() {
			m.doc.CursorLine = m.doc.BlockCount() - 1
		} else {
			m.doc.CursorLine = start
		}
		m.doc.CursorCol = 0
	}

	// Record multi-delete for undo
	if len(deletedStates) > 0 {
		m.undoManager.RecordMultiDelete(start, deletedStates, cursorLine, cursorCol)
	}

	m.clearMultiLineSelection()
	m.ensureCursorVisible()
	return m.markDirty()
}

// deleteHandleSelectedLines deletes all lines selected via handle
func (m *Model) deleteHandleSelectedLines() tea.Cmd {
	if !m.hasHandleSelection() {
		return nil
	}

	// Get sorted list of selected lines (descending for safe deletion)
	var selectedIndices []int
	for idx := range m.selectedLines {
		selectedIndices = append(selectedIndices, idx)
	}
	// Sort descending
	for i := 0; i < len(selectedIndices)-1; i++ {
		for j := i + 1; j < len(selectedIndices); j++ {
			if selectedIndices[i] < selectedIndices[j] {
				selectedIndices[i], selectedIndices[j] = selectedIndices[j], selectedIndices[i]
			}
		}
	}

	cursorLine := m.doc.CursorLine
	cursorCol := m.doc.CursorCol

	// Capture states for undo (in ascending order for proper restore)
	var deletedStates []undo.BlockState
	minIdx := selectedIndices[len(selectedIndices)-1]
	for i := len(selectedIndices) - 1; i >= 0; i-- {
		idx := selectedIndices[i]
		blk := m.doc.BlockAt(idx)
		if blk != nil {
			state := undo.CaptureBlockState(blk)
			if state != nil {
				deletedStates = append(deletedStates, *state)
			}
		}
	}

	// Don't delete all blocks - keep at least one
	if len(selectedIndices) >= m.doc.BlockCount() {
		for m.doc.BlockCount() > 1 {
			m.doc.RemoveBlock(m.doc.BlockCount() - 1)
		}
		m.doc.Blocks[0].SetContent("")
		m.doc.Blocks[0].SetSpans(nil)
		m.doc.CursorLine = 0
		m.doc.CursorCol = 0
	} else {
		// Delete from highest index to lowest
		for _, idx := range selectedIndices {
			if idx < m.doc.BlockCount() {
				m.doc.RemoveBlock(idx)
			}
		}
		// Adjust cursor
		if minIdx >= m.doc.BlockCount() {
			m.doc.CursorLine = m.doc.BlockCount() - 1
		} else {
			m.doc.CursorLine = minIdx
		}
		m.doc.CursorCol = 0
	}

	if len(deletedStates) > 0 {
		m.undoManager.RecordMultiDelete(minIdx, deletedStates, cursorLine, cursorCol)
	}

	m.clearHandleSelection()
	m.ensureCursorVisible()
	return m.markDirty()
}
