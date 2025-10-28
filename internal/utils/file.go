package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	if !FileExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GetHomeDir returns the user's home directory
func GetHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "."
	}
	return homeDir
}

// JoinPath safely joins path components
func JoinPath(components ...string) string {
	return filepath.Join(components...)
}

// SanitizeFilename removes unsafe characters from filename
func SanitizeFilename(filename string) string {
	// Replace unsafe characters with dashes
	unsafeChars := []string{"@", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	
	for _, char := range unsafeChars {
		result = strings.ReplaceAll(result, char, "-")
	}
	
	// Remove multiple consecutive dashes
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	
	// Remove leading/trailing dashes
	result = strings.Trim(result, "-")
	
	return result
}

// FormatPackageFilename creates a safe filename for package tarball
func FormatPackageFilename(pkgName, version string) string {
	safeName := SanitizeFilename(pkgName)
	return fmt.Sprintf("%s-%s.tgz", safeName, version)
}

// FormatPackageDirname creates a safe directory name for extracted package
func FormatPackageDirname(pkgName, version string) string {
	safeName := SanitizeFilename(pkgName)
	return fmt.Sprintf("%s-%s", safeName, version)
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// IsDir checks if the path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if the path is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetDirSize calculates the total size of a directory
func GetDirSize(path string) (int64, error) {
	var size int64
	
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// CleanPath removes extra path separators and resolves relative paths
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// GetRelativePath returns the relative path from base to target
func GetRelativePath(base, target string) (string, error) {
	return filepath.Rel(base, target)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
