package link

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetGetRemove(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	// Unknown key: not present, no error.
	if _, ok, err := m.Get("/some/repo"); err != nil || ok {
		t.Fatalf("Get on empty store = (%v, %v), want (false, nil)", ok, err)
	}

	b := Binding{
		OrganizationID: "org_1",
		ClusterID:      "cl_1",
		EnvironmentID:  "env_1",
		DeploymentID:   "dep_1",
	}
	if err := m.Set("/some/repo", b); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok, err := m.Get("/some/repo")
	if err != nil || !ok {
		t.Fatalf("Get after Set = (%v, %v), want (true, nil)", ok, err)
	}
	if got != b {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, b)
	}

	// The file is created 0600.
	fi, err := os.Stat(filepath.Join(dir, "links.yaml"))
	if err != nil {
		t.Fatalf("stat links.yaml: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Fatalf("links.yaml perm = %o, want 600", perm)
	}

	removed, err := m.Remove("/some/repo")
	if err != nil || !removed {
		t.Fatalf("Remove = (%v, %v), want (true, nil)", removed, err)
	}
	if _, ok, _ := m.Get("/some/repo"); ok {
		t.Fatal("Get after Remove still present")
	}

	// Removing an absent key reports false, no error.
	if removed, err := m.Remove("/gone"); err != nil || removed {
		t.Fatalf("Remove absent = (%v, %v), want (false, nil)", removed, err)
	}
}

func TestSetReplaces(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	if err := m.Set("/repo", Binding{OrganizationID: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Set("/repo", Binding{OrganizationID: "b"}); err != nil {
		t.Fatal(err)
	}
	got, _, _ := m.Get("/repo")
	if got.OrganizationID != "b" {
		t.Fatalf("Set did not replace: got org %q, want b", got.OrganizationID)
	}
}

func TestMultipleKeysIndependent(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	_ = m.Set("/a", Binding{OrganizationID: "org_a"})
	_ = m.Set("/b", Binding{OrganizationID: "org_b"})

	a, _, _ := m.Get("/a")
	b, _, _ := m.Get("/b")
	if a.OrganizationID != "org_a" || b.OrganizationID != "org_b" {
		t.Fatalf("keys collided: a=%q b=%q", a.OrganizationID, b.OrganizationID)
	}
	// Removing one leaves the other.
	_, _ = m.Remove("/a")
	if _, ok, _ := m.Get("/b"); !ok {
		t.Fatal("removing /a dropped /b")
	}
}

func TestResolveKeyFindsRepoRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	key, err := ResolveKey(sub)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}
	// EvalSymlinks because macOS TempDir may live under /var -> /private/var.
	wantRoot, _ := filepath.EvalSymlinks(root)
	gotKey, _ := filepath.EvalSymlinks(key)
	if gotKey != wantRoot {
		t.Fatalf("ResolveKey(sub) = %q, want repo root %q", gotKey, wantRoot)
	}
}

func TestResolveKeyNoRepoUsesDir(t *testing.T) {
	dir := t.TempDir()
	key, err := ResolveKey(dir)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}
	wantDir, _ := filepath.EvalSymlinks(dir)
	gotKey, _ := filepath.EvalSymlinks(key)
	if gotKey != wantDir {
		t.Fatalf("ResolveKey with no repo = %q, want dir %q", gotKey, wantDir)
	}
}

func TestGitFileIsTreatedAsRoot(t *testing.T) {
	// Worktrees/submodules use a .git *file*, not a directory.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: ..."), 0644); err != nil {
		t.Fatal(err)
	}
	key, err := ResolveKey(root)
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, _ := filepath.EvalSymlinks(root)
	gotKey, _ := filepath.EvalSymlinks(key)
	if gotKey != wantRoot {
		t.Fatalf("ResolveKey with .git file = %q, want %q", gotKey, wantRoot)
	}
}
