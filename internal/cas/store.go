package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func baseStoreDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".npgo", "store", "v3"), nil
}

func extractedCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".npgo", "extracted-cache"), nil
}

func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashStream(r io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func PackagePath(hash string) (string, error) {
	root, err := baseStoreDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, hash, "package"), nil
}

func ExtractedCachePath(hash string) (string, error) {
	root, err := extractedCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, hash), nil
}

func Exists(hash string) (bool, error) {
	p, err := PackagePath(hash)
	if err != nil {
		return false, err
	}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return true, nil
	}
	return false, nil
}

func EnsureExtractedCache(hash string) (string, error) {
	casPath, err := PackagePath(hash)
	if err != nil {
		return "", err
	}
	out, err := ExtractedCachePath(hash)
	if err != nil {
		return "", err
	}
	if fi, err := os.Lstat(out); err == nil {
		_ = fi
		return out, nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return "", err
	}
	if err := os.Symlink(casPath, out); err == nil {
		return out, nil
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return "", err
	}
	return out, nil
}

func EnsureDirs(hash string) (string, error) {
	p, err := PackagePath(hash)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return "", fmt.Errorf("failed to create CAS dir: %w", err)
	}
	return p, nil
}
