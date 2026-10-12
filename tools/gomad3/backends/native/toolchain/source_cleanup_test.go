package toolchain

import (
	"archive/tar"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

type sourceCleanupTransport func(*http.Request) (*http.Response, error)

func (transport sourceCleanupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type sourceCleanupBody struct {
	io.Reader
	file     *os.File
	closes   int
	closeErr error
}

func (body *sourceCleanupBody) Close() error {
	body.closes++
	if body.file != nil {
		body.closeErr = body.file.Close()
	}
	return body.closeErr
}

func sourceCleanupClosedFile(t *testing.T) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "closed-body")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file
}

func sourceCleanupSpec(t *testing.T, contents []byte, transport sourceCleanupTransport) SourceSpec {
	t.Helper()
	return SourceSpec{
		CacheDir: t.TempDir(), Name: "go.src.tar.gz", URL: "https://source.invalid/archive",
		SHA256: fmt.Sprintf("%x", sha256.Sum256(contents)), Retries: 1,
		Client: &http.Client{Transport: transport},
	}
}

func sourceCleanupResponse(body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: body, Header: make(http.Header)}
}

func TestSourceCleanupPublicationRetriesAndCache(t *testing.T) {
	for _, test := range []struct {
		name        string
		retries     int
		attempts    int
		healthyLast bool
	}{
		{name: "one", retries: 1, attempts: 1},
		{name: "explicit", retries: 2, attempts: 2},
		{name: "default", retries: 0, attempts: 3},
		{name: "later-success", retries: 2, attempts: 2, healthyLast: true},
		{name: "healthy", retries: 1, attempts: 1, healthyLast: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			contents := []byte("verified archive bytes\n")
			var bodies []*sourceCleanupBody
			config := sourceCleanupSpec(t, contents, func(request *http.Request) (*http.Response, error) {
				body := &sourceCleanupBody{Reader: bytes.NewReader(contents)}
				if !test.healthyLast || len(bodies)+1 != test.attempts {
					body.file = sourceCleanupClosedFile(t)
				}
				bodies = append(bodies, body)
				return sourceCleanupResponse(body), nil
			})
			config.Retries = test.retries
			published := filepath.Join(config.CacheDir, config.Name)
			got, err := EnsureSource(context.Background(), config)
			if test.healthyLast {
				if err != nil || got != published {
					t.Fatalf("EnsureSource() = %q, %v", got, err)
				}
			} else {
				if got != "" || !errors.Is(err, os.ErrClosed) {
					t.Fatalf("EnsureSource() = %q, %v; want closed-file failure despite publication", got, err)
				}
				want := fmt.Sprintf("download source archive after %d attempt(s): %s", test.attempts, bodies[len(bodies)-1].closeErr)
				if err.Error() != want || errors.Unwrap(err) != bodies[len(bodies)-1].closeErr {
					t.Fatalf("final error = %v, want %s and actual Close identity", err, want)
				}
			}
			if len(bodies) != test.attempts {
				t.Fatalf("requests = %d, want %d", len(bodies), test.attempts)
			}
			for _, body := range bodies {
				if body.closes != 1 {
					t.Fatalf("body closes = %d, want 1", body.closes)
				}
			}
			stored, readErr := os.ReadFile(published)
			if readErr != nil || !bytes.Equal(stored, contents) {
				t.Fatalf("published bytes = %q, %v", stored, readErr)
			}
			info, statErr := os.Stat(published)
			if statErr != nil || info.Mode().Perm() != 0o644 {
				t.Fatalf("published mode = %v, %v", info, statErr)
			}
			entries, readErr := os.ReadDir(config.CacheDir)
			if readErr != nil || len(entries) != 1 || entries[0].Name() != config.Name {
				t.Fatalf("cache entries = %v, %v", entries, readErr)
			}
			cached, cacheErr := EnsureSource(context.Background(), config)
			if cacheErr != nil || cached != published || len(bodies) != test.attempts {
				t.Fatalf("cached EnsureSource() = %q, %v; requests = %d", cached, cacheErr, len(bodies))
			}
		})
	}
}

