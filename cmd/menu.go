package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudenelson/editor"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const pageSize = 20

// DocumentInfo holds the metadata for a document in the menu
type DocumentInfo struct {
	FilePath   string
	DocTitle   string
	DocPreview string // First 3 lines combined
	Loaded     bool   // Whether the document has been parsed
}

// menuState represents the current state of the menu
type menuState int

const (
	stateLoading menuState = iota
	stateList
	stateEmpty
)

// menuModel is the BubbleTea model for the menu
type menuModel struct {
	state      menuState
	spinner    spinner.Model
	documents  []DocumentInfo // All documents (lazily loaded)
	filePaths  []string       // All file paths found
	selected   *DocumentInfo
	cursor     int // Current cursor position (global index)
	viewOffset int // First visible item index within current page
	err        error
	width      int
	height     int
	searchPath string
}

// filesFoundMsg is sent when file paths are discovered
type filesFoundMsg struct {
	paths []string
	err   error
}

// pageLoadedMsg is sent when a page of documents is parsed
type pageLoadedMsg struct {
	startIndex int
	documents  []DocumentInfo
}

// Styles for the menu
var (
	menuTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	menuSpinnerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))

	menuLoadingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	menuEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	menuSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("212")).
				Bold(true)

	menuDescStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(lipgloss.Color("241"))

	menuSelectedDescStyle = lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(lipgloss.Color("248"))

	menuPaginationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				PaddingLeft(2).
				MarginTop(1)
)

func newMenuModel(searchPath string) menuModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = menuSpinnerStyle

	return menuModel{
		state:      stateLoading,
		spinner:    s,
		searchPath: searchPath,
		cursor:     0,
		viewOffset: 0,
	}
}

func (m menuModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		findFiles(m.searchPath),
	)
}

// currentPage returns the current page number (0-indexed)
func (m menuModel) currentPage() int {
	return m.cursor / pageSize
}

// totalPages returns the total number of pages
func (m menuModel) totalPages() int {
	if len(m.documents) == 0 {
		return 0
	}
	return (len(m.documents) + pageSize - 1) / pageSize
}

// pageStart returns the start index of the current page
func (m menuModel) pageStart() int {
	return m.currentPage() * pageSize
}

// pageEnd returns the end index (exclusive) of the current page
func (m menuModel) pageEnd() int {
	end := m.pageStart() + pageSize
	if end > len(m.documents) {
		end = len(m.documents)
	}
	return end
}

// visibleItems returns how many items can fit on screen
// Each item takes 1 line for title + up to 3 lines for preview + 1 blank line = 5 lines max
// Reserve 4 lines for header and footer
func (m menuModel) visibleItems() int {
	if m.height <= 6 {
		return 1
	}
	// Each item: 1 title + 3 preview + 1 blank = 5 lines
	available := m.height - 4 // header + footer
	items := available / 5
	if items < 1 {
		items = 1
	}
	if items > pageSize {
		items = pageSize
	}
	return items
}

// needsPageLoad checks if current page has unloaded documents
func (m menuModel) needsPageLoad() bool {
	start := m.pageStart()
	end := m.pageEnd()
	for i := start; i < end; i++ {
		if !m.documents[i].Loaded {
			return true
		}
	}
	return false
}

