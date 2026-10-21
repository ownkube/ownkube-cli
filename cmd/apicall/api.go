// Package apicall implements `okctl api`: a raw, authenticated request to the
// Ownkube CLI API, for endpoints and options okctl has no command for yet.
package apicall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var methods = map[string]bool{
	http.MethodGet: true, http.MethodPost: true, http.MethodPut: true,
	http.MethodPatch: true, http.MethodDelete: true,
}

// New returns the `okctl api` command.
func New() *cobra.Command {
	var (
		data    string
		include bool
	)
	cmd := &cobra.Command{
		Use:   "api <method> <path>",
		Short: "Make an authenticated request to the Ownkube API",
		Long: "Send a request to the Ownkube CLI API with your credentials and print the " +
			"response body. The path is relative to the API, e.g. /v1/projects. --data " +
			"(-d) is a JSON body, @file to read one from a file, or - for stdin. " +
			"Exits non-zero when the response is an error status.",
		Example: "  okctl api GET /v1/projects\n" +
			"  okctl api GET '/v1/deployments?projectId=prj_123'\n" +
			"  okctl api PATCH /v1/deployments/dep_123/name -d '{\"displayName\":\"API\"}'\n" +
			"  okctl api POST /v1/deployments/dep_123/restart",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			if !methods[method] {
				return fmt.Errorf("method must be GET, POST, PUT, PATCH or DELETE, got %q", args[0])
			}
			path := args[1]
			if strings.Contains(path, "://") {
				// Never send the API key to a host other than the configured API.
				return fmt.Errorf("pass a path like /v1/projects, not a URL")
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			var body io.Reader
			if data != "" {
				raw, err := readData(data)
				if err != nil {
					return err
				}
				if !json.Valid(raw) {
					return fmt.Errorf("--data is not valid JSON")
				}
				body = bytes.NewReader(raw)
			}

			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			resp, err := cl.Do(cmd.Context(), method, path, body)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			if include {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n", resp.Proto, resp.Status)
				for k, vs := range resp.Header {
					for _, v := range vs {
						fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", k, v)
					}
				}
				fmt.Fprintln(cmd.ErrOrStderr())
			}
			out := cmd.OutOrStdout()
			if len(raw) > 0 {
				if f, ok := out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
					var pretty bytes.Buffer
					if json.Indent(&pretty, raw, "", "  ") == nil {
						raw = pretty.Bytes()
					}
				}
				out.Write(raw)
				if raw[len(raw)-1] != '\n' {
					fmt.Fprintln(out)
				}
			}
			if resp.StatusCode >= 400 {
				return fmt.Errorf("%s %s: %s", method, path, resp.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&data, "data", "d", "", "JSON request body, @file, or - for stdin")
	cmd.Flags().BoolVar(&include, "include", false, "Print the response status and headers to stderr")
	return cmd
}

func readData(data string) ([]byte, error) {
	switch {
	case data == "-":
		return io.ReadAll(os.Stdin)
	case strings.HasPrefix(data, "@"):
		return os.ReadFile(data[1:])
	default:
		return []byte(data), nil
	}
}
