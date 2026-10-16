// Package installation resolves explicitly pinned WASM operator installations.
package installation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

const Schema = "gomad-wasm.installation/v1"
const Environment = "GOMAD_WASM_INSTALLATION"
const Sibling = "gomad-wasm.json"
const maximumManifestBytes = 16 << 20

type Manifest struct {
	Schema                      string          `json:"schema"`
	Options                     backend.Options `json:"options"`
	RuntimeImplementationSHA256 string          `json:"runtime_implementation_sha256,omitempty"`
}

type Spec struct {
	Executable, ExplicitManifest, EnvironmentManifest string
}

type Description struct {
	Path     string        `json:"path"`
	Source   string        `json:"source"`
	SHA256   record.SHA256 `json:"sha256"`
	Manifest Manifest      `json:"manifest"`
}

func Resolve(spec Spec) (Description, error) {
	path, source := spec.ExplicitManifest, "explicit"
	if path == "" {
		path, source = spec.EnvironmentManifest, "environment"
	}
	if path == "" {
		if spec.Executable == "" {
			return Description{}, errors.New("WASM executable path is required for sibling discovery")
		}
		path, source = filepath.Join(filepath.Dir(spec.Executable), Sibling), "executable-sibling"
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Description{}, err
	}
	data, err := hostfs.ReadBounded(absolute, maximumManifestBytes)
	if err != nil {
		return Description{}, fmt.Errorf("read WASM installation: %w", err)
	}
	var manifest Manifest
	if err := decode(data, &manifest); err != nil {
		return Description{}, fmt.Errorf("decode WASM installation: %w", err)
	}
	if err := validate(manifest); err != nil {
		return Description{}, err
	}
	if err := verifyExecutable(manifest.Options.HelperPath, manifest.Options.HelperSHA256); err != nil {
		return Description{}, fmt.Errorf("installed WASM helper: %w", err)
	}
	return Description{absolute, source, record.HashBytes(data), manifest}, nil
}

func (description Description) Provider() (*backend.Provider, error) {
	if err := validate(description.Manifest); err != nil {
		return nil, err
	}
	return backend.New(description.Manifest.Options)
}

// Build pins existing host tools. It does not install a compiler or change the
// guest configuration; preparation will validate selected compiler inputs.
func Build(options backend.Options) (Manifest, error) {
	var err error
	for _, path := range []*string{&options.CompilerPath, &options.HelperPath, &options.CacheRoot} {
		if *path == "" {
			return Manifest{}, errors.New("compiler, helper, and cache paths are required")
		}
		*path, err = filepath.Abs(*path)
		if err != nil {
			return Manifest{}, err
		}
	}
	if options.CompileCacheRoot != "" {
		options.CompileCacheRoot, err = filepath.Abs(options.CompileCacheRoot)
		if err != nil {
			return Manifest{}, err
		}
	}
	if options.RuntimeRoot != "" {
		options.RuntimeRoot, err = filepath.Abs(options.RuntimeRoot)
		if err != nil {
			return Manifest{}, err
		}
	}
	options.CompilerSHA256, err = executableDigest(options.CompilerPath)
	if err != nil {
		return Manifest{}, err
	}
	options.HelperSHA256, err = executableDigest(options.HelperPath)
	if err != nil {
		return Manifest{}, err
	}
	options.ModelSHA256 = wasi.ImplementationSHA256()
	options.Engine = wasi.PinnedEngine()
	manifest := Manifest{Schema: Schema, Options: options}
	if options.Config.Profile == wasi.CooperativeProfile {
		manifest.RuntimeImplementationSHA256 = toolchain.ImplementationSHA256()
	}
	return manifest, validate(manifest)
}

func Write(path string, manifest Manifest) (retErr error) {
	if err := validate(manifest); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if len(data)+1 > maximumManifestBytes {
		return errors.New("WASM installation manifest exceeds its size bound")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	_, err = file.Write(append(data, '\n'))
	return err
}

func ReadConfig(path string) (wasi.Config, error) {
	data, err := hostfs.ReadBounded(path, maximumManifestBytes)
	if err != nil {
		return wasi.Config{}, err
	}
	var config wasi.Config
	err = decode(data, &config)
	return config, err
}

func DefaultConfig(profile string) wasi.Config {
	config := wasi.Config{Profile: profile, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: wasi.Limits{OutputBytes: 8 << 20, TranscriptBytes: 64 << 20, Calls: 1000000, Descriptors: 64, Files: 1000, FilesystemBytes: 64 << 20, PendingEvents: 4096}}
	if profile == wasi.CooperativeProfile {
		config.Clock.ReadStepNanos = 0
	}
	return config
}

func validate(manifest Manifest) error {
	options := manifest.Options
	if manifest.Schema != Schema || !filepath.IsAbs(options.CompilerPath) || !filepath.IsAbs(options.HelperPath) || !filepath.IsAbs(options.CacheRoot) {
		return errors.New("WASM installation schema and absolute tool/cache paths are required")
	}
	if (options.Config.NetworkV2 != nil || options.Config.Storage != nil) && options.Config.Profile != wasi.CooperativeProfile {
		return errors.New("operator network and storage require the cooperative profile")
	}
	if options.Config.Profile == wasi.CooperativeProfile {
		if !filepath.IsAbs(options.RuntimeRoot) || manifest.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() {
			return errors.New("pinned cooperative runtime installation is required")
		}
	} else if options.RuntimeRoot != "" || manifest.RuntimeImplementationSHA256 != "" {
		return errors.New("stock installation cannot select a cooperative runtime")
	}
	_, err := backend.New(options)
	return err
}

func verifyExecutable(path, expected string) error {
	digest, err := executableDigest(path)
	if err != nil {
		return err
	}
	if digest != expected {
		return errors.New("executable identity changed")
	}
	return nil
}

func executableDigest(path string) (string, error) {
	data, err := hostfs.ReadBounded(path, 512<<20)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return "", fmt.Errorf("not a regular executable: %s", path)
	}
	return string(record.HashBytes(data)), nil
}

func decode(data []byte, value any) error {
	trailing, err := strictjson.DecodeFirst(data, value)
	if err != nil {
		return err
	}
	if trailing {
		return errors.New("WASM installation input has trailing JSON")
	}
	return nil
}
