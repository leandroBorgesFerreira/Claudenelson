package editor

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/block"
	"claudenelson/editor/document"
	"claudenelson/editor/drawer"
	"claudenelson/editor/factory"
	"claudenelson/editor/format"
	"claudenelson/editor/persistence"
	"claudenelson/editor/styles"
	"claudenelson/editor/undo"
)

const saveDelay = 500 * time.Millisecond

// saveTickMsg is sent when the save timer fires
type saveTickMsg time.Time

// Sample document content in markdown-like format
var sampleContent = `# Welcome to the Block Editor
This is a Notion-like block-based editor.
## Features
- Text blocks for paragraphs
- Headings (H1 through H4)
- Checkbox/todo items
### Todo List
[x] Implement block rendering
[x] Add navigation with arrow keys
[] Add editing capabilities
#### Navigation
Use Up/Down arrows to move between blocks.`

// Model represents the editor state
type Model struct {
	doc       *document.Document
	factory   *factory.BlockFactory
	registry  *drawer.DrawerRegistry
	width     int
	height    int
	savePath  string    // Path to save document
	dirty     bool      // Document has unsaved changes
	saveTimer time.Time // Last modification time
	// Formatting modes
	boldMode      bool
	italicMode    bool
	underlineMode bool
	// Highlight selection mode (within a line)
	highlightMode   bool // When true, arrow keys select text for highlighting
	selectionStart  int  // Starting position of selection
	// Multi-line selection mode (whole lines)
	multiLineSelect    bool // When true, whole lines are being selected
	lineSelectionStart int  // Starting line of multi-line selection
	lineSelectionEnd   int  // Ending line of multi-line selection
	// Character selection mode (character by character, can span lines)
	charSelect         bool // When true, characters are being selected
	charSelStartLine   int  // Starting line of character selection
	charSelStartCol    int  // Starting column of character selection
	charSelEndLine     int  // Ending line of character selection
	charSelEndCol      int  // Ending column of character selection
	// Undo/Redo
	undoManager       *undo.Manager
	lastBlockState    *undo.BlockState // State before current edit session
	lastBlockIndex    int              // Index of block being edited
	pendingUndoRecord bool             // Whether we need to record an undo entry
}

// getBlockAtY returns the block index at the given Y position, or -1 if none
func (m Model) getBlockAtY(y int) int {
	// View structure:
	// Line 0: Title
	// Line 1: Empty
	// Line 2+: Blocks (one per line)
	// Note: offset may need adjustment based on terminal behavior
	const headerLines = 8
	blockIndex := y - headerLines
	if blockIndex >= 0 && blockIndex < m.doc.BlockCount() {
		return blockIndex
	}
	return -1
}

// New creates a new editor model with sample content or loads from file
func New(savePath string) Model {
	f := factory.NewBlockFactory()
	r := drawer.NewDrawerRegistry()
	r.RegisterAll()

	var doc *document.Document

	// Try to load existing document from file
	if savePath != "" {
		if _, err := os.Stat(savePath); err == nil {
			if loadedDoc, err := persistence.Load(savePath, f); err == nil {
				doc = loadedDoc
			}
		}
	}

	// If no document was loaded, create with sample content
	if doc == nil {
		doc = document.NewDocument()
		lines := strings.Split(sampleContent, "\n")
		for _, line := range lines {
			b := f.CreateFromLine(line)
			doc.AddBlock(b)
		}
	}

	return Model{
		doc:            doc,
		factory:        f,
		registry:       r,
		width:          80,
		height:         24,
		savePath:       savePath,
		dirty:          false,
		undoManager:    undo.NewManager(100), // Keep up to 100 undo entries
		lastBlockIndex: -1,
	}
}

// scheduleSave returns a tea.Cmd that fires after saveDelay
func (m Model) scheduleSave() tea.Cmd {
	return tea.Tick(saveDelay, func(t time.Time) tea.Msg {
		return saveTickMsg(t)
	})
}

// markDirty marks the document as having unsaved changes and schedules a save
func (m *Model) markDirty() tea.Cmd {
	m.dirty = true
	m.saveTimer = time.Now()
	return m.scheduleSave()
}