type sourceCleanupReader func([]byte) (int, error)

func (reader sourceCleanupReader) Read(buffer []byte) (int, error) { return reader(buffer) }

type sourceCleanupCallbackError struct {
	fd       int
	observed []error
}

func (err *sourceCleanupCallbackError) Error() string {
	var stat syscall.Stat_t
	err.observed = append(err.observed, syscall.Fstat(err.fd, &stat))
	return "response callback failed"
}

func TestSourceCleanupClosesTemporaryBeforeFormattingPrimary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor observation requires Linux procfs")
	}
	if _, err := os.ReadDir("/proc/self/fd"); err != nil {
		t.Skipf("procfs unavailable: %v", err)
	}
	primary := &sourceCleanupCallbackError{fd: -1}
	body := &sourceCleanupBody{}
	config := sourceCleanupSpec(t, []byte("archive"), func(*http.Request) (*http.Response, error) {
		return sourceCleanupResponse(body), nil
	})
	body.Reader = sourceCleanupReader(func([]byte) (int, error) {
		paths, err := filepath.Glob(filepath.Join(config.CacheDir, ".source-archive-*"))
		if err != nil || len(paths) != 1 {
			t.Fatalf("temporary paths = %v, %v", paths, err)
		}
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if name == paths[0] {
				primary.fd, err = strconv.Atoi(entry.Name())
				if err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		var stat syscall.Stat_t
		if primary.fd < 0 || syscall.Fstat(primary.fd, &stat) != nil {
			t.Fatal("temporary descriptor was not captured open")
		}
		return 0, primary
	})
	got, err := EnsureSource(context.Background(), config)
	if len(primary.observed) != 1 || !errors.Is(primary.observed[0], syscall.EBADF) {
		t.Fatalf("primary Error callback descriptor observations = %v; want one EBADF", primary.observed)
	}
	if got != "" || errors.Unwrap(errors.Unwrap(err)) != primary || err.Error() != "download source archive after 1 attempt(s): download source archive: response callback failed" {
		t.Fatalf("EnsureSource() = %q, %v; primary identity or message changed", got, err)
	}
	if body.closes != 1 {
		t.Fatalf("body closes = %d", body.closes)
	}
}

func TestSourceCleanupPrimaryErrors(t *testing.T) {
	readErr := io.ErrClosedPipe
	for _, primary := range []string{"http", "read", "checksum", "cancel"} {
		for _, failedClose := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/failed-close=%t", primary, failedClose), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				contents := []byte("archive")
				body := &sourceCleanupBody{Reader: bytes.NewReader(contents)}
				if failedClose {
					body.file = sourceCleanupClosedFile(t)
				}
				want := "request source archive: HTTP status 503 Service Unavailable"
				var cause error
				switch primary {
				case "read":
					reader, writer := io.Pipe()
					if err := reader.Close(); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					body.Reader = reader
					want, cause = "download source archive: io: read/write on closed pipe", readErr
				case "checksum":
					want = "source archive checksum mismatch: got 0eb3e36bfb24dcd9bb1d1bece1531216b59539a8fde17ee80224af0653c92aa3, want " + strings.Repeat("a", 64)
				case "cancel":
					body.Reader = sourceCleanupReader(func([]byte) (int, error) { cancel(); return 0, nil })
					want, cause = "download source archive: context canceled", context.Canceled
				}
				config := sourceCleanupSpec(t, contents, func(request *http.Request) (*http.Response, error) {
					response := sourceCleanupResponse(body)
					if primary == "http" {
						response.StatusCode, response.Status = 503, "503 Service Unavailable"
					}
					return response, nil
				})
				if primary == "checksum" {
					config.SHA256 = strings.Repeat("a", 64)
				}
				got, err := EnsureSource(ctx, config)
				if got != "" || err == nil || !strings.HasPrefix(err.Error(), "download source archive after 1 attempt(s): ") {
					t.Fatalf("EnsureSource() = %q, %v", got, err)
				}
				err = errors.Unwrap(err)
				if failedClose {
					want += "\n" + body.closeErr.Error()
					if !errors.Is(err, body.closeErr) || !errors.Is(err, os.ErrClosed) {
						t.Fatalf("error = %v; missing actual Close identity", err)
					}
				} else if cause != nil && errors.Unwrap(err) != cause {
					t.Fatalf("nil cleanup changed primary wrapper identity: %v", err)
				}
				if err == nil || err.Error() != want || (cause != nil && !errors.Is(err, cause)) || body.closes != 1 {
					t.Fatalf("download() = %v, closes = %d; want %s", err, body.closes, want)
				}
				entries, err := os.ReadDir(config.CacheDir)
				if err != nil || len(entries) != 0 {
					t.Fatalf("failed download leaves entries = %v, %v", entries, err)
				}
			})
		}
	}
}

