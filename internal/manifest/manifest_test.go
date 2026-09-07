package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{
		Name:     "api",
		Type:     "web",
		Image:    "ghcr.io/acme/api",
		Tag:      "v1.2.3",
		Port:     8080,
		Replicas: 2,
		Public:   true,
		Env: []EnvVar{
			{Name: "LOG_LEVEL", Value: "info"},
			{Name: "DB_PASSWORD", Secret: true},
		},
	}
	if err := Write(dir, m, false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != m.Name || got.Type != m.Type || got.Port != m.Port || got.Replicas != m.Replicas || got.Public != m.Public {
		t.Fatalf("scalar round-trip mismatch: got %+v", got)
	}
	if len(got.Env) != 2 || got.Env[1].Name != "DB_PASSWORD" || !got.Env[1].Secret {
		t.Fatalf("env round-trip mismatch: got %+v", got.Env)
	}
}

func TestSecretValueNotWritten(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{
		Name: "api",
		Env:  []EnvVar{{Name: "DB_PASSWORD", Secret: true}},
	}
	if err := Write(dir, m, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	// A secret entry carries only a reference; there is no value to leak, and
	// omitempty keeps the empty value out of the file entirely.
	if strings.Contains(string(data), "value:") {
		t.Fatalf("secret-only manifest wrote a value: field:\n%s", data)
	}
}

func TestWriteRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{Name: "api"}
	if err := Write(dir, m, false); err != nil {
		t.Fatal(err)
	}
	// Second write without force must fail and not clobber.
	if err := Write(dir, &Manifest{Name: "changed"}, false); err == nil {
		t.Fatal("Write without force overwrote an existing file")
	}
	got, _ := Load(dir)
	if got.Name != "api" {
		t.Fatalf("existing spec was clobbered: name = %q", got.Name)
	}
	// With force it overwrites.
	if err := Write(dir, &Manifest{Name: "changed"}, true); err != nil {
		t.Fatalf("Write force: %v", err)
	}
	got, _ = Load(dir)
	if got.Name != "changed" {
		t.Fatalf("force write did not take: name = %q", got.Name)
	}
}

func TestLoadRequiresName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(Path(dir), []byte("type: web\nport: 8080\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("Load accepted a manifest with no name")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if Exists(dir) {
		t.Fatal("Exists true on empty dir")
	}
	_ = os.WriteFile(Path(dir), []byte("name: api\n"), 0644)
	if !Exists(dir) {
		t.Fatal("Exists false after write")
	}
}

func TestJSONProjection(t *testing.T) {
	m := &Manifest{Name: "api", Type: "web", Port: 8080}
	b, err := m.JSON()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"name":"api"`) || !strings.Contains(s, `"port":8080`) {
		t.Fatalf("unexpected JSON: %s", s)
	}
}

func TestPath(t *testing.T) {
	if Path("/x/y") != filepath.Join("/x/y", FileName) {
		t.Fatalf("Path wrong: %s", Path("/x/y"))
	}
}
