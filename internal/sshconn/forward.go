package sshconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// redialWindow bounds how long a dropped tunnel keeps trying to reconnect.
const redialWindow = 60 * time.Second

// ForwardHooks reports tunnel events. Any field may be nil.
type ForwardHooks struct {
	// ConnFailed: one local connection could not be forwarded.
	ConnFailed func(error)
	// Reconnecting: the SSH connection dropped and is being redialed.
	Reconnecting func()
	// Reconnected: a redial succeeded.
	Reconnected func()
}

// Forward accepts connections on ln and pipes each one through the SSH
// connection to remote (host:port as the server sees it). When the connection
// drops, Forward redials and keeps ln open: connections already open end with
// the old session, new ones use the new one. With a nil redial it stops when
// the connection drops instead. It returns nil when ctx ends, or an error when
// ln fails or the connection is lost for good.
func Forward(ctx context.Context, c *ssh.Client, redial Redialer, ln net.Listener, remote string, hooks ForwardHooks) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cur := newLive(c)
	defer cur.close()

	lost := make(chan error, 1)
	go func() {
		defer ln.Close()
		lost <- supervise(ctx, cur, redial, hooks)
	}()
	for {
		local, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// supervise sends before closing ln, so a lost connection is here.
			select {
			case lostErr := <-lost:
				return lostErr
			default:
				return err
			}
		}
		go forwardOne(ctx, cur, local, remote, hooks)
	}
}

// supervise redials whenever the current connection ends, until ctx ends
// (nil) or a redial gives up (its error).
func supervise(ctx context.Context, cur *live, redial Redialer, hooks ForwardHooks) error {
	for {
		_, dead, _ := cur.get()
		select {
		case <-ctx.Done():
			return nil
		case <-dead:
		}
		if redial == nil {
			return errors.New("the SSH connection closed")
		}
		call(hooks.Reconnecting)
		c, err := redialFor(ctx, redial)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		cur.swap(c)
		call(hooks.Reconnected)
	}
}

// redialFor retries redial with backoff for up to redialWindow. A refused key
// is final: the same key won't be accepted on the next try either.
func redialFor(ctx context.Context, redial Redialer) (*ssh.Client, error) {
	deadline := time.Now().Add(redialWindow)
	wait := 250 * time.Millisecond
	for {
		c, err := redial(ctx)
		if err == nil {
			return c, nil
		}
		if errors.Is(err, ErrKeyNotAccepted) || time.Now().Add(wait).After(deadline) {
			return nil, fmt.Errorf("reconnecting: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		wait = min(wait*2, 5*time.Second)
	}
}

// forwardOne pipes one local connection. A dial on a connection that is going
// away waits for the redial and tries again; any other failure is reported.
func forwardOne(ctx context.Context, cur *live, local net.Conn, remote string, hooks ForwardHooks) {
	defer local.Close()
	for {
		c, dead, swapped := cur.get()
		upstream, err := c.Dial("tcp", remote)
		if err == nil {
			pipe(local, upstream)
			upstream.Close()
			return
		}
		select {
		case <-dead:
		case <-time.After(time.Second):
			if hooks.ConnFailed != nil {
				hooks.ConnFailed(err)
			}
			return
		}
		select {
		case <-swapped:
		case <-ctx.Done():
			return
		}
	}
}

// live is the tunnel's current SSH connection, replaced on each redial.
type live struct {
	mu      sync.Mutex
	c       *ssh.Client
	dead    <-chan struct{} // closed when c's connection ends
	swapped chan struct{}   // closed when c is replaced
}

func newLive(c *ssh.Client) *live {
	l := &live{}
	l.swap(c)
	return l
}

func (l *live) get() (*ssh.Client, <-chan struct{}, <-chan struct{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.c, l.dead, l.swapped
}

func (l *live) swap(c *ssh.Client) {
	l.mu.Lock()
	prev := l.swapped
	l.c, l.dead, l.swapped = c, waitClosed(c), make(chan struct{})
	l.mu.Unlock()
	if prev != nil {
		close(prev)
	}
}

func (l *live) close() {
	c, _, _ := l.get()
	_ = c.Close()
}

func waitClosed(c *ssh.Client) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		_ = c.Wait()
		close(done)
	}()
	return done
}

func call(f func()) {
	if f != nil {
		f()
	}
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
