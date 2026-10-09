//go:build windows

package sshconn

import (
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// watchResize polls the console size (Windows has no SIGWINCH) and forwards
// changes to the session.
func watchResize(sess *ssh.Session, fd int) (stop func()) {
	done := make(chan struct{})
	go func() {
		lastCols, lastRows, _ := term.GetSize(fd)
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				cols, rows, err := term.GetSize(fd)
				if err != nil || (cols == lastCols && rows == lastRows) {
					continue
				}
				lastCols, lastRows = cols, rows
				_ = sess.WindowChange(rows, cols)
			}
		}
	}()
	return func() { close(done) }
}
