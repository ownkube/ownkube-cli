package sourcearchive

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExcluded(t *testing.T) {
	cases := map[string]bool{
		"main.go":                      false,
		"src/app.ts":                   false,
		"environment.ts":               false,
		"docs/env.md":                  false,
		".gitignore":                   false,
		".github/workflows/ci.yml":     false,
		".env":                         true,
		".env.local":                   true,
		".env.production":              true,
		"apps/web/.env":                true,
		".git/config":                  true,
		"node_modules/react/index.js":  true,
		"apps/web/node_modules/a/b.js": true,
		"packages/x/.git/HEAD":         true,
		"node_modules":                 true,
	}
	for path, want := range cases {
		if got := Excluded(path); got != want {
			t.Errorf("Excluded(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestWalkFilesSkipsExcluded(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		"index.js",
		"src/app.js",
		".env",
		".env.local",
		".git/config",
		"node_modules/pkg/index.js",
	} {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := walkFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"index.js", filepath.Join("src", "app.js")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("walkFiles = %v, want %v", got, want)
	}
}