// currentStyle returns the current formatting style based on active modes
func (m *Model) currentStyle() format.Style {
	return format.Style{
		Bold:      m.boldMode,
		Italic:    m.italicMode,
		Underline: m.underlineMode,
	}
}

// applyHighlightAndExit applies or removes highlight from the selected range and exits highlight mode
func (m *Model) applyHighlightAndExit() tea.Cmd {
	if !m.highlightMode {
		return nil
	}

	currentBlock := m.doc.CurrentBlock()
	if currentBlock != nil {
		start, end := m.selectionStart, m.doc.CursorCol
		if start > end {
			start, end = end, start
		}
		if start != end {
			// Toggle highlight span on current block
			spans := currentBlock.Spans()
			newSpans := spans.ToggleHighlight(start, end)
			currentBlock.SetSpans(newSpans)
		}
	}

	m.highlightMode = false
	return m.markDirty()
}

// exitHighlightMode exits highlight mode without applying
func (m *Model) exitHighlightMode() {
	m.highlightMode = false
}

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
	return m.markDirty()
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.EnableMouseCellMotion
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case saveTickMsg:
		// Handle debounced save
		if m.dirty && m.savePath != "" && time.Since(m.saveTimer) >= saveDelay {
			persistence.Save(m.doc, m.savePath)
			m.dirty = false
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Save before quitting if dirty
			if m.dirty && m.savePath != "" {
				persistence.Save(m.doc, m.savePath)
			}
			return m, tea.Quit

		case "alt+up", "alt+k":
			// Start or extend multi-line selection upward
			if !m.multiLineSelect {
				m.multiLineSelect = true
				m.lineSelectionStart = m.doc.CursorLine
				m.lineSelectionEnd = m.doc.CursorLine
			}
			if m.lineSelectionEnd > 0 {
				m.lineSelectionEnd--
				m.doc.CursorLine = m.lineSelectionEnd
			}

		case "alt+down", "alt+j":
			// Start or extend multi-line selection downward
			if !m.multiLineSelect {
				m.multiLineSelect = true
				m.lineSelectionStart = m.doc.CursorLine
				m.lineSelectionEnd = m.doc.CursorLine
			}
			if m.lineSelectionEnd < m.doc.BlockCount()-1 {
				m.lineSelectionEnd++
				m.doc.CursorLine = m.lineSelectionEnd
			}

		case "alt+left", "alt+h":
			// Start or extend character selection leftward
			m.clearMultiLineSelection()
			if !m.charSelect {
				m.charSelect = true
				m.charSelStartLine = m.doc.CursorLine
				m.charSelStartCol = m.doc.CursorCol
				m.charSelEndLine = m.doc.CursorLine
				m.charSelEndCol = m.doc.CursorCol
			}
			// Move selection end left
			if m.charSelEndCol > 0 {
				m.charSelEndCol--
				m.doc.CursorCol = m.charSelEndCol
			} else if m.charSelEndLine > 0 {
				// Move to end of previous line
				m.charSelEndLine--
				m.doc.CursorLine = m.charSelEndLine
				prevBlock := m.doc.BlockAt(m.charSelEndLine)
				if prevBlock != nil {
					m.charSelEndCol = len([]rune(prevBlock.Content()))
					m.doc.CursorCol = m.charSelEndCol
				}
			}

		case "alt+right", "alt+l":
			// Start or extend character selection rightward
			m.clearMultiLineSelection()
			if !m.charSelect {
				m.charSelect = true
				m.charSelStartLine = m.doc.CursorLine
				m.charSelStartCol = m.doc.CursorCol
				m.charSelEndLine = m.doc.CursorLine
				m.charSelEndCol = m.doc.CursorCol
			}
			// Move selection end right
			currentBlock := m.doc.BlockAt(m.charSelEndLine)
			if currentBlock != nil {
				lineLen := len([]rune(currentBlock.Content()))
				if m.charSelEndCol < lineLen {
					m.charSelEndCol++
					m.doc.CursorCol = m.charSelEndCol
				} else if m.charSelEndLine < m.doc.BlockCount()-1 {
					// Move to start of next line
					m.charSelEndLine++
					m.charSelEndCol = 0
					m.doc.CursorLine = m.charSelEndLine
					m.doc.CursorCol = 0
				}
			}

		case "up":
			if m.highlightMode {
				// In highlight mode, up/down exits without applying
				m.highlightMode = false
			}
			m.recordBlockModification() // Commit any pending changes
			m.clearAllSelections()
			m.doc.MoveUp()

		case "down":
			if m.highlightMode {
				// In highlight mode, up/down exits without applying
				m.highlightMode = false
			}
			m.recordBlockModification() // Commit any pending changes
			m.clearAllSelections()
			m.doc.MoveDown()

		case "left":
			m.clearAllSelections()
			m.doc.MoveLeft()

		case "right":
			m.clearAllSelections()
			m.doc.MoveRight()

		case "home", "ctrl+a":
			m.clearAllSelections()
			if m.highlightMode {
				// Extend selection to start of line
				m.doc.MoveToLineStart()
			} else {
				m.doc.MoveToLineStart()
			}

		case "end", "ctrl+e":
			m.clearAllSelections()
			if m.highlightMode {
				// Extend selection to end of line
				m.doc.MoveToLineEnd()
			} else {
				m.doc.MoveToLineEnd()
			}

		case "ctrl+b":
			m.boldMode = !m.boldMode

		case "ctrl+i":
			m.italicMode = !m.italicMode

		case "ctrl+u":
			m.underlineMode = !m.underlineMode

		case "ctrl+h":
			// Enter highlight mode - start selection at current cursor
			m.highlightMode = true
			m.selectionStart = m.doc.CursorCol

		case "ctrl+z":
			// Undo
			cmd = m.performUndo()

		case "ctrl+y", "ctrl+shift+z":
			// Redo
			cmd = m.performRedo()

		case "backspace":
			if m.multiLineSelect {
				cmd = m.deleteSelectedLines()
			} else if m.charSelect {
				cmd = m.deleteCharSelection()
			} else if m.highlightMode {
				cmd = m.applyHighlightAndExit()
			} else {
				m.captureCurrentBlockState()
				if !m.doc.DeleteCharBackward() {
					// At start of line - try to convert special blocks to text first
					if !m.convertToTextBlock() {
						m.mergeWithPreviousBlock()
					}
				}
				cmd = m.markDirty()
			}

		case "delete":
			if m.multiLineSelect {
				cmd = m.deleteSelectedLines()
			} else if m.charSelect {
				cmd = m.deleteCharSelection()
			} else if m.highlightMode {
				cmd = m.applyHighlightAndExit()
			} else {
				m.captureCurrentBlockState()
				if !m.doc.DeleteCharForward() {
					m.mergeWithNextBlock()
				}
				cmd = m.markDirty()
			}

		case "enter":
			if m.multiLineSelect {
				cmd = m.deleteSelectedLines()
			} else if m.charSelect {
				cmd = m.deleteCharSelection()
			} else if m.highlightMode {
				cmd = m.applyHighlightAndExit()
			} else {
				m.handleEnter()
				cmd = m.markDirty()
			}

		case " ":
			if m.charSelect {
				cmd = m.deleteCharSelection()
			} else {
				m.clearAllSelections()
				if m.highlightMode {
					cmd = m.applyHighlightAndExit()
				} else {
					m.captureCurrentBlockState()
					// Insert space character with formatting
					m.doc.InsertCharWithFormat(' ', m.currentStyle())
					// Check for block type triggers
					m.checkBlockTriggers()
					cmd = m.markDirty()
				}
			}

		case "esc":
			// Escape key exits selection modes without applying
			if m.multiLineSelect {
				m.clearMultiLineSelection()
			} else if m.charSelect {
				m.clearCharSelection()
			} else if m.highlightMode {
				m.exitHighlightMode()
			}

		default:
			// Handle regular character input with formatting
			if len(msg.Runes) > 0 {
				if m.charSelect {
					// Delete selection then insert character
					m.deleteCharSelection()
					m.captureCurrentBlockState()
					style := m.currentStyle()
					for _, r := range msg.Runes {
						m.doc.InsertCharWithFormat(r, style)
					}
					cmd = m.markDirty()
				} else {
					m.clearAllSelections()
					if m.highlightMode {
						cmd = m.applyHighlightAndExit()
					} else {
						m.captureCurrentBlockState()
						style := m.currentStyle()
						for _, r := range msg.Runes {
							m.doc.InsertCharWithFormat(r, style)
						}
						cmd = m.markDirty()
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			blockIndex := m.getBlockAtY(msg.Y)
			if blockIndex >= 0 {
				m.doc.SetCursor(blockIndex)
				// Toggle checkbox if clicked
				if cb, ok := m.doc.CurrentBlock().(*block.CheckboxBlock); ok {
					cb.Toggle()
					cmd = m.markDirty()
				}
			}
		}
	}

	return m, cmd
}

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

// View renders the editor
func (m Model) View() string {
	var b strings.Builder

	// Title (line 0)
	title := styles.TitleStyle.Render("claudenelson Block Editor")
	b.WriteString(title)
	b.WriteString("\n")

	// Empty line (line 1)
	b.WriteString("\n")

	// Blocks start at line 2, one block per line
	for i := 0; i < m.doc.BlockCount(); i++ {
		blk := m.doc.BlockAt(i)
		isFocused := i == m.doc.CursorLine

		// Calculate selection range for this block
		selStart, selEnd := -1, -1
		if m.highlightMode && isFocused {
			// Highlight mode (for applying highlight formatting)
			selStart = m.selectionStart
			selEnd = m.doc.CursorCol
		} else if m.charSelect {
			// Character selection mode
			selStart, selEnd = m.getCharSelectionForLine(i)
		}

		// Check if this line is part of multi-line selection
		lineSelected := m.isLineSelected(i)

		ctx := drawer.DrawContext{
			Width:          m.width,
			IsFocused:      isFocused,
			CursorPos:      m.doc.CursorCol,
			LineNumber:     i,
			ShowCursor:     true,
			SelectionStart: selStart,
			SelectionEnd:   selEnd,
			LineSelected:   lineSelected,
		}

		// Consistent indentation for all blocks
		indent := "  "

		// Render the block content
		content := m.registry.Draw(blk, ctx)

		b.WriteString(indent)
		b.WriteString(content)
		b.WriteString("\n")
	}

	// Formatting indicators
	b.WriteString("\n")
	var formatIndicators []string
	if m.boldMode {
		formatIndicators = append(formatIndicators, styles.BoldIndicator.Render("B"))
	}
	if m.italicMode {
		formatIndicators = append(formatIndicators, styles.ItalicIndicator.Render("I"))
	}
	if m.underlineMode {
		formatIndicators = append(formatIndicators, styles.UnderlineIndicator.Render("U"))
	}
	if m.highlightMode {
		formatIndicators = append(formatIndicators, styles.HighlightIndicator.Render("H"))
	}
	if m.multiLineSelect {
		start, end := m.getSelectedLineRange()
		count := end - start + 1
		formatIndicators = append(formatIndicators, styles.SelectionIndicator.Render(fmt.Sprintf("LINES:%d", count)))
	}
	if m.charSelect {
		formatIndicators = append(formatIndicators, styles.SelectionIndicator.Render("SELECT"))
	}
	if len(formatIndicators) > 0 {
		b.WriteString("  ")
		b.WriteString(strings.Join(formatIndicators, " "))
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	var helpText string
	if m.multiLineSelect {
		helpText = "LINE SELECT: ⌥↑/↓: Extend • Backspace/Del: Delete lines • Esc: Cancel"
	} else if m.charSelect {
		helpText = "CHAR SELECT: ⌥←/→: Extend • Backspace/Del: Delete • Esc: Cancel"
	} else if m.highlightMode {
		helpText = "HIGHLIGHT MODE: ←/→: Select • Enter/Space: Apply • Esc: Cancel"
	} else {
		helpText = "←/→: Cursor • ↑/↓: Block • ^Z: Undo • ^Y: Redo • ^B/I/U/H: Format • ^C: Quit"
	}
	help := styles.HelpStyle.Render(helpText)
	b.WriteString(help)
	b.WriteString("\n")

	return b.String()
}
