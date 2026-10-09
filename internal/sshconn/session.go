package sshconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ownkube/okctl/internal/version"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// keepAliveInterval keeps NAT/LB state alive on quiet sessions. The proxy's
// idle timeout counts channel data only, so this does not hold sessions open.
const keepAliveInterval = 30 * time.Second

// Target is where to dial: a region's SSH front door and the username that
// selects the deployment (and optionally the instance, "<id>+<n>").
type Target struct {
	Host     string
	Port     int
	User     string
	HostKeys []string
}

// Dial opens an authenticated SSH connection to the region front door,
// verifying the server against the published host keys, and starts
// keepalives that stop when ctx ends or the connection closes.
func Dial(ctx context.Context, t Target, signer ssh.Signer) (*ssh.Client, error) {
	cb, algos, err := pinnedHostKeys(t.Host, t.HostKeys)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:              t.User,
		Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback:   cb,
		HostKeyAlgorithms: algos,
		ClientVersion:     "SSH-2.0-okctl_" + strings.ReplaceAll(version.Version, " ", "_"),
		Timeout:           15 * time.Second,
	}

	addr := net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
	d := net.Dialer{Timeout: cfg.Timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		if strings.Contains(err.Error(), "unable to authenticate") {
			return nil, fmt.Errorf("Ownkube did not accept your SSH key for this service. " +
				"Check that you are an owner or admin of its organization, and see 'okctl ssh keys list'")
		}
		return nil, fmt.Errorf("SSH handshake with %s: %w", addr, err)
	}
	client := ssh.NewClient(c, chans, reqs)
	go keepAlive(ctx, client)
	return client, nil
}

func keepAlive(ctx context.Context, c *ssh.Client) {
	t := time.NewTicker(keepAliveInterval)
	defer t.Stop()
	done := make(chan struct{})
	go func() {
		_ = c.Wait()
		close(done)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-t.C:
			if _, _, err := c.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				return
			}
		}
	}
}

// Shell runs an interactive login shell on the connection and returns the
// remote exit code.
func Shell(c *ssh.Client) (int, error) {
	return run(c, "", true)
}

// Run executes command remotely and returns its exit code. It allocates a
// terminal only when both stdin and stdout are terminals, so pipes and
// scripts get clean, untranslated bytes.
func Run(c *ssh.Client, command string) (int, error) {
	return run(c, command, false)
}

func run(c *ssh.Client, command string, shell bool) (int, error) {
	sess, err := c.NewSession()
	if err != nil {
		return 255, fmt.Errorf("opening session: %w", err)
	}
	defer sess.Close()

	inFd, outFd := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	// Like OpenSSH: with stdin piped and no command, the remote shell reads
	// commands from stdin without a terminal.
	tty := term.IsTerminal(inFd) && term.IsTerminal(outFd)

	sess.Stdin = os.Stdin
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	if tty {
		cols, rows, err := term.GetSize(outFd)
		if err != nil {
			cols, rows = 80, 24
		}
		modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
		if err := sess.RequestPty(termName(), rows, cols, modes); err != nil {
			return 255, fmt.Errorf("requesting a terminal: %w", err)
		}
		state, err := term.MakeRaw(inFd)
		if err != nil {
			return 255, fmt.Errorf("setting terminal raw mode: %w", err)
		}
		defer term.Restore(inFd, state)
		stop := watchResize(sess, outFd)
		defer stop()
	}

	if shell {
		err = sess.Shell()
	} else {
		err = sess.Start(command)
	}
	if err != nil {
		return 255, fmt.Errorf("starting session: %w", err)
	}
	return exitCode(sess.Wait())
}

func exitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitStatus(), nil
	}
	var missing *ssh.ExitMissingError
	if errors.As(err, &missing) {
		return 255, fmt.Errorf("the connection closed before the command reported an exit status")
	}
	if errors.Is(err, io.EOF) {
		return 255, fmt.Errorf("the connection closed unexpectedly")
	}
	return 255, err
}

func termName() string {
	if t := os.Getenv("TERM"); t != "" {
		return t
	}
	return "xterm-256color"
}

// JoinArgs builds the remote command line the way OpenSSH does: the words are
// joined with spaces and the remote shell parses the result, so
// `okctl ssh -- 'id; hostname'` runs both commands, as `ssh host 'id; hostname'` would.
func JoinArgs(args []string) string {
	return strings.Join(args, " ")
}

// Forward accepts connections on ln and pipes each one through the SSH
// connection to remote (host:port as the server sees it). It returns when ln
// is closed or the SSH connection drops.
func Forward(ctx context.Context, c *ssh.Client, ln net.Listener, remote string, onErr func(error)) error {
	go func() {
		select {
		case <-ctx.Done():
		case <-waitClosed(c):
		}
		ln.Close()
	}()
	for {
		local, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go func() {
			defer local.Close()
			upstream, err := c.Dial("tcp", remote)
			if err != nil {
				if onErr != nil {
					onErr(err)
				}
				return
			}
			defer upstream.Close()
			pipe(local, upstream)
		}()
	}
}

func waitClosed(c *ssh.Client) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		_ = c.Wait()
		close(done)
	}()
	return done
}

// pipe copies both directions, half-closing writes when one side finishes.
func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		if cw, ok := dst.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
	<-done
}
