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

// isClickOnHandle returns true if the X position is on the "||" handle
func (m Model) isClickOnHandle(x int) bool {
	return x < 2 // "||" is at positions 0 and 1
}

// getBlockDrawerAtY finds or creates the BlockDrawer at the given screen Y position
func (m *Model) getBlockDrawerAtY(y int) *drawer.BlockDrawer {
	// Calculate header offset
	// View structure: Title (0), Empty line (1), [scroll indicator (2) if scrollOffset > 0], then blocks
	headerLines := 1
	if m.scrollOffset > 0 {
		headerLines = 2
	}

	// Calculate which block index this Y corresponds to
	blockY := y - headerLines
	if blockY < 0 {
		return nil
	}

	blockIndex := blockY + m.scrollOffset
	if blockIndex < 0 || blockIndex >= m.doc.BlockCount() {
		return nil
	}

	blk := m.doc.BlockAt(blockIndex)
	if blk == nil {
		return nil
	}

	// Create a BlockDrawer for this position
	return m.registry.CreateBlockDrawer(blk, y, blockIndex)
}

// handleMouseEvent processes mouse events by delegating to block drawers
func (m *Model) handleMouseEvent(msg tea.MouseMsg) tea.Cmd {
	// Convert tea mouse action to drawer event type
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

	// Find the drawer at this Y position
	targetDrawer := m.getBlockDrawerAtY(msg.Y)

	// Update hover state
	if targetDrawer != nil {
		m.hoveredLine = targetDrawer.BlockIndex
	} else {
		m.hoveredLine = -1
	}

	// Only handle left button for clicks
	if msg.Button != tea.MouseButtonLeft && msg.Action != tea.MouseActionMotion {
		return nil
	}

	if targetDrawer == nil {
		return nil
	}

	blockIndex := targetDrawer.BlockIndex

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
	prefixWidth := targetDrawer.PrefixWidth()
	rawX := msg.X - handleWidth    // X relative to block start (includes prefix)
	contentX := rawX - prefixWidth // X relative to content start

	// Build mouse context for the drawer
	mouseCtx := drawer.MouseContext{
		X:          contentX,
		RawX:       rawX,
		EventType:  eventType,
		ClickCount: m.clickCount,
		IsDragging: m.isDragging,
	}

	// Let the drawer handle the mouse event
	action, handled := targetDrawer.HandleMouse(msg.Y, mouseCtx)
	if !handled {
		return nil
	}

	// Process the action returned by the drawer
	return m.processDrawerAction(action, targetDrawer.Block, blockIndex)
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
