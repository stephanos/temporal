package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"google.golang.org/protobuf/proto"
)

// RunExtension ends every published Run's name, after its digest.
const RunExtension = ".binpb"

const maxRunBytes = 64 << 20

var (
	hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
	runName   = regexp.MustCompile(`^([0-9a-f]{64})\` + RunExtension + `$`)
)

// Publication is where a published Run stands. Key is relative to the store root.
type Publication struct {
	Key      string
	Path     string
	Existing bool
}

// ConflictError says a key holds something other than the Run's bytes. Nothing was overwritten.
type ConflictError struct {
	Key    string
	Detail string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s: %s; a published Run is never overwritten", e.Key, e.Detail)
}

// Canonical is a Run's canonical payload and its SHA-256 digest. The digest is outside the
// payload, and the payload holds no wall-clock time.
func Canonical(run *Run) ([]byte, string, error) {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(run)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}

// Publish stores a Run under root at model hash / seed / digest, exclusively and atomically: a
// synced temporary file hard-linked to the final name, which fails when the name exists. The
// same Run published again, concurrently or later, finds itself already published; a key holding
// anything else is a ConflictError.
func Publish(ctx context.Context, root string, run *Run) (Publication, error) {
	if !hexDigest.MatchString(run.GetModelHash()) {
		return Publication{}, fmt.Errorf("model hash %q is not a lower-case hex SHA-256", run.GetModelHash())
	}
	data, digest, err := Canonical(run)
	if err != nil {
		return Publication{}, err
	}
	seed := strconv.FormatUint(run.GetSeed(), 10)
	key := filepath.Join(run.GetModelHash(), seed, digest+RunExtension)
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return Publication{}, fmt.Errorf("store root: %w", err)
	}
	directory := resolved
	for _, part := range []string{run.GetModelHash(), seed} {
		directory = filepath.Join(directory, part)
		if err := os.Mkdir(directory, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
			return Publication{}, fmt.Errorf("store %s: %w", key, err)
		}
		if info, err := os.Lstat(directory); err != nil {
			return Publication{}, fmt.Errorf("store %s: %w", key, err)
		} else if !info.IsDir() {
			return Publication{}, &ConflictError{Key: key, Detail: "a directory of the key is not a directory"}
		}
	}
	final := filepath.Join(directory, digest+RunExtension)
	temporary, err := os.CreateTemp(directory, "."+digest+".tmp-*")
	if err != nil {
		return Publication{}, fmt.Errorf("store %s: %w", key, err)
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	_, err = temporary.Write(data)
	if err == nil {
		err = temporary.Chmod(0o644)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if err = errors.Join(err, temporary.Close()); err != nil {
		return Publication{}, fmt.Errorf("store %s: %w", key, err)
	}
	if err := ctx.Err(); err != nil {
		return Publication{}, fmt.Errorf("store %s: interrupted before publishing: %w", key, err)
	}
	err = os.Link(temporary.Name(), final)
	if err == nil {
		return Publication{Key: key, Path: final}, nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return Publication{}, fmt.Errorf("store %s: %w", key, err)
	}
	if err := sameContents(key, final, data); err != nil {
		return Publication{}, err
	}
	return Publication{Key: key, Path: final, Existing: true}, nil
}

// sameContents requires an existing key to be a regular file holding exactly data, reading it
// without following a symlink and at most one byte past data.
func sameContents(key, path string, data []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("store %s: %w", key, err)
	}
	if !info.Mode().IsRegular() {
		return &ConflictError{Key: key, Detail: fmt.Sprintf("the key is not a regular file (%s)", info.Mode().Type())}
	}
	file, err := openExisting(path)
	if err != nil {
		return &ConflictError{Key: key, Detail: fmt.Sprintf("the key cannot be read as a file: %v", err)}
	}
	defer func() { _ = file.Close() }()
	if opened, err := file.Stat(); err != nil || !opened.Mode().IsRegular() {
		return &ConflictError{Key: key, Detail: "the key changed to something other than a regular file"}
	}
	existing, err := io.ReadAll(io.LimitReader(file, int64(len(data))+1))
	if err != nil {
		return fmt.Errorf("store %s: %w", key, err)
	}
	if !bytes.Equal(existing, data) {
		return &ConflictError{Key: key, Detail: "the key holds other bytes"}
	}
	return nil
}

// Load reads a Run. Its bytes must be canonical, and a file named by a digest must hash to it.
func Load(path string) (*Run, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxRunBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxRunBytes {
		return nil, fmt.Errorf("%s: a Run is at most %d bytes", filepath.Base(path), maxRunBytes)
	}
	run := new(Run)
	if err := proto.Unmarshal(data, run); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	canonical, digest, err := Canonical(run)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, data) {
		return nil, fmt.Errorf("%s: not a canonical Run", filepath.Base(path))
	}
	if match := runName.FindStringSubmatch(filepath.Base(path)); match != nil && match[1] != digest {
		return nil, fmt.Errorf("%s: the Run's digest is %s", filepath.Base(path), digest)
	}
	return run, nil
}
