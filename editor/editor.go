package editor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"claudenelson/editor/block"
	"claudenelson/editor/document"
	"claudenelson/editor/drawer"
	"claudenelson/editor/factory"
	"claudenelson/editor/styles"
)

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
	doc      *document.Document
	factory  *factory.BlockFactory
	registry *drawer.DrawerRegistry
	width    int
	height   int
}

// New creates a new editor model with sample content
func New() Model {
	f := factory.NewBlockFactory()
	r := drawer.NewDrawerRegistry()
	r.RegisterAll()

	doc := document.NewDocument()

	// Parse sample content into blocks
	lines := strings.Split(sampleContent, "\n")
	for _, line := range lines {
		b := f.CreateFromLine(line)
		doc.AddBlock(b)
	}

	return Model{
		doc:      doc,
		factory:  f,
		registry: r,
		width:    80,
		height:   24,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "up":
			m.doc.MoveUp()

		case "down":
			m.doc.MoveDown()

		case "left":
			m.doc.MoveLeft()

		case "right":
			m.doc.MoveRight()

		case "home":
			m.doc.MoveToLineStart()

		case "end":
			m.doc.MoveToLineEnd()

		case "backspace":
			if !m.doc.DeleteCharBackward() {
				m.mergeWithPreviousBlock()
			}

		case "delete":
			if !m.doc.DeleteCharForward() {
				m.mergeWithNextBlock()
			}

		case "enter":
			m.handleEnter()

		case " ":
			// Insert space character
			m.doc.InsertChar(' ')

		default:
			// Handle regular character input
			if len(msg.Runes) > 0 {
				for _, r := range msg.Runes {
					m.doc.InsertChar(r)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// handleEnter splits the current block at cursor and creates a new block
func (m *Model) handleEnter() {
	currentBlock := m.doc.CurrentBlock()
	if currentBlock == nil {
		return
	}

	// Get text after cursor
	rightPart := m.doc.SplitBlockAtCursor()

	// Determine new block type
	var newBlock block.Block
	switch currentBlock.Type() {
	case block.TypeListItem:
		newBlock = m.factory.CreateListItem(rightPart)
	case block.TypeCheckboxItem:
		newBlock = m.factory.CreateCheckbox(rightPart, false)
	default:
		newBlock = m.factory.CreateText(rightPart)
	}

	// Insert new block after current
	m.doc.InsertBlock(m.doc.CursorLine+1, newBlock)

	// Move cursor to start of new block
	m.doc.MoveDown()
	m.doc.CursorCol = 0
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

	// Title
	title := styles.TitleStyle.Render("claudenelson Block Editor")
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Render each block
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

	// Help text
	b.WriteString("\n")
	help := styles.HelpStyle.Render("←/→: Move cursor • ↑/↓: Move block • Enter: Split block • Backspace/Delete: Edit • Ctrl+C: Quit")
	b.WriteString(help)
	b.WriteString("\n")

	return b.String()
}
