package undo

import (
	"claudenelson/editor/block"
	"claudenelson/editor/format"
)

// OpType represents the type of operation for undo/redo
type OpType int

const (
	OpModify OpType = iota // Block content/formatting was modified
	OpAdd                  // Block was added
	OpDelete               // Block was deleted
	OpMultiDelete          // Multiple blocks were deleted
)

// BlockState stores the state of a block for undo/redo
type BlockState struct {
	Content string
	Spans   format.Spans
	Checked *bool // For checkbox blocks
	Type    block.BlockType
}

// Operation represents a single undoable operation
type Operation struct {
	Type       OpType
	Index      int          // Block index where operation occurred
	OldState   *BlockState  // Previous state (for modify/delete)
	NewState   *BlockState  // New state (for modify/add)
	OldBlocks  []BlockState // For multi-delete: all deleted blocks
	StartIndex int          // For multi-delete: starting index
	CursorLine int          // Cursor position before operation
	CursorCol  int
}

// Stack is a stack of operations for undo/redo
type Stack struct {
	items    []Operation
	maxSize  int
}

// NewStack creates a new operation stack with a maximum size
func NewStack(maxSize int) *Stack {
	return &Stack{
		items:   make([]Operation, 0, maxSize),
		maxSize: maxSize,
	}
}

// Push adds an operation to the stack
func (s *Stack) Push(op Operation) {
	if len(s.items) >= s.maxSize {
		// Remove oldest item
		s.items = s.items[1:]
	}
	s.items = append(s.items, op)
}

// Pop removes and returns the top operation
func (s *Stack) Pop() (Operation, bool) {
	if len(s.items) == 0 {
		return Operation{}, false
	}
	op := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return op, true
}

// Clear removes all operations from the stack
func (s *Stack) Clear() {
	s.items = s.items[:0]
}

// IsEmpty returns true if the stack is empty
func (s *Stack) IsEmpty() bool {
	return len(s.items) == 0
}

// Len returns the number of operations in the stack
func (s *Stack) Len() int {
	return len(s.items)
}

// Manager manages undo and redo stacks
type Manager struct {
	undoStack *Stack
	redoStack *Stack
}

// NewManager creates a new undo/redo manager
func NewManager(maxSize int) *Manager {
	return &Manager{
		undoStack: NewStack(maxSize),
		redoStack: NewStack(maxSize),
	}
}

// CaptureBlockState captures the current state of a block
func CaptureBlockState(b block.Block) *BlockState {
	if b == nil {
		return nil
	}
	state := &BlockState{
		Content: b.Content(),
		Spans:   copySpans(b.Spans()),
		Type:    b.Type(),
	}
	if cb, ok := b.(*block.CheckboxBlock); ok {
		checked := cb.IsChecked()
		state.Checked = &checked
	}
	return state
}

// copySpans creates a deep copy of spans
func copySpans(spans format.Spans) format.Spans {
	if spans == nil {
		return nil
	}
	result := make(format.Spans, len(spans))
	copy(result, spans)
	return result
}

// RecordModify records a block modification
func (m *Manager) RecordModify(index int, oldState, newState *BlockState, cursorLine, cursorCol int) {
	m.undoStack.Push(Operation{
		Type:       OpModify,
		Index:      index,
		OldState:   oldState,
		NewState:   newState,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
	m.redoStack.Clear() // Clear redo stack on new operation
}

// RecordAdd records a block addition
func (m *Manager) RecordAdd(index int, newState *BlockState, cursorLine, cursorCol int) {
	m.undoStack.Push(Operation{
		Type:       OpAdd,
		Index:      index,
		NewState:   newState,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
	m.redoStack.Clear()
}

// RecordDelete records a block deletion
func (m *Manager) RecordDelete(index int, oldState *BlockState, cursorLine, cursorCol int) {
	m.undoStack.Push(Operation{
		Type:       OpDelete,
		Index:      index,
		OldState:   oldState,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
	m.redoStack.Clear()
}

// RecordMultiDelete records deletion of multiple blocks
func (m *Manager) RecordMultiDelete(startIndex int, blocks []BlockState, cursorLine, cursorCol int) {
	m.undoStack.Push(Operation{
		Type:       OpMultiDelete,
		StartIndex: startIndex,
		OldBlocks:  blocks,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
	m.redoStack.Clear()
}

// Undo returns the operation to undo, moving it to redo stack
func (m *Manager) Undo() (Operation, bool) {
	op, ok := m.undoStack.Pop()
	if ok {
		m.redoStack.Push(op)
	}
	return op, ok
}

// Redo returns the operation to redo, moving it to undo stack
func (m *Manager) Redo() (Operation, bool) {
	op, ok := m.redoStack.Pop()
	if ok {
		m.undoStack.Push(op)
	}
	return op, ok
}

// CanUndo returns true if there are operations to undo
func (m *Manager) CanUndo() bool {
	return !m.undoStack.IsEmpty()
}

// CanRedo returns true if there are operations to redo
func (m *Manager) CanRedo() bool {
	return !m.redoStack.IsEmpty()
}

// UndoCount returns the number of undoable operations
func (m *Manager) UndoCount() int {
	return m.undoStack.Len()
}

// RedoCount returns the number of redoable operations
func (m *Manager) RedoCount() int {
	return m.redoStack.Len()
}
