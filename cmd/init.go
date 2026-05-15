package cmd

import (
	"fmt"
	"os"

	"claudenelson/editor"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Launch the text editor",
	Long:  "Launches the BubbleTea-based text editor with interactive color-changing text.",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(
			editor.New(),
			tea.WithMouseCellMotion(),
		)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
