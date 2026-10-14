package backend

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

// PreparedIdentity exposes already frozen build inventories to qualification,
// without giving qualification a second source-discovery or compiler owner.
type PreparedIdentity struct {
	WorkloadSourceSHA256, RuntimeSHA256 record.SHA256
}

func (p *Provider) QualificationIdentity(prepared target.Prepared) (PreparedIdentity, error) {
	if prepared.Backend == nil || len(prepared.BackendPayloads) != 1 {
		return PreparedIdentity{}, errors.New("prepared WASM provenance is required")
	}
	pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
	if err != nil {
		return PreparedIdentity{}, err
	}
	if err := p.validateProvenance(pr, prepared); err != nil {
		return PreparedIdentity{}, err
	}
	compilerRoot := filepath.Dir(filepath.Dir(p.options.CompilerPath))
	within := func(path, root string) bool {
		if root == "" {
			return false
		}
		relative, err := filepath.Rel(root, path)
		return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
	}
	var source, runtime []sourceIdentity
	for _, entry := range pr.Sources {
		if err := checkFile(entry.Path, string(entry.SHA256), 512<<20); err != nil {
			return PreparedIdentity{}, err
		}
		if within(entry.Path, compilerRoot) || within(entry.Path, p.options.RuntimeRoot) || within(entry.Path, filepath.Join(p.options.CacheRoot, "runtime-overlay")) {
			runtime = append(runtime, entry)
		} else {
			source = append(source, entry)
		}
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		return PreparedIdentity{}, err
	}
	identity := PreparedIdentity{WorkloadSourceSHA256: record.HashBytes(encoded)}
	encoded, err = json.Marshal(runtime)
	if err != nil {
		return PreparedIdentity{}, err
	}
	identity.RuntimeSHA256 = record.HashBytes(encoded)
	return identity, nil
}
