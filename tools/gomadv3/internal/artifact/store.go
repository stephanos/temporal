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
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
)

type Store struct {
	Root         string
	Context      context.Context
	MaximumBytes uint64
}

type Input struct {
	Manifest       record.Manifest
	TargetPath     string
	Stdout         []byte
	Stderr         []byte
	IOTranscript   []byte
	ReadOnlyMounts *romount.ArtifactRecord
	World          record.WorldPayloads
}

type Artifact struct {
	Path        string
	Manifest    record.Manifest
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

func (store Store) Publish(input Input) (_ Artifact, retErr error) {
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
	if input.Manifest.World.Initial.File != "world/snapshot.json" || input.Manifest.World.Transitions.File != "world/transitions.jsonl" || input.Manifest.World.Final.File != "world/final-snapshot.json" {
		return Artifact{}, fmt.Errorf("World payload paths must use the canonical artifact layout")
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

	manifest := input.Manifest
	manifest.Target.File = "target"
	manifest.Streams.Stdout.File = "stdout"
	manifest.Streams.Stderr.File = "stderr"
	if manifest.IOProfile.Transcript != nil {
		manifest.IOProfile.Transcript.File = "io/transcript.bin"
	}
	manifest.Files = nil

	targetFile, err := copyPayload(ctx, input.TargetPath, filepath.Join(staging, manifest.Target.File), manifest.Target.File, 0o700)
	if err != nil {
		return Artifact{}, err
	}
	if targetFile.SHA256 != manifest.Target.SHA256 || targetFile.Size != manifest.Target.Size {
		return Artifact{}, fmt.Errorf("prepared target identity changed during publication")
	}
	manifest.Target.SHA256 = targetFile.SHA256
	manifest.Target.Size = targetFile.Size
	stdoutFile, err := writePayload(ctx, filepath.Join(staging, manifest.Streams.Stdout.File), manifest.Streams.Stdout.File, input.Stdout, 0o600)
	if err != nil {
		return Artifact{}, err
	}
	stderrFile, err := writePayload(ctx, filepath.Join(staging, manifest.Streams.Stderr.File), manifest.Streams.Stderr.File, input.Stderr, 0o600)
	if err != nil {
		return Artifact{}, err
	}
	manifest.Streams.Stdout.RetainedSHA256 = stdoutFile.SHA256
	manifest.Streams.Stderr.RetainedSHA256 = stderrFile.SHA256
	files := []record.File{targetFile, stdoutFile, stderrFile}
	if manifest.IOProfile.Transcript != nil {
		transcript := manifest.IOProfile.Transcript
		transcriptFile, err := writePayload(ctx, filepath.Join(staging, filepath.FromSlash(transcript.File)), transcript.File, input.IOTranscript, 0o600)
		if err != nil {
			return Artifact{}, err
		}
		if transcriptFile.SHA256 != transcript.SHA256 || transcriptFile.Size != transcript.Bytes {
			return Artifact{}, errors.New("I/O transcript identity changed during publication")
		}
		files = append(files, transcriptFile)
	}
	if manifest.IOProfile.ReadOnlyMounts != nil {
		if input.ReadOnlyMounts == nil {
			return Artifact{}, errors.New("read-only mount artifact payload is required")
		}
		manifestBytes, err := record.CanonicalJSON(manifest.IOProfile.ReadOnlyMounts)
		if err != nil {
			return Artifact{}, fmt.Errorf("encode manifest read-only mount identity: %w", err)
		}
		inputBytes, err := record.CanonicalJSON(input.ReadOnlyMounts.Manifest)
		if err != nil || !bytes.Equal(manifestBytes, inputBytes) {
			return Artifact{}, errors.Join(errors.New("read-only mount artifact identity changed during publication"), err)
		}
		mounts := manifest.IOProfile.ReadOnlyMounts
		descriptorFile, err := writePayload(ctx, filepath.Join(staging, filepath.FromSlash(mounts.File)), mounts.File, input.ReadOnlyMounts.Descriptor, 0o600)
		if err != nil {
			return Artifact{}, err
		}
		if descriptorFile.SHA256 != mounts.SHA256 || descriptorFile.Size != mounts.Bytes {
			return Artifact{}, errors.New("read-only mount descriptor identity changed during publication")
		}
		files = append(files, descriptorFile)
		payloadPaths := make([]string, 0, len(input.ReadOnlyMounts.Payloads))
		for payloadPath := range input.ReadOnlyMounts.Payloads {
			payloadPaths = append(payloadPaths, payloadPath)
		}
		sort.Strings(payloadPaths)
		for _, payloadPath := range payloadPaths {
			payloadFile, err := writePayload(ctx, filepath.Join(staging, filepath.FromSlash(payloadPath)), payloadPath, input.ReadOnlyMounts.Payloads[payloadPath], 0o600)
			if err != nil {
				return Artifact{}, err
			}
			files = append(files, payloadFile)
		}
	} else if input.ReadOnlyMounts != nil {
		return Artifact{}, errors.New("unexpected read-only mount artifact payload")
	}
	if manifest.IOProfile.Transcript != nil || manifest.IOProfile.ReadOnlyMounts != nil {
		if err := syncDirectoryContext(ctx, filepath.Join(staging, "io")); err != nil {
			return Artifact{}, fmt.Errorf("sync I/O artifact directory: %w", err)
		}
	}

	worldDirectory := filepath.Join(staging, "world")
	if err := os.Mkdir(worldDirectory, 0o700); err != nil {
		return Artifact{}, fmt.Errorf("create World artifact directory: %w", err)
	}
	if err := os.Chmod(worldDirectory, 0o700); err != nil {
		return Artifact{}, fmt.Errorf("make World artifact directory private: %w", err)
	}
	worldFiles := []struct {
		path string
		data []byte
		hash *record.SHA256
	}{
		{path: manifest.World.Initial.File, data: input.World.Initial, hash: &manifest.World.Initial.RawSHA256},
		{path: manifest.World.Transitions.File, data: input.World.Transitions, hash: &manifest.World.Transitions.RawSHA256},
		{path: manifest.World.Final.File, data: input.World.Final, hash: &manifest.World.Final.RawSHA256},
	}
	for _, payload := range worldFiles {
		file, writeErr := writePayload(ctx, filepath.Join(staging, filepath.FromSlash(payload.path)), payload.path, payload.data, 0o600)
		if writeErr != nil {
			return Artifact{}, writeErr
		}
		*payload.hash = file.SHA256
		files = append(files, file)
	}
	if err := syncDirectoryContext(ctx, worldDirectory); err != nil {
		return Artifact{}, fmt.Errorf("sync World artifact directory: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest.Files = files
	manifest, manifestBytes, err := record.FinalizeManifest(manifest)
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

	finalPath := filepath.Join(store.Root, signatureDirectory(manifest.Outcome.FailureSignature, false))
	for {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		if err := renameNoReplace(staging, finalPath); err == nil {
			break
		} else if !errors.Is(err, os.ErrExist) {
			return Artifact{}, fmt.Errorf("publish artifact: %w", err)
		}
		existing, openErr := Open(finalPath)
		if openErr != nil {
			return Artifact{}, fmt.Errorf("existing artifact %s failed validation: %w", finalPath, openErr)
		}
		if existing.Manifest.Outcome.FailureSignature == manifest.Outcome.FailureSignature {
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
		completePath := filepath.Join(store.Root, signatureDirectory(manifest.Outcome.FailureSignature, true))
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

func artifactStoredBytes(manifest record.Manifest, manifestBytes uint64) (uint64, error) {
	total := manifestBytes
	for _, file := range manifest.Files {
		if uint64(file.Size) > ^uint64(0)-total {
			return 0, errors.New("artifact byte count overflows uint64")
		}
		total += uint64(file.Size)
	}
	return total, nil
}

func signatureDirectory(signature record.SHA256, complete bool) string {
	hex := strings.TrimPrefix(string(signature), "sha256:")
	if !complete && len(hex) >= 32 {
		hex = hex[:32]
	}
	return "sha256-" + hex
}

func copyPayload(ctx context.Context, source, destination, relativePath string, mode os.FileMode) (record.File, error) {
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
