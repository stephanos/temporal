package pinimpact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	resolveTimeout     = 10 * time.Minute
	resolveOutputLimit = 64 << 20
)

// module is the identity view of one go.mod and go.sum: what the module
// requires and replaces, the zip sums it records, and the versions its module
// graph selects.
type moduleState struct {
	goDirective string
	required    map[string][]string
	replaced    map[string]bool
	sums        map[string]map[string][]string
	selected    map[string]string
}

type moduleMatch struct {
	ok     bool
	absent bool
	reason string
}

func loadModule(ctx context.Context, name string, files ModuleFiles, resolver Resolver) (moduleState, error) {
	parsed, err := parseModule(files)
	if err != nil {
		return moduleState{}, &InputError{Err: fmt.Errorf("%s: %w", name, err)}
	}
	parsed.selected, err = resolver.Resolve(ctx, files)
	if err != nil {
		return moduleState{}, fmt.Errorf("resolve %s module graph: %w", name, err)
	}
	return parsed, nil
}

func parseModule(files ModuleFiles) (moduleState, error) {
	parsed, err := modfile.Parse("go.mod", files.GoMod, nil)
	if err != nil {
		return moduleState{}, fmt.Errorf("parse go.mod: %w", err)
	}
	if parsed.Module == nil {
		return moduleState{}, errors.New("go.mod has no module directive")
	}
	result := moduleState{required: map[string][]string{}, replaced: map[string]bool{}, sums: map[string]map[string][]string{}}
	if parsed.Go != nil {
		result.goDirective = parsed.Go.Version
	}
	for _, requirement := range parsed.Require {
		result.required[requirement.Mod.Path] = append(result.required[requirement.Mod.Path], requirement.Mod.Version)
	}
	for _, replacement := range parsed.Replace {
		result.replaced[replacement.Old.Path] = true
	}
	for number, line := range strings.Split(string(files.GoSum), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return moduleState{}, fmt.Errorf("go.sum line %d is malformed", number+1)
		}
		path, version, sum := fields[0], fields[1], fields[2]
		if strings.HasSuffix(version, "/go.mod") {
			continue
		}
		if err := module.Check(path, version); err != nil {
			return moduleState{}, fmt.Errorf("go.sum line %d: %w", number+1, err)
		}
		if result.sums[path] == nil {
			result.sums[path] = map[string][]string{}
		}
		result.sums[path][version] = append(result.sums[path][version], sum)
	}
	return result, nil
}

func (target moduleState) requires(path string) bool {
	return len(target.required[path]) != 0
}

// observed returns the version the module requires and its single recorded
// sum, each empty when absent or ambiguous.
func (target moduleState) observed(path string) (string, string) {
	versions := target.required[path]
	if len(versions) != 1 {
		return "", ""
	}
	sums := target.sums[path][versions[0]]
	if len(sums) != 1 {
		return versions[0], ""
	}
	return versions[0], sums[0]
}

// match reports whether the module pins path at exactly version and sum. It
// follows the adapter registry's check: a replacement of the path, a
// duplicated or different requirement, or any recorded sum other than the
// pinned one is a mismatch. It also requires the module graph to select the
// required version, which an untidy go.mod would not.
func (target moduleState) match(path, version, sum string) moduleMatch {
	if target.replaced[path] {
		return moduleMatch{reason: "candidate replaces " + path}
	}
	versions := target.required[path]
	switch {
	case len(versions) == 0:
		return moduleMatch{absent: true, reason: "candidate does not require " + path}
	case len(versions) > 1:
		return moduleMatch{reason: "candidate requires " + path + " more than once"}
	case versions[0] != version:
		return moduleMatch{reason: fmt.Sprintf("candidate requires %s@%s; pinned %s", path, versions[0], version)}
	}
	if selected := target.selected[path]; selected != version {
		return moduleMatch{reason: fmt.Sprintf("candidate module graph selects %s@%s although go.mod requires %s", path, selected, version)}
	}
	sums := target.sums[path][version]
	if len(sums) == 0 {
		return moduleMatch{reason: fmt.Sprintf("candidate go.sum has no sum for %s@%s", path, version)}
	}
	for _, recorded := range sums {
		if recorded != sum {
			return moduleMatch{reason: fmt.Sprintf("candidate go.sum records %s for %s@%s; pinned %s", recorded, path, version, sum)}
		}
	}
	return moduleMatch{ok: true}
}

// GoResolver resolves module graphs with the go command in a scratch copy of
// the module and a private module cache. Inside the target module, the go
// command would rewrite its go.mod and go.sum.
type GoResolver struct {
	command     string
	environment []string
	cache       string
}

