// Package connect implements `okctl connect`: open psql or valkey-cli against
// an Ownkube Compute database or cache, or hold a local tunnel to it for GUI
// tools. Uses the public endpoint when it is on, otherwise an SSH tunnel
// through the region's SSH endpoint.
package connect

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/sshconn"
	"github.com/spf13/cobra"
)

// New returns the `okctl connect` command.
func New() *cobra.Command {
	var (
		serviceFlag  string
		tunnelOnly   bool
		portFlag     int
		viaSSH       bool
		identityFlag string
		yesFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "connect [deployment-id]",
		Short: "Open a client for a database or cache, or a local tunnel to it",
		Long: "Open psql (databases) or valkey-cli / redis-cli (caches) connected to an\n" +
			"Ownkube Compute database or cache.\n\n" +
			"When public access is on, okctl connects to the public endpoint. Otherwise\n" +
			"(or with --ssh) it opens a private tunnel over SSH to a local port; the first\n" +
			"time, okctl sets up an SSH key for you as 'okctl ssh' does.\n\n" +
			"--tunnel-only keeps the tunnel open and prints a local connection string\n" +
			"for GUI tools (TablePlus, DBeaver, RedisInsight) until you press Ctrl+C.",
		Example: "  okctl connect dep_db\n" +
			"  okctl connect dep_db --tunnel-only --port 15432",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flag := serviceFlag
			if len(args) == 1 {
				flag = args[0]
			}
			depID, err := ux.ResolveDeployment(flag, ".")
			if err != nil {
				return err
			}
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			info, err := cl.ConnectionInfo(ctx, depID)
			if err != nil {
				return err
			}
			if info.Paused {
				return fmt.Errorf("this %s is paused for billing and accepts no connections until it resumes", info.ResourceType)
			}

			if !viaSSH && !tunnelOnly && portFlag == 0 && info.PublicReady && info.PublicUri != "" {
				return launchClient(ctx, info.ResourceType, info.PublicUri)
			}

			target, err := sshconn.GetTarget(ctx, cl, depID)
			if err != nil {
				return err
			}
			if target.Kind == "app" {
				return fmt.Errorf("%s is an app; 'okctl connect' is for databases and caches. Use 'okctl ssh %s' for a shell",
					target.Name, depID)
			}

			c, redial, err := sshconn.OpenRedialable(ctx, cl, target, sshconn.OpenOptions{
				EnsureOptions: sshconn.EnsureOptions{IdentityFile: identityFlag, Yes: yesFlag, Log: cmd.ErrOrStderr()},
			})
			if err != nil {
				return err
			}
			defer c.Close()

			ln, err := listen(portFlag, int(info.Port), tunnelOnly)
			if err != nil {
				return err
			}
			localPort := ln.Addr().(*net.TCPAddr).Port
			uri := localURI(info, localPort)

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			remote := net.JoinHostPort("localhost", strconv.Itoa(int(info.Port)))
			// If the SSH session drops, Forward redials and keeps the local
			// port, so only open client connections need to reconnect.
			stderr := cmd.ErrOrStderr()
			fwdErr := make(chan error, 1)
			go func() {
				fwdErr <- sshconn.Forward(ctx, c, redial, ln, remote, sshconn.ForwardHooks{
					ConnFailed: func(err error) {
						fmt.Fprintf(stderr, "Tunnel connection failed: %v\n", err)
					},
					Reconnecting: func() {
						fmt.Fprintln(stderr, "Tunnel dropped, reconnecting...")
					},
					Reconnected: func() {
						fmt.Fprintln(stderr, "Tunnel reconnected.")
					},
				})
			}()

			if tunnelOnly {
				if ux.IsStructured() {
					if err := ux.Print(cmd.OutOrStdout(), map[string]any{
						"host": "127.0.0.1", "port": localPort, "uri": uri,
					}); err != nil {
						return err
					}
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Tunnel open on 127.0.0.1:%d\n", localPort)
					fmt.Fprintf(cmd.OutOrStdout(), "Connection string: %s\n", uri)
					fmt.Fprintln(cmd.ErrOrStderr(), "Press Ctrl+C to close.")
				}
				sig, stop := signal.NotifyContext(ctx, os.Interrupt)
				defer stop()
				select {
				case <-sig.Done():
					return nil
				case err := <-fwdErr:
					if err != nil {
						return fmt.Errorf("tunnel closed: %w", err)
					}
					return fmt.Errorf("tunnel closed")
				}
			}

			return launchClient(ctx, info.ResourceType, uri)
		},
	}

	cmd.Flags().StringVar(&serviceFlag, "service", "", "Deployment ID (defaults to this directory's link)")
	cmd.Flags().BoolVar(&tunnelOnly, "tunnel-only", false, "Hold a local tunnel open instead of launching a client")
	cmd.Flags().IntVar(&portFlag, "port", 0, "Local port for the tunnel (default: the standard port if free, else any free port)")
	cmd.Flags().BoolVar(&viaSSH, "ssh", false, "Use a private SSH tunnel even when public access is on")
	cmd.Flags().StringVarP(&identityFlag, "identity-file", "i", "", "SSH private key to use (added to your account if needed)")
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Create and add an SSH key without asking when none is set up")
	return cmd
}

