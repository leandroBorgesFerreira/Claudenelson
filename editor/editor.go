package editor

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/document"
	"claudenelson/editor/drawer"
	"claudenelson/editor/factory"
	"claudenelson/editor/persistence"
	"claudenelson/editor/styles"
	"claudenelson/editor/undo"
)

const saveDelay = 500 * time.Millisecond

// saveTickMsg is sent when the save timer fires
type saveTickMsg time.Time

// Sample document content in markdown-like format
var sampleContent = `# Start Writing`

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
	// Viewport scrolling
	scrollOffset int // First visible line (block index)
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
	// Multi-line handle selection
	selectedLines map[int]bool // Lines selected via handle clicks
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
		selectedLines:  make(map[int]bool),
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

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil
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
			m.ensureCursorVisible()

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
			m.ensureCursorVisible()

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
			m.ensureCursorVisible()

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
			m.ensureCursorVisible()

		case "up":
			if m.highlightMode {
				// In highlight mode, up/down exits without applying
				m.highlightMode = false
			}
			m.recordBlockModification() // Commit any pending changes
			m.clearAllSelections()
			m.doc.MoveUp()
			m.ensureCursorVisible()

		case "down":
			if m.highlightMode {
				// In highlight mode, up/down exits without applying
				m.highlightMode = false
			}
			m.recordBlockModification() // Commit any pending changes
			m.clearAllSelections()
			m.doc.MoveDown()
			m.ensureCursorVisible()

		case "pgup":
			m.recordBlockModification()
			m.clearAllSelections()
			// Move cursor up by visible page
			pageSize := m.getVisibleLineCount()
			for i := 0; i < pageSize && m.doc.CursorLine > 0; i++ {
				m.doc.MoveUp()
			}
			m.ensureCursorVisible()

		case "pgdown":
			m.recordBlockModification()
			m.clearAllSelections()
			// Move cursor down by visible page
			pageSize := m.getVisibleLineCount()
			for i := 0; i < pageSize && m.doc.CursorLine < m.doc.BlockCount()-1; i++ {
				m.doc.MoveDown()
			}
			m.ensureCursorVisible()

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
			if m.hasHandleSelection() {
				cmd = m.toggleFormatOnHandleSelectedLines(true, false, false)
			} else if m.charSelect {
				cmd = m.toggleFormatOnCharSelection(true, false, false)
			} else {
				m.boldMode = !m.boldMode
			}

		case "ctrl+i":
			if m.hasHandleSelection() {
				cmd = m.toggleFormatOnHandleSelectedLines(false, true, false)
			} else if m.charSelect {
				cmd = m.toggleFormatOnCharSelection(false, true, false)
			} else {
				m.italicMode = !m.italicMode
			}

		case "ctrl+u":
			if m.hasHandleSelection() {
				cmd = m.toggleFormatOnHandleSelectedLines(false, false, true)
			} else if m.charSelect {
				cmd = m.toggleFormatOnCharSelection(false, false, true)
			} else {
				m.underlineMode = !m.underlineMode
			}

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
			if m.hasHandleSelection() {
				cmd = m.deleteHandleSelectedLines()
			} else if m.multiLineSelect {
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
			if m.hasHandleSelection() {
				cmd = m.deleteHandleSelectedLines()
			} else if m.multiLineSelect {
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
			// Escape key exits selection modes without applying, or quits editor
			if m.multiLineSelect {
				m.clearMultiLineSelection()
			} else if m.charSelect {
				m.clearCharSelection()
			} else if m.highlightMode {
				m.exitHighlightMode()
			} else {
				// No selection mode active - save and quit
				if m.dirty {
					persistence.Save(m.doc, m.savePath)
				}
				return m, tea.Quit
			}

		default:
			// Handle regular character input with formatting
			if len(msg.Runes) > 0 {
				if m.charSelect {
					// Delete selection then insert character
					m.deleteCharSelection()
				}
				m.clearAllSelections()
				if m.highlightMode {
					cmd = m.applyHighlightAndExit()
				} else {
					m.captureCurrentBlockState()
					style := m.currentStyle()
					for _, r := range msg.Runes {
						// Handle newline characters by creating new blocks
						if r == '\n' || r == '\r' {
							m.handleEnter()
						} else {
							m.doc.InsertCharWithFormat(r, style)
						}
					}
					cmd = m.markDirty()
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
	}

	return m, cmd
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

	// Calculate visible range
	visibleLines := m.getVisibleLineCount()
	startLine := m.scrollOffset
	endLine := m.scrollOffset + visibleLines
	if endLine > m.doc.BlockCount() {
		endLine = m.doc.BlockCount()
	}

	// Show scroll indicator if not at top
	if m.scrollOffset > 0 {
		b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("  ↑ %d more lines above", m.scrollOffset)))
		b.WriteString("\n")
	}

	// Render only visible blocks
	for i := startLine; i < endLine; i++ {
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

		// Check if this line is part of multi-line selection or handle-selected
		lineSelected := m.isLineSelected(i) || m.isLineHandleSelected(i)
		handleSelected := m.isLineHandleSelected(i)

		ctx := drawer.DrawContext{
			Width:            m.width,
			IsFocused:        isFocused,
			CursorPos:        m.doc.CursorCol,
			LineNumber:       i,
			ShowCursor:       true,
			SelectionStart:   selStart,
			SelectionEnd:     selEnd,
			LineSelected:     lineSelected,
			IsHandleSelected: handleSelected,
		}

		// Render the block content (drawer handles the handle rendering)
		content := m.registry.Draw(blk, ctx)
		b.WriteString(content)
		b.WriteString("\n")
	}

	// Show scroll indicator if not at bottom
	remainingLines := m.doc.BlockCount() - endLine
	if remainingLines > 0 {
		b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("  ↓ %d more lines below", remainingLines)))
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
	if len(formatIndicators) > 0 {
		b.WriteString("  ")
		b.WriteString(strings.Join(formatIndicators, " "))
		b.WriteString("\n")
	}

	// Line count indicator
	lineInfo := fmt.Sprintf("Line %d/%d", m.doc.CursorLine+1, m.doc.BlockCount())
	b.WriteString("  ")
	b.WriteString(styles.HelpStyle.Render(lineInfo))
	b.WriteString("\n")

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
		helpText = "←/→: Cursor • ↑/↓: Block • PgUp/Dn: Scroll • ^Z: Undo • ^B/I/U/H: Format • ^C: Quit"
	}
	help := styles.HelpStyle.Render(helpText)
	b.WriteString(help)
	b.WriteString("\n")

	return b.String()
}
