package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/record"
)

type Store struct {
	Root         string
	Context      context.Context
	MaximumBytes uint64
	Key          StoreKey
}

type StoreKey uint8

const (
	StoreKeyFailureSignature StoreKey = iota
	StoreKeyRecord
)

type Payload struct {
	Path       string
	Mode       os.FileMode
	SourcePath string
	Data       []byte
	SHA256     record.SHA256
	Size       record.Uint64String
}

type Publication struct {
	Record   record.ExecutionRecord
	Payloads []Payload
}

type Artifact struct {
	Path        string
	Manifest    record.ExecutionRecord
	StoredBytes uint64
	root        *os.Root
}

type CapacityError struct {
	Required uint64
	Maximum  uint64
}

func (err *CapacityError) Error() string {
	return fmt.Sprintf("artifact requires %d bytes, exceeding the %d-byte capacity", err.Required, err.Maximum)
}

func (store Store) PublishArtifact(publication Publication) (_ Artifact, retErr error) {
	ctx := store.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if store.Root == "" {
		return Artifact{}, fmt.Errorf("artifact store root is required")
	}
	if publication.Record.SchemaVersion != record.SchemaVersion || publication.Record.Runner.RecordContract != record.RecordContract {
		return Artifact{}, fmt.Errorf("artifact publication requires the current run-record contract")
	}
	if len(publication.Payloads) == 0 {
		return Artifact{}, fmt.Errorf("artifact publication requires payloads")
	}
	if err := validatePublicationPayloads(publication.Payloads); err != nil {
		return Artifact{}, err
	}
	if err := os.MkdirAll(store.Root, 0o700); err != nil {
		return Artifact{}, fmt.Errorf("create artifact store: %w", err)
	}
	if err := os.Chmod(store.Root, 0o700); err != nil {
		return Artifact{}, fmt.Errorf("make artifact store private: %w", err)
	}
	staging, err := os.MkdirTemp(store.Root, ".publish-")
	if err != nil {
		return Artifact{}, fmt.Errorf("create artifact staging directory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(staging))
	}()
	if err := os.Chmod(staging, 0o700); err != nil {
		return Artifact{}, fmt.Errorf("make artifact staging directory private: %w", err)
	}

	manifest := publication.Record
	manifest.Files = nil
	files := make([]record.File, 0, len(publication.Payloads))
	for _, payload := range publication.Payloads {
		destination := filepath.Join(staging, filepath.FromSlash(payload.Path))
		var file record.File
		var err error
		if payload.SourcePath != "" {
			file, err = copyPayload(ctx, payload.SourcePath, destination, payload.Path, payload.Mode)
		} else {
			file, err = writePayload(ctx, destination, payload.Path, payload.Data, payload.Mode)
		}
		if err != nil {
			return Artifact{}, err
		}
		if file.SHA256 != payload.SHA256 || file.Size != payload.Size {
			return Artifact{}, fmt.Errorf("artifact payload %s identity changed during publication", payload.Path)
		}
		files = append(files, file)
	}
	if err := syncPayloadDirectories(ctx, staging, publication.Payloads); err != nil {
		return Artifact{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest.Files = files
	manifest, manifestBytes, err := record.FinalizeExecutionRecord(manifest)
	if err != nil {
		return Artifact{}, fmt.Errorf("finalize artifact manifest: %w", err)
	}
	storedBytes, err := artifactStoredBytes(manifest, uint64(len(manifestBytes)))
	if err != nil {
		return Artifact{}, err
	}
	if store.MaximumBytes != 0 && storedBytes > store.MaximumBytes {
		return Artifact{}, &CapacityError{Required: storedBytes, Maximum: store.MaximumBytes}
	}
	if _, err := writePayload(ctx, filepath.Join(staging, "manifest.json"), "manifest.json", manifestBytes, 0o600); err != nil {
		return Artifact{}, err
	}
	if err := syncDirectoryContext(ctx, staging); err != nil {
		return Artifact{}, fmt.Errorf("sync artifact staging directory: %w", err)
	}

	identity, err := storeIdentity(store.Key, manifest)
	if err != nil {
		return Artifact{}, err
	}
	finalPath := filepath.Join(store.Root, identityDirectory(identity, false))
	for {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		if err := renameNoReplace(staging, finalPath); err == nil {
			break
		} else if !errors.Is(err, os.ErrExist) {
			return Artifact{}, fmt.Errorf("publish artifact: %w", err)
		}
		existing, openErr := OpenArtifact(finalPath)
		if openErr != nil {
			return Artifact{}, fmt.Errorf("existing artifact %s failed validation: %w", finalPath, openErr)
		}
		existingIdentity, identityErr := storeIdentity(store.Key, existing.Manifest)
		if identityErr != nil {
			return Artifact{}, errors.Join(identityErr, existing.Close())
		}
		if existingIdentity == identity {
			identity := Artifact{Path: existing.Path, Manifest: existing.Manifest, StoredBytes: existing.StoredBytes}
			if closeErr := existing.Close(); closeErr != nil {
				return Artifact{}, fmt.Errorf("close existing artifact: %w", closeErr)
			}
			if removeErr := os.RemoveAll(staging); removeErr != nil {
				return Artifact{}, fmt.Errorf("remove redundant artifact staging directory: %w", removeErr)
			}
			return identity, nil
		}
		if closeErr := existing.Close(); closeErr != nil {
			return Artifact{}, fmt.Errorf("close colliding artifact: %w", closeErr)
		}
		completePath := filepath.Join(store.Root, identityDirectory(identity, true))
		if finalPath == completePath {
			return Artifact{}, fmt.Errorf("artifact signature collision at %s", finalPath)
		}
		finalPath = completePath
	}
	if err := syncDirectoryContext(ctx, store.Root); err != nil {
		return Artifact{}, fmt.Errorf("sync artifact store: %w", err)
	}
	return Artifact{Path: finalPath, Manifest: manifest, StoredBytes: storedBytes}, nil
}

func validatePublicationPayloads(payloads []Payload) error {
	seen := make(map[string]struct{}, len(payloads))
	for _, payload := range payloads {
		if payload.Path == "" || payload.Path == "manifest.json" || path.Clean(payload.Path) != payload.Path || filepath.IsAbs(filepath.FromSlash(payload.Path)) || strings.Contains(payload.Path, `\`) {
			return fmt.Errorf("invalid artifact payload path %q", payload.Path)
		}
		if _, duplicate := seen[payload.Path]; duplicate {
			return fmt.Errorf("duplicate artifact payload path %q", payload.Path)
		}
		seen[payload.Path] = struct{}{}
		if payload.Mode != 0o600 && payload.Mode != 0o700 {
			return fmt.Errorf("invalid artifact payload mode %#o for %s", payload.Mode, payload.Path)
		}
		if payload.SourcePath != "" && payload.Data != nil {
			return fmt.Errorf("artifact payload %s has both source and inline data", payload.Path)
		}
		if payload.SHA256 == "" {
			return fmt.Errorf("artifact payload %s has no expected identity", payload.Path)
		}
		if _, err := payload.SHA256.Bytes(); err != nil {
			return fmt.Errorf("artifact payload %s expected identity: %w", payload.Path, err)
		}
	}
	return nil
}

func artifactStoredBytes(manifest record.ExecutionRecord, manifestBytes uint64) (uint64, error) {
	total := manifestBytes
	for _, file := range manifest.Files {
		if uint64(file.Size) > ^uint64(0)-total {
			return 0, errors.New("artifact byte count overflows uint64")
		}
		total += uint64(file.Size)
	}
	return total, nil
}

func storeIdentity(key StoreKey, manifest record.ExecutionRecord) (record.SHA256, error) {
	switch key {
	case StoreKeyFailureSignature:
		return manifest.Outcome.FailureSignature, nil
	case StoreKeyRecord:
		return manifest.RecordHash, nil
	default:
		return "", fmt.Errorf("unknown artifact store key %d", key)
	}
}

func identityDirectory(identity record.SHA256, complete bool) string {
	hex := strings.TrimPrefix(string(identity), "sha256:")
	if !complete && len(hex) >= 32 {
		hex = hex[:32]
	}
	return "sha256-" + hex
}

func copyPayload(ctx context.Context, source, destination, relativePath string, mode os.FileMode) (record.File, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return record.File{}, fmt.Errorf("create payload parent %s: %w", relativePath, err)
	}
	if err := os.Chmod(filepath.Dir(destination), 0o700); err != nil {
		return record.File{}, fmt.Errorf("make payload parent private %s: %w", relativePath, err)
	}
	input, err := os.Open(source)
	if err != nil {
		return record.File{}, fmt.Errorf("open payload %s: %w", relativePath, err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return record.File{}, fmt.Errorf("stat payload %s: %w", relativePath, err)
	}
	if !info.Mode().IsRegular() {
		return record.File{}, fmt.Errorf("payload %s is not a regular file", relativePath)
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return record.File{}, fmt.Errorf("create payload %s: %w", relativePath, err)
	}
	if err := output.Chmod(mode); err != nil {
		output.Close()
		return record.File{}, fmt.Errorf("set payload mode %s: %w", relativePath, err)
	}
	hasher := recordHashWriter{writer: output, hasher: sha256.New()}
	_, copyErr := copyWithContext(ctx, &hasher, input)
	if copyErr != nil {
		output.Close()
		return record.File{}, fmt.Errorf("copy payload %s: %w", relativePath, copyErr)
	}
	if err := syncFileContext(ctx, output); err != nil {
		output.Close()
		return record.File{}, fmt.Errorf("sync payload %s: %w", relativePath, err)
	}
	if err := output.Close(); err != nil {
		return record.File{}, fmt.Errorf("close payload %s: %w", relativePath, err)
	}
	if err := input.Close(); err != nil {
		return record.File{}, fmt.Errorf("close source payload %s: %w", relativePath, err)
	}
	return record.File{Path: relativePath, Mode: formatMode(mode), Size: record.Uint64String(hasher.size), SHA256: record.SHA256("sha256:" + hex.EncodeToString(hasher.hasher.Sum(nil)))}, nil
}

func syncPayloadDirectories(ctx context.Context, staging string, payloads []Payload) error {
	directories := make(map[string]struct{})
	for _, payload := range payloads {
		directory := filepath.Dir(filepath.FromSlash(payload.Path))
		for directory != "." {
			directories[directory] = struct{}{}
			directory = filepath.Dir(directory)
		}
	}
	ordered := make([]string, 0, len(directories))
	for directory := range directories {
		ordered = append(ordered, directory)
	}
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	for _, directory := range ordered {
		if err := syncDirectoryContext(ctx, filepath.Join(staging, directory)); err != nil {
			return fmt.Errorf("sync artifact payload directory %s: %w", filepath.ToSlash(directory), err)
		}
	}
	return nil
}

func writePayload(ctx context.Context, destination, relativePath string, data []byte, mode os.FileMode) (record.File, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return record.File{}, fmt.Errorf("create payload parent %s: %w", relativePath, err)
	}
	if err := os.Chmod(filepath.Dir(destination), 0o700); err != nil {
		return record.File{}, fmt.Errorf("make payload parent private %s: %w", relativePath, err)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return record.File{}, fmt.Errorf("create payload %s: %w", relativePath, err)
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return record.File{}, fmt.Errorf("set payload mode %s: %w", relativePath, err)
	}
	if _, err := copyWithContext(ctx, file, bytes.NewReader(data)); err != nil {
		file.Close()
		return record.File{}, fmt.Errorf("write payload %s: %w", relativePath, err)
	}
	if err := syncFileContext(ctx, file); err != nil {
		file.Close()
		return record.File{}, fmt.Errorf("sync payload %s: %w", relativePath, err)
	}
	if err := file.Close(); err != nil {
		return record.File{}, fmt.Errorf("close payload %s: %w", relativePath, err)
	}
	return record.File{Path: relativePath, Mode: formatMode(mode), Size: record.Uint64String(len(data)), SHA256: record.HashBytes(data)}, nil
}

func copyWithContext(ctx context.Context, destination io.Writer, source io.Reader) (uint64, error) {
	var total uint64
	var buffer [64 << 10]byte
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := source.Read(buffer[:])
		if read > 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += uint64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return total, nil
			}
			return total, readErr
		}
	}
}

type recordHashWriter struct {
	writer io.Writer
	hasher hash.Hash
	size   uint64
}

func (writer *recordHashWriter) Write(data []byte) (int, error) {
	written, err := writer.writer.Write(data)
	if _, hashErr := writer.hasher.Write(data[:written]); err == nil && hashErr != nil {
		err = hashErr
	}
	writer.size += uint64(written)
	return written, err
}

func formatMode(mode os.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func syncDirectoryContext(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := syncDirectory(path); err != nil {
		return err
	}
	return ctx.Err()
}

func syncFileContext(ctx context.Context, file *os.File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return ctx.Err()
}
