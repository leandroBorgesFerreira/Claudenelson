package editor

import (
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
		doc:      doc,
		factory:  f,
		registry: r,
		width:    80,
		height:   24,
		savePath: savePath,
		dirty:    false,
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

		case "up":
			m.doc.MoveUp()

		case "down":
			m.doc.MoveDown()

		case "left":
			m.doc.MoveLeft()

		case "right":
			m.doc.MoveRight()

		case "home", "ctrl+a":
			m.doc.MoveToLineStart()

		case "end", "ctrl+e":
			m.doc.MoveToLineEnd()

		case "ctrl+b":
			m.boldMode = !m.boldMode

		case "ctrl+i":
			m.italicMode = !m.italicMode

		case "ctrl+u":
			m.underlineMode = !m.underlineMode

		case "backspace":
			if !m.doc.DeleteCharBackward() {
				// At start of line - try to convert special blocks to text first
				if !m.convertToTextBlock() {
					m.mergeWithPreviousBlock()
				}
			}
			cmd = m.markDirty()

		case "delete":
			if !m.doc.DeleteCharForward() {
				m.mergeWithNextBlock()
			}
			cmd = m.markDirty()

		case "enter":
			m.handleEnter()
			cmd = m.markDirty()

		case " ":
			// Insert space character with formatting
			m.doc.InsertCharWithFormat(' ', m.currentStyle())
			// Check for block type triggers
			m.checkBlockTriggers()
			cmd = m.markDirty()

		default:
			// Handle regular character input with formatting
			if len(msg.Runes) > 0 {
				style := m.currentStyle()
				for _, r := range msg.Runes {
					m.doc.InsertCharWithFormat(r, style)
				}
				cmd = m.markDirty()
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

	// Move cursor to start of new block
	m.doc.MoveDown()
	m.doc.CursorCol = 0
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

		ctx := drawer.DrawContext{
			Width:      m.width,
			IsFocused:  isFocused,
			CursorPos:  m.doc.CursorCol,
			LineNumber: i,
			ShowCursor: true,
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
	if len(formatIndicators) > 0 {
		b.WriteString("  ")
		b.WriteString(strings.Join(formatIndicators, " "))
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	help := styles.HelpStyle.Render("←/→: Cursor • ↑/↓: Block • ^B: Bold • ^I: Italic • ^U: Underline • ^C: Quit")
	b.WriteString(help)
	b.WriteString("\n")

	return b.String()
}
