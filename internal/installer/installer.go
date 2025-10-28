package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"npgo/internal/cache"
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

	// Check if already installed
	installedPath := filepath.Join(i.nodeModulesPath, name)
	if _, err := os.Stat(installedPath); err == nil {
		return "", fmt.Errorf("package %s is already installed", name)
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
			// Download tarball
			tarballPath, err := registry.DownloadTarball(metadata.TarballURL, name, metadata.Version)
			if err != nil {
				return "", fmt.Errorf("failed to download tarball: %w", err)
			}

			// Extract
			extractPath := cache.GetExtractPath(name, metadata.Version)
			if err := extractor.ExtractTarGz(tarballPath, extractPath); err != nil {
				return "", fmt.Errorf("failed to extract: %w", err)
			}
		}
	}

	// Get extract path with resolved version
	extractPath := cache.GetExtractPath(name, resolvedVersion)

	// Create symlink
	if err := i.createSymlink(name, extractPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	return resolvedVersion, nil
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

// copyDir recursively copies a directory tree from src to dst
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
		if err := copyFile(sPath, dPath); err != nil {
			return err
		}
	}
	return nil
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
	for name, version := range packages {
		if _, err := i.InstallPackage(name, version); err != nil {
			return fmt.Errorf("failed to install %s: %w", name, err)
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
