// Package agents installs okctl into AI coding agents: the Ownkube MCP server
// entry (which runs `okctl mcp serve`, so no API key is written into an
// agent's config) and the okctl skill.
package agents

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ServerName is the name okctl's MCP server entry gets in every agent.
const ServerName = "ownkube"

// SkillName is the directory the okctl skill is installed under.
const SkillName = "okctl"

// Agent is an AI coding agent okctl can set up.
type Agent struct {
	ID   string // claude, cursor, codex
	Name string
}

// All lists the agents okctl knows how to set up.
var All = []Agent{
	{ID: "claude", Name: "Claude Code"},
	{ID: "cursor", Name: "Cursor"},
	{ID: "codex", Name: "Codex"},
}

// IDs is All's ids, for help text and errors.
func IDs() []string {
	ids := make([]string, len(All))
	for i, a := range All {
		ids[i] = a.ID
	}
	return ids
}

// Lookup returns the agent with id.
func Lookup(id string) (Agent, error) {
	for _, a := range All {
		if a.ID == id {
			return a, nil
		}
	}
	return Agent{}, fmt.Errorf("unknown agent %q: pick one of %s", id, strings.Join(IDs(), ", "))
}

// Select resolves ids to agents, or with none, every agent installed here.
func Select(ids []string) ([]Agent, error) {
	if len(ids) == 0 {
		var found []Agent
		for _, a := range All {
			if a.Detected() {
				found = append(found, a)
			}
		}
		if len(found) == 0 {
			return nil, fmt.Errorf("found no Claude Code, Cursor or Codex install: pick one with --agent %s", strings.Join(IDs(), "|"))
		}
		return found, nil
	}
	var out []Agent
	seen := map[string]bool{}
	for _, id := range ids {
		a, err := Lookup(strings.ToLower(strings.TrimSpace(id)))
		if err != nil {
			return nil, err
		}
		if !seen[a.ID] {
			seen[a.ID] = true
			out = append(out, a)
		}
	}
	return out, nil
}

