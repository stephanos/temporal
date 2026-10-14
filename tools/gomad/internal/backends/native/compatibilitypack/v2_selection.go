package compatibility

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

type ValidatedPack struct {
	pack   Pack
	digest string
}

type selectedPack struct {
	ValidatedPack
}

// LoadPacks returns the packs the build selects from: the embedded ones and
// those ExternalPacksEnvironment names, under the same validation.
func LoadPacks() ([]ValidatedPack, error) {
	return loadPacksV2()
}

// ExternalPacksEnvironment names a directory of reviewed packs that a module
// outside this repository owns. Its packs are loaded next to the embedded ones
// under the same validation; each selected pack's ID and digest enter the
// target identity, so replay and resume without the same pack fail closed.
const ExternalPacksEnvironment = "GOMAD3_COMPATIBILITY_PACKS"

func loadPacksV2() ([]ValidatedPack, error) {
	entries, err := packFiles.ReadDir("packs")
	if err != nil {
		return nil, fmt.Errorf("read compatibility packs: %w", err)
	}
	if len(entries) == 0 || len(entries) > maximumPackRules {
		return nil, errors.New("compatibility pack count is invalid")
	}
	result := make([]ValidatedPack, 0, len(entries))
	for _, entry := range entries {
		contents, err := packFiles.ReadFile("packs/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read compatibility pack %s: %w", entry.Name(), err)
		}
		candidate, err := loadPackEntry(entry.Name(), entry.IsDir(), contents)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	external, err := loadExternalPacks(os.Getenv(ExternalPacksEnvironment))
	if err != nil {
		return nil, err
	}
	for _, candidate := range external {
		if slices.ContainsFunc(result, func(embedded ValidatedPack) bool { return embedded.pack.ID == candidate.pack.ID }) {
			return nil, fmt.Errorf("external compatibility pack %s collides with an embedded pack", candidate.pack.ID)
		}
		result = append(result, candidate)
	}
	if len(result) > maximumPackRules {
		return nil, errors.New("compatibility pack count is invalid")
	}
	return result, nil
}

func loadPackEntry(name string, directory bool, contents []byte) (ValidatedPack, error) {
	if directory || !strings.HasSuffix(name, ".json") {
		return ValidatedPack{}, fmt.Errorf("compatibility pack entry %s is invalid", name)
	}
	candidate, err := LoadPack(contents)
	if err != nil {
		return ValidatedPack{}, fmt.Errorf("compatibility pack %s: %w", name, err)
	}
	if name != candidate.pack.ID+".json" {
		return ValidatedPack{}, fmt.Errorf("compatibility pack %s does not match ID %s", name, candidate.pack.ID)
	}
	return candidate, nil
}

// loadExternalPacks reads every pack in directory. An unset directory loads
// none; a set one must be an absolute, clean path to an existing directory, so
// a typo cannot silently fall back to the embedded packs alone.
func loadExternalPacks(directory string) ([]ValidatedPack, error) {
	if directory == "" {
		return nil, nil
	}
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return nil, fmt.Errorf("%s=%q must be an absolute, clean path", ExternalPacksEnvironment, directory)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read external compatibility packs: %w", err)
	}
	if len(entries) > maximumPackRules {
		return nil, errors.New("external compatibility pack count is invalid")
	}
	result := make([]ValidatedPack, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return nil, fmt.Errorf("external compatibility pack entry %s is not a regular file", entry.Name())
		}
		var contents []byte
		if !entry.IsDir() {
			contents, err = os.ReadFile(filepath.Join(directory, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("read external compatibility pack %s: %w", entry.Name(), err)
			}
		}
		candidate, err := loadPackEntry(entry.Name(), entry.IsDir(), contents)
		if err != nil {
			return nil, fmt.Errorf("external %w", err)
		}
		result = append(result, candidate)
	}
	return result, nil
}

// LoadPackDirectory returns the packs in directory, such as an authoring
// root's packs/, under the validation external packs receive. A missing
// directory holds no packs.
func LoadPackDirectory(directory string) ([]ValidatedPack, error) {
	if _, err := os.Stat(directory); errors.Is(err, os.ErrNotExist) {
		return []ValidatedPack{}, nil
	}
	return loadExternalPacks(directory)
}

func SelectPacks(packs []ValidatedPack, packages []Package) (Selection, error) {
	return SelectPacksForPlatform(packs, packages, runtime.GOOS+"/"+runtime.GOARCH)
}

// PinEvidence reads the same validated packs used by target preparation.
func PinEvidence() ([]PackEvidence, error) {
	packs, err := loadPacksV2()
	if err != nil {
		return nil, err
	}
	evidence := make([]PackEvidence, len(packs))
	for index, pack := range packs {
		evidence[index] = projectPackEvidence(selectedPack{ValidatedPack: pack})
	}
	return evidence, nil
}
