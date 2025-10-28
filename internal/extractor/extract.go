package extractor

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTarGz extracts a .tgz file to the destination directory
func ExtractTarGz(src, dest string) error {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Ensure destination directory exists
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip if it's not a regular file
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Clean the path and remove leading directory components
		// npm packages typically have a package/ prefix
		cleanPath := cleanTarPath(header.Name)
		if cleanPath == "" {
			continue // Skip empty paths
		}

		// Create full destination path
		fullPath := filepath.Join(dest, cleanPath)

		// Ensure parent directory exists
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Create the file
		destFile, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}

		// Copy file contents
		_, err = io.Copy(destFile, tarReader)
		destFile.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file contents: %w", err)
		}

		// Set file permissions
		if err := os.Chmod(fullPath, os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	return nil
}

// cleanTarPath removes the package prefix from tar paths
// npm packages typically have structure like: package/package.json, package/lib/index.js
func cleanTarPath(path string) string {
	// Split path into components
	parts := strings.Split(path, "/")

	// Find the package directory (usually the first non-empty component)
	var packageIndex = -1
	for i, part := range parts {
		if part != "" && part != "." {
			packageIndex = i
			break
		}
	}

	if packageIndex == -1 {
		return ""
	}

	// Skip the package directory and return the rest
	if packageIndex+1 < len(parts) {
		return strings.Join(parts[packageIndex+1:], "/")
	}

	return ""
}
