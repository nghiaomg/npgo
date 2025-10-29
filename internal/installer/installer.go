package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"npgo/internal/ui"
)

type Installer struct {
	nodeModulesPath string
	debug           bool
}

// PackageSpec is a minimal spec for pipeline install
type PackageSpec struct {
	Name       string
	Version    string
	TarballURL string
}

func NewInstaller(nodeModulesPath string) *Installer {
	return &Installer{nodeModulesPath: nodeModulesPath, debug: false}
}

func NewInstallerWithDebug(nodeModulesPath string, debug bool) *Installer {
	return &Installer{nodeModulesPath: nodeModulesPath, debug: debug}
}

func (i *Installer) InstallPackage(name, version string) (string, error) {
	resolvedVersion := version

	installedPath := filepath.Join(i.nodeModulesPath, name)
	if _, err := os.Stat(installedPath); err == nil {
		if iv, _ := readIntegrity(installedPath); iv == version {
			return version, nil
		}
		_ = os.RemoveAll(installedPath)
	}

	if err := os.MkdirAll(i.nodeModulesPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create node_modules: %w", err)
	}

	cachePath := cache.GetCachePath(name, version)

	if !cache.Exists(cachePath) {
		metadata, err := registry.FetchMetadata(name, version)
		if err != nil {
			return "", fmt.Errorf("failed to fetch metadata: %w", err)
		}

		resolvedVersion = metadata.Version
		if i.debug {
			ui.InstallStep("🔗", fmt.Sprintf("Tarball URL: %s", metadata.TarballURL))
			ui.Muted.Printf("   Package: %s@%s (spec: %s)\n", name, resolvedVersion, version)
		}

		cachePath = cache.GetCachePath(name, resolvedVersion)
		if !cache.Exists(cachePath) {
			stream, err := registry.StreamTarball(metadata.TarballURL)
			if err != nil {
				return "", fmt.Errorf("failed to stream tarball: %w", err)
			}
			defer stream.Close()
			h := sha256.New()
			tee := io.TeeReader(stream, h)
			tmpDir, err := os.MkdirTemp("", "npgo-extract-*")
			if err != nil {
				return "", err
			}
			tmpPkg := filepath.Join(tmpDir, "package")
			if err := extractor.ExtractFromReader(tee, tmpPkg); err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
			hash := hex.EncodeToString(h.Sum(nil))
			if i.debug {
				ui.InstallStep("🔐", fmt.Sprintf("SHA256: %s", hash))
			}
			casPath, err := cas.EnsureDirs(hash)
			if err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
			exists, _ := cas.Exists(hash)
			if !exists {
				if err := os.MkdirAll(filepath.Dir(casPath), 0755); err != nil {
					os.RemoveAll(tmpDir)
					return "", err
				}
				if err := os.Rename(tmpPkg, casPath); err != nil {
					if err := createTreeLinkOrCopy(tmpPkg, casPath); err != nil {
						os.RemoveAll(tmpDir)
						return "", err
					}
				}
			}
			os.RemoveAll(tmpDir)
			_, _ = cas.EnsureExtractedCache(hash)
			extractPath := cache.GetExtractPath(name, metadata.Version)
			if err := linkDirPreferSymlink(casPath, extractPath); err != nil {
				return "", err
			}
			if i.debug {
				ui.InstallStep("🔗", fmt.Sprintf("Linked extract → %s", extractPath))
				files, dirs, samples := summarizeDir(extractPath, 15)
				ui.Muted.Printf("   node cache view → Files: %d, Dirs: %d\n", files, dirs)
				for _, s := range samples {
					ui.Muted.Printf("     - %s\n", s)
				}
			}
		}
	}

	extractPath := cache.GetExtractPath(name, resolvedVersion)

	if err := i.createSymlink(name, extractPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}
	if i.debug {
		ui.InstallStep("🔗", fmt.Sprintf("node_modules/%s → %s", name, extractPath))
	}

	if err := ensureGlobalPackageLink(name, extractPath); err != nil {
	}

	_ = i.linkPackageBinaries(name, extractPath)
	if i.debug {
		binDir := filepath.Join(i.nodeModulesPath, ".bin")
		files, _, samples := summarizeDir(binDir, 10)
		if files > 0 {
			ui.InstallStep("⚙️", fmt.Sprintf("Created shims in %s", binDir))
			for _, s := range samples {
				ui.Muted.Printf("     - %s\n", s)
			}
		}
	}

	_ = writeIntegrity(installedPath, name, resolvedVersion, "")

	return resolvedVersion, nil
}

