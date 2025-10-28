# npgo Install Sequence Diagram

## Mermaid Sequence Diagram

```mermaid
sequenceDiagram
    participant User
    participant CLI as npgo CLI
    participant Registry as npm Registry
    participant Cache as Local Cache
    participant Extractor as Package Extractor
    participant FS as File System

    User->>CLI: npgo install
    CLI->>FS: Read package.json
    FS-->>CLI: Return dependencies
    
    CLI->>CLI: Parse dependencies
    CLI->>CLI: Build dependency tree
    
    loop For each dependency
        CLI->>Cache: Check if cached
        Cache-->>CLI: Cache status
        
        alt Not cached
            CLI->>Registry: Fetch metadata
            Registry-->>CLI: Package info
            CLI->>Registry: Download tarball
            Registry-->>CLI: .tgz file
            CLI->>Cache: Save tarball
            CLI->>Extractor: Extract package
            Extractor->>FS: Write extracted files
            FS-->>Extractor: Success
            Extractor-->>CLI: Extraction complete
        else Already cached
            CLI->>Cache: Use cached package
        end
        
        CLI->>FS: Create symlink in node_modules
        FS-->>CLI: Symlink created
    end
    
    CLI-->>User: Installation complete
```

## Future Install Flow (Detailed)

### Phase 1: Dependency Resolution
```
1. Read package.json
2. Parse dependencies and devDependencies
3. Resolve version ranges (^, ~, >=, etc.)
4. Build dependency tree
5. Detect conflicts
```

### Phase 2: Package Fetching
```
For each package in dependency tree:
  1. Check cache existence
  2. If not cached:
     - Fetch metadata from registry
     - Download tarball
     - Extract to cache
  3. If cached:
     - Skip download
     - Use existing extraction
```

### Phase 3: node_modules Linking
```
1. Create node_modules directory
2. For each package:
   - Create symlink: node_modules/package -> ~/.npgo/extracted/package-version
   - Handle nested dependencies
3. Create .bin symlinks for executables
```

## Error Scenarios

### Dependency Resolution Errors
```mermaid
graph TD
    A[Dependency Conflict] --> B[Version Mismatch]
    A --> C[Circular Dependency]
    A --> D[Missing Package]
    
    B --> E[Report conflict to user]
    C --> F[Detect cycle and report]
    D --> G[Suggest alternatives]
```

### Network Errors
```mermaid
graph TD
    A[Network Error] --> B[Registry Unavailable]
    A --> C[Download Timeout]
    A --> D[Invalid Response]
    
    B --> E[Retry with backoff]
    C --> F[Increase timeout]
    D --> G[Report error and exit]
```

## Performance Optimizations

### Parallel Downloads
```
- Use goroutines for concurrent downloads
- Limit concurrent connections (e.g., 10)
- Implement download queue
```

### Smart Caching
```
- Cache validation with checksums
- TTL-based cache invalidation
- Incremental updates
```

### Dependency Tree Optimization
```
- Flatten dependency tree where possible
- Deduplicate common dependencies
- Optimize symlink creation
```

## Lock File Strategy (Future)

### npgo.lock Structure
```json
{
  "lockfileVersion": 1,
  "packages": {
    "express": {
      "version": "4.18.2",
      "resolved": "https://registry.npmjs.org/express/-/express-4.18.2.tgz",
      "integrity": "sha512-...",
      "dependencies": {
        "accepts": "~1.3.8",
        "array-flatten": "1.1.1"
      }
    }
  }
}
```

### Lock File Benefits
- Reproducible builds
- Faster installs
- Dependency integrity
- Version pinning
