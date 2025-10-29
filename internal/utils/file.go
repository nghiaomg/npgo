package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func EnsureDir(path string) error {
	if !FileExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func GetHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return homeDir
}

func JoinPath(components ...string) string {
	return filepath.Join(components...)
}

func SanitizeFilename(filename string) string {
	unsafeChars := []string{"@", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range unsafeChars {
		result = strings.ReplaceAll(result, char, "-")
	}

	// Remove multiple consecutive dashes
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	result = strings.Trim(result, "-")

	return result
}

func FormatPackageFilename(pkgName, version string) string {
	safeName := SanitizeFilename(pkgName)
	return fmt.Sprintf("%s-%s.tgz", safeName, version)
}

func FormatPackageDirname(pkgName, version string) string {
	safeName := SanitizeFilename(pkgName)
	return fmt.Sprintf("%s-%s", safeName, version)
}

func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

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

func CleanPath(path string) string {
	return filepath.Clean(path)
}

func GetRelativePath(base, target string) (string, error) {
	return filepath.Rel(base, target)
}

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
