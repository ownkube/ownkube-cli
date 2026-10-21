package ux

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	linkstore "github.com/ownkube/okctl/internal/link"
)

// Commands take apps, environments and projects by name, slug or id. An id is
// used as-is; anything else is matched case-insensitively against the
// resource's name (and slug or display name) from the list endpoints, so a
// name that matches more than one resource is an error listing every match.

// prefixedID matches the "<kind>_<base32>" id form some resources use.
var prefixedID = regexp.MustCompile(`^[a-z]{2,10}_[0-9A-Za-z]{8,}$`)

// LooksLikeID reports whether ref is an id rather than a name, so it can skip
// the list lookup.
func LooksLikeID(ref string) bool {
	if _, err := uuid.Parse(ref); err == nil {
		return true
	}
	return prefixedID.MatchString(ref)
}

// Scope narrows an app lookup to one environment or project ("" = unset).
type Scope struct {
	ProjectID     string
	EnvironmentID string
}

type candidate struct {
	id, label string
}

// pickOne returns the single candidate's id, "" when there are none, and an
// error naming every match when there are several.
func pickOne(kind, ref string, matches []candidate) (string, error) {
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0].id, nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].label < matches[j].label })
	var b strings.Builder
	fmt.Fprintf(&b, "%q matches %d %ss; pass one of these ids instead:", ref, len(matches), kind)
	for _, m := range matches {
		fmt.Fprintf(&b, "\n  %s  %s", m.id, m.label)
	}
	return "", fmt.Errorf("%s", b.String())
}

func nameMatches(ref string, names ...string) bool {
	for _, n := range names {
		if n != "" && strings.EqualFold(n, ref) {
			return true
		}
	}
	return false
}

// ResolveProject turns a project name, slug or id into its id ("" stays "").
func ResolveProject(ctx context.Context, cl *client.Client, ref string) (string, error) {
	if ref == "" || LooksLikeID(ref) {
		return ref, nil
	}
	projects, err := cl.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	var matches []candidate
	for _, p := range projects {
		if p.Id == ref || nameMatches(ref, p.Name, p.Slug) {
			matches = append(matches, candidate{p.Id, p.Name})
		}
	}
	id, err := pickOne("project", ref, matches)
	if err != nil || id != "" {
		return id, err
	}
	if _, err := cl.GetProject(ctx, ref); err == nil {
		return ref, nil
	}
	return "", fmt.Errorf("no project named %q (see 'okctl projects list')", ref)
}

// ResolveEnvironment turns an environment name, slug or id into its id ("" stays
// ""). projectID, when set, limits the match to that project: every project
// has its own "Production", so a bare name is often ambiguous without it.
func ResolveEnvironment(ctx context.Context, cl *client.Client, ref, projectID string) (string, error) {
	if ref == "" || LooksLikeID(ref) {
		return ref, nil
	}
	envs, err := cl.ListEnvironments(ctx)
	if err != nil {
		return "", err
	}
	var hits []api.Environment
	for _, e := range envs {
		if projectID != "" && Deref(e.ProjectId) != projectID {
			continue
		}
		if e.Id == ref || nameMatches(ref, e.Name, e.Slug) {
			hits = append(hits, e)
		}
	}
	// Several projects can each have an environment by this name; label each
	// match with its project so the error is readable.
	projectNames := map[string]string{}
	if len(hits) > 1 {
		if projects, err := cl.ListProjects(ctx); err == nil {
			for _, p := range projects {
				projectNames[p.Id] = p.Name
			}
		}
	}
	matches := make([]candidate, len(hits))
	for i, e := range hits {
		pid := Deref(e.ProjectId)
		matches[i] = candidate{e.Id, fmt.Sprintf("%s (project %s; pass --project to narrow)", e.Name, orID(projectNames[pid], pid))}
	}
	id, err := pickOne("environment", ref, matches)
	if err != nil || id != "" {
		return id, err
	}
	if _, err := cl.GetEnvironment(ctx, ref); err == nil {
		return ref, nil
	}
	return "", fmt.Errorf("no environment named %q (see 'okctl environments list')", ref)
}

func orID(name, id string) string {
	if name == "" {
		return id
	}
	return name
}

// ListAllDeployments lists every deployment the organization can see, across
// every project.
func ListAllDeployments(ctx context.Context, cl *client.Client) ([]api.Deployment, error) {
	projects, err := cl.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var all []api.Deployment
	for _, p := range projects {
		deps, err := cl.ListDeployments(ctx, client.ListDeploymentsFilter{ProjectID: p.Id})
		if err != nil {
			return nil, err
		}
		for _, d := range deps {
			if !seen[d.Id] {
				seen[d.Id] = true
				all = append(all, d)
			}
		}
	}
	return all, nil
}

// ListScopedDeployments lists the deployments in scope: one environment, one
// project, or (with neither) every project.
func ListScopedDeployments(ctx context.Context, cl *client.Client, s Scope) ([]api.Deployment, error) {
	switch {
	case s.EnvironmentID != "":
		return cl.ListDeployments(ctx, client.ListDeploymentsFilter{EnvironmentID: s.EnvironmentID})
	case s.ProjectID != "":
		return cl.ListDeployments(ctx, client.ListDeploymentsFilter{ProjectID: s.ProjectID})
	default:
		return ListAllDeployments(ctx, cl)
	}
}

// ResolveApp turns an app name or id into its deployment id. With ref empty it
// falls back to this directory's link. A name is looked up in the given scope;
// with no scope it tries the linked environment first, then everything.
func ResolveApp(ctx context.Context, cl *client.Client, ref string, s Scope, dir string) (string, error) {
	if ref == "" {
		return ResolveDeployment("", dir)
	}
	if LooksLikeID(ref) {
		return ref, nil
	}

	scopes := []Scope{s}
	if s == (Scope{}) {
		if b, ok, _ := Linked(dir); ok && b.EnvironmentID != "" {
			scopes = []Scope{{EnvironmentID: b.EnvironmentID}, {}}
		}
	}
	for _, sc := range scopes {
		deps, err := ListScopedDeployments(ctx, cl, sc)
		if err != nil {
			return "", err
		}
		var matches []candidate
		for _, d := range deps {
			if d.Id == ref || nameMatches(ref, d.Name, Deref(d.DisplayName)) {
				matches = append(matches, candidate{d.Id, fmt.Sprintf("%s (%s, environment %s; pass --environment to narrow)", d.Name, d.ResourceType, d.EnvironmentId)})
			}
		}
		id, err := pickOne("app", ref, matches)
		if err != nil || id != "" {
			return id, err
		}
	}
	if _, err := cl.GetDeployment(ctx, ref); err == nil {
		return ref, nil
	}
	return "", fmt.Errorf("no app named %q (see 'okctl deploy list')", ref)
}

// Linked returns the binding for dir's link, if any.
func Linked(dir string) (linkstore.Binding, bool, error) {
	if dir == "" {
		dir = "."
	}
	key, err := linkstore.ResolveKey(dir)
	if err != nil {
		return linkstore.Binding{}, false, err
	}
	return linkstore.NewManager(g.Config.Dir()).Get(key)
}