func summarizeDir(dir string, maxSamples int) (int, int, []string) {
	var files, dirs int
	samples := make([]string, 0, maxSamples)
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if p == dir {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if info.IsDir() {
			dirs++
		} else {
			files++
		}
		if len(samples) < maxSamples {
			samples = append(samples, rel)
		}
		return nil
	})
	return files, dirs, samples
}

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
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		target = resolved
	}
	if _, err := os.Lstat(link); err == nil {
		_ = os.RemoveAll(link)
	}
	if runtime.GOOS == "windows" {
		if err := createJunctionWindows(link, target); err == nil {
			return nil
		}
		return createTreeLinkOrCopy(target, link)
	}
	if err := os.Symlink(target, link); err != nil {
		return createTreeLinkOrCopy(target, link)
	}
	return nil
}

func (i *Installer) linkPackageBinaries(pkgName, extractPath string) error {
	pkgJSON := filepath.Join(extractPath, "package.json")
	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		return nil
	}
	var raw struct {
		Bin any `json:"bin"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	binDir := filepath.Join(i.nodeModulesPath, ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	switch v := raw.Bin.(type) {
	case string:
		if v == "" {
			return nil
		}
		return createBinShim(binDir, pkgName, v)
	case map[string]any:
		for name, p := range v {
			rel, _ := p.(string)
			if rel == "" {
				continue
			}
			if err := createBinShim(binDir, pkgName, rel); err != nil {
				return err
			}
			if name != pkgName {
				if err := createBinShimNamed(binDir, name, pkgName, rel); err != nil {
					return err
				}
			}
		}
	default:
		return nil
	}
	return nil
}

func createBinShim(binDir, pkgName, relPath string) error {
	return createBinShimNamed(binDir, pkgName, pkgName, relPath)
}

func createBinShimNamed(binDir, binName, pkgName, relPath string) error {
	targetRel := filepath.Join("..", pkgName, filepath.FromSlash(relPath))
	linkPath := filepath.Join(binDir, binName)
	_ = os.RemoveAll(linkPath)
	_ = os.RemoveAll(linkPath + ".cmd")
	if runtime.GOOS == "windows" {
		content := "@ECHO OFF\r\n" + "node \"%~dp0\\" + filepath.ToSlash(targetRel) + "\" %*\r\n"
		return os.WriteFile(linkPath+".cmd", []byte(content), 0644)
	}
	return os.Symlink(targetRel, linkPath)
}

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

func (i *Installer) createSymlink(name, targetPath string) error {
	linkPath := filepath.Join(i.nodeModulesPath, name)

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("target path does not exist: %s", targetPath)
	}

	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.RemoveAll(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing link: %w", err)
		}
	}

	relPath, err := filepath.Rel(i.nodeModulesPath, targetPath)
	if err == nil {
		if err := os.Symlink(relPath, linkPath); err == nil {
			return nil
		} else {
			if runtime.GOOS == "windows" {
				if mkErr := createJunctionWindows(linkPath, targetPath); mkErr == nil {
					return nil
				}
				return copyDir(targetPath, linkPath)
			}
			if absErr := os.Symlink(targetPath, linkPath); absErr == nil {
				return nil
			}
			return copyDir(targetPath, linkPath)
		}
	}

	if err := os.Symlink(targetPath, linkPath); err == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		if mkErr := createJunctionWindows(linkPath, targetPath); mkErr == nil {
			return nil
		}
	}

	return copyDir(targetPath, linkPath)
}

func createJunctionWindows(linkPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, targetPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink failed: %v: %s", err, string(out))
	}
	return nil
}

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
		if err := linkFile(sPath, dPath); err != nil {
			if err := copyFile(sPath, dPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func linkFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
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

func linkDirPreferSymlink(src, dst string) error {
	_ = os.RemoveAll(dst)
	if runtime.GOOS == "windows" {
		if err := createJunctionWindows(dst, src); err == nil {
			return nil
		}
		return copyDir(src, dst)
	}
	if err := os.Symlink(src, dst); err == nil {
		return nil
	}
	return copyDir(src, dst)
}

func (i *Installer) InstallAll(packages map[string]string) error {
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

func (i *Installer) Clean() error {
	if err := os.RemoveAll(i.nodeModulesPath); err != nil {
		return fmt.Errorf("failed to clean node_modules: %w", err)
	}
	return nil
}

// InstallPipeline installs packages using two-stage pipeline: download/extract → link
func (i *Installer) InstallPipeline(pkgs []PackageSpec, downloadWorkers, linkWorkers int) error {
	if downloadWorkers <= 0 {
		downloadWorkers = 8
	}
	if linkWorkers <= 0 {
		linkWorkers = 8
	}

	type linkItem struct{ name, version, casPath string }
	dlJobs := make(chan PackageSpec, len(pkgs))
	linkJobs := make(chan linkItem, len(pkgs))
	errs := make(chan error, len(pkgs))

	var wgDL sync.WaitGroup
	var wgLink sync.WaitGroup

	// stage 1: download+extract to CAS
	dlWorker := func() {
		defer wgDL.Done()
		for p := range dlJobs {
			// Ensure CAS path via single-pass pipeline
			// Fast path: if CAS already has content, skip
			// Hash requires downloading; we attempt metadata tarball stream
			stream, err := registry.StreamTarball(p.TarballURL)
			if err != nil {
				errs <- fmt.Errorf("failed to stream %s: %w", p.Name, err)
				continue
			}
			h := sha256.New()
			tee := io.TeeReader(stream, h)
			tmpDir, err := os.MkdirTemp("", "npgo-extract-*")
			if err != nil {
				stream.Close()
				errs <- err
				continue
			}
			tmpPkg := filepath.Join(tmpDir, "package")
			if err := extractor.ExtractFromReader(tee, tmpPkg); err != nil {
				stream.Close()
				os.RemoveAll(tmpDir)
				errs <- err
				continue
			}
			stream.Close()
			hash := hex.EncodeToString(h.Sum(nil))
			casPath, err := cas.EnsureDirs(hash)
			if err != nil {
				os.RemoveAll(tmpDir)
				errs <- err
				continue
			}
			exists, _ := cas.Exists(hash)
			if !exists {
				if err := os.MkdirAll(filepath.Dir(casPath), 0755); err != nil {
					os.RemoveAll(tmpDir)
					errs <- err
					continue
				}
				if err := os.Rename(tmpPkg, casPath); err != nil {
					if err := createTreeLinkOrCopy(tmpPkg, casPath); err != nil {
						os.RemoveAll(tmpDir)
						errs <- err
						continue
					}
				}
			}
			os.RemoveAll(tmpDir)
			_, _ = cas.EnsureExtractedCache(hash)
			linkJobs <- linkItem{name: p.Name, version: p.Version, casPath: casPath}
		}
	}

	// stage 2: link to project
	linkWorker := func() {
		defer wgLink.Done()
		for it := range linkJobs {
			extractPath := cache.GetExtractPath(it.name, it.version)
			if err := linkDirPreferSymlink(it.casPath, extractPath); err != nil {
				errs <- err
				continue
			}
			_ = ensureGlobalPackageLink(it.name, extractPath)
			_ = i.linkPackageBinaries(it.name, extractPath)
			_ = writeIntegrity(filepath.Join(i.nodeModulesPath, it.name), it.name, it.version, "")
			// symlink node_modules/<name> → extractPath
			if err := i.createSymlink(it.name, extractPath); err != nil {
				errs <- err
				continue
			}
		}
	}

	for w := 0; w < downloadWorkers; w++ {
		wgDL.Add(1)
		go dlWorker()
	}
	for w := 0; w < linkWorkers; w++ {
		wgLink.Add(1)
		go linkWorker()
	}

	for _, p := range pkgs {
		dlJobs <- p
	}
	close(dlJobs)
	wgDL.Wait()
	close(linkJobs)
	wgLink.Wait()

	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
