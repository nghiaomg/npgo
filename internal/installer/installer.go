package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"npgo/internal/cache"
	"npgo/internal/cas"
	"npgo/internal/extractor"
	"npgo/internal/registry"
)

// Installer handles package installation and linking
type Installer struct {
	nodeModulesPath string
}

// NewInstaller creates a new installer
func NewInstaller(nodeModulesPath string) *Installer {
	return &Installer{
		nodeModulesPath: nodeModulesPath,
	}
}

// InstallPackage installs a single package and returns resolved version
func (i *Installer) InstallPackage(name, version string) (string, error) {
	resolvedVersion := version

	// Check if already installed (idempotent): if integrity matches, skip
	installedPath := filepath.Join(i.nodeModulesPath, name)
	if _, err := os.Stat(installedPath); err == nil {
		if iv, _ := readIntegrity(installedPath); iv == version {
			return version, nil
		}
		// If version differs, attempt relink (remove and continue)
		_ = os.RemoveAll(installedPath)
	}

	// Ensure node_modules exists
	if err := os.MkdirAll(i.nodeModulesPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create node_modules: %w", err)
	}

	// Check cache with original version first
	cachePath := cache.GetCachePath(name, version)

	// If not found, fetch metadata to get resolved version
	if !cache.Exists(cachePath) {
		// Fetch metadata
		metadata, err := registry.FetchMetadata(name, version)
		if err != nil {
			return "", fmt.Errorf("failed to fetch metadata: %w", err)
		}

		resolvedVersion = metadata.Version

		// Check cache again with resolved version
		cachePath = cache.GetCachePath(name, resolvedVersion)
		if !cache.Exists(cachePath) {
			// Streaming download, hash, store in CAS, then extract once
			body, err := registry.StreamTarball(metadata.TarballURL)
			if err != nil {
				return "", fmt.Errorf("failed to stream tarball: %w", err)
			}
			// Tee reader to compute hash
			pr, pw := io.Pipe()
			tee := io.TeeReader(body, pw)
			// hash in background
			var hash string
			var hashErr error
			done := make(chan struct{})
			go func() {
				defer close(done)
				hash, hashErr = cas.HashStream(tee)
				pw.Close()
				body.Close()
			}()
			// consume pipe to /dev/null
			go func() { io.Copy(io.Discard, pr); pr.Close() }()
			<-done
			if hashErr != nil {
				return "", fmt.Errorf("failed to hash tarball: %w", hashErr)
			}

			// Ensure CAS path
			casPath, err := cas.EnsureDirs(hash)
			if err != nil {
				return "", err
			}
			// If CAS already contains extraction, skip
			// Otherwise, download again (stream) and extract into CAS package dir
			exists, _ := cas.Exists(hash)
			if !exists {
				body2, err := registry.StreamTarball(metadata.TarballURL)
				if err != nil {
					return "", err
				}
				if err := extractor.ExtractFromReader(body2, casPath); err != nil {
					body2.Close()
					return "", err
				}
				body2.Close()
			}
			// Link from CAS to user cache's extracted path for compatibility
			extractPath := cache.GetExtractPath(name, metadata.Version)
			if err := createTreeLinkOrCopy(casPath, extractPath); err != nil {
				return "", err
			}
		}
	}

	// Get extract path with resolved version
	extractPath := cache.GetExtractPath(name, resolvedVersion)

	// Create symlink
	if err := i.createSymlink(name, extractPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	// Create/update global node_modules link for cross-project resolution
	if err := ensureGlobalPackageLink(name, extractPath); err != nil {
		// non-fatal
	}

	// Write per-package integrity file
	_ = writeIntegrity(installedPath, name, resolvedVersion, "")

	return resolvedVersion, nil
}

// createTreeLinkOrCopy links (hardlink) a tree from src to dst; copies if link fails
func createTreeLinkOrCopy(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("source is not a directory")
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := createTreeLinkOrCopy(s, d); err != nil {
				return err
			}
			continue
		}
		if err := linkFile(s, d); err != nil {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// Global node_modules helpers
func globalNodeModulesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".npgo", "node_modules")
	}
	return filepath.Join(home, ".npgo", "node_modules")
}

func ensureGlobalPackageLink(name, target string) error {
	base := globalNodeModulesPath()
	if err := os.MkdirAll(base, 0755); err != nil {
		return err
	}
	link := filepath.Join(base, name)
	// remove existing
	if _, err := os.Lstat(link); err == nil {
		_ = os.RemoveAll(link)
	}
	// create symlink/junction
	if runtime.GOOS == "windows" {
		if err := createJunctionWindows(link, target); err == nil {
			return nil
		}
		// fallback hardlink tree
		return createTreeLinkOrCopy(target, link)
	}
	if err := os.Symlink(target, link); err != nil {
		return createTreeLinkOrCopy(target, link)
	}
	return nil
}

