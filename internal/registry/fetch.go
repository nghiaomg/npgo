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
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		MaxConnsPerHost:       64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	},
	Timeout: 30 * time.Second,
}

type PackageMetadata struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	TarballURL string `json:"dist.tarball"`
	Dist       struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
}

type RegistryResponse struct {
	Name     string                 `json:"name"`
	Versions map[string]interface{} `json:"versions"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
}

func FetchMetadata(pkgName, version string) (*PackageMetadata, error) {
	// Use cached registry document with ETag/Last-Modified support
	registryResp, err := getRegistryResponseCached(pkgName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry data: %w", err)
	}

	targetVersion := version
	if version == "latest" {
		targetVersion = registryResp.DistTags.Latest
	}

	versionData, exists := registryResp.Versions[targetVersion]
	if !exists {
		if resolved := resolveVersionFromMap(registryResp.Versions, targetVersion); resolved != "" {
			targetVersion = resolved
			versionData = registryResp.Versions[targetVersion]
		} else {
			return nil, fmt.Errorf("version %s not found for package %s", targetVersion, pkgName)
		}
	}

	versionJSON, err := json.Marshal(versionData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version data: %w", err)
	}

	var metadata PackageMetadata
	if err := json.Unmarshal(versionJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse version metadata: %w", err)
	}

	if metadata.TarballURL == "" {
		metadata.TarballURL = metadata.Dist.Tarball
	}

	if metadata.TarballURL == "" {
		return nil, fmt.Errorf("no tarball URL found for package %s@%s", pkgName, targetVersion)
	}

	return &metadata, nil
}

func resolveVersionFromMap(versions map[string]interface{}, spec string) string {
	s := spec
	if len(s) > 2 && (s[len(s)-2:] == ".x" || s[len(s)-2:] == ".*") {
		s = s[:len(s)-2]
	}
	parts := make([]int, 0, 3)
	segs := 0
	cur := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			parts = append(parts, cur)
			cur = 0
			segs++
			continue
		}
		if c < '0' || c > '9' {
			return ""
		}
		cur = cur*10 + int(c-'0')
	}
	if len(s) > 0 {
		parts = append(parts, cur)
		segs++
	}
	if segs == 0 || segs > 2 {
		return ""
	}

	best := ""
	bestMaj, bestMin, bestPatch := -1, -1, -1
	for v := range versions {
		maj, min, pat := parseSemver(v)
		if maj < 0 {
			continue
		}
		if segs == 1 {
			if maj != parts[0] {
				continue
			}
		} else if segs == 2 {
			if maj != parts[0] || min != parts[1] {
				continue
			}
		}
		if maj > bestMaj || (maj == bestMaj && (min > bestMin || (min == bestMin && pat > bestPatch))) {
			bestMaj, bestMin, bestPatch = maj, min, pat
			best = v
		}
	}
	return best
}

func parseSemver(v string) (int, int, int) {
	n1, n2, n3 := -1, -1, -1
	cur := 0
	seg := 0
	for i := 0; i <= len(v); i++ {
		if i == len(v) || v[i] == '.' {
			if seg == 0 {
				n1 = cur
			} else if seg == 1 {
				n2 = cur
			} else if seg == 2 {
				n3 = cur
			}
			seg++
			cur = 0
			continue
		}
		c := v[i]
		if c < '0' || c > '9' {
			return -1, -1, -1
		}
		cur = cur*10 + int(c-'0')
	}
	if n1 < 0 {
		return -1, -1, -1
	}
	if n2 < 0 {
		n2 = 0
	}
	if n3 < 0 {
		n3 = 0
	}
	return n1, n2, n3
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

	cacheDir := getCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.tgz", pkgName, version)
	filepath := filepath.Join(cacheDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write tarball to file: %w", err)
	}

	return filepath, nil
}

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
		return ".npgo/cache"
	}
	return filepath.Join(homeDir, ".npgo", "cache")
}