func TestSourceCleanupRemoveFailure(t *testing.T) {
	for _, failedClose := range []bool{false, true} {
		t.Run(fmt.Sprintf("failed-close=%t", failedClose), func(t *testing.T) {
			readErr := io.ErrClosedPipe
			reader, writer := io.Pipe()
			if err := reader.Close(); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			var obstruction string
			body := &sourceCleanupBody{}
			if failedClose {
				body.file = sourceCleanupClosedFile(t)
			}
			config := sourceCleanupSpec(t, []byte("archive"), func(request *http.Request) (*http.Response, error) {
				return sourceCleanupResponse(body), nil
			})
			body.Reader = sourceCleanupReader(func([]byte) (int, error) {
				paths, err := filepath.Glob(filepath.Join(config.CacheDir, ".source-archive-*"))
				if err != nil || len(paths) != 1 {
					t.Fatalf("temporary paths = %v, %v", paths, err)
				}
				obstruction = paths[0]
				if err := os.Remove(obstruction); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(obstruction, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(obstruction, "owned"), []byte("retain obstruction"), 0o600); err != nil {
					t.Fatal(err)
				}
				return reader.Read(nil)
			})
			got, err := EnsureSource(context.Background(), config)
			if got != "" || !errors.Is(err, readErr) || !errors.Is(err, syscall.ENOTEMPTY) {
				t.Fatalf("EnsureSource() = %q, %v; want read failure and actual ENOTEMPTY", got, err)
			}
			var removal *os.PathError
			if !errors.As(err, &removal) || removal.Op != "remove" || removal.Path != obstruction {
				t.Fatalf("removal error = %v", err)
			}
			want := "download source archive after 1 attempt(s): download source archive: io: read/write on closed pipe\n" + removal.Error()
			if failedClose {
				want += "\n" + body.closeErr.Error()
				if !errors.Is(err, body.closeErr) || !errors.Is(err, os.ErrClosed) {
					t.Fatalf("body Close cause missing: %v", err)
				}
			}
			if err.Error() != want || body.closes != 1 {
				t.Fatalf("error = %v, closes = %d; want %s", err, body.closes, want)
			}
			contents, err := os.ReadFile(filepath.Join(obstruction, "owned"))
			if err != nil || string(contents) != "retain obstruction" {
				t.Fatalf("obstruction contents = %q, %v", contents, err)
			}
			if err := os.Remove(filepath.Join(obstruction, "owned")); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(obstruction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSourceCleanupCancellationBetweenAttempts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := &sourceCleanupBody{
		Reader: sourceCleanupReader(func([]byte) (int, error) { cancel(); return 0, nil }),
		file:   sourceCleanupClosedFile(t),
	}
	requests := 0
	config := sourceCleanupSpec(t, []byte("archive"), func(*http.Request) (*http.Response, error) {
		requests++
		return sourceCleanupResponse(body), nil
	})
	config.Retries = 2
	got, err := EnsureSource(ctx, config)
	if got != "" || err != context.Canceled || requests != 1 || body.closes != 1 {
		t.Fatalf("EnsureSource() = %q, %v; requests = %d, closes = %d", got, err, requests, body.closes)
	}
	if !errors.Is(body.closeErr, os.ErrClosed) {
		t.Fatalf("actual body Close = %v", body.closeErr)
	}
}

func TestSourceCleanupLegacyTarNormalization(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		size string
		body string
		flag byte
	}{
		{name: "regular", path: "go", size: "00000000010", body: "go1.27.1", flag: tar.TypeReg},
		{name: "directory", path: "go/", size: "00000000000", flag: tar.TypeDir},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := make([]byte, 2048)
			copy(raw[:100], test.path)
			copy(raw[100:108], "0000755\x00")
			copy(raw[108:116], "0000000\x00")
			copy(raw[116:124], "0000000\x00")
			copy(raw[124:136], test.size+"\x00")
			copy(raw[136:148], "00000000000\x00")
			copy(raw[148:156], "        ")
			raw[156] = 0
			copy(raw[512:], test.body)
			var checksum int
			for _, value := range raw[:512] {
				checksum += int(value)
			}
			copy(raw[148:156], fmt.Sprintf("%06o\x00 ", checksum))
			header, err := tar.NewReader(bytes.NewReader(raw)).Next()
			if err != nil || header.Typeflag != test.flag {
				t.Fatalf("raw NUL header normalized to %v, %v", header, err)
			}
			var compressed bytes.Buffer
			zipper := gzip.NewWriter(&compressed)
			if _, err := zipper.Write(raw); err != nil {
				t.Fatal(err)
			}
			if err := zipper.Close(); err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(t.TempDir(), "legacy.tar.gz")
			if err := os.WriteFile(archive, compressed.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			destination := t.TempDir()
			if err := ExtractSource(context.Background(), archive, destination); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(filepath.Join(destination, test.path))
			if err != nil || info.IsDir() != (test.flag == tar.TypeDir) || info.Mode().Perm() != 0o755 {
				t.Fatalf("extracted entry = %v, %v", info, err)
			}
			if test.body != "" {
				contents, err := os.ReadFile(filepath.Join(destination, test.path))
				if err != nil || string(contents) != "go1.27.1" {
					t.Fatalf("contents = %q, %v", contents, err)
				}
			}
		})
	}
}

func TestSourceCleanupCorruptDeflateReadAndClose(t *testing.T) {
	archive := []byte{0x1f, 0x8b, 0x08, 0, 0, 0, 0, 0, 0, 0x03, 0x07}
	zipper, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	_, readErr := io.ReadAll(zipper)
	closeErr := zipper.Close()
	if readErr != flate.CorruptInputError(1) || closeErr != flate.CorruptInputError(1) {
		t.Fatalf("literal invalid deflate Read = %v, Close = %v", readErr, closeErr)
	}
	archivePath := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	err = ExtractSource(context.Background(), archivePath, t.TempDir())
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("ExtractSource() = %v; want read and Close errors", err)
	}
	causes := joined.Unwrap()
	if len(causes) != 2 || errors.Unwrap(causes[0]) != readErr || causes[1] != closeErr {
		t.Fatalf("ExtractSource() causes = %v; want primary read then actual Close", causes)
	}
	if err.Error() != "read source archive: flate: corrupt input before offset 1\nflate: corrupt input before offset 1" {
		t.Fatalf("ExtractSource() = %v", err)
	}
}

func TestSourceCleanupCacheAndExtractionErrors(t *testing.T) {
	config := sourceCleanupSpec(t, []byte("archive"), func(*http.Request) (*http.Response, error) {
		t.Fatal("cache inspection failure reached network")
		return nil, errors.New("unexpected network")
	})
	if err := os.Mkdir(filepath.Join(config.CacheDir, config.Name), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureSource(context.Background(), config); err == nil || !strings.HasPrefix(err.Error(), "inspect cached source archive: ") {
		t.Fatalf("cache error = %v", err)
	}
	for _, test := range []struct{ name, contents, prefix string }{
		{name: "missing", prefix: "open source archive: "},
		{name: "gzip", contents: "not compressed", prefix: "open compressed source archive: "},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "archive")
			if test.contents != "" {
				if err := os.WriteFile(archive, []byte(test.contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := ExtractSource(context.Background(), archive, t.TempDir())
			if err == nil || !strings.HasPrefix(err.Error(), test.prefix) {
				t.Fatalf("extraction error = %v", err)
			}
		})
	}
}