// listen binds the tunnel's local port on loopback: the requested one, else
// the datastore's standard port when free (tunnel-only, so GUI tools can use
// their defaults), else any free port.
func listen(requested, standard int, preferStandard bool) (net.Listener, error) {
	if requested != 0 {
		ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(requested)))
		if err != nil {
			return nil, fmt.Errorf("local port %d is not available: %w", requested, err)
		}
		return ln, nil
	}
	if preferStandard && standard != 0 {
		if ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(standard))); err == nil {
			return ln, nil
		}
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

// localURI builds a connection string for the tunnel's local end. The tunnel
// itself is encrypted, so the local hop is plaintext.
func localURI(info *api.ConnectionInfo, port int) string {
	host := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	switch info.ResourceType {
	case api.ConnectionInfoResourceTypeDatabase:
		u := url.URL{
			Scheme:   "postgresql",
			User:     url.UserPassword(info.Username, info.Password),
			Host:     host,
			Path:     "/" + info.Database,
			RawQuery: "sslmode=disable",
		}
		return u.String()
	default:
		u := url.URL{Scheme: "redis", Host: host}
		if info.Password != "" {
			u.User = url.UserPassword("default", info.Password)
		}
		return u.String()
	}
}

// launchClient runs psql or valkey-cli/redis-cli against uri, attached to
// this terminal, and exits with its status.
func launchClient(ctx context.Context, kind api.ConnectionInfoResourceType, uri string) error {
	var (
		bin  string
		argv []string
	)
	switch kind {
	case api.ConnectionInfoResourceTypeDatabase:
		if p, err := exec.LookPath("psql"); err == nil {
			bin, argv = p, []string{uri}
		}
	default:
		for _, name := range []string{"valkey-cli", "redis-cli"} {
			if p, err := exec.LookPath(name); err == nil {
				bin, argv = p, []string{"-u", uri}
				// The public cache edge routes by SNI, and valkey-cli/redis-cli
				// send none over TLS unless told to.
				if u, err := url.Parse(uri); err == nil && u.Scheme == "rediss" {
					argv = append(argv, "--sni", u.Hostname())
				}
				break
			}
		}
	}
	if bin == "" {
		want := "psql"
		if kind != api.ConnectionInfoResourceTypeDatabase {
			want = "valkey-cli or redis-cli"
		}
		return fmt.Errorf("%s is not installed. Install it, or run with --tunnel-only and use any client", want)
	}

	// The client owns Ctrl+C (cancel a query, clear a line); okctl swallows it
	// while the client runs so the tunnel stays up. Notify, not Ignore: an
	// ignored signal would be inherited by the client.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	defer signal.Stop(sigs)

	child := exec.CommandContext(ctx, bin, argv...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := child.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return err
}
