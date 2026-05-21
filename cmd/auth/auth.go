// Package auth holds login, logout, and status commands — the browser-based
// auth flow lives in login.go, the others are thin wrappers around the config
// manager and /v1/info.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/config"
	"github.com/spf13/cobra"
)

// Login returns the `okctl login` command.
func Login() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Ownkube via browser",
		Long: `Opens your browser to authorize the CLI with your Ownkube account.

The CLI starts a local callback server, opens the Ownkube authorization page
in your browser, and waits for you to grant access. Once authorized, your
API key is stored locally for subsequent commands.`,
		RunE: runLogin,
	}
}

// Logout returns the `okctl logout` command.
func Logout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ux.Config().DeleteCredentials(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out successfully.")
			return nil
		},
	}
}

// Status returns the `okctl status` command.
func Status() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current login status and user info",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := ux.Config().LoadCredentials()
			if err != nil {
				return err
			}
			if creds == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Not logged in.")
				return nil
			}

			c, err := client.New(ux.APIURL(), creds.APIKey)
			if err != nil {
				return err
			}

			info, err := c.GetUserInfo(cmd.Context())
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Logged in locally but API key is invalid or expired.")
				fmt.Fprintln(cmd.OutOrStdout(), "Run 'okctl login' to re-authenticate.")
				return nil
			}

			if ux.IsStructured() {
				return ux.Print(cmd.OutOrStdout(), map[string]string{
					"user_id": info.ID,
					"name":    info.Name,
					"email":   info.Email,
					"api_url": ux.APIURL(),
				})
			}
			return ux.Print(cmd.OutOrStdout(), [][]string{
				{"FIELD", "VALUE"},
				{"Name", info.Name},
				{"Email", info.Email},
				{"User ID", info.ID},
				{"API URL", ux.APIURL()},
			})
		},
	}
}

func runLogin(cmd *cobra.Command, args []string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	state, err := randomState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	type callbackResult struct {
		apiKey string
		err    error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		receivedState := r.URL.Query().Get("state")
		apiKey := r.URL.Query().Get("api_key")

		if receivedState != state {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, htmlPage("Authorization Failed", "State mismatch — possible CSRF attack. Please try again."))
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch")}
			return
		}

		if apiKey == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no API key received"
			}
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, htmlPage("Authorization Failed", errMsg))
			resultCh <- callbackResult{err: fmt.Errorf("authorization failed: %s", errMsg)}
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, htmlPage("Authorization Successful", "You can close this window and return to the terminal."))
		resultCh <- callbackResult{apiKey: apiKey}
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultCh <- callbackResult{err: fmt.Errorf("callback server error: %w", err)}
		}
	}()
	defer server.Close()

	authURL := fmt.Sprintf("%s/cli-authorize?port=%d&state=%s", ux.APIURL(), port, state)
	fmt.Fprintf(cmd.OutOrStdout(), "Opening browser for authorization...\n")
	fmt.Fprintf(cmd.OutOrStdout(), "If the browser doesn't open, visit:\n  %s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Could not open browser automatically: %v\n", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Waiting for authorization...\n")

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	var apiKey string
	select {
	case result := <-resultCh:
		if result.err != nil {
			return result.err
		}
		apiKey = result.apiKey
	case <-ctx.Done():
		return fmt.Errorf("authorization timed out — please try again")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Verifying credentials...\n")
	c, err := client.New(ux.APIURL(), apiKey)
	if err != nil {
		return err
	}

	info, err := c.GetUserInfo(cmd.Context())
	if err != nil {
		return fmt.Errorf("API key verification failed: %w", err)
	}

	creds := &config.Credentials{
		APIKey:        apiKey,
		UserID:        info.ID,
		UserName:      info.Name,
		UserEmail:     info.Email,
		EmailVerified: info.EmailVerified,
	}
	if err := ux.Config().SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nLogged in as %s (%s)\n", info.Name, info.Email)
	return nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var binary string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		binary = "open"
	case "linux":
		binary = "xdg-open"
	case "windows":
		binary = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		return fmt.Errorf("unsupported platform")
	}
	args = append(args, url)
	return exec.Command(binary, args...).Start()
}

func htmlPage(title, message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>okctl — %s</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #0a0a0a; color: #fafafa; }
.card { text-align: center; padding: 2rem; }
h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
p { color: #a0a0a0; }
</style></head>
<body><div class="card"><h1>%s</h1><p>%s</p></div></body>
</html>`, title, title, message)
}
