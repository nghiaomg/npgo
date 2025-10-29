package extractor

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	pgzip "github.com/klauspost/pgzip"

	mmap "golang.org/x/exp/mmap"
)

func ExtractTarGz(src, dest string) error {
	mm, err := mmap.Open(src)
	if err == nil {
		defer mm.Close()
		reader := io.NewSectionReader(mm, 0, int64(mm.Len()))
		if gz, gzErr := pgzip.NewReader(reader); gzErr == nil {
			gz.Multistream(true)
			defer gz.Close()
			tr := tar.NewReader(gz)
			if err := extractTarReaderParallel(tr, dest); err != nil {
				return err
			}
			return nil
		}
		tr := tar.NewReader(reader)
		if err := extractTarReaderParallel(tr, dest); err != nil {
			return err
		}
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	gzReader, err := pgzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	gzReader.Multistream(true)
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	return extractTarReaderParallel(tarReader, dest)
}

func ExtractFromReader(r io.Reader, dest string) error {
	var tr *tar.Reader

	if gz, err := pgzip.NewReader(noCloseReader{r}); err == nil {
		gz.Multistream(true)
		defer gz.Close()
		tr = tar.NewReader(gz)
	} else {
		tr = tar.NewReader(r)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return extractTarReaderParallel(tr, dest)
}

type noCloseReader struct{ io.Reader }

func (noCloseReader) Close() error { return nil }

var copyBufPool = sync.Pool{New: func() any { return make([]byte, 64*1024) }}

type fileJob struct {
	path string
	mode int64
	data []byte
}

func extractTarReaderParallel(tr *tar.Reader, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	jobs := make(chan fileJob, 128)
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if err := os.MkdirAll(filepath.Dir(j.path), 0755); err != nil {
					continue
				}
				f, err := os.OpenFile(j.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
				if err != nil {
					continue
				}
				if len(j.data) > 0 {
					_, _ = f.Write(j.data)
				}
				_ = f.Close()
				if runtime.GOOS != "windows" {
					_ = os.Chmod(j.path, os.FileMode(j.mode))
				}
			}
		}()
	}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			close(jobs)
			wg.Wait()
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		clean := cleanTarPath(header.Name)
		if clean == "" {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(filepath.Join(dest, clean), 0755)
		case tar.TypeReg:
			full := filepath.Join(dest, clean)
			var out []byte
			if header.Size > 0 {
				buf := copyBufPool.Get().([]byte)
				for {
					n, er := tr.Read(buf)
					if n > 0 {
						out = append(out, buf[:n]...)
					}
					if er == io.EOF {
						break
					}
					if er != nil {
						break
					}
				}
				copyBufPool.Put(buf)
			}
			jobs <- fileJob{path: full, mode: header.Mode, data: out}
		default:
		}
	}
	close(jobs)
	wg.Wait()
	return nil
}

func cleanTarPath(p string) string {
	if strings.HasPrefix(p, "./") {
		p = p[2:]
	}
	if len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "package/") {
		return p[len("package/"):]
	}
	if p == "package" {
		return ""
	}
	return p
}