// NewGoResolver returns a resolver that runs goCommand with environment, minus
// the settings it owns. Proxy and checksum settings pass through unchanged.
func NewGoResolver(goCommand string, environment []string) (*GoResolver, error) {
	if !filepath.IsAbs(goCommand) {
		return nil, fmt.Errorf("go command %q must be an absolute path", goCommand)
	}
	cache, err := os.MkdirTemp("", "gomad3-pin-impact-modcache-")
	if err != nil {
		return nil, fmt.Errorf("create private module cache: %w", err)
	}
	owned := map[string]bool{"GOFLAGS": true, "GOMODCACHE": true, "GOTOOLCHAIN": true, "GOWORK": true, "GOMADSEED": true, "GOMAD3_CHILD_SEED": true}
	filtered := make([]string, 0, len(environment)+4)
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !owned[name] {
			filtered = append(filtered, entry)
		}
	}
	filtered = append(filtered, "GOFLAGS=-mod=mod", "GOMODCACHE="+cache, "GOTOOLCHAIN=local", "GOWORK=off")
	return &GoResolver{command: goCommand, environment: filtered, cache: cache}, nil
}

// Close removes the private module cache, whose files the go command makes
// read-only.
func (resolver *GoResolver) Close() error {
	err := filepath.WalkDir(resolver.cache, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return nil
	})
	return errors.Join(err, os.RemoveAll(resolver.cache))
}

func (resolver *GoResolver) Resolve(ctx context.Context, files ModuleFiles) (_ map[string]string, retErr error) {
	moduleFile, err := absoluteLocalReplacements(files)
	if err != nil {
		return nil, &InputError{Err: err}
	}
	scratch, err := os.MkdirTemp("", "gomad3-pin-impact-module-")
	if err != nil {
		return nil, fmt.Errorf("create scratch module: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(scratch)) }()
	if err := os.WriteFile(filepath.Join(scratch, "go.mod"), moduleFile, 0o600); err != nil {
		return nil, fmt.Errorf("write scratch go.mod: %w", err)
	}
	if err := os.WriteFile(filepath.Join(scratch, "go.sum"), files.GoSum, 0o600); err != nil {
		return nil, fmt.Errorf("write scratch go.sum: %w", err)
	}
	result, err := hostexec.Run(ctx, hostexec.Request{
		Command: []string{resolver.command, "list", "-m", "-json", "all"}, Dir: scratch, Env: resolver.environment,
		Timeout: resolveTimeout, TerminateGrace: time.Second, OutputLimit: resolveOutputLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("run go list: %w", err)
	}
	if result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || result.WatchdogTimeout {
		return nil, fmt.Errorf("go list -m all failed: %s", strings.TrimSpace(string(result.Stderr.RawBytes)))
	}
	if result.Stdout.Truncated {
		return nil, errors.New("go list -m all output exceeds its bound")
	}
	return decodeModuleList(result.Stdout.RawBytes)
}

func decodeModuleList(output []byte) (map[string]string, error) {
	selected := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var listed struct {
			Path    string
			Version string
			Main    bool
			Error   *struct{ Err string }
		}
		if err := decoder.Decode(&listed); errors.Is(err, io.EOF) {
			return selected, nil
		} else if err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		if listed.Error != nil {
			return nil, fmt.Errorf("resolve %s: %s", listed.Path, listed.Error.Err)
		}
		if !listed.Main {
			selected[listed.Path] = listed.Version
		}
	}
}

// absoluteLocalReplacements rewrites relative directory replacements against
// the module's directory, so the scratch copy reads the same replacements.
func absoluteLocalReplacements(files ModuleFiles) ([]byte, error) {
	parsed, err := modfile.Parse("go.mod", files.GoMod, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}
	changed := false
	for _, replacement := range parsed.Replace {
		if replacement.New.Version != "" || filepath.IsAbs(replacement.New.Path) {
			continue
		}
		if files.Directory == "" {
			return nil, fmt.Errorf("local replacement of %s needs the module directory", replacement.Old.Path)
		}
		if err := parsed.AddReplace(replacement.Old.Path, replacement.Old.Version, filepath.Join(files.Directory, replacement.New.Path), ""); err != nil {
			return nil, fmt.Errorf("rewrite local replacement of %s: %w", replacement.Old.Path, err)
		}
		changed = true
	}
	if !changed {
		return files.GoMod, nil
	}
	return parsed.Format()
}