// adjustViewOffset ensures cursor is visible within the viewport
func (m *menuModel) adjustViewOffset() {
	cursorInPage := m.cursor - m.pageStart()
	visible := m.visibleItems()

	// If cursor is above viewport, scroll up
	if cursorInPage < m.viewOffset {
		m.viewOffset = cursorInPage
	}
	// If cursor is below viewport, scroll down
	if cursorInPage >= m.viewOffset+visible {
		m.viewOffset = cursorInPage - visible + 1
	}
	// Clamp viewOffset
	if m.viewOffset < 0 {
		m.viewOffset = 0
	}
	maxOffset := m.pageEnd() - m.pageStart() - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.state == stateList {
				doc := m.documents[m.cursor]
				if doc.Loaded {
					m.selected = &doc
					return m, tea.Quit
				}
			}
			if m.state == stateEmpty {
				return m, tea.Quit
			}
		case "up", "k":
			if m.state == stateList && m.cursor > 0 {
				oldPage := m.currentPage()
				m.cursor--
				newPage := m.currentPage()
				// If we moved to a new page, reset view offset and load
				if newPage != oldPage {
					m.viewOffset = pageSize - 1 // Start at bottom of previous page
					if m.needsPageLoad() {
						m.adjustViewOffset()
						return m, loadPage(m.filePaths, m.pageStart(), m.pageEnd())
					}
				}
				m.adjustViewOffset()
			}
			return m, nil
		case "down", "j":
			if m.state == stateList && m.cursor < len(m.documents)-1 {
				oldPage := m.currentPage()
				m.cursor++
				newPage := m.currentPage()
				// If we moved to a new page, reset view offset and load
				if newPage != oldPage {
					m.viewOffset = 0 // Start at top of new page
					if m.needsPageLoad() {
						m.adjustViewOffset()
						return m, loadPage(m.filePaths, m.pageStart(), m.pageEnd())
					}
				}
				m.adjustViewOffset()
			}
			return m, nil
		case "pgup":
			if m.state == stateList {
				oldPage := m.currentPage()
				m.cursor -= pageSize
				if m.cursor < 0 {
					m.cursor = 0
				}
				newPage := m.currentPage()
				if newPage != oldPage {
					m.viewOffset = 0
					if m.needsPageLoad() {
						m.adjustViewOffset()
						return m, loadPage(m.filePaths, m.pageStart(), m.pageEnd())
					}
				}
				m.adjustViewOffset()
			}
			return m, nil
		case "pgdown":
			if m.state == stateList {
				oldPage := m.currentPage()
				m.cursor += pageSize
				if m.cursor >= len(m.documents) {
					m.cursor = len(m.documents) - 1
				}
				newPage := m.currentPage()
				if newPage != oldPage {
					m.viewOffset = 0
					if m.needsPageLoad() {
						m.adjustViewOffset()
						return m, loadPage(m.filePaths, m.pageStart(), m.pageEnd())
					}
				}
				m.adjustViewOffset()
			}
			return m, nil
		case "home":
			if m.state == stateList {
				oldPage := m.currentPage()
				m.cursor = 0
				m.viewOffset = 0
				newPage := m.currentPage()
				if newPage != oldPage && m.needsPageLoad() {
					return m, loadPage(m.filePaths, m.pageStart(), m.pageEnd())
				}
			}
			return m, nil
		case "end":
			if m.state == stateList && len(m.documents) > 0 {
				oldPage := m.currentPage()
				m.cursor = len(m.documents) - 1
				newPage := m.currentPage()
				if newPage != oldPage {
					m.viewOffset = 0
					if m.needsPageLoad() {
						m.adjustViewOffset()
						return m, loadPage(m.filePaths, m.pageStart(), m.pageEnd())
					}
				}
				m.adjustViewOffset()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustViewOffset()
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case filesFoundMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateEmpty
			return m, nil
		}

		if len(msg.paths) == 0 {
			m.state = stateEmpty
			return m, nil
		}

		m.filePaths = msg.paths
		// Initialize documents with just file paths (not loaded)
		m.documents = make([]DocumentInfo, len(msg.paths))
		for i, path := range msg.paths {
			m.documents[i] = DocumentInfo{
				FilePath: path,
				DocTitle: filepath.Base(path),
				Loaded:   false,
			}
		}

		m.state = stateList
		m.cursor = 0
		m.viewOffset = 0
		// Load the first page
		return m, loadPage(m.filePaths, 0, min(pageSize, len(m.filePaths)))

	case pageLoadedMsg:
		// Update loaded documents
		for i, doc := range msg.documents {
			idx := msg.startIndex + i
			if idx < len(m.documents) {
				m.documents[idx] = doc
			}
		}
		return m, nil
	}

	return m, nil
}

func (m menuModel) View() string {
	switch m.state {
	case stateLoading:
		return fmt.Sprintf("\n  %s %s\n\n",
			m.spinner.View(),
			menuLoadingStyle.Render("Loading documents..."))

	case stateEmpty:
		if m.err != nil {
			return fmt.Sprintf("\n  %s\n\n  Press q to quit.\n",
				menuEmptyStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		}
		return fmt.Sprintf("\n  %s\n\n  Press q to quit.\n",
			menuEmptyStyle.Render("No documents found. Create one with: claudenelson write -f <filename>.wrdoc.json"))

	case stateList:
		var b strings.Builder

		// Title
		b.WriteString(menuTitleStyle.Render("Select a document to edit"))
		b.WriteString("\n\n")

		pageStart := m.pageStart()
		pageEnd := m.pageEnd()
		visible := m.visibleItems()

		// Calculate visible range within the page
		viewStart := pageStart + m.viewOffset
		viewEnd := viewStart + visible
		if viewEnd > pageEnd {
			viewEnd = pageEnd
		}

		for i := viewStart; i < viewEnd; i++ {
			doc := m.documents[i]
			isSelected := i == m.cursor

			// Title
			title := doc.DocTitle
			if !doc.Loaded {
				title = filepath.Base(doc.FilePath) + " (loading...)"
			}

			if isSelected {
				b.WriteString(menuSelectedStyle.Render("▸ " + title))
			} else {
				b.WriteString(menuItemStyle.Render("  " + title))
			}
			b.WriteString("\n")

			// Preview (description) - show first 3 lines of content
			if doc.Loaded && doc.DocPreview != "" {
				lines := strings.Split(doc.DocPreview, "\n")
				for _, line := range lines {
					if isSelected {
						b.WriteString(menuSelectedDescStyle.Render(line))
					} else {
						b.WriteString(menuDescStyle.Render(line))
					}
					b.WriteString("\n")
				}
			}
			b.WriteString("\n")
		}

		// Pagination info
		totalDocs := len(m.documents)
		currentPage := m.currentPage() + 1
		totalPages := m.totalPages()

		b.WriteString(menuPaginationStyle.Render(
			fmt.Sprintf("Page %d/%d • Item %d/%d • ↑↓ navigate • enter select • q quit",
				currentPage, totalPages, m.cursor+1, totalDocs)))

		return b.String()
	}

	return ""
}

// findFiles scans for .wrdoc.json files without parsing them
func findFiles(searchPath string) tea.Cmd {
	return func() tea.Msg {
		pattern := filepath.Join(searchPath, "*.wrdoc.json")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return filesFoundMsg{err: err}
		}

		// Filter out folder files
		var paths []string
		for _, path := range matches {
			if !strings.HasSuffix(path, ".wrfolder.json") {
				paths = append(paths, path)
			}
		}

		return filesFoundMsg{paths: paths}
	}
}

// loadPage parses documents for a specific range
func loadPage(filePaths []string, start, end int) tea.Cmd {
	return func() tea.Msg {
		var documents []DocumentInfo

		for i := start; i < end && i < len(filePaths); i++ {
			path := filePaths[i]
			info, err := parseDocumentInfo(path)
			if err != nil {
				// Use filename as fallback
				info = DocumentInfo{
					FilePath:   path,
					DocTitle:   filepath.Base(path),
					DocPreview: "(failed to parse)",
					Loaded:     true,
				}
			}
			documents = append(documents, info)
		}

		return pageLoadedMsg{
			startIndex: start,
			documents:  documents,
		}
	}
}

// parseDocumentInfo reads a document file and extracts title and first 3 lines
func parseDocumentInfo(path string) (DocumentInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DocumentInfo{}, err
	}

	var doc struct {
		Title   string `json:"title"`
		Content []struct {
			Text string `json:"text"`
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
		} `json:"content"`
	}

	if err := json.Unmarshal(data, &doc); err != nil {
		return DocumentInfo{}, err
	}

	// Get title
	title := doc.Title
	if title == "" {
		title = filepath.Base(path)
	}

	// Get first 3 non-empty, non-title lines for preview
	var previewLines []string
	for _, step := range doc.Content {
		// Skip title blocks
		if step.Type.Name == "title" {
			continue
		}
		// Skip empty lines
		text := strings.TrimSpace(step.Text)
		if text == "" {
			continue
		}
		// Truncate long lines
		if len(text) > 60 {
			text = text[:57] + "..."
		}
		previewLines = append(previewLines, text)
		if len(previewLines) >= 3 {
			break
		}
	}

	preview := strings.Join(previewLines, "\n")
	if preview == "" {
		preview = "(empty document)"
	}

	return DocumentInfo{
		FilePath:   path,
		DocTitle:   title,
		DocPreview: preview,
		Loaded:     true,
	}, nil
}

var menuSearchPath string

var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Browse and select documents to edit",
	Long:  "Displays a list of all Writeopia documents (.wrdoc.json) in the specified directory with titles and preview text.",
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve search path
		searchPath := menuSearchPath
		if searchPath == "" || searchPath == "." {
			var err error
			searchPath, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Loop: menu -> editor -> menu
		for {
			// Run the menu
			model := newMenuModel(searchPath)
			p := tea.NewProgram(model, tea.WithAltScreen())

			finalModel, err := p.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running menu: %v\n", err)
				os.Exit(1)
			}

			// Check if a document was selected
			m, ok := finalModel.(menuModel)
			if !ok || m.selected == nil {
				// User quit without selecting - exit
				break
			}

			// Launch the editor with the selected document
			editorProgram := tea.NewProgram(
				editor.New(m.selected.FilePath),
				tea.WithAltScreen(),
			)

			if _, err := editorProgram.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
				os.Exit(1)
			}
			// After editor exits, loop back to menu
		}
	},
}

func init() {
	menuCmd.Flags().StringVarP(&menuSearchPath, "path", "p", ".", "Directory to search for documents")
	rootCmd.AddCommand(menuCmd)
}
