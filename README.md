# npgo - Fast Node Package Manager

npgo là một trình quản lý package Node.js nhanh được viết bằng Go, tập trung vào tốc độ và hiệu quả.

## 🚀 Giai đoạn 1 - Fetch Engine

Hiện tại npgo đã hoàn thành giai đoạn đầu tiên với khả năng fetch và cache packages từ npm registry.

### Tính năng hiện tại

- ✅ Fetch package từ npm registry
- ✅ Cache tarball files cục bộ
- ✅ Giải nén package content
- ✅ CLI interface với Cobra
- ✅ Quản lý cache thông minh

### Cài đặt và Build

```bash
# Clone repository
git clone https://github.com/nghiaomg/npgo.git
cd npgo

# Download dependencies
go mod tidy

# Build binary
go build -o npgo cmd/npgo/main.go
```

### Sử dụng

```bash
# Fetch một package cụ thể
./npgo fetch express@4.18.2

# Fetch latest version
./npgo fetch express

# Fetch với version tag
./npgo fetch express@latest
```

### Cấu trúc Cache

npgo sẽ tạo cache tại `~/.npgo/`:

```
~/.npgo/
├── cache/           # Tarball files (.tgz)
│   └── express-4.18.2.tgz
└── extracted/       # Extracted package content
    └── express-4.18.2/
        ├── package.json
        ├── lib/
        └── ...
```

### Workflow Fetch

Khi chạy `npgo fetch express@4.18.2`:

1. **Parse package specification** - Tách tên package và version
2. **Check cache** - Kiểm tra xem đã có trong cache chưa
3. **Fetch metadata** - Lấy thông tin từ npm registry
4. **Download tarball** - Tải file .tgz về cache
5. **Extract package** - Giải nén vào thư mục extracted

### Cấu trúc Dự án

```
npgo/
├── cmd/
│   └── npgo/
│       └── main.go          # CLI entry point
├── internal/
│   ├── registry/
│   │   └── fetch.go         # npm registry integration
│   ├── cache/
│   │   └── cache.go         # cache management
│   ├── extractor/
│   │   └── extract.go       # tarball extraction
│   └── utils/
│       └── file.go          # file utilities
├── diagrams/
│   ├── architecture.md      # system architecture overview
│   ├── fetch_flow.md        # fetch command flow
│   ├── install_sequence.md  # install sequence diagram
│   └── caching_strategy.md  # caching strategy details
├── scripts/
│   └── export-diagrams.sh   # export diagrams to PNG
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

## 📊 Architecture Diagrams

Dự án bao gồm các diagram chi tiết để hiểu rõ kiến trúc hệ thống:

- **[Architecture Overview](diagrams/architecture.md)** - Tổng quan kiến trúc hệ thống
- **[Fetch Flow](diagrams/fetch_flow.md)** - Luồng thực thi lệnh `npgo fetch`
- **[Install Sequence](diagrams/install_sequence.md)** - Sequence diagram cho `npgo install` (tương lai)
- **[Caching Strategy](diagrams/caching_strategy.md)** - Chiến lược cache chi tiết

### Export Diagrams

Để export diagrams từ Mermaid sang PNG:

```bash
# Cài đặt mermaid-cli
npm install -g @mermaid-js/mermaid-cli

# Export diagrams
chmod +x scripts/export-diagrams.sh
./scripts/export-diagrams.sh
```

## 🔜 Roadmap

### Giai đoạn tiếp theo
- [ ] `npgo install` - Link packages vào node_modules
- [ ] Package.json support
- [ ] Dependency resolution
- [ ] npm-compatible commands

### Mục tiêu dài hạn
- [ ] Parallel downloads với goroutines
- [ ] Advanced caching với TTL
- [ ] Workspace support
- [ ] Performance optimizations

## Development

```bash
# Run tests
go test ./...

# Format code
go fmt ./...

# Lint code
golangci-lint run
```

## License

MIT License
