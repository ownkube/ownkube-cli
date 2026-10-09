package sshconn

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/ownkube/okctl/internal/client"
	"golang.org/x/crypto/ssh"
)

// TargetAPI is the slice of the API client GetTarget needs.
type TargetAPI interface {
	GetSSHTarget(ctx context.Context, deploymentID string) (*client.SSHTarget, error)
}

// OpenOptions selects the instance and key for Open.
type OpenOptions struct {
	EnsureOptions
	// Instance is the 1-based instance to reach; 0 means any running one.
	Instance int
}

// Open makes sure a registered key is available (creating one on first
// connect), records the region host keys in ~/.ssh/known_hosts, and dials
// target (from GetTarget). The caller closes the returned client.
func Open(ctx context.Context, api KeyAPI, target *client.SSHTarget, opts OpenOptions) (*ssh.Client, error) {
	user := target.Username
	if opts.Instance > 0 {
		if !hasInstance(target, opts.Instance) {
			return nil, fmt.Errorf("%s has no instance %d (it is running %d)",
				target.Name, opts.Instance, len(target.Instances))
		}
		user += "+" + strconv.Itoa(opts.Instance)
	}

	key, err := EnsureKey(ctx, api, opts.EnsureOptions)
	if err != nil {
		return nil, err
	}
	signer, err := key.Signer()
	if err != nil {
		return nil, err
	}

	log := opts.Log
	if log == nil {
		log = io.Discard
	}
	if err := RememberHostKeys(target.SSHHost, target.SSHPort, target.HostKeys); err != nil {
		fmt.Fprintf(log, "Warning: could not update ~/.ssh/known_hosts: %v\n", err)
	}

	return Dial(ctx, Target{
		Host:     target.SSHHost,
		Port:     target.SSHPort,
		User:     user,
		HostKeys: target.HostKeys,
	}, signer)
}

// GetTarget fetches the SSH target, turning "not available" into guidance.
func GetTarget(ctx context.Context, api TargetAPI, deploymentID string) (*client.SSHTarget, error) {
	target, err := api.GetSSHTarget(ctx, deploymentID)
	if client.IsAPICode(err, "SSH_NOT_AVAILABLE") {
		return nil, fmt.Errorf("SSH access is not available for this service. It needs to run on Ownkube Compute, " +
			"and you need to be an owner or admin of its organization")
	}
	if err != nil {
		return nil, err
	}
	if target.SSHPort == 0 {
		target.SSHPort = 22
	}
	return target, nil
}

func hasInstance(t *client.SSHTarget, n int) bool {
	for _, in := range t.Instances {
		if in.Index == n {
			return true
		}
	}
	return false
}
