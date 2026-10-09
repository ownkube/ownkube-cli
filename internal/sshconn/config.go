package sshconn

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HostEntry is one okctl-managed Host block in ~/.ssh/config.
type HostEntry struct {
	Alias        string
	HostName     string
	Port         int
	User         string
	IdentityFile string
}

func (e HostEntry) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", beginMarker(e.Alias))
	fmt.Fprintf(&b, "Host %s\n", e.Alias)
	fmt.Fprintf(&b, "  HostName %s\n", e.HostName)
	if e.Port != 0 && e.Port != 22 {
		fmt.Fprintf(&b, "  Port %s\n", strconv.Itoa(e.Port))
	}
	fmt.Fprintf(&b, "  User %s\n", e.User)
	if e.IdentityFile != "" {
		fmt.Fprintf(&b, "  IdentityFile %s\n", quoteConfigValue(e.IdentityFile))
		fmt.Fprintf(&b, "  IdentitiesOnly yes\n")
	}
	fmt.Fprintf(&b, "%s\n", endMarker(e.Alias))
	return b.String()
}

func beginMarker(alias string) string { return "# okctl:begin " + alias }
func endMarker(alias string) string   { return "# okctl:end " + alias }

// RenderHostEntries renders entries as they would be written (for --dry-run).
func RenderHostEntries(entries []HostEntry) string {
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = e.render()
	}
	return strings.Join(parts, "\n")
}

// SSHConfigPath returns ~/.ssh/config.
func SSHConfigPath() (string, error) {
	dir, err := SSHDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config"), nil
}

// WriteHostEntries upserts entries into ~/.ssh/config. Blocks okctl wrote
// before (between its begin/end markers) are replaced in place; new ones are
// prepended, because ssh uses the first matching Host and a trailing
// "Host *" block would otherwise win.
func WriteHostEntries(entries []HostEntry) (string, error) {
	path, err := SSHConfigPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("reading %s: %w", tildePath(path), err)
	}
	content := string(raw)

	var prepend []string
	for _, e := range entries {
		begin, end := beginMarker(e.Alias), endMarker(e.Alias)
		i := strings.Index(content, begin+"\n")
		j := strings.Index(content, end+"\n")
		if i >= 0 && j > i {
			content = content[:i] + e.render() + content[j+len(end)+1:]
			continue
		}
		prepend = append(prepend, e.render())
	}
	if len(prepend) > 0 {
		head := strings.Join(prepend, "\n") + "\n"
		content = head + content
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", tildePath(path), err)
	}
	return path, nil
}

func quoteConfigValue(v string) string {
	if strings.ContainsAny(v, " \t") {
		return `"` + v + `"`
	}
	return v
}
