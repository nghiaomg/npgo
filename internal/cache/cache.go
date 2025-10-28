package cache

import (
	"os"
	"path/filepath"
	"strings"
)

// GetCachePath returns the cache file path for a package
func GetCachePath(pkgName, version string) string {
	cacheDir := getCacheDir()
	filename := formatFilename(pkgName, version)
	return filepath.Join(cacheDir, filename)
}

// GetExtractPath returns the extraction directory path for a package
func GetExtractPath(pkgName, version string) string {
	extractDir := getExtractDir()
	dirname := formatDirname(pkgName, version)
	return filepath.Join(extractDir, dirname)
}

// Exists checks if a cached file exists
func Exists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// EnsureCacheDir ensures the cache directory exists
func EnsureCacheDir() error {
	cacheDir := getCacheDir()
	return os.MkdirAll(cacheDir, 0755)
}

// EnsureExtractDir ensures the extract directory exists
func EnsureExtractDir() error {
	extractDir := getExtractDir()
	return os.MkdirAll(extractDir, 0755)
}

func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".npgo/cache"
	}
	return filepath.Join(homeDir, ".npgo", "cache")
}

func getExtractDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".npgo/extracted"
	}
	return filepath.Join(homeDir, ".npgo", "extracted")
}

func formatFilename(pkgName, version string) string {
	// Replace @ and / with - for safe filenames
	safeName := replaceUnsafeChars(pkgName)
	return safeName + "-" + version + ".tgz"
}

func formatDirname(pkgName, version string) string {
	// Replace @ and / with - for safe directory names
	safeName := replaceUnsafeChars(pkgName)
	return safeName + "-" + version
}

func replaceUnsafeChars(name string) string {
	// Replace @ and / with - for safe filesystem names
	result := name
	for _, char := range []string{"@", "/", "\\"} {
		result = strings.ReplaceAll(result, char, "-")
	}
	return result
}
