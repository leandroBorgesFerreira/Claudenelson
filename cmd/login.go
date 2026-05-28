package cmd

import (  
  "fmt"

  "github.com/spf13/cobra"

  "os"
  "runtime"
  "os/exec"
  "context"
  "net/http"
  "time"
  "path/filepath"

  "encoding/json"
  "claudenelson/auth"
)

// loginSuccessHTML is what the user sees in the browser after successful login
const loginSuccessHTML = `
<!DOCTYPE html>
<html>
<head><title>Logged In</title></head>
<body style="font-family: sans-serif; text-align: center; margin-top: 50px;">
	<h1>🎉 Authenticated with Writeopia!</h1>
	<p>You can close this browser tab and return to your terminal.</p>
</body>
</html>
`

var loginCmd = &cobra.Command{
  Use:   "auth login",
  Short: "Login in Writeopia",
  Long:  `Login in Writeopia. Currently it is not possible to login in self deployed version of Writeopia`,
  RunE: func(cmd *cobra.Command, args []string) error {
	tokenChan := make(chan string)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {		
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Token missing from request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(loginSuccessHTML))
		
		tokenChan <- token
	})

	server := &http.Server{
		Addr:    ":9999",
		Handler: mux, 
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Local server error: %v\n", err)
		}
	}()

	authURL := "https://writeopia.io/login?redirect_uri=http://localhost:9999/callback"
	fmt.Println("Opening your browser to authenticate...")

	if err := openBrowser(authURL); err != nil {
		return fmt.Errorf("could not open browser automatically: %w. Please open this link manually: %s", err, authURL)
	}

	var finalToken string

	select {
		case token := <-tokenChan:
			_ = server.Shutdown(context.Background())
			finalToken = token
		case <-time.After(2 * time.Minute):
			_ = server.Shutdown(context.Background())
			return fmt.Errorf("login timed out after 2 minutes")
	}


	saveToken(finalToken)
	fmt.Printf("\nSuccessfully authenticated!")

	return nil
  },
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = append(args, "url.dll,FileProtocolHandler", url)
	case "darwin": // macOS
		cmd = "open"
		args = append(args, url)
	default: // Linux
		cmd = "xdg-open"
		args = append(args, url)
	}
	return exec.Command(cmd, args...).Start()
}

func saveToken(token string) error {
	home, _ := os.UserHomeDir()

	dirPath := filepath.Join(home, ".claudenelson")
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return err
	}

	cfg := auth.AuthConfig{
		Token: token,
	}

	filePath := filepath.Join(dirPath, "auth.json")

	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0600)
}

func init() {
  rootCmd.AddCommand(loginCmd)
}