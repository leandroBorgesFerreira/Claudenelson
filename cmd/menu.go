package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudenelson/editor"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// DocumentInfo holds the metadata for a document in the menu
type DocumentInfo struct {
	FilePath    string
	DocTitle    string
	DocPreview  string // First 3 lines combined
}

// Implement list.Item interface
func (d DocumentInfo) FilterValue() string { return d.DocTitle }
func (d DocumentInfo) Title() string       { return d.DocTitle }
func (d DocumentInfo) Description() string { return d.DocPreview }

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
	list       list.Model
	documents  []DocumentInfo
	selected   *DocumentInfo
	err        error
	width      int
	height     int
	searchPath string
}

// documentsLoadedMsg is sent when documents are loaded
type documentsLoadedMsg struct {
	documents []DocumentInfo
	err       error
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
)

func newMenuModel(searchPath string) menuModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = menuSpinnerStyle

	return menuModel{
		state:      stateLoading,
		spinner:    s,
		searchPath: searchPath,
	}
}

func (m menuModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadDocuments(m.searchPath),
	)
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.state == stateList {
				if i, ok := m.list.SelectedItem().(DocumentInfo); ok {
					m.selected = &i
					return m, tea.Quit
				}
			}
			if m.state == stateEmpty {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == stateList && m.width > 0 && m.height > 0 {
			m.list.SetSize(msg.Width, msg.Height-4)
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case documentsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateEmpty
			return m, nil
		}

		m.documents = msg.documents
		if len(m.documents) == 0 {
			m.state = stateEmpty
			return m, nil
		}

		// Create list items
		items := make([]list.Item, len(m.documents))
		for i, doc := range m.documents {
			items[i] = doc
		}

		// Create and configure the list
		delegate := list.NewDefaultDelegate()
		delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
			Foreground(lipgloss.Color("212")).
			BorderForeground(lipgloss.Color("212"))
		delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
			Foreground(lipgloss.Color("241"))
		// Set height to accommodate title (1) + 3 preview lines + spacing
		delegate.SetHeight(5)

		// Use sensible defaults if dimensions not yet set
		width := m.width
		height := m.height
		if width == 0 {
			width = 80
		}
		if height == 0 {
			height = 24
		}

		m.list = list.New(items, delegate, width, height-4)
		m.list.Title = "Select a document to edit"
		m.list.SetShowStatusBar(true)
		m.list.SetFilteringEnabled(true)
		m.list.Styles.Title = menuTitleStyle

		// Ensure we start at the first item
		m.list.Select(0)

		m.state = stateList
		return m, nil
	}

	// Update list if in list state
	if m.state == stateList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
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
		return "\n" + m.list.View()
	}

	return ""
}

// loadDocuments scans for .wrdoc.json files and returns their info
func loadDocuments(searchPath string) tea.Cmd {
	return func() tea.Msg {
		var documents []DocumentInfo

		// Find all .wrdoc.json files
		pattern := filepath.Join(searchPath, "*.wrdoc.json")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return documentsLoadedMsg{err: err}
		}

		for _, path := range matches {
			// Skip folders (.wrfolder.json)
			if strings.HasSuffix(path, ".wrfolder.json") {
				continue
			}

			info, err := parseDocumentInfo(path)
			if err != nil {
				// Skip files that can't be parsed
				continue
			}

			documents = append(documents, info)
		}

		return documentsLoadedMsg{documents: documents}
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

		// Run the menu
		model := newMenuModel(searchPath)
		p := tea.NewProgram(model, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running menu: %v\n", err)
			os.Exit(1)
		}

		// Check if a document was selected
		if m, ok := finalModel.(menuModel); ok && m.selected != nil {
			// Launch the editor with the selected document
			fmt.Printf("Opening: %s\n", m.selected.FilePath)

			editorProgram := tea.NewProgram(
				editor.New(m.selected.FilePath),
				tea.WithMouseCellMotion(),
			)

			if _, err := editorProgram.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	menuCmd.Flags().StringVarP(&menuSearchPath, "path", "p", ".", "Directory to search for documents")
	rootCmd.AddCommand(menuCmd)
}