// Detected reports whether the agent looks installed for this user.
func (a Agent) Detected() bool {
	if bin := a.cli(); bin != "" {
		if _, err := exec.LookPath(bin); err == nil {
			return true
		}
	}
	dir, err := a.home()
	if err != nil {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// cli is the agent's command-line tool, if it has one okctl drives.
func (a Agent) cli() string {
	switch a.ID {
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	}
	return ""
}

// home is the agent's per-user config directory.
func (a Agent) home() (string, error) {
	switch a.ID {
	case "claude":
		if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
			return d, nil
		}
	case "codex":
		if d := os.Getenv("CODEX_HOME"); d != "" {
			return d, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "."+a.ID), nil
}

// Server is the command an agent runs to start okctl's MCP server.
type Server struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// OkctlServer is the entry that starts this okctl binary's MCP bridge; flags
// are global flags to pin for it (e.g. --api-url).
func OkctlServer(flags ...string) Server {
	return Server{Command: executable(), Args: append([]string{"mcp", "serve"}, flags...)}
}

// executable is this binary's path, as the okctl on PATH when that's the same
// file, so a package manager's stable link survives upgrades. It's absolute
// because an agent started from a desktop launcher may not share the shell's
// PATH.
func executable() string {
	self, err := os.Executable()
	if err != nil {
		return "okctl"
	}
	if onPath, err := exec.LookPath("okctl"); err == nil {
		if abs, err := filepath.Abs(onPath); err == nil && sameFile(abs, self) {
			return abs
		}
	}
	return self
}

func sameFile(a, b string) bool {
	ia, err := os.Stat(a)
	if err != nil {
		return false
	}
	ib, err := os.Stat(b)
	return err == nil && os.SameFile(ia, ib)
}

// CommandLine renders the server as a shell command, for display.
func (s Server) CommandLine() string {
	parts := []string{shellQuote(s.Command)}
	for _, a := range s.Args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// Result says what installing into one agent did.
type Result struct {
	Agent  string `json:"agent"`
	Target string `json:"target"`
	Note   string `json:"note,omitempty"`
}

// Snippet is the config InstallMCP writes for a, for --print.
func (a Agent) Snippet(s Server) string {
	switch a.ID {
	case "claude":
		return "claude mcp add --scope user " + ServerName + " -- " + s.CommandLine()
	case "codex":
		return codexBlock(s)
	default:
		b, _ := json.MarshalIndent(map[string]any{"mcpServers": map[string]Server{ServerName: s}}, "", "  ")
		return string(b)
	}
}

// InstallMCP adds (or replaces) okctl's MCP server entry in a's user config.
// Other servers in that config are left as they are.
func (a Agent) InstallMCP(s Server) (Result, error) {
	switch a.ID {
	case "claude":
		return a.installClaude(s)
	case "cursor":
		return a.installCursor(s)
	case "codex":
		return a.installCodex(s)
	}
	return Result{}, fmt.Errorf("unknown agent %q", a.ID)
}

// installClaude goes through Claude Code's own CLI, which owns its config file.
func (a Agent) installClaude(s Server) (Result, error) {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return Result{}, fmt.Errorf("the claude command isn't on your PATH; add the server yourself with:\n  %s", a.Snippet(s))
	}
	// Replace an earlier entry; a missing one makes remove fail, which is fine.
	_ = exec.Command(bin, "mcp", "remove", "--scope", "user", ServerName).Run()
	args := append([]string{"mcp", "add", "--scope", "user", ServerName, "--", s.Command}, s.Args...)
	if out, err := exec.Command(bin, args...).CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("claude mcp add: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return Result{Agent: a.Name, Target: "user config (claude mcp add --scope user)"}, nil
}

func (a Agent) installCursor(s Server) (Result, error) {
	dir, err := a.home()
	if err != nil {
		return Result{}, err
	}
	path := filepath.Join(dir, "mcp.json")
	doc := map[string]any{}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return Result{}, err
	case len(bytes.TrimSpace(raw)) > 0:
		if err := json.Unmarshal(raw, &doc); err != nil {
			return Result{}, fmt.Errorf("%s isn't valid JSON, so okctl left it alone: %w", path, err)
		}
	}
	servers, _ := doc["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[ServerName] = s
	doc["mcpServers"] = servers
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return Result{}, err
	}
	if err := writeFile(path, append(out, '\n')); err != nil {
		return Result{}, err
	}
	return Result{Agent: a.Name, Target: path}, nil
}

// codexHeader opens okctl's table in Codex's config.toml.
const codexHeader = "[mcp_servers." + ServerName + "]"

func codexBlock(s Server) string {
	args := make([]string, len(s.Args))
	for i, a := range s.Args {
		args[i] = strconv.Quote(a)
	}
	return codexHeader + "\n" +
		"command = " + strconv.Quote(s.Command) + "\n" +
		"args = [" + strings.Join(args, ", ") + "]\n"
}

// installCodex replaces okctl's table in config.toml, or appends one, and
// keeps every other line of the file as it was.
func (a Agent) installCodex(s Server) (Result, error) {
	dir, err := a.home()
	if err != nil {
		return Result{}, err
	}
	path := filepath.Join(dir, "config.toml")
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
	}
	var kept []string
	skipping := false
	for _, line := range strings.SplitAfter(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			// Drop okctl's table and its subtables (e.g. [mcp_servers.ownkube.env]).
			skipping = t == codexHeader || strings.HasPrefix(t, "[mcp_servers."+ServerName+".")
		}
		if !skipping && line != "" {
			kept = append(kept, line)
		}
	}
	text := strings.TrimRight(strings.Join(kept, ""), "\n")
	if text != "" {
		text += "\n\n"
	}
	text += codexBlock(s)
	if err := writeFile(path, []byte(text)); err != nil {
		return Result{}, err
	}
	return Result{Agent: a.Name, Target: path}, nil
}

// SkillPath is where a's copy of the okctl skill goes.
func (a Agent) SkillPath() (string, error) {
	dir, err := a.home()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skills", SkillName, "SKILL.md"), nil
}

// InstallSkill writes the okctl skill into a's user skills directory,
// replacing an earlier copy.
func (a Agent) InstallSkill(skill []byte) (Result, error) {
	path, err := a.SkillPath()
	if err != nil {
		return Result{}, err
	}
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, skill) {
		return Result{Agent: a.Name, Target: path, Note: "already up to date"}, nil
	}
	if err := writeFile(path, skill); err != nil {
		return Result{}, err
	}
	return Result{Agent: a.Name, Target: path}, nil
}

// writeFile replaces path's contents atomically, keeping its mode when it
// already exists. A symlinked file (a dotfiles checkout) is written through.
func writeFile(path string, data []byte) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`!*?[]{}()<>|&;#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
