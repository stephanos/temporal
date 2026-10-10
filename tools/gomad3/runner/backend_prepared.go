package runner

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func loadPreparedBackend(campaignPath, modulePath string, prepared *target.Prepared) (retErr error) {
	if err := record.ValidateBackendMetadata(*prepared.Backend); err != nil {
		return err
	}
	root, err := os.OpenRoot(campaignPath)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	for _, reference := range record.BackendReferences(*prepared.Backend) {
		name := path.Join(path.Dir(modulePath), reference.File)
		file, err := root.Open(filepath.FromSlash(name))
		if err != nil {
			return err
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() < 0 || uint64(info.Size()) != uint64(reference.Bytes) {
			return errors.Join(errors.New("prepared backend payload mode or size mismatch"), statErr, file.Close())
		}
		data, readErr := io.ReadAll(io.LimitReader(file, int64(reference.Bytes)+1))
		if err := errors.Join(readErr, file.Close()); err != nil {
			return err
		}
		prepared.BackendPayloads = append(prepared.BackendPayloads, target.BackendPayload{Reference: reference, Data: data})
	}
	return prepared.ValidateBackendPayloads()
}
