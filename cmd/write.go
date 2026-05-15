package cmd

import (
	"fmt"
	"os"

	"claudenelson/editor"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var filePath string

var writeCmd = &cobra.Command{
	Use:   "write",
	Short: "Launch the text editor",
	Long:  "Launches the BubbleTea-based text editor. Loads existing document if file exists, auto-saves on changes.",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(
			editor.New(filePath),
			tea.WithMouseCellMotion(),
		)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	writeCmd.Flags().StringVarP(&filePath, "file", "f", "document.json", "Path to save/load document")
	rootCmd.AddCommand(writeCmd)
}
