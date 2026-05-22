package editor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/block"
	"claudenelson/editor/undo"
)

// handleEnter splits the current block at cursor and creates a new block
func (m *Model) handleEnter() {
	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return
	}

	// Capture state before split for undo
	oldState := undo.CaptureBlockState(currentBlock)
	cursorLine := m.doc.CursorLine
	cursorCol := m.doc.CursorCol

	// Get text and spans after cursor
	rightPart, rightSpans := m.doc.SplitBlockAtCursor()

	// Determine new block type and create with spans
	var newBlock block.Block
	switch currentBlock.Type() {
	case block.TypeListItem:
		newBlock = m.factory.CreateListItemWithSpans(rightPart, rightSpans)
	case block.TypeCheckboxItem:
		newBlock = m.factory.CreateCheckboxWithSpans(rightPart, false, rightSpans)
	default:
		newBlock = m.factory.CreateTextWithSpans(rightPart, rightSpans)
	}

	// Insert new block after current
	m.doc.InsertBlock(m.doc.CursorLine+1, newBlock)

	// Record the modification of current block
	newState := undo.CaptureBlockState(currentBlock)
	m.undoManager.RecordModify(cursorLine, oldState, newState, cursorLine, cursorCol)

	// Record the addition of the new block
	m.recordBlockAdd(m.doc.CursorLine + 1)

	// Move cursor to start of new block
	m.doc.MoveDown()
	m.doc.CursorCol = 0
	m.ensureCursorVisible()

	// Reset undo tracking state
	m.lastBlockState = nil
	m.lastBlockIndex = -1
	m.pendingUndoRecord = false
}

// checkBlockTriggers checks if the current content matches a block trigger pattern
// and converts the block type accordingly
func (m *Model) checkBlockTriggers() {
	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return
	}

	// Only trigger on text blocks and headings
	blockType := currentBlock.Type()
	isText := blockType == block.TypeText
	isHeading := blockType == block.TypeH1 || blockType == block.TypeH2 ||
		blockType == block.TypeH3 || blockType == block.TypeH4

	if !isText && !isHeading {
		return
	}

	content := currentBlock.Content()

	// Check for checkbox patterns at start of line
	if strings.HasPrefix(content, "[] ") {
		// Convert to unchecked checkbox
		newContent := strings.TrimPrefix(content, "[] ")
		newBlock := m.factory.CreateCheckbox(newContent, false)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}

	if strings.HasPrefix(content, "[x] ") || strings.HasPrefix(content, "[X] ") {
		// Convert to checked checkbox
		newContent := content[4:] // Remove "[x] " or "[X] "
		newBlock := m.factory.CreateCheckbox(newContent, true)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}

	// Check for list item pattern
	if strings.HasPrefix(content, "- ") {
		newContent := strings.TrimPrefix(content, "- ")
		newBlock := m.factory.CreateListItem(newContent)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}

	// Check for heading patterns (longest first)
	if strings.HasPrefix(content, "#### ") {
		newContent := strings.TrimPrefix(content, "#### ")
		newBlock := m.factory.CreateHeading(newContent, 4)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}
	if strings.HasPrefix(content, "### ") {
		newContent := strings.TrimPrefix(content, "### ")
		newBlock := m.factory.CreateHeading(newContent, 3)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}
	if strings.HasPrefix(content, "## ") {
		newContent := strings.TrimPrefix(content, "## ")
		newBlock := m.factory.CreateHeading(newContent, 2)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}
	if strings.HasPrefix(content, "# ") {
		newContent := strings.TrimPrefix(content, "# ")
		newBlock := m.factory.CreateHeading(newContent, 1)
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		m.doc.CursorCol = 0
		return
	}
}

// convertToTextBlock converts checkbox/list/heading blocks to text blocks
// Returns true if a conversion was made, false otherwise
func (m *Model) convertToTextBlock() bool {
	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return false
	}

	blockType := currentBlock.Type()
	switch blockType {
	case block.TypeCheckboxItem, block.TypeListItem,
		block.TypeH1, block.TypeH2, block.TypeH3, block.TypeH4:
		// Create a new text block with the same content
		newBlock := m.factory.CreateText(currentBlock.Content())
		// Replace the current block
		m.doc.Blocks[m.doc.CursorLine] = newBlock
		return true
	default:
		return false
	}
}

// mergeWithPreviousBlock merges current block content into previous block
func (m *Model) mergeWithPreviousBlock() {
	if m.doc.CursorLine == 0 {
		return
	}

	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return
	}

	prevBlock := m.doc.BlockAt(m.doc.CursorLine - 1)
	if prevBlock == nil {
		return
	}

	// Remember the join point (end of previous block)
	joinPoint := len([]rune(prevBlock.Content()))

	// Append current content to previous block
	prevBlock.SetContent(prevBlock.Content() + currentBlock.Content())

	// Remove current block
	m.doc.RemoveBlock(m.doc.CursorLine)

	// Move cursor to previous block at join point
	m.doc.CursorLine--
	m.doc.CursorCol = joinPoint
}

// mergeWithNextBlock merges next block content into current block
func (m *Model) mergeWithNextBlock() {
	if m.doc.CursorLine >= m.doc.BlockCount()-1 {
		return
	}

	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return
	}

	nextBlock := m.doc.BlockAt(m.doc.CursorLine + 1)
	if nextBlock == nil {
		return
	}

	// Append next content to current block
	currentBlock.SetContent(currentBlock.Content() + nextBlock.Content())

	// Remove next block
	m.doc.RemoveBlock(m.doc.CursorLine + 1)
}

func (m *Model) deleteCurrentLine() tea.Cmd {
	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return nil
	}

	// Capture state for undo
	state := undo.CaptureBlockState(currentBlock)
	cursorLine := m.doc.CursorLine
	cursorCol := m.doc.CursorCol

	// If this is the only block, just clear its content
	if m.doc.BlockCount() == 1 {
		currentBlock.SetContent("")
		currentBlock.SetSpans(nil)
		m.doc.CursorCol = 0
	} else {
		// Remove the current block
		m.doc.RemoveBlock(m.doc.CursorLine)

		// Adjust cursor position
		if m.doc.CursorLine >= m.doc.BlockCount() {
			m.doc.CursorLine = m.doc.BlockCount() - 1
		}
		m.doc.CursorCol = 0
	}

	// Record for undo
	if state != nil {
		m.undoManager.RecordMultiDelete(cursorLine, []undo.BlockState{*state}, cursorLine, cursorCol)
	}

	m.ensureCursorVisible()
	return m.markDirty()
}

// toggleCheckbox toggles the checked state of a checkbox block
func (m *Model) toggleCheckbox() tea.Cmd {
	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return nil
	}

	// Check if it's a checkbox block
	checkbox, ok := currentBlock.(*block.CheckboxBlock)
	if !ok {
		return nil
	}

	// Capture state for undo
	oldState := undo.CaptureBlockState(currentBlock)
	cursorLine := m.doc.CursorLine
	cursorCol := m.doc.CursorCol

	// Toggle the checkbox
	checkbox.Toggle()

	// Record for undo
	newState := undo.CaptureBlockState(currentBlock)
	m.undoManager.RecordModify(cursorLine, oldState, newState, cursorLine, cursorCol)

	return m.markDirty()
}