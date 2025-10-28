# npgo Caching Strategy

## Cache Architecture

### Directory Structure
```
~/.npgo/
├── cache/                    # Tarball storage
│   ├── express-4.18.2.tgz
│   ├── lodash-4.17.21.tgz
│   └── ...
├── extracted/               # Extracted packages
│   ├── express-4.18.2/
│   │   ├── package.json
│   │   ├── lib/
│   │   └── ...
│   └── lodash-4.17.21/
│       ├── package.json
│       ├── lodash.js
│       └── ...
└── lock/                    # Lock files (future)
    └── npgo.lock
```

## Caching Levels

### Level 1: Tarball Cache
- **Purpose**: Store downloaded .tgz files
- **Location**: `~/.npgo/cache/`
- **Naming**: `{package-name}-{version}.tgz`
- **Benefits**: Avoid re-downloading packages

### Level 2: Extracted Cache
- **Purpose**: Store extracted package contents
- **Location**: `~/.npgo/extracted/`
- **Naming**: `{package-name}-{version}/`
- **Benefits**: Fast symlink creation for node_modules

### Level 3: Metadata Cache (Future)
- **Purpose**: Cache package metadata
- **Location**: `~/.npgo/metadata/`
- **Format**: JSON files
- **Benefits**: Faster dependency resolution

## Cache Management

### Cache Validation

#### File Existence Check
```go
func Exists(filePath string) bool {
    _, err := os.Stat(filePath)
    return !os.IsNotExist(err)
}
```

#### Integrity Validation (Future)
```go
func ValidateCache(pkgName, version string) bool {
    // Check file size
    // Verify checksum
    // Validate extraction
    return true
}
```

### Cache Cleanup

#### TTL-based Cleanup
```go
type CacheEntry struct {
    Package   string
    Version   string
    Timestamp time.Time
    TTL       time.Duration
}

func CleanExpiredCache() error {
    // Remove entries older than TTL
    // Default TTL: 30 days
}
```

#### Size-based Cleanup
```go
func CleanOldCache(maxSize int64) error {
    // Remove oldest entries when cache exceeds maxSize
    // Default maxSize: 1GB
}
```

## Cache Operations

### Write Operations
```go
// Download and cache tarball
func CacheTarball(url, pkgName, version string) error {
    // 1. Download from URL
    // 2. Save to cache directory
    // 3. Update cache index
}

// Extract and cache package
func CacheExtracted(tarballPath, pkgName, version string) error {
    // 1. Extract tarball
    // 2. Save to extracted directory
    // 3. Update extraction index
}
```

### Read Operations
```go
// Check if package is cached
func IsCached(pkgName, version string) bool {
    cachePath := GetCachePath(pkgName, version)
    return Exists(cachePath)
}

// Get cached package path
func GetCachedPath(pkgName, version string) string {
    return GetExtractPath(pkgName, version)
}
```

## Performance Optimizations

### Concurrent Access
```go
type CacheManager struct {
    mutex sync.RWMutex
    index map[string]CacheEntry
}

func (cm *CacheManager) Get(key string) (CacheEntry, bool) {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    entry, exists := cm.index[key]
    return entry, exists
}
```

### Batch Operations
```go
// Cache multiple packages concurrently
func CachePackages(packages []PackageSpec) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(packages))
    
    for _, pkg := range packages {
        wg.Add(1)
        go func(p PackageSpec) {
            defer wg.Done()
            if err := CachePackage(p.Name, p.Version); err != nil {
                errChan <- err
            }
        }(pkg)
    }
    
    wg.Wait()
    close(errChan)
    
    // Handle errors
    return nil
}
```

## Cache Configuration

### Environment Variables
```bash
# Cache directory
export NPGO_CACHE_DIR="/custom/cache/path"

# Cache TTL (days)
export NPGO_CACHE_TTL="30"

# Max cache size (bytes)
export NPGO_MAX_CACHE_SIZE="1073741824"  # 1GB
```

### Configuration File
```json
{
  "cache": {
    "dir": "~/.npgo",
    "ttl": "30d",
    "maxSize": "1GB",
    "cleanupInterval": "24h"
  }
}
```

## Cache Statistics

### Metrics Collection
```go
type CacheStats struct {
    TotalPackages   int
    TotalSize       int64
    HitRate         float64
    MissRate        float64
    LastCleanup     time.Time
}

func GetCacheStats() CacheStats {
    // Calculate cache statistics
    return CacheStats{}
}
```

### Cache Health Check
```go
func HealthCheck() error {
    // Check cache directory permissions
    // Verify cache integrity
    // Check available disk space
    return nil
}
```

## Future Enhancements

### Incremental Updates
- Only download changed files
- Delta compression for updates
- Smart diff algorithms

### Distributed Caching
- Share cache across team
- CDN integration
- Peer-to-peer sharing

### Advanced Compression
- Brotli compression for tarballs
- Deduplication across packages
- Compression ratio optimization
