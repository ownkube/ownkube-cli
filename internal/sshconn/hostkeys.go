package sshconn

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// parseHostKeys parses the region's published host keys
// ("ssh-ed25519 AAAA…" lines).
func parseHostKeys(lines []string) ([]ssh.PublicKey, error) {
	var keys []ssh.PublicKey
	for _, l := range lines {
		pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(l))
		if err != nil {
			return nil, fmt.Errorf("parsing published host key: %w", err)
		}
		keys = append(keys, pub)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("Ownkube has not published a host key for this region yet")
	}
	return keys, nil
}

// pinnedHostKeys returns a callback that accepts only the published keys
// (no trust on first use) and the algorithms to negotiate for them.
func pinnedHostKeys(host string, lines []string) (ssh.HostKeyCallback, []string, error) {
	keys, err := parseHostKeys(lines)
	if err != nil {
		return nil, nil, err
	}
	var algos []string
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k.Type()] {
			continue
		}
		seen[k.Type()] = true
		if k.Type() == ssh.KeyAlgoRSA {
			algos = append(algos, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256)
			continue
		}
		algos = append(algos, k.Type())
	}
	cb := func(_ string, _ net.Addr, got ssh.PublicKey) error {
		for _, k := range keys {
			if bytes.Equal(k.Marshal(), got.Marshal()) {
				return nil
			}
		}
		return fmt.Errorf("the server at %s presented host key %s, which is not a key Ownkube publishes; refusing to connect",
			host, ssh.FingerprintSHA256(got))
	}
	return cb, algos, nil
}

// KnownHostsLines renders the published keys as known_hosts entries.
func KnownHostsLines(host string, port int, lines []string) ([]string, error) {
	keys, err := parseHostKeys(lines)
	if err != nil {
		return nil, err
	}
	addr := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, knownhosts.Line([]string{addr}, k))
	}
	return out, nil
}

// RememberHostKeys appends the published keys to ~/.ssh/known_hosts so stock
// ssh/scp/sftp trust the region too. Entries already present are skipped.
func RememberHostKeys(host string, port int, lines []string) error {
	entries, err := KnownHostsLines(host, port, lines)
	if err != nil {
		return err
	}
	dir, err := SSHDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "known_hosts")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading known_hosts: %w", err)
	}
	have := map[string]bool{}
	for _, l := range strings.Split(string(existing), "\n") {
		have[strings.TrimSpace(l)] = true
	}

	var add []string
	for _, e := range entries {
		if !have[e] {
			add = append(add, e)
		}
	}
	if len(add) == 0 {
		return nil
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("writing known_hosts: %w", err)
	}
	defer f.Close()
	prefix := ""
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		prefix = "\n"
	}
	_, err = f.WriteString(prefix + strings.Join(add, "\n") + "\n")
	return err
}
