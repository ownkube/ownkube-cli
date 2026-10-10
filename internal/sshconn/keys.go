// Package sshconn is okctl's SSH client side of Ownkube shell access:
// local key discovery and creation, registering keys with
// the API, pinning region host keys, and running shells, commands, and port
// forwards over golang.org/x/crypto/ssh (no OpenSSH dependency, so it works
// on Windows too).
package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/ownkube/okctl/internal/prompt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// DefaultKeyName is the key okctl creates on first connect, under ~/.ssh.
const DefaultKeyName = "ownkube_ed25519"

// candidateNames are the ~/.ssh keys okctl looks at, in preference order.
var candidateNames = []string{DefaultKeyName, "id_ed25519", "id_ecdsa", "id_rsa"}

// LocalKey is a public key okctl can authenticate with: a key file on disk
// (Path set) or an identity held by the running ssh-agent (Path empty).
type LocalKey struct {
	Path        string
	Public      ssh.PublicKey
	Fingerprint string
	Comment     string

	agentSigner ssh.Signer
	signer      ssh.Signer
}

// AuthorizedKey renders the key as a single authorized_keys line.
func (k *LocalKey) AuthorizedKey() string {
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k.Public)))
	if k.Comment != "" {
		line += " " + k.Comment
	}
	return line
}

// Label is a short human description: the file path, or the agent comment.
func (k *LocalKey) Label() string {
	if k.Path != "" {
		return tildePath(k.Path)
	}
	if k.Comment != "" {
		return k.Comment + " (ssh-agent)"
	}
	return "ssh-agent key"
}

// Signer returns a signer for the key, preferring the ssh-agent copy (no
// passphrase prompt) and otherwise reading the private key file, prompting
// for its passphrase when it is encrypted.
func (k *LocalKey) Signer() (ssh.Signer, error) {
	if k.agentSigner != nil {
		return k.agentSigner, nil
	}
	if k.signer != nil {
		return k.signer, nil
	}
	if s := agentSignerFor(k.Public); s != nil {
		k.agentSigner = s
		return s, nil
	}
	if k.Path == "" {
		return nil, fmt.Errorf("key %s is no longer available in ssh-agent", k.Fingerprint)
	}
	pemBytes, err := os.ReadFile(k.Path)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(pemBytes)
	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		pass, perr := prompt.ReadSecret(fmt.Sprintf("Passphrase for %s: ", tildePath(k.Path)))
		if perr != nil {
			return nil, perr
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(pass))
	}
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", tildePath(k.Path), err)
	}
	k.signer = signer
	return signer, nil
}

// SSHDir returns ~/.ssh.
func SSHDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	return filepath.Join(home, ".ssh"), nil
}

// DefaultKeyPath returns ~/.ssh/ownkube_ed25519.
func DefaultKeyPath() (string, error) {
	dir, err := SSHDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultKeyName), nil
}

// GenerateKey writes a new unencrypted ed25519 key pair to path (0600) and
// path+".pub". It refuses to overwrite an existing file.
func GenerateKey(path, comment string) (*LocalKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return nil, fmt.Errorf("encoding key: %w", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("writing private key: %w", err)
	}
	if _, err := f.Write(pem.EncodeToMemory(block)); err != nil {
		f.Close()
		return nil, fmt.Errorf("writing private key: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	key := &LocalKey{
		Path:        path,
		Public:      sshPub,
		Fingerprint: ssh.FingerprintSHA256(sshPub),
		Comment:     comment,
	}
	if err := os.WriteFile(path+".pub", []byte(key.AuthorizedKey()+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing public key: %w", err)
	}
	return key, nil
}

// LoadKey reads the public half of the key at path (a private key path or
// its .pub). It reads path+".pub" when present, otherwise derives the public
// key from the private file (which may prompt for a passphrase).
func LoadKey(path string) (*LocalKey, error) {
	path = strings.TrimSuffix(expandHome(path), ".pub")
	if raw, err := os.ReadFile(path + ".pub"); err == nil {
		pub, comment, _, _, perr := ssh.ParseAuthorizedKey(raw)
		if perr != nil {
			return nil, fmt.Errorf("parsing %s.pub: %w", tildePath(path), perr)
		}
		return &LocalKey{Path: path, Public: pub, Fingerprint: ssh.FingerprintSHA256(pub), Comment: comment}, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("key %s: %w", tildePath(path), err)
	}
	k := &LocalKey{Path: path}
	signer, err := k.Signer()
	if err != nil {
		return nil, err
	}
	k.Public = signer.PublicKey()
	k.Fingerprint = ssh.FingerprintSHA256(k.Public)
	return k, nil
}

// CandidateKeys lists keys okctl could use without creating one: ssh-agent
// identities first, then well-known ~/.ssh key files that have a .pub.
// Duplicates (same key in the agent and on disk) are merged, keeping the path.
func CandidateKeys() []*LocalKey {
	var out []*LocalKey
	seen := map[string]*LocalKey{}

	for _, s := range agentSigners() {
		pub := s.PublicKey()
		k := &LocalKey{Public: pub, Fingerprint: ssh.FingerprintSHA256(pub), agentSigner: s}
		if c, ok := pub.(*agent.Key); ok {
			k.Comment = c.Comment
		}
		seen[k.Fingerprint] = k
		out = append(out, k)
	}

	dir, err := SSHDir()
	if err != nil {
		return out
	}
	for _, name := range candidateNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path + ".pub"); err != nil {
			continue
		}
		k, err := LoadKey(path)
		if err != nil {
			continue
		}
		if existing, ok := seen[k.Fingerprint]; ok {
			existing.Path = path
			continue
		}
		seen[k.Fingerprint] = k
		out = append(out, k)
	}
	return out
}

// agentSigners returns the running ssh-agent's identities, or nil when no
// agent is reachable. The agent connection stays open for the process.
func agentSigners() []ssh.Signer {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	signers, err := agent.NewClient(conn).Signers()
	if err != nil {
		conn.Close()
		return nil
	}
	return signers
}

func agentSignerFor(pub ssh.PublicKey) ssh.Signer {
	want := string(pub.Marshal())
	for _, s := range agentSigners() {
		if string(s.PublicKey().Marshal()) == want {
			return s
		}
	}
	return nil
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}
