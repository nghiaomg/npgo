package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type cacheMeta struct {
	ETag         string    `json:"etag"`
	LastModified string    `json:"lastModified"`
	CachedAt     time.Time `json:"cachedAt"`
}

func registryCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".npgo", "registry-cache")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

var httpSem = make(chan struct{}, 64)

func getRegistryResponseCached(pkgName string) (*RegistryResponse, error) {
	dir, err := registryCacheDir()
	if err != nil {
		return nil, err
	}
	dataPath := filepath.Join(dir, pkgName+".json")
	metaPath := filepath.Join(dir, pkgName+".meta.json")
	if err := os.MkdirAll(filepath.Dir(dataPath), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(metaPath), 0755); err != nil {
		return nil, err
	}

	var meta cacheMeta
	if b, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(b, &meta)
	}

	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkgName)
	// bound request with timeout to avoid goroutine leaks
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if meta.ETag != "" {
		req.Header.Set("If-None-Match", meta.ETag)
	}
	if meta.LastModified != "" {
		req.Header.Set("If-Modified-Since", meta.LastModified)
	}

	httpSem <- struct{}{}
	resp, err := HTTPClient.Do(req)
	<-httpSem
	if err != nil {
		if b, err2 := os.ReadFile(dataPath); err2 == nil {
			var rr RegistryResponse
			if jsonErr := json.Unmarshal(b, &rr); jsonErr == nil {
				return &rr, nil
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		b, err := os.ReadFile(dataPath)
		if err != nil {
			return nil, fmt.Errorf("cache miss after 304: %w", err)
		}
		var rr RegistryResponse
		if err := json.Unmarshal(b, &rr); err != nil {
			return nil, err
		}
		return &rr, nil
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(dataPath, body, 0644); err != nil {
			return nil, err
		}
		meta.ETag = resp.Header.Get("ETag")
		meta.LastModified = resp.Header.Get("Last-Modified")
		meta.CachedAt = time.Now()
		if mb, err := json.MarshalIndent(meta, "", "  "); err == nil {
			_ = os.WriteFile(metaPath, mb, 0644)
		}
		var rr RegistryResponse
		if err := json.Unmarshal(body, &rr); err != nil {
			return nil, err
		}
		return &rr, nil
	default:
		if b, err2 := os.ReadFile(dataPath); err2 == nil {
			var rr RegistryResponse
			if jsonErr := json.Unmarshal(b, &rr); jsonErr == nil {
				return &rr, nil
			}
		}
		return nil, fmt.Errorf("registry status %d", resp.StatusCode)
	}
}

func PrefetchRegistry(pkgs []string, concurrency int) {
	if concurrency <= 0 {
		concurrency = 64
	}
	if concurrency > 64 {
		concurrency = 64
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, name := range pkgs {
		n := name
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			_, _ = getRegistryResponseCached(n)
		}()
	}
	wg.Wait()
}
