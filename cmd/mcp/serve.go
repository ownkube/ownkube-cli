package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/client"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the Ownkube MCP server over stdio, for an AI agent",
		Long: "Bridge the Ownkube MCP server to stdin and stdout: one JSON-RPC message per " +
			"line in, each reply on its own line out. An agent starts this itself; set it " +
			"up with 'okctl mcp install'. Requests carry the credentials from 'okctl login' " +
			"and the organization picked with --organization, OKCTL_ORGANIZATION or " +
			"'okctl config set organization'. Diagnostics go to stderr.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			b := &bridge{cl: cl, out: cmd.OutOrStdout(), log: cmd.ErrOrStderr()}
			return b.run(cmd.Context(), cmd.InOrStdin())
		},
	}
}

// maxMessage caps one JSON-RPC message read from the agent.
const maxMessage = 32 << 20

// bridge relays an agent's stdio JSON-RPC messages to the Ownkube MCP server.
type bridge struct {
	cl  *client.Client
	log io.Writer

	outMu sync.Mutex
	out   io.Writer

	mu      sync.Mutex
	session string
	init    []byte // the agent's initialize request, replayed if the session expires

	renew sync.Mutex
	wg    sync.WaitGroup
}

func (b *bridge) run(ctx context.Context, in io.Reader) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64<<10), maxMessage)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		msg := append([]byte(nil), line...)
		// Until the server has issued a session, handle messages in order so
		// initialize settles first; after that a slow tool call mustn't hold up
		// the rest (a cancellation, say).
		if b.sessionID() == "" {
			b.handle(ctx, msg)
			continue
		}
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.handle(ctx, msg)
		}()
	}
	b.wg.Wait()
	b.end()
	if err := sc.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	return nil
}

func (b *bridge) sessionID() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.session
}

// message is the part of a JSON-RPC message the bridge looks at. A batch
// (a JSON array) leaves it empty.
type message struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
}

func (b *bridge) handle(ctx context.Context, msg []byte) {
	var m message
	_ = json.Unmarshal(msg, &m)
	if m.Method == "initialize" {
		b.mu.Lock()
		b.init = msg
		b.mu.Unlock()
	}

	resp, err := b.send(ctx, msg, m.Method == "initialize")
	if err != nil {
		b.fail(m, err)
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusAccepted:
	case resp.StatusCode >= 400:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if isJSONRPC(raw) {
			b.emit(raw)
			return
		}
		b.fail(m, httpError(resp.StatusCode, raw))
	case strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"):
		b.stream(resp.Body)
	default:
		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxMessage))
		if err != nil {
			b.fail(m, err)
			return
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			b.emit(raw)
		}
	}
}

// send posts msg in the current session. When the server no longer knows the
// session, it starts a new one by replaying the agent's initialize and sends
// msg again.
func (b *bridge) send(ctx context.Context, msg []byte, initialize bool) (*http.Response, error) {
	sid := b.sessionID()
	if initialize {
		sid = ""
	}
	resp, err := b.cl.MCP(ctx, http.MethodPost, msg, sid)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound && sid != "" {
		resp.Body.Close()
		if err := b.reinitialize(ctx, sid); err != nil {
			return nil, err
		}
		if resp, err = b.cl.MCP(ctx, http.MethodPost, msg, b.sessionID()); err != nil {
			return nil, err
		}
	}
	if s := resp.Header.Get(client.MCPSessionHeader); s != "" {
		b.mu.Lock()
		b.session = s
		b.mu.Unlock()
	}
	return resp, nil
}

// reinitialize replaces the expired session stale, once however many
// requests found it expired.
func (b *bridge) reinitialize(ctx context.Context, stale string) error {
	b.renew.Lock()
	defer b.renew.Unlock()
	b.mu.Lock()
	current, init := b.session, b.init
	b.mu.Unlock()
	if current != stale {
		return nil
	}
	if init == nil {
		return fmt.Errorf("the MCP session expired; restart the MCP server in your agent")
	}
	fmt.Fprintln(b.log, "okctl mcp: session expired, starting a new one")

	resp, err := b.cl.MCP(ctx, http.MethodPost, init, "")
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	sid := resp.Header.Get(client.MCPSessionHeader)
	if resp.StatusCode >= 400 || sid == "" {
		return fmt.Errorf("the MCP session expired and a new one couldn't be started (%s)", resp.Status)
	}
	b.mu.Lock()
	b.session = sid
	b.mu.Unlock()

	resp, err = b.cl.MCP(ctx, http.MethodPost, []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`), sid)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

// stream relays each event of a server-sent event stream as one message.
func (b *bridge) stream(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), maxMessage)
	var data []string
	flush := func() {
		if len(data) > 0 {
			b.emit([]byte(strings.Join(data, "\n")))
			data = nil
		}
	}
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		fmt.Fprintln(b.log, "okctl mcp: reading the server's event stream:", err)
	}
}

// emit writes one message to the agent on a single line.
func (b *bridge) emit(raw []byte) {
	var line bytes.Buffer
	if err := json.Compact(&line, raw); err != nil {
		fmt.Fprintln(b.log, "okctl mcp: dropped a reply that isn't JSON:", err)
		return
	}
	line.WriteByte('\n')
	b.outMu.Lock()
	defer b.outMu.Unlock()
	b.out.Write(line.Bytes())
}

// fail reports err on stderr and, for a request, answers it with a JSON-RPC
// error so the agent isn't left waiting.
func (b *bridge) fail(m message, err error) {
	fmt.Fprintln(b.log, "okctl mcp:", err)
	if m.Method == "" || len(m.ID) == 0 || string(m.ID) == "null" {
		return
	}
	reply, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      m.ID,
		"error":   map[string]any{"code": -32603, "message": err.Error()},
	})
	b.emit(reply)
}

// end closes the session when the agent goes away.
func (b *bridge) end() {
	sid := b.sessionID()
	if sid == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if resp, err := b.cl.MCP(ctx, http.MethodDelete, nil, sid); err == nil {
		resp.Body.Close()
	}
}

func isJSONRPC(raw []byte) bool {
	var m struct {
		JSONRPC string `json:"jsonrpc"`
	}
	return json.Unmarshal(raw, &m) == nil && m.JSONRPC != ""
}

// httpError explains a failed request in terms of what the user can fix.
func httpError(status int, raw []byte) error {
	var e struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	switch {
	case status == http.StatusUnauthorized:
		return fmt.Errorf("okctl's sign-in was rejected (%s): run 'okctl login' and restart the MCP server", orStatus(e.Code, status))
	case e.Code == "ORGANIZATION_FORBIDDEN":
		return fmt.Errorf("this account can't use the selected organization: check 'okctl config get organization' or the --organization in the agent's MCP config")
	case e.Error != "":
		return fmt.Errorf("the Ownkube MCP server said: %s (%s)", e.Error, orStatus(e.Code, status))
	}
	return fmt.Errorf("the Ownkube MCP server returned HTTP %d", status)
}

func orStatus(code string, status int) string {
	if code != "" {
		return code
	}
	return fmt.Sprintf("HTTP %d", status)
}
