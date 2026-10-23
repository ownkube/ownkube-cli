// Package sourcearchive packages a local working tree as a gzipped tarball for
// upload, the source behind `okctl up`.
package sourcearchive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Create writes a gzipped tarball of dir to a temp file and returns its path
// and byte size. The caller owns the file and must remove it. File selection
// honours .gitignore when dir is a git work tree (via `git ls-files`); outside
// a repo it walks the tree. Either way .git, node_modules and .env* files are
// always left out (see Excluded).
func Create(ctx context.Context, dir string) (string, int64, error) {
	files, err := collectFiles(ctx, dir)
	if err != nil {
		return "", 0, err
	}
	if len(files) == 0 {
		return "", 0, fmt.Errorf("no files to deploy in %s", dir)
	}

	tmp, err := os.CreateTemp("", "okctl-source-*.tar.gz")
	if err != nil {
		return "", 0, fmt.Errorf("creating archive: %w", err)
	}
	// On any failure below, don't leak the temp file.
	cleanup := func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)

	if err := writeEntries(tw, dir, files); err != nil {
		cleanup()
		return "", 0, err
	}
	if err := tw.Close(); err != nil {
		cleanup()
		return "", 0, fmt.Errorf("finalizing archive: %w", err)
	}
	if err := gz.Close(); err != nil {
		cleanup()
		return "", 0, fmt.Errorf("finalizing archive: %w", err)
	}

	size, err := tmp.Seek(0, io.SeekCurrent)
	if err != nil {
		cleanup()
		return "", 0, fmt.Errorf("sizing archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("closing archive: %w", err)
	}
	return tmp.Name(), size, nil
}

// Excluded reports whether a path relative to the archive root is always left
// out, whatever .gitignore says: anything under a .git or node_modules
// directory (dependencies are installed by the build, not uploaded) and any
// .env* file (local secrets stay on this machine).
func Excluded(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if excludedDir(p) {
			return true
		}
	}
	base := parts[len(parts)-1]
	return excludedDir(base) || strings.HasPrefix(base, ".env")
}

func excludedDir(name string) bool {
	return name == ".git" || name == "node_modules"
}

// writeEntries adds each relative path (plus its parent directories, once) to
// the tar writer. Directory entries keep extractors that don't auto-create
// parents happy; files are streamed with their on-disk mode.
func writeEntries(tw *tar.Writer, dir string, rel []string) error {
	seenDir := map[string]bool{}
	for _, r := range rel {
		if err := writeParentDirs(tw, r, seenDir); err != nil {
			return err
		}
		if err := writeFile(tw, dir, r); err != nil {
			return err
		}
	}
	return nil
}

func writeParentDirs(tw *tar.Writer, rel string, seen map[string]bool) error {
	parent := filepath.Dir(rel)
	if parent == "." || parent == "/" {
		return nil
	}
	// Walk from the top down so parents precede children in the archive.
	parts := strings.Split(filepath.ToSlash(parent), "/")
	acc := ""
	for _, p := range parts {
		if acc == "" {
			acc = p
		} else {
			acc += "/" + p
		}
		if seen[acc] {
			continue
		}
		seen[acc] = true
		hdr := &tar.Header{
			Name:     acc + "/",
			Typeflag: tar.TypeDir,
			Mode:     0o755,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("writing dir %s: %w", acc, err)
		}
	}
	return nil
}

func writeFile(tw *tar.Writer, dir, rel string) error {
	abs := filepath.Join(dir, rel)
	info, err := os.Lstat(abs)
	if err != nil {
		return fmt.Errorf("reading %s: %w", rel, err)
	}
	// Skip anything that isn't a regular file (symlinks, sockets, devices):
	// a build context is source, not special files.
	if !info.Mode().IsRegular() {
		return nil
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("header for %s: %w", rel, err)
	}
	hdr.Name = filepath.ToSlash(rel)

	f, err := os.Open(abs)
	if err != nil {
		return fmt.Errorf("opening %s: %w", rel, err)
	}
	defer f.Close()

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing header for %s: %w", rel, err)
	}
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}

// collectFiles returns the working-tree files to archive, relative to dir. It
// prefers `git ls-files` (tracked + untracked-not-ignored) so .gitignore is
// honoured; outside a git work tree it walks the directory. Excluded paths
// are dropped either way.
func collectFiles(ctx context.Context, dir string) ([]string, error) {
	files, ok := gitTrackedFiles(ctx, dir)
	if !ok {
		return walkFiles(dir)
	}
	kept := files[:0]
	for _, f := range files {
		if !Excluded(f) {
			kept = append(kept, f)
		}
	}
	sort.Strings(kept)
	return kept, nil
}

// gitTrackedFiles lists the working tree via git, honouring the ignore rules.
// Returns ok=false when git is unavailable or dir is not a work tree, so the
// caller can fall back to a plain walk.
func gitTrackedFiles(ctx context.Context, dir string) ([]string, bool) {
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, false
	}
	// -c (cached/tracked) -o (others/untracked) --exclude-standard (apply
	// .gitignore/.git/info/exclude), NUL-separated for path safety.
	cmd := exec.CommandContext(ctx, git, "-C", dir, "ls-files", "-co", "--exclude-standard", "-z")
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	var files []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, true
}

// walkFiles is the non-git fallback: every regular file under dir except the
// excluded ones, as paths relative to dir.
func walkFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir && excludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if !Excluded(rel) {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}
