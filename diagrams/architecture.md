# npgo Architecture Overview

## System Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI (Cobra)   │    │  npm Registry   │    │  Local Cache    │
│                 │    │                 │    │                 │
│  npgo fetch     │◄──►│ registry.npmjs  │◄──►│ ~/.npgo/cache/  │
│  npgo install   │    │     .org        │    │ ~/.npgo/extract/│
│  npgo list      │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Registry       │    │   Extractor     │    │   Cache         │
│   Module         │    │   Module         │    │   Module        │
│                 │    │                 │    │                 │
│ • FetchMetadata │    │ • ExtractTarGz  │    │ • GetCachePath  │
│ • DownloadTarball│    │ • CleanTarPath  │    │ • Exists        │
│ • ParseVersions  │    │ • ValidateFiles │    │ • EnsureDirs    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Data Flow

### Fetch Command Flow
1. **Parse Input**: `express@4.18.2` → `{name: "express", version: "4.18.2"}`
2. **Check Cache**: Look for existing `express-4.18.2.tgz`
3. **Fetch Metadata**: GET `https://registry.npmjs.org/express`
4. **Download Tarball**: Download `.tgz` file to cache
5. **Extract Package**: Extract to `~/.npgo/extracted/express-4.18.2/`

### Install Command Flow (Future)
1. **Read package.json**: Parse dependencies
2. **Resolve Dependencies**: Build dependency tree
3. **Fetch Missing**: Download packages not in cache
4. **Link node_modules**: Create symlinks to extracted packages

## File Structure

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

## Module Responsibilities

### Registry Module
- **Purpose**: Interface with npm registry
- **Key Functions**:
  - `FetchMetadata(pkgName, version)` - Get package info
  - `DownloadTarball(url, pkgName, version)` - Download .tgz
  - Handle version resolution (latest, semver ranges)

### Cache Module  
- **Purpose**: Manage local package storage
- **Key Functions**:
  - `GetCachePath(pkgName, version)` - Generate cache paths
  - `Exists(path)` - Check if cached
  - `EnsureDirs()` - Create cache directories

### Extractor Module
- **Purpose**: Handle tarball extraction
- **Key Functions**:
  - `ExtractTarGz(src, dest)` - Extract .tgz files
  - `CleanTarPath(path)` - Remove package/ prefix
  - Validate extracted files

### Utils Module (Future)
- **Purpose**: Common utilities
- **Key Functions**:
  - File operations
  - Path manipulation
  - Error handling helpers
