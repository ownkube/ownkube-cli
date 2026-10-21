// Package update checks GitHub releases for a newer okctl, works out how this
// binary was installed, and replaces a release binary in place.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub repository okctl is released from.
const Repo = "ownkube/ownkube-cli"

// releasesAPI is overridable in tests.
var releasesAPI = "https://api.github.com/repos/" + Repo + "/releases/latest"

// Release is the latest published release.
type Release struct {
	Tag    string  `json:"tag_name"`
	URL    string  `json:"html_url"`
	Assets []Asset `json:"assets"`
}

// Asset is one downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Version is the release's version without the leading "v".
func (r Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }

func (r Release) asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// Latest fetches the latest release.
func Latest(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("checking for updates: GitHub returned %s", resp.Status)
	}
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Release{}, fmt.Errorf("reading the latest release: %w", err)
	}
	if r.Tag == "" {
		return Release{}, fmt.Errorf("the latest release has no tag")
	}
	return r, nil
}

// Newer reports whether version a is newer than b. Both are dotted numbers
// with an optional "v" prefix and "-suffix"; a build that isn't a release
// ("dev") is never older than anything.
func Newer(a, b string) bool {
	pa, okA := parse(a)
	pb, okB := parse(b)
	if !okA || !okB {
		return false
	}
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

// IsRelease reports whether v is a release version rather than a dev build.
func IsRelease(v string) bool {
	_, ok := parse(v)
	return ok
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Method is how this binary was installed.
type Method int

const (
	// Binary is a release archive unpacked by hand: okctl can replace it.
	Binary Method = iota
	// Homebrew installs upgrade with brew.
	Homebrew
	// GoInstall builds upgrade with go install.
	GoInstall
)

// InstallMethod works out how the binary at exe was installed.
func InstallMethod(exe string) Method {
	p := filepath.ToSlash(exe)
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		p = filepath.ToSlash(resolved)
	}
	if strings.Contains(p, "/Cellar/") || strings.Contains(p, "/homebrew/") || strings.Contains(p, "/linuxbrew/") {
		return Homebrew
	}
	for _, dir := range goBinDirs() {
		if dir != "" && filepath.ToSlash(filepath.Dir(p)) == filepath.ToSlash(dir) {
			return GoInstall
		}
	}
	return Binary
}

func goBinDirs() []string {
	dirs := []string{os.Getenv("GOBIN")}
	if gp := os.Getenv("GOPATH"); gp != "" {
		for _, p := range filepath.SplitList(gp) {
			dirs = append(dirs, filepath.Join(p, "bin"))
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "go", "bin"))
	}
	return dirs
}

// ArchiveName is the release archive for this platform. Archives are named
// after the repository, not the binary.
func ArchiveName() string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("ownkube-cli_%s_%s.%s", runtime.GOOS, runtime.GOARCH, ext)
}

// Replace downloads r's archive for this platform, checks it against the
// release's checksums.txt, and swaps it in for the binary at exe.
func Replace(ctx context.Context, r Release, exe string) error {
	name := ArchiveName()
	archive, ok := r.asset(name)
	if !ok {
		return fmt.Errorf("release %s has no %s", r.Tag, name)
	}
	sums, ok := r.asset("checksums.txt")
	if !ok {
		return fmt.Errorf("release %s has no checksums.txt", r.Tag)
	}

	sumsBody, err := download(ctx, sums.URL)
	if err != nil {
		return err
	}
	want, err := checksumFor(sumsBody, name)
	if err != nil {
		return err
	}
	body, err := download(ctx, archive.URL)
	if err != nil {
		return err
	}
	got := sha256.Sum256(body)
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("%s doesn't match its checksum; not installing it", name)
	}

	bin, err := extract(name, body)
	if err != nil {
		return err
	}
	return swap(exe, bin)
}

func download(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// checksumFor finds name's sha256 in a checksums.txt ("<hex>  <name>" lines).
func checksumFor(sums []byte, name string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[1] == name {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("checksums.txt has no entry for %s", name)
}

// extract pulls the okctl binary out of a release archive.
func extract(name string, body []byte) ([]byte, error) {
	want := "okctl"
	if runtime.GOOS == "windows" {
		want = "okctl.exe"
	}
	if strings.HasSuffix(name, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == want {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("%s has no %s", name, want)
	}
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s has no %s", name, want)
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == want {
			return io.ReadAll(tr)
		}
	}
}

// swap writes bin next to exe and renames it over exe, so a failure part-way
// leaves the old binary in place.
func swap(exe string, bin []byte) error {
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".okctl-upgrade-*")
	if err != nil {
		return fmt.Errorf("can't write to %s (%w); reinstall okctl or rerun with permission to write there", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		// A running .exe can't be overwritten, but it can be renamed aside.
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return err
		}
	}
	return os.Rename(tmpName, exe)
}
