package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Paths
func baseStoreDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".npgo", "store", "v3"), nil
}

// HashBytes computes SHA-256 of data
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashStream computes SHA-256 of a stream
func HashStream(r io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// PackagePath returns the path to the package directory in CAS
func PackagePath(hash string) (string, error) {
	root, err := baseStoreDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, hash, "package"), nil
}

// Exists checks if a stored package exists
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

// EnsureDirs creates base CAS directories
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
