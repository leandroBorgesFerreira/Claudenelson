package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Heading styles
	H1Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))

	H2Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69"))

	H3Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	H4Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("57"))

	// Heading prefix styles (dimmed)
	HeadingPrefixStyle = lipgloss.NewStyle().
				Faint(true)

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
)
