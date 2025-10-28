package extractor

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	mmap "golang.org/x/exp/mmap"
)

// ExtractTarGz extracts a .tgz file to the destination directory
func ExtractTarGz(src, dest string) error {
	// Try mmap for faster read
	mm, err := mmap.Open(src)
	if err == nil {
		defer mm.Close()
		// mmap.ReaderAt implements ReadAt; wrap into io.Reader via NewSectionReader
		reader := io.NewSectionReader(mm, 0, int64(mm.Len()))
		// Attempt gzip reader
		if gz, gzErr := gzip.NewReader(reader); gzErr == nil {
			defer gz.Close()
			tr := tar.NewReader(gz)
			if err := extractTarReader(tr, dest); err != nil {
				return err
			}
			return nil
		}
		// Fallback: plain tar
		tr := tar.NewReader(reader)
		if err := extractTarReader(tr, dest); err != nil {
			return err
		}
		return nil
	}

	// Fallback to regular file IO
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
	return extractTarReader(tarReader, dest)
}

// ExtractFromReader streams extract from an io.Reader (HTTP body)
func ExtractFromReader(r io.Reader, dest string) error {
	// Support gzip if Content-Encoding is gzip or data is gzip
	var tr *tar.Reader

	// Try to detect gzip by peeking via http or assume gzip
	// We will attempt gzip first; if fails, treat as plain tar
	if gz, err := gzip.NewReader(noCloseReader{r}); err == nil {
		defer gz.Close()
		tr = tar.NewReader(gz)
	} else {
		tr = tar.NewReader(r)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return extractTarReader(tr, dest)
}

// noCloseReader prevents closing the underlying reader (HTTP Body handled elsewhere)
type noCloseReader struct{ io.Reader }

func (noCloseReader) Close() error { return nil }

// extractTarReader extracts files from a tar reader to dest
func extractTarReader(tr *tar.Reader, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			cleanPath := cleanTarPath(header.Name)
			if cleanPath == "" {
				continue
			}
			fullPath := filepath.Join(dest, cleanPath)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir: %w", err)
			}
		case tar.TypeReg:
			cleanPath := cleanTarPath(header.Name)
			if cleanPath == "" {
				continue
			}
			fullPath := filepath.Join(dest, cleanPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir: %w", err)
			}
			f, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to copy file: %w", err)
			}
			f.Close()
			if err := os.Chmod(fullPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to chmod: %w", err)
			}
		default:
			// skip
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
