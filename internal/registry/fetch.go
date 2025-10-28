package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// HTTPClient is a shared HTTP client with keep-alive pooling
var HTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 0,
}

// PackageMetadata represents the metadata structure from npm registry
type PackageMetadata struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	TarballURL string `json:"dist.tarball"`
	Dist       struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
	Dependencies map[string]string `json:"dependencies"`
}

// RegistryResponse represents the response from npm registry
type RegistryResponse struct {
	Name     string                 `json:"name"`
	Versions map[string]interface{} `json:"versions"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
}

// FetchMetadata fetches package metadata from npm registry
func FetchMetadata(pkgName, version string) (*PackageMetadata, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkgName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var registryResp RegistryResponse
	if err := json.Unmarshal(body, &registryResp); err != nil {
		return nil, fmt.Errorf("failed to parse registry response: %w", err)
	}

	// Determine actual version to fetch
	targetVersion := version
	if version == "latest" {
		targetVersion = registryResp.DistTags.Latest
	}

	// Extract version-specific metadata
	versionData, exists := registryResp.Versions[targetVersion]
	if !exists {
		return nil, fmt.Errorf("version %s not found for package %s", targetVersion, pkgName)
	}

	// Convert to JSON and parse as PackageMetadata
	versionJSON, err := json.Marshal(versionData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version data: %w", err)
	}

	var metadata PackageMetadata
	if err := json.Unmarshal(versionJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse version metadata: %w", err)
	}

	// Ensure we have the tarball URL
	if metadata.TarballURL == "" {
		metadata.TarballURL = metadata.Dist.Tarball
	}

	if metadata.TarballURL == "" {
		return nil, fmt.Errorf("no tarball URL found for package %s@%s", pkgName, targetVersion)
	}

	return &metadata, nil
}

// DownloadTarball downloads the package tarball to cache directory
func DownloadTarball(tarballURL, pkgName, version string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, tarballURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create cache directory if it doesn't exist
	cacheDir := getCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create filename
	filename := fmt.Sprintf("%s-%s.tgz", pkgName, version)
	filepath := filepath.Join(cacheDir, filename)

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	// Copy response body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write tarball to file: %w", err)
	}

	return filepath, nil
}

// StreamTarball returns a streaming reader for the tarball
func StreamTarball(tarballURL string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, tarballURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download tarball: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".npgo/cache"
	}
	return filepath.Join(homeDir, ".npgo", "cache")
}
