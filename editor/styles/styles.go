package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Heading styles - visually distinct without # prefix
	H1Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Underline(true)

	H2Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213"))

	H3Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141"))

	H4Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("147")).
		Italic(true)

	// Text style
	TextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// List item styles
	ListPrefixStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	ListContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// Checkbox styles
	CheckboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	CheckboxCheckedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46"))

	CheckboxContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	CheckboxCheckedContentStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("243")).
					Strikethrough(true)

	// Focus indicator
	FocusIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true)

	UnfocusedIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Faint(true)

	// Title
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	// Formatting indicators
	BoldIndicator = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	ItalicIndicator = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("33")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	UnderlineIndicator = lipgloss.NewStyle().
				Underline(true).
				Foreground(lipgloss.Color("33")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)

	// Highlight style (yellow background)
	HighlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("0"))

	HighlightIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("226")).
				Padding(0, 1)

	// Selection style (for highlight mode)
	SelectionStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("39")).
			Foreground(lipgloss.Color("0"))

	// Multi-line selection indicator
	SelectionIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("62")).
				Padding(0, 1)
)
