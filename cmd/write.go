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
	Use:   "write [file]",
	Short: "Launch the text editor",
	Long:  "Launches the BubbleTea-based text editor. Loads existing document if file exists, auto-saves on changes.",
	Args: cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		filePath = "document.wrdoc.json"

        // Overwrite it if the user provided a file path positional argument
        if len(args) > 0 {
            filePath = args[0]
        }

		p := tea.NewProgram(
			editor.New(filePath),
		)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {	
	rootCmd.AddCommand(writeCmd)
}
