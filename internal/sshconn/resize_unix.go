//go:build !windows

package sshconn

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// watchResize forwards terminal size changes (SIGWINCH) to the session.
func watchResize(sess *ssh.Session, fd int) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ch:
				if cols, rows, err := term.GetSize(fd); err == nil {
					_ = sess.WindowChange(rows, cols)
				}
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
