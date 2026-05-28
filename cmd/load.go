package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "cloud load [file]",
	Short: "Loads a document from backend into current folder",
	Long:  "Loads a document from backend into current folder",
	Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) {
		token, err := auth.LoadToken()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}

		docTitle := args[0]

		baseURL := "https://api.writeopia.dev" 
		apiURL := fmt.Sprintf("%s/documents/%s", baseURL, docTitle)

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 15 * time.Second}
		fmt.Printf("Fetching document '%s' from cloud...\n", docTitle)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("document %s not found on the server", docTitle)
		} else if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned unexpected status: %s", resp.Status)
		}

		filename := docTitle + "wrdoc.json" 
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create local file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save document content: %w", err)
		}

		fmt.Printf("🎉 Success! Document saved locally as %s \n", filename)
		return nil
	},
}

func init() {	
	rootCmd.AddCommand(loadCmd)
}