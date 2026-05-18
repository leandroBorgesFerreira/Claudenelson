package editor

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/block"
	"claudenelson/editor/drawer"
)

const (
	handleWidth          = 3 // "|| " = 3 characters
	doubleClickThreshold = 500 * time.Millisecond
)

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

// isClickOnHandle returns true if the X position is on the "||" handle
func (m Model) isClickOnHandle(x int) bool {
	return x < 2 // "||" is at positions 0 and 1
}

// handleBlockMouse processes mouse events by delegating to the block's drawer
func (m *Model) handleBlockMouse(blk block.Block, blockIndex int, msg tea.MouseMsg) tea.Cmd {
	// Convert tea.MouseMsg to drawer.MouseContext
	var eventType drawer.MouseEventType
	switch msg.Action {
	case tea.MouseActionPress:
		eventType = drawer.MousePress
	case tea.MouseActionRelease:
		eventType = drawer.MouseRelease
	case tea.MouseActionMotion:
		eventType = drawer.MouseMotion
	default:
		return nil
	}

	// Only handle left button clicks
	if msg.Button != tea.MouseButtonLeft && msg.Action != tea.MouseActionMotion {
		return nil
	}

	// Check if click is on the handle first
	if msg.Action == tea.MouseActionPress && m.isClickOnHandle(msg.X) {
		m.doc.SetCursor(blockIndex)
		m.clearCharSelection()
		m.toggleLineHandleSelection(blockIndex)
		return nil
	}

	// Track click count for double/triple clicks
	now := time.Now()
	if msg.Action == tea.MouseActionPress {
		if now.Sub(m.lastClickTime) < doubleClickThreshold && blockIndex == m.lastClickLine {
			m.clickCount++
			if m.clickCount > 3 {
				m.clickCount = 3
			}
		} else {
			m.clickCount = 1
		}
		m.lastClickTime = now
		m.lastClickLine = blockIndex
	}

	// Calculate X position relative to content (after handle and prefix)
	prefixWidth := m.registry.PrefixWidth(blk)
	rawX := msg.X - handleWidth          // X relative to block start (includes prefix)
	contentX := rawX - prefixWidth       // X relative to content start

	// Build mouse context for the drawer
	mouseCtx := drawer.MouseContext{
		X:          contentX,
		RawX:       rawX,
		EventType:  eventType,
		ClickCount: m.clickCount,
		IsDragging: m.isDragging,
	}

	// Delegate to the drawer
	action := m.registry.HandleMouse(blk, mouseCtx)

	// Process the action returned by the drawer
	return m.processDrawerAction(action, blk, blockIndex)
}

// processDrawerAction applies the action returned by a drawer
func (m *Model) processDrawerAction(action drawer.Action, blk block.Block, blockIndex int) tea.Cmd {
	switch action.Type {
	case drawer.ActionNone:
		return nil

	case drawer.ActionSetCursor:
		m.clearAllSelections()
		m.doc.SetCursor(blockIndex)
		m.doc.CursorCol = action.CursorCol
		m.ensureCursorVisible()

	case drawer.ActionStartDrag:
		m.clearAllSelections()
		m.doc.SetCursor(blockIndex)
		m.doc.CursorCol = action.CursorCol
		m.isDragging = true
		m.charSelect = true
		m.charSelStartLine = blockIndex
		m.charSelStartCol = action.SelStart
		m.charSelEndLine = blockIndex
		m.charSelEndCol = action.SelEnd
		m.ensureCursorVisible()

	case drawer.ActionExtendDrag:
		if m.isDragging && blockIndex == m.charSelStartLine {
			m.charSelEndLine = blockIndex
			m.charSelEndCol = action.CursorCol
			m.doc.CursorCol = action.CursorCol
		}

	case drawer.ActionEndDrag:
		if m.isDragging {
			m.isDragging = false
			// If no actual selection was made, clear it
			if m.charSelStartLine == m.charSelEndLine && m.charSelStartCol == m.charSelEndCol {
				m.clearCharSelection()
			}
		}

	case drawer.ActionSelectWord:
		m.isDragging = false
		m.doc.SetCursor(blockIndex)
		m.charSelect = true
		m.charSelStartLine = blockIndex
		m.charSelStartCol = action.SelStart
		m.charSelEndLine = blockIndex
		m.charSelEndCol = action.SelEnd
		m.doc.CursorCol = action.CursorCol
		m.ensureCursorVisible()

	case drawer.ActionSelectLine:
		m.isDragging = false
		m.doc.SetCursor(blockIndex)
		m.charSelect = true
		m.charSelStartLine = blockIndex
		m.charSelStartCol = action.SelStart
		m.charSelEndLine = blockIndex
		m.charSelEndCol = action.SelEnd
		m.doc.CursorCol = action.CursorCol
		m.ensureCursorVisible()

	case drawer.ActionToggleCheck:
		if cb, ok := blk.(*block.CheckboxBlock); ok {
			cb.Toggle()
			m.isDragging = false
			m.clearCharSelection()
			return m.markDirty()
		}
	}

	return nil
}
