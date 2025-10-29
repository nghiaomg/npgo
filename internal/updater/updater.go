package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"npgo/internal/registry"
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		Name               string `json:"name"`
	} `json:"assets"`
}

func fetchLatestRelease() (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/nghiaomg/npgo/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	resp, err := registry.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func CheckUpdate(currentVersion string) (latest string, hasNew bool, err error) {
	rel, err := fetchLatestRelease()
	if err != nil {
		return "", false, err
	}
	if rel.TagName == "" {
		return "", false, fmt.Errorf("no release info")
	}
	if rel.TagName == currentVersion {
		return rel.TagName, false, nil
	}
	return rel.TagName, true, nil
}

func DownloadLatest(destDir string) (string, string, error) {
	rel, err := fetchLatestRelease()
	if err != nil {
		return "", "", err
	}
	var url, name string
	for _, a := range rel.Assets {
		switch runtime.GOOS {
		case "windows":
			if a.Name == "npgo.exe" {
				url = a.BrowserDownloadURL
				name = a.Name
			}
		default:
			if a.Name == "npgo" {
				url = a.BrowserDownloadURL
				name = a.Name
			}
		}
		if url != "" {
			break
		}
	}
	if url == "" {
		return "", "", fmt.Errorf("no matching binary asset")
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", "", err
	}
	outPath := filepath.Join(destDir, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := registry.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return "", "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return "", "", err
	}
	f.Close()
	return outPath, rel.TagName, nil
}
