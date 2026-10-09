package sshconn

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/prompt"
	"golang.org/x/term"
)

// KeyAPI is the slice of the API client EnsureKey needs.
type KeyAPI interface {
	ListSSHKeys(ctx context.Context) ([]client.SSHKey, error)
	AddSSHKey(ctx context.Context, name, publicKey string) (*client.SSHKey, bool, error)
}

// EnsureOptions controls first-connect key setup.
type EnsureOptions struct {
	// IdentityFile forces a specific key (registered on the fly if needed).
	IdentityFile string
	// Yes skips prompts: a new key is created when none is registered.
	Yes bool
	// Log receives progress lines (stderr in practice).
	Log io.Writer
}

// EnsureKey returns a local key that is registered with Ownkube, creating
// and uploading one on first connect. Order: the -i key; else the first
// local key (agent or ~/.ssh) already registered; else ~/.ssh/ownkube_ed25519
// if it exists; else prompt to create one or pick an existing key (created
// without prompting under --yes or when stdin is not a terminal).
func EnsureKey(ctx context.Context, api KeyAPI, opts EnsureOptions) (*LocalKey, error) {
	if opts.Log == nil {
		opts.Log = io.Discard
	}

	registered, err := api.ListSSHKeys(ctx)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(registered))
	for _, k := range registered {
		known[k.Fingerprint] = true
	}

	if opts.IdentityFile != "" {
		key, err := LoadKey(opts.IdentityFile)
		if err != nil {
			return nil, err
		}
		if !known[key.Fingerprint] {
			if err := Register(ctx, api, key, "", opts.Log); err != nil {
				return nil, err
			}
		}
		return key, nil
	}

	candidates := CandidateKeys()
	for _, k := range candidates {
		if known[k.Fingerprint] {
			return k, nil
		}
	}

	defaultPath, err := DefaultKeyPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(defaultPath); err == nil {
		key, err := LoadKey(defaultPath)
		if err != nil {
			return nil, err
		}
		return key, Register(ctx, api, key, "", opts.Log)
	}

	var files []*LocalKey
	for _, k := range candidates {
		if k.Path != "" {
			files = append(files, k)
		}
	}

	interactive := !opts.Yes && term.IsTerminal(int(os.Stdin.Fd()))
	choice := 0
	if interactive && len(files) > 0 {
		options := []string{fmt.Sprintf("Create a new key at %s (recommended)", tildePath(defaultPath))}
		for _, k := range files {
			options = append(options, fmt.Sprintf("Use %s.pub (%s)", k.Label(), k.Fingerprint))
		}
		fmt.Fprintln(opts.Log, "No SSH key on this machine is registered with Ownkube yet.")
		choice, err = prompt.Select("Which key should Ownkube use?", options)
		if err != nil {
			return nil, err
		}
	}

	var key *LocalKey
	if choice == 0 {
		key, err = GenerateKey(defaultPath, keyComment())
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(opts.Log, "Created a new SSH key at %s\n", tildePath(defaultPath))
	} else {
		key = files[choice-1]
	}
	return key, Register(ctx, api, key, "", opts.Log)
}

// Register uploads key's public half under name (default: this machine).
func Register(ctx context.Context, api KeyAPI, key *LocalKey, name string, log io.Writer) error {
	if name == "" {
		name = keyComment()
	}
	_, created, err := api.AddSSHKey(ctx, name, key.AuthorizedKey())
	if client.IsAPICode(err, "SSH_KEY_IN_USE") {
		return fmt.Errorf("the key %s is registered to a different Ownkube account; "+
			"pass -i with another key, or remove it from that account", key.Label())
	}
	if client.IsAPICode(err, "SSH_KEY_LIMIT") {
		return fmt.Errorf("you have reached the SSH key limit; remove one with 'okctl ssh keys remove <id>'")
	}
	if err != nil {
		return err
	}
	if created {
		fmt.Fprintf(log, "Added SSH key %s to your Ownkube account as %q\n", key.Fingerprint, name)
	}
	return nil
}

// DefaultKeyLabel names a key after this machine, e.g. "okctl@laptop".
func DefaultKeyLabel() string { return keyComment() }

// keyComment names a key after this machine, e.g. "okctl@laptop".
func keyComment() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}
	return "okctl@" + host
}
