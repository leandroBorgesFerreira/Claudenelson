package cmd

import (
	"fmt"
	"os"
	"strings"

	"claudenelson/editor/block"
	"claudenelson/editor/factory"
	"claudenelson/editor/persistence"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var readFilePath string

// Styles for terminal output
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Underline(true)

	h1Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	h2Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213"))

	h3Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141"))

	h4Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("147")).
		Italic(true)

	textStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	listStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	checkboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	checkboxCheckedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46"))

	checkedContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Strikethrough(true)

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("0"))
)

var readCmd = &cobra.Command{
	Use:   "read [file]",
	Short: "Print document contents to terminal",
	Long:  "Reads a document file and prints its formatted contents to the terminal.",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
            readFilePath = args[0]
        }

		if _, err := os.Stat(readFilePath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", readFilePath)
			os.Exit(1)
		}

		f := factory.NewBlockFactory()
		doc, err := persistence.Load(readFilePath, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading document: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		for i, b := range doc.Blocks {
			line := renderBlock(b, i == 0)
			fmt.Println(line)
		}
		fmt.Println()
	},
}

func renderBlock(b block.Block, isFirst bool) string {
	content := b.Content()
	content = applySpanFormatting(content, b)

	switch b.Type() {
	case block.TypeH1:
		if isFirst {
			return titleStyle.Render(strings.ToUpper(content))
		}
		return h1Style.Render(strings.ToUpper(content))
	case block.TypeH2:
		return h2Style.Render(content)
	case block.TypeH3:
		return h3Style.Render(content)
	case block.TypeH4:
		return h4Style.Render(content)
	case block.TypeListItem:
		return listStyle.Render("• ") + textStyle.Render(content)
	case block.TypeCheckboxItem:
		if cb, ok := b.(*block.CheckboxBlock); ok {
			if cb.IsChecked() {
				return checkboxCheckedStyle.Render("[x] ") + checkedContentStyle.Render(content)
			}
			return checkboxUncheckedStyle.Render("[ ] ") + textStyle.Render(content)
		}
		return textStyle.Render(content)
	default:
		return textStyle.Render(content)
	}
}

func applySpanFormatting(content string, b block.Block) string {
	spans := b.Spans()
	if len(spans) == 0 {
		return content
	}

	runes := []rune(content)
	if len(runes) == 0 {
		return content
	}

	// Build style map for each character
	type charStyle struct {
		bold, italic, underline, highlight bool
	}
	styleMap := make([]charStyle, len(runes))

	for _, span := range spans {
		for i := span.Start; i < span.End && i < len(runes); i++ {
			if i >= 0 {
				if span.Style.Bold {
					styleMap[i].bold = true
				}
				if span.Style.Italic {
					styleMap[i].italic = true
				}
				if span.Style.Underline {
					styleMap[i].underline = true
				}
				if span.Style.Highlight {
					styleMap[i].highlight = true
				}
			}
		}
	}

	// Group consecutive characters with same style
	var result strings.Builder
	i := 0
	for i < len(runes) {
		currentStyle := styleMap[i]
		j := i + 1
		for j < len(runes) && styleMap[j] == currentStyle {
			j++
		}

		segment := string(runes[i:j])
		style := lipgloss.NewStyle()

		if currentStyle.bold {
			style = style.Bold(true)
		}
		if currentStyle.italic {
			style = style.Italic(true)
		}
		if currentStyle.underline {
			style = style.Underline(true)
		}
		if currentStyle.highlight {
			style = style.Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))
		}

		if currentStyle.bold || currentStyle.italic || currentStyle.underline || currentStyle.highlight {
			result.WriteString(style.Render(segment))
		} else {
			result.WriteString(segment)
		}

		i = j
	}

	return result.String()
}

func init() {
	rootCmd.AddCommand(readCmd)
}
