package editor

import (
	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/block"
	"claudenelson/editor/format"
	"claudenelson/editor/undo"
)

// captureCurrentBlockState captures the state of the current block for undo
func (m *Model) captureCurrentBlockState() {
	if m.doc.CursorLine != m.lastBlockIndex || m.lastBlockState == nil {
		m.lastBlockIndex = m.doc.CursorLine
		m.lastBlockState = undo.CaptureBlockState(m.doc.CurrentBlock())
		m.pendingUndoRecord = true
	}
}

// recordBlockModification records the current block modification if changed
func (m *Model) recordBlockModification() {
	if !m.pendingUndoRecord || m.lastBlockState == nil {
		return
	}

	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return
	}

	newState := undo.CaptureBlockState(currentBlock)

	// Only record if content actually changed
	if m.lastBlockState.Content != newState.Content ||
		!spansEqual(m.lastBlockState.Spans, newState.Spans) {
		m.undoManager.RecordModify(
			m.lastBlockIndex,
			m.lastBlockState,
			newState,
			m.doc.CursorLine,
			m.doc.CursorCol,
		)
	}

	m.lastBlockState = nil
	m.pendingUndoRecord = false
}

// spansEqual checks if two span slices are equal
func spansEqual(a, b format.Spans) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// recordBlockAdd records a block addition for undo
func (m *Model) recordBlockAdd(index int) {
	blk := m.doc.BlockAt(index)
	if blk == nil {
		return
	}
	m.undoManager.RecordAdd(
		index,
		undo.CaptureBlockState(blk),
		m.doc.CursorLine,
		m.doc.CursorCol,
	)
}

// recordBlockDelete records a block deletion for undo
func (m *Model) recordBlockDelete(index int, state *undo.BlockState) {
	m.undoManager.RecordDelete(
		index,
		state,
		m.doc.CursorLine,
		m.doc.CursorCol,
	)
}

// performUndo undoes the last operation
func (m *Model) performUndo() tea.Cmd {
	// First, commit any pending modification
	m.recordBlockModification()

	op, ok := m.undoManager.Undo()
	if !ok {
		return nil
	}

	switch op.Type {
	case undo.OpModify:
		// Restore the old block state
		if op.OldState != nil && op.Index < m.doc.BlockCount() {
			m.restoreBlockState(op.Index, op.OldState)
		}
	case undo.OpAdd:
		// Remove the added block
		if op.Index < m.doc.BlockCount() {
			m.doc.RemoveBlock(op.Index)
		}
	case undo.OpDelete:
		// Re-add the deleted block
		if op.OldState != nil {
			blk := m.createBlockFromState(op.OldState)
			if op.Index >= m.doc.BlockCount() {
				m.doc.AddBlock(blk)
			} else {
				m.doc.InsertBlock(op.Index, blk)
			}
		}
	case undo.OpMultiDelete:
		// Re-add all deleted blocks
		for i, state := range op.OldBlocks {
			blk := m.createBlockFromState(&state)
			insertIdx := op.StartIndex + i
			if insertIdx >= m.doc.BlockCount() {
				m.doc.AddBlock(blk)
			} else {
				m.doc.InsertBlock(insertIdx, blk)
			}
		}
	}

	// Restore cursor position
	m.doc.CursorLine = op.CursorLine
	m.doc.CursorCol = op.CursorCol
	if m.doc.CursorLine >= m.doc.BlockCount() {
		m.doc.CursorLine = m.doc.BlockCount() - 1
	}
	if m.doc.CursorLine < 0 {
		m.doc.CursorLine = 0
	}
	m.ensureCursorVisible()

	// Reset tracking state
	m.lastBlockState = nil
	m.lastBlockIndex = -1
	m.pendingUndoRecord = false

	return m.markDirty()
}

// performRedo redoes the last undone operation
func (m *Model) performRedo() tea.Cmd {
	op, ok := m.undoManager.Redo()
	if !ok {
		return nil
	}

	switch op.Type {
	case undo.OpModify:
		// Apply the new block state
		if op.NewState != nil && op.Index < m.doc.BlockCount() {
			m.restoreBlockState(op.Index, op.NewState)
		}
	case undo.OpAdd:
		// Re-add the block
		if op.NewState != nil {
			blk := m.createBlockFromState(op.NewState)
			if op.Index >= m.doc.BlockCount() {
				m.doc.AddBlock(blk)
			} else {
				m.doc.InsertBlock(op.Index, blk)
			}
		}
	case undo.OpDelete:
		// Remove the block again
		if op.Index < m.doc.BlockCount() {
			m.doc.RemoveBlock(op.Index)
		}
	case undo.OpMultiDelete:
		// Remove all the blocks again
		for i := len(op.OldBlocks) - 1; i >= 0; i-- {
			idx := op.StartIndex + i
			if idx < m.doc.BlockCount() {
				m.doc.RemoveBlock(idx)
			}
		}
	}

	m.ensureCursorVisible()

	// Reset tracking state
	m.lastBlockState = nil
	m.lastBlockIndex = -1
	m.pendingUndoRecord = false

	return m.markDirty()
}

// restoreBlockState restores a block to a saved state
func (m *Model) restoreBlockState(index int, state *undo.BlockState) {
	blk := m.doc.BlockAt(index)
	if blk == nil {
		return
	}
	blk.SetContent(state.Content)
	blk.SetSpans(state.Spans)
	if state.Checked != nil {
		if cb, ok := blk.(*block.CheckboxBlock); ok {
			cb.SetChecked(*state.Checked)
		}
	}
}

// createBlockFromState creates a new block from a saved state
func (m *Model) createBlockFromState(state *undo.BlockState) block.Block {
	var blk block.Block
	switch state.Type {
	case block.TypeH1:
		blk = m.factory.CreateHeading(state.Content, 1)
	case block.TypeH2:
		blk = m.factory.CreateHeading(state.Content, 2)
	case block.TypeH3:
		blk = m.factory.CreateHeading(state.Content, 3)
	case block.TypeH4:
		blk = m.factory.CreateHeading(state.Content, 4)
	case block.TypeListItem:
		blk = m.factory.CreateListItemWithSpans(state.Content, state.Spans)
	case block.TypeCheckboxItem:
		checked := false
		if state.Checked != nil {
			checked = *state.Checked
		}
		blk = m.factory.CreateCheckboxWithSpans(state.Content, checked, state.Spans)
	default:
		blk = m.factory.CreateTextWithSpans(state.Content, state.Spans)
	}
	if state.Spans != nil {
		blk.SetSpans(state.Spans)
	}
	return blk
}