// Integrity metadata helpers
func integrityFile(dir string) string { return filepath.Join(dir, ".npgo-integrity.json") }

func writeIntegrity(dir, name, version, hash string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data := fmt.Sprintf("{\n  \"name\": \"%s\",\n  \"version\": \"%s\",\n  \"hash\": \"%s\"\n}\n", name, version, hash)
	return os.WriteFile(integrityFile(dir), []byte(data), 0644)
}

func readIntegrity(dir string) (string, error) {
	b, err := os.ReadFile(integrityFile(dir))
	if err != nil {
		return "", err
	}
	// naive parse to avoid extra dep: extract version field
	// expect: "version": "..."
	bs := string(b)
	const key = "\"version\""
	idx := strings.Index(bs, key)
	if idx == -1 {
		return "", fmt.Errorf("no version in integrity")
	}
	rest := bs[idx+len(key):]
	q1 := strings.Index(rest, "\"")
	if q1 == -1 {
		return "", fmt.Errorf("parse error")
	}
	rest = rest[q1+1:]
	q2 := strings.Index(rest, "\"")
	if q2 == -1 {
		return "", fmt.Errorf("parse error")
	}
	return rest[:q2], nil
}

// createSymlink creates a symbolic link from node_modules to cache
func (i *Installer) createSymlink(name, targetPath string) error {
	linkPath := filepath.Join(i.nodeModulesPath, name)

	// Check if target exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("target path does not exist: %s", targetPath)
	}

	// Remove existing link if exists
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing link: %w", err)
		}
	}

	// Try relative symlink first
	relPath, err := filepath.Rel(i.nodeModulesPath, targetPath)
	if err == nil {
		if err := os.Symlink(relPath, linkPath); err == nil {
			return nil
		} else {
			// On Windows, lack of privilege often causes EPERM
			if runtime.GOOS == "windows" {
				if mkErr := createJunctionWindows(linkPath, targetPath); mkErr == nil {
					return nil
				}
				// Fallback: copy directory
				return copyDir(targetPath, linkPath)
			}
			// Non-windows: retry absolute symlink; if fails, fallback to copy
			if absErr := os.Symlink(targetPath, linkPath); absErr == nil {
				return nil
			}
			return copyDir(targetPath, linkPath)
		}
	}

	// If cannot compute relative, attempt absolute symlink
	if err := os.Symlink(targetPath, linkPath); err == nil {
		return nil
	}

	// Windows junction fallback
	if runtime.GOOS == "windows" {
		if mkErr := createJunctionWindows(linkPath, targetPath); mkErr == nil {
			return nil
		}
	}

	// Final fallback: copy
	return copyDir(targetPath, linkPath)
}

// createJunctionWindows creates a directory junction using mklink /J
func createJunctionWindows(linkPath, targetPath string) error {
	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	// mklink requires cmd.exe
	// mklink /J link target
	cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, targetPath)
	// Inherit environment; hide window
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink failed: %v: %s", err, string(out))
	}
	return nil
}

// copyDir recursively links (hardlink) files from src to dst when possible, otherwise copies
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("source is not a directory")
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		sPath := filepath.Join(src, e.Name())
		dPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(sPath, dPath); err != nil {
				return err
			}
			continue
		}
		// Try hardlink first
		if err := linkFile(sPath, dPath); err != nil {
			// Fallback to copy
			if err := copyFile(sPath, dPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// linkFile tries to create a hardlink from src to dst
func linkFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	// Remove existing destination if any
	if _, err := os.Lstat(dst); err == nil {
		_ = os.Remove(dst)
	}
	return os.Link(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// InstallAll installs all dependencies
func (i *Installer) InstallAll(packages map[string]string) error {
	// parallel install with worker pool
	type job struct{ name, version string }
	jobs := make(chan job, len(packages))
	errs := make(chan error, len(packages))

	const maxWorkers = 16
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			if _, err := i.InstallPackage(j.name, j.version); err != nil {
				errs <- fmt.Errorf("failed to install %s: %w", j.name, err)
			} else {
				errs <- nil
			}
		}
	}

	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go worker()
	}

	for name, version := range packages {
		jobs <- job{name: name, version: version}
	}
	close(jobs)

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Clean removes all installed packages
func (i *Installer) Clean() error {
	if err := os.RemoveAll(i.nodeModulesPath); err != nil {
		return fmt.Errorf("failed to clean node_modules: %w", err)
	}
	return nil
}
