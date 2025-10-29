## Báo cáo cài đặt NPGO – Cách `npgo install` hoạt động hiện tại

### 1) Tổng quan

- Mục tiêu: Cài đặt toàn bộ dependencies (và devDependencies khi `--dev`) từ `package.json` vào `node_modules/`, với tốc độ cao, ít I/O và có thể lặp lại (idempotent).
- Thành phần chính:
  - CLI (Cobra): `npgo install | npgo i [--dev]`
  - Resolver: xây đồ thị phụ thuộc (DAG-ish), chuẩn hoá version, tải metadata
  - Registry: lấy metadata và tarball từ npm (có cache ETag/Last-Modified)
  - CAS Store: lưu trữ/extract theo hash tarball để tái sử dụng
  - Extractor: giải nén (hỗ trợ stream + mmap), chuẩn hoá đường dẫn
  - Installer: link/copy vào `node_modules`, tạo `.bin` shims, integrity
  - UI: spinner/progress/log debug

---

### 2) Luồng chi tiết cài đặt

1. Khởi động lệnh
   - `npgo i` hoặc `npgo install`.
   - Nếu có `--dev`, bật log chi tiết (điểm debug) và cài cả devDependencies.

2. Đọc `package.json`
   - Parse `dependencies` (+ `devDependencies` khi `--dev`).
   - Hiển thị tổng số dependency phát hiện.

3. Resolve phụ thuộc (song song + cache metadata)
   - Chuẩn hoá version spec: bỏ `^`, `~`, các toán tử so sánh, giữ lại `latest` hoặc cụm đơn giản như `1`, `1.2`.
   - Truy cập `https://registry.npmjs.org/<pkg>` để lấy metadata toàn gói (toàn bộ versions) qua lớp Registry Cache:
     - Lưu JSON vào `~/.npgo/registry-cache/<pkg>.json` + meta ETag/Last-Modified.
     - Lần sau gửi `If-None-Match`/`If-Modified-Since` → nếu `304 Not Modified` thì đọc từ cache local (thời gian gần như 0ms).
   - Dò phiên bản phù hợp (bao gồm xử lý nhanh các spec đơn giản như `1`, `1.*`, `1.2`).
   - Lưu `Dependencies` của phiên bản đã resolve làm `RawDeps` để dùng ngay, tránh gọi lại registry lần 2.
   - Duyệt cây phụ thuộc bằng `BuildGraph` với semaphore (mặc định 32 concurrent) để fetch metadata song song.
   - Sắp xếp topo bằng `TopoOrder`. Nếu phát hiện chu trình, không dừng – nối phần còn lại theo thứ tự ổn định để tiếp tục cài đặt.

4. Tải và extract tarball theo CAS
   - Với mỗi package (theo topo):
     - Streaming tải tarball (không lưu `.tgz` trung gian), tính SHA256 khi stream.
     - Dùng SHA256 làm khoá CAS: `~/.npgo/store/v3/<sha256>/package/`.
     - Nếu mục CAS chưa có hoặc rỗng, giải nén trực tiếp vào CAS.
     - Liên kết (symlink/junction/hardlink/copy) từ CAS sang cache extracted: `~/.npgo/extracted/<name-version>/`.
     - Ghi thống kê (khi `--dev`): số file/dir, mẫu tên file.

5. Link vào `node_modules/` và tạo `.bin` shims
   - Tạo liên kết `node_modules/<pkg>` → `~/.npgo/extracted/<name-version>`.
   - Trên Windows: ưu tiên junction, fallback hardlink/copy khi thiếu quyền symlink.
   - Đọc `package.json` của package để tạo shims trong `node_modules/.bin` (POSIX: symlink; Windows: `.cmd`).
   - Ghi per-package integrity: `node_modules/<pkg>/.npgo-integrity.json` để idempotent (skip nếu phiên bản trùng).
   - Tạo liên kết global vào `~/.npgo/node_modules/<pkg>` để hỗ trợ resolve xuyên dự án.

6. Lockfile (snapshot tối thiểu)
   - Ghi `.npgo-lock.yaml` với danh sách gói (name, version, resolved URL, integrity placeholder).
   - Tương lai: dùng lockfile để bỏ qua bước resolve.

---

### 3) Debug & Logs (`--dev`)

- Resolver: in "Resolving <name> (spec → normalized)", in lỗi resolve cụ thể.
- Registry: in tarball URL, SHA256.
- Extractor/Installer: sau extract/link hiển thị tổng số file/dir và một số path mẫu.
- Tiện cho việc dò lỗi "mất file" (đã sửa `cleanTarPath` – chỉ bỏ `package/` khi tồn tại và giữ nguyên các path khác).

---

### 4) Tối ưu hiệu năng hiện có

- Registry cache (ETag/Last-Modified) giảm số request và độ trễ mạng.
- Resolve song song (semaphore 32) – cắt thời gian resolve lớn.
- Tránh fetch metadata lặp lại nhờ `RawDeps` lưu kèm mỗi lần resolve.
- Streaming + CAS: giải nén một lần cho mỗi tarball; dùng link cho các project khác.

Gợi ý tiếp theo (đề xuất):
- Tăng semaphore lên 64 khi mạng/băng thông cho phép.
- Memory cache process-level cho metadata vừa đọc (tránh I/O đĩa lần 2 trong cùng tiến trình).
- Giảm chi tiết log (hoặc thêm `--trace`) để hạn chế overhead khi có rất nhiều dependencies.

---

### 5) Hành vi trong các tình huống đặc biệt

- Chu trình phụ thuộc: không thất bại; cài tiếp các node còn lại theo thứ tự ổn định.
- Package có scope (ví dụ `@pm2/agent`): registry-cache tạo thư mục theo scope trước khi ghi file.
- Thiếu quyền symlink (Windows): tự động fallback junction/hardlink/copy.
- Mạng lỗi khi dùng cache: sẽ fallback đọc cache local (nếu có) thay vì fail ngay.

---

### 6) Đường dẫn quan trọng

- Registry cache: `~/.npgo/registry-cache/<pkg>.json` (+ `.meta.json`)
- Tarball CAS: `~/.npgo/store/v3/<sha256>/package/`
- Extract cache: `~/.npgo/extracted/<name-version>/`
- Global link: `~/.npgo/node_modules/<pkg>`
- Project node_modules: `<project>/node_modules/<pkg>`
- Shims: `<project>/node_modules/.bin/`
- Integrity: `<project>/node_modules/<pkg>/.npgo-integrity.json`
- Lockfile: `<project>/.npgo-lock.yaml`

---

### 7) Tóm tắt

`npgo install` hiện đã:
- Resolve metadata nhanh (cache + song song),
- Tối ưu I/O (streaming + CAS),
- Idempotent qua integrity,
- Khả năng hoạt động tốt trên Windows/POSIX với chiến lược link linh hoạt,
- Log debug đầy đủ để truy vết.