package toolchain

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
)

const maximumDownloadBytes = 256 << 20
const maximumExpandedBytes = 4 << 30
const maximumArchiveEntries = 100_000

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type SourceSpec struct {
	CacheDir string
	Name     string
	URL      string
	SHA256   string
	Retries  int
	Client   *http.Client
}

func EnsureSource(ctx context.Context, config SourceSpec) (string, error) {
	if err := validateSourceSpec(config); err != nil {
		return "", err
	}
	cacheDir, err := filepath.Abs(config.CacheDir)
	if err != nil || cacheDir == string(filepath.Separator) {
		return "", errors.Join(errors.New("source archive cache must be an absolute non-root directory"), err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create source archive cache: %w", err)
	}
	archivePath := filepath.Join(cacheDir, config.Name)
	match, err := matchesDigest(archivePath, config.SHA256)
	if err == nil && match {
		return archivePath, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect cached source archive: %w", err)
	}

	retries := config.Retries
	if retries == 0 {
		retries = 3
	}
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}
	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if err := download(ctx, client, config, archivePath); err == nil {
			return archivePath, nil
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("download source archive after %d attempt(s): %w", retries, lastErr)
}

func ExtractSource(ctx context.Context, archivePath, destination string) error {
	if archivePath == "" || destination == "" {
		return errors.New("source archive and destination are required")
	}
	destination, err := filepath.Abs(destination)
	if err != nil || destination == string(filepath.Separator) {
		return errors.Join(errors.New("source archive destination must be an absolute non-root directory"), err)
	}
	if err := prepareDestination(destination); err != nil {
		return err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return fmt.Errorf("open source archive destination: %w", err)
	}
	defer root.Close()

	file, _, err := hostfs.OpenPath(archivePath)
	if err != nil {
		return fmt.Errorf("open source archive: %w", err)
	}
	defer file.Close()
	zipper, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open compressed source archive: %w", err)
	}
	defer zipper.Close()

	reader := tar.NewReader(zipper)
	seen := make(map[string]struct{})
	var entries int
	var expanded int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read source archive: %w", err)
		}
		entries++
		if entries > maximumArchiveEntries {
			return fmt.Errorf("source archive has more than %d entries", maximumArchiveEntries)
		}
		name, err := archivePathName(header.Name)
		if err != nil {
			return err
		}
		if _, found := seen[name]; found {
			return fmt.Errorf("source archive contains duplicate path: %s", name)
		}
		seen[name] = struct{}{}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(name, fs.FileMode(header.Mode)&0o777); err != nil {
				return fmt.Errorf("create source archive directory %s: %w", name, err)
			}
			if err := root.Chmod(name, fs.FileMode(header.Mode)&0o777); err != nil {
				return fmt.Errorf("set source archive directory mode %s: %w", name, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || expanded > maximumExpandedBytes-header.Size {
				return fmt.Errorf("source archive expands beyond %d bytes", maximumExpandedBytes)
			}
			expanded += header.Size
			if err := root.MkdirAll(path.Dir(name), 0o755); err != nil {
				return fmt.Errorf("create source archive parent %s: %w", name, err)
			}
			output, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fs.FileMode(header.Mode)&0o777)
			if err != nil {
				return fmt.Errorf("create source archive file %s: %w", name, err)
			}
			copyErr := copyExactly(ctx, output, reader, header.Size)
			closeErr := output.Close()
			if copyErr != nil || closeErr != nil {
				return errors.Join(fmt.Errorf("extract source archive file %s: %w", name, copyErr), closeErr)
			}
		default:
			return fmt.Errorf("source archive contains unsupported entry %s of type %d", name, header.Typeflag)
		}
	}
	if _, found := seen["go"]; !found {
		return errors.New("source archive does not contain the expected go root")
	}
	return nil
}

func validateSourceSpec(config SourceSpec) error {
	if config.CacheDir == "" || config.URL == "" || config.Name == "" {
		return errors.New("source archive cache, name, and URL are required")
	}
	if filepath.Base(config.Name) != config.Name || config.Name == "." || config.Name == ".." {
		return errors.New("source archive name must be a base name")
	}
	if !sha256Pattern.MatchString(config.SHA256) {
		return errors.New("source archive SHA-256 is invalid")
	}
	if config.Retries < 0 {
		return errors.New("source archive retries must not be negative")
	}
	return nil
}

func matchesDigest(filePath, want string) (bool, error) {
	file, info, err := hostfs.OpenPath(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	if info.Size() < 0 || info.Size() > maximumDownloadBytes {
		return false, nil
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return false, err
	}
	return fmt.Sprintf("%x", digest.Sum(nil)) == want, nil
}

func download(ctx context.Context, client *http.Client, config SourceSpec, archivePath string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, config.URL, nil)
	if err != nil {
		return fmt.Errorf("construct source archive request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request source archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("request source archive: HTTP status %s", response.Status)
	}
	temporary, err := os.CreateTemp(config.CacheDir, ".source-archive-*")
	if err != nil {
		return fmt.Errorf("create source archive download: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	digest := sha256.New()
	written, copyErr := copyWithContext(ctx, io.MultiWriter(temporary, digest), io.LimitReader(response.Body, maximumDownloadBytes+1))
	if copyErr != nil {
		temporary.Close()
		return fmt.Errorf("download source archive: %w", copyErr)
	}
	if written > maximumDownloadBytes {
		temporary.Close()
		return fmt.Errorf("source archive download exceeds %d bytes", maximumDownloadBytes)
	}
	actual := fmt.Sprintf("%x", digest.Sum(nil))
	if actual != config.SHA256 {
		temporary.Close()
		return fmt.Errorf("source archive checksum mismatch: got %s, want %s", actual, config.SHA256)
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set source archive mode: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync source archive download: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close source archive download: %w", err)
	}
	if err := os.Rename(temporaryPath, archivePath); err != nil {
		return fmt.Errorf("publish source archive download: %w", err)
	}
	return syncSourceDirectory(filepath.Dir(archivePath))
}

func prepareDestination(destination string) error {
	info, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(destination, 0o700); err != nil {
			return fmt.Errorf("create source archive destination: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect source archive destination: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("source archive destination must be a real directory")
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return fmt.Errorf("read source archive destination: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("source archive destination must be empty")
	}
	return nil
}

func archivePathName(name string) (string, error) {
	withoutSlash := strings.TrimSuffix(name, "/")
	clean := path.Clean(withoutSlash)
	if name == "" || strings.Contains(name, "\\") || path.IsAbs(name) || clean != withoutSlash || !fs.ValidPath(clean) {
		return "", fmt.Errorf("source archive contains invalid path: %q", name)
	}
	if clean != "go" && !strings.HasPrefix(clean, "go/") {
		return "", fmt.Errorf("source archive path is outside the go root: %s", clean)
	}
	return clean, nil
}

func copyExactly(ctx context.Context, destination io.Writer, source io.Reader, size int64) error {
	written, err := copyWithContext(ctx, destination, io.LimitReader(source, size))
	if err != nil {
		return err
	}
	if written != size {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func copyWithContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}

func syncSourceDirectory(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}
