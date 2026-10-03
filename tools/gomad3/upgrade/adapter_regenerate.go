package upgrade

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func RegenerateAdapter(root, module, version, approval string, stdout io.Writer) (int, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return 2, err
	}
	checkoutRoot := filepath.Clean(filepath.Join(absRoot, "..", ".."))
	if _, err := os.Lstat(adapterPublicationMarker(checkoutRoot)); err == nil {
		if err := withAdapterPublicationLock(checkoutRoot, func() error { return recoverAdapterPublication(checkoutRoot) }); err != nil {
			return 1, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return 1, err
	}
	descriptor, err := gomadversion.Load(absRoot)
	if err != nil {
		return 1, err
	}
	oldVersion, oldSum := "", ""
	for _, adapter := range descriptor.Adapters {
		if adapter.Module == module {
			oldVersion, oldSum = adapter.Version, adapter.Sum
			break
		}
	}
	if oldVersion == "" {
		return 2, fmt.Errorf("adapter %s is not in the version descriptor", module)
	}
	return runAdapterRegenerationForPin(absRoot, checkoutRoot, module, version, approval, oldVersion, oldSum, stdout)
}

func runAdapterRegenerationForPin(absRoot, checkoutRoot, module, version, approval, oldVersion, oldSum string, stdout io.Writer) (status int, retErr error) {
	private, err := os.MkdirTemp("", "gomad-adapter-download-*")
	if err != nil {
		return 3, err
	}
	defer func() {
		if err := removeAdapterPrivate(private); err != nil {
			status, retErr = 1, errors.Join(retErr, err)
		}
	}()
	oldRoot, oldDownloadedSum, oldGoModSum, err := downloadAdapterModule(module, oldVersion, private)
	if err != nil {
		return 3, err
	}
	if oldDownloadedSum != oldSum {
		return 1, errors.New("pinned adapter module sum disagrees with the downloaded module")
	}
	newRoot, newSum, newGoModSum, err := downloadAdapterModule(module, version, private)
	if err != nil {
		return 3, err
	}
	result, err := deterministicio.RegenerateAdapter(module, version, newSum, newRoot, filepath.Join(private, "replacement"))
	if err != nil {
		return 1, err
	}
	result.PreparedSourceSets, err = preparedAdapterSourceSets(result)
	if err != nil {
		return 1, err
	}
	digest, err := adapterRegenerationApproval(result)
	if err != nil {
		return 1, err
	}
	if approval == "" {
		if err := writeAdapterRegenerationReview(stdout, oldRoot, result, digest); err != nil {
			return 1, err
		}
		return 0, nil
	}
	if approval != digest {
		return 1, errors.New("adapter regeneration approval does not match changed upstream source and proposed anchors")
	}
	var staleReport bytes.Buffer
	if err := reportStaleAdapterPacks(&staleReport, module, version); err != nil {
		return 1, err
	}
	files, snapshot, err := stageAdapterRegeneration(absRoot, result, oldVersion, oldSum, oldGoModSum, newGoModSum, filepath.Join(private, "cache"))
	if err != nil {
		return 1, err
	}
	if err := publishAdapterFilesWithSnapshot(checkoutRoot, files, snapshot, adapterRegenerationInputTrees); err != nil {
		return 1, err
	}
	if _, err := fmt.Fprintf(stdout, "regenerated %s@%s with approval %s\n", module, version, digest); err != nil {
		return 1, err
	}
	if _, err := io.Copy(stdout, &staleReport); err != nil {
		return 1, err
	}
	return 0, nil
}

func removeAdapterPrivate(root string) error {
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(root)
}

func downloadAdapterModule(module, version, private string) (directory, moduleSum, goModSum string, retErr error) {
	cache := filepath.Join(private, "cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return "", "", "", err
	}
	command := exec.CommandContext(context.Background(), "go", "mod", "download", "-json", module+"@"+version)
	command.Dir = private
	command.Env = append(os.Environ(), "GOMODCACHE="+cache, "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", "", "", fmt.Errorf("download %s@%s: %w: %s", module, version, err, output)
	}
	var result struct{ Path, Version, Sum, GoModSum, Dir, Error string }
	if err := json.Unmarshal(output, &result); err != nil {
		return "", "", "", fmt.Errorf("decode module download: %w", err)
	}
	if result.Error != "" || result.Path != module || result.Version != version || result.Sum == "" || result.GoModSum == "" || result.Dir == "" {
		return "", "", "", fmt.Errorf("module download identity mismatch or failure: %s", result.Error)
	}
	return result.Dir, result.Sum, result.GoModSum, nil
}

func preparedAdapterSourceSets(result deterministicio.AdapterRegeneration) (map[string]string, error) {
	prepared := make(map[string]string, 2)
	for _, platform := range []struct{ host, goos, goarch string }{{"darwin/arm64", "darwin", "arm64"}, {"linux/amd64", "linux", "amd64"}} {
		context := build.Default
		context.GOOS, context.GOARCH, context.CgoEnabled = platform.goos, platform.goarch, false
		context.BuildTags = []string{"gomad", "hashicorpmetrics", "integration", "test_dep"}
		packageDir := filepath.Join(result.ReplacementRoot, strings.TrimPrefix(result.PreparedPackage, result.Module))
		pkg, err := context.ImportDir(packageDir, 0)
		if err != nil {
			return nil, fmt.Errorf("select %s source set on %s: %w", result.PreparedPackage, platform.host, err)
		}
		names := append([]string{}, pkg.GoFiles...)
		names = append(names, pkg.CgoFiles...)
		slices.Sort(names)
		sources := make([]compatibility.Source, 0, len(names))
		for _, name := range names {
			contents, err := os.ReadFile(filepath.Join(packageDir, name))
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(contents)
			sources = append(sources, compatibility.Source{Name: name, SHA256: fmt.Sprintf("sha256:%x", digest)})
		}
		prepared[platform.host] = compatibility.DigestSources(sources)
	}
	return prepared, nil
}

func adapterRegenerationApproval(result deterministicio.AdapterRegeneration) (string, error) {
	encoded, err := canonicaljson.CanonicalJSON(result)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("gomad3.adapter-regeneration-review/v1\x00"))
	_, _ = hash.Write(encoded)
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func writeAdapterRegenerationReview(output io.Writer, oldRoot string, result deterministicio.AdapterRegeneration, digest string) error {
	if _, err := fmt.Fprintf(output, "Adapter: %s@%s\nModule sum: %s\nOriginal inventory: %s\nReplacement inventory: %s\n", result.Module, result.Version, result.Sum, result.OriginalInventorySHA256, result.ReplacementInventorySHA256); err != nil {
		return err
	}
	for _, source := range result.Sources {
		old, err := os.ReadFile(filepath.Join(oldRoot, filepath.FromSlash(source.Path)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if _, err := fmt.Fprintf(output, "\nSource: %s\nOld source: %s\nNew source: %s\nNew replacement: %s\n", source.Path, source.OldSourceSHA256, source.SourceSHA256, source.ReplacementSHA256); err != nil {
			return err
		}
		if !bytes.Equal(old, source.Source) {
			if _, err := fmt.Fprintf(output, "Changed upstream source %s:\n%s\n", source.Path, source.Source); err != nil {
				return err
			}
		}
	}
	for _, host := range []string{"darwin/arm64", "linux/amd64"} {
		if _, err := fmt.Fprintf(output, "Prepared source set %s: %s\n", host, result.PreparedSourceSets[host]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(output, "Approval SHA-256: %s\n", digest); err != nil {
		return err
	}
	return reportStaleAdapterPacks(output, result.Module, result.Version)
}

func reportStaleAdapterPacks(output io.Writer, module, version string) error {
	packs, err := compatibility.PinEvidence()
	if err != nil {
		return err
	}
	for _, pack := range packs {
		stale := false
		for _, item := range pack.Activation {
			stale = stale || item.Path == module && item.Version != version || item.Adapter != nil && item.Adapter.Module == module && item.Adapter.Version != version
		}
		for _, rule := range pack.Rules {
			item := rule.Module
			stale = stale || item.Path == module && item.Version != version || item.Adapter != nil && item.Adapter.Module == module && item.Adapter.Version != version
		}
		if stale {
			if _, err := fmt.Fprintf(output, "Stale adapter-bound pack: %s\n", pack.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

type goPinEdit struct {
	start, end int
	value      string
}

func editAdapterPins(contents []byte, result deterministicio.AdapterRegeneration, oldVersion, oldSum string) ([]byte, error) {
	replacements, err := adapterPinReplacements(result, oldVersion, oldSum)
	if err != nil {
		return nil, err
	}
	edits, hits, hostHits, err := adapterPinEdits(contents, result.PreparedSourceSets, replacements)
	if err != nil {
		return nil, err
	}
	for old := range replacements {
		libcVersionOrSum := result.Module == "modernc.org/libc" && (old == oldVersion || old == oldSum)
		if hits[old] != 1 && (!libcVersionOrSum || hits[old] != 0) {
			return nil, fmt.Errorf("missing pin or duplicate pin %s: %d", old, hits[old])
		}
	}
	for host := range result.PreparedSourceSets {
		if hostHits[host] != 1 {
			return nil, fmt.Errorf("missing prepared source-set pin for %s: %d", host, hostHits[host])
		}
	}
	slices.SortFunc(edits, func(a, b goPinEdit) int { return b.start - a.start })
	updated := append([]byte{}, contents...)
	for _, edit := range edits {
		updated = append(append(append([]byte{}, updated[:edit.start]...), edit.value...), updated[edit.end:]...)
	}
	return updated, nil
}

func adapterPinReplacements(result deterministicio.AdapterRegeneration, oldVersion, oldSum string) (map[string]string, error) {
	replacements := map[string]string{}
	add := func(old, next string) error {
		if old == "" || old == next {
			return nil
		}
		if other, found := replacements[old]; found && other != next {
			return fmt.Errorf("conflicting proposed pin for %s", old)
		}
		replacements[old] = next
		return nil
	}
	for _, pair := range [][2]string{{result.OldOriginalInventorySHA256, result.OriginalInventorySHA256}, {result.OldReplacementInventorySHA256, result.ReplacementInventorySHA256}, {oldVersion, result.Version}, {oldSum, result.Sum}} {
		if err := add(pair[0], pair[1]); err != nil {
			return nil, err
		}
	}
	for _, source := range result.Sources {
		if err := add(source.OldSourceSHA256, source.SourceSHA256); err != nil {
			return nil, err
		}
		if err := add(source.OldReplacementSHA256, source.ReplacementSHA256); err != nil {
			return nil, err
		}
	}
	return replacements, nil
}

func adapterPinEdits(contents []byte, prepared map[string]string, replacements map[string]string) (edits []goPinEdit, hits, hostHits map[string]int, retErr error) {
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, "adapter.go", contents, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}
	hits = map[string]int{}
	hostHits = map[string]int{}
	edits = []goPinEdit{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		if field, ok := node.(*ast.KeyValueExpr); ok {
			key, keyOK := field.Key.(*ast.BasicLit)
			value, valueOK := field.Value.(*ast.BasicLit)
			if keyOK && valueOK && key.Kind == token.STRING && value.Kind == token.STRING {
				host, _ := strconv.Unquote(key.Value)
				if next, found := prepared[host]; found {
					hostHits[host]++
					edits = append(edits, goPinEdit{start: files.Position(value.Pos()).Offset, end: files.Position(value.End()).Offset, value: strconv.Quote(next)})
					return false
				}
			}
		}
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		old, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		if next, found := replacements[old]; found {
			hits[old]++
			edits = append(edits, goPinEdit{start: files.Position(literal.Pos()).Offset, end: files.Position(literal.End()).Offset, value: strconv.Quote(next)})
		}
		return true
	})
	return edits, hits, hostHits, nil
}

func stageAdapterRegeneration(root string, result deterministicio.AdapterRegeneration, oldVersion, oldSum, oldGoModSum, newGoModSum, privateCache string) (files []adapterPublicationFile, snapshot map[string]publicationContent, retErr error) {
	work, err := os.MkdirTemp("", "gomad-adapter-stage-*")
	if err != nil {
		return nil, nil, err
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(work)) }()
	stage := filepath.Join(work, adapterRegenerationInputTrees[0])
	checkoutRoot := filepath.Clean(filepath.Join(root, "..", ".."))
	snapshot, err = copyAdapterStageInputs(root, checkoutRoot, work)
	if err != nil {
		return nil, nil, err
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		contents, err := os.ReadFile(filepath.Join(checkoutRoot, name))
		if err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(filepath.Join(work, name), contents, 0o600); err != nil {
			return nil, nil, err
		}
		snapshot[name] = publicationContent{Present: true, Bytes: contents}
	}
	cacheDir := filepath.Join(stage, ".toolchain")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, nil, err
	}
	if err := os.Symlink(filepath.Join(root, ".toolchain", "generator-cache"), filepath.Join(cacheDir, "generator-cache")); err != nil {
		return nil, nil, err
	}
	adapterFile := filepath.Join(stage, "deterministicio", result.SourceFile)
	contents, err := os.ReadFile(adapterFile)
	if err != nil {
		return nil, nil, err
	}
	updated, err := editAdapterPins(contents, result, oldVersion, oldSum)
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(adapterFile, updated, 0o644); err != nil {
		return nil, nil, err
	}
	if err := editAdapterDescriptor(filepath.Join(stage, "toolchain", "version", "version.json"), result.Module, oldVersion, oldSum, result.Version, result.Sum); err != nil {
		return nil, nil, err
	}
	if err := editAdapterFixtures(filepath.Join(stage, "deterministicio", "testdata"), result.Module, oldVersion, oldSum, oldGoModSum, result.Version, result.Sum, newGoModSum, privateCache); err != nil {
		return nil, nil, err
	}
	generate := exec.CommandContext(context.Background(), "make", "-C", stage, "generate")
	generate.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err := generate.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("generate staged adapter artifacts: %w\n%s", err, output)
	}
	validate := exec.CommandContext(context.Background(), "make", "-C", stage, "validate-toolchain", "validate-qualification")
	validate.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err = validate.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("validate staged adapter artifacts: %w\n%s", err, output)
	}
	compile := exec.CommandContext(context.Background(), "go", "test", "-tags", "test_dep", "-run", "^$", "./deterministicio")
	compile.Dir = stage
	compile.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err = compile.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("compile staged adapter: %w\n%s", err, output)
	}
	files = []adapterPublicationFile{}
	for _, subtree := range adapterRegenerationInputTrees[:2] {
		err = collectAdapterStageChanges(filepath.Join(work, subtree), subtree, snapshot, &files)
		if err != nil {
			return nil, nil, err
		}
	}
	return files, snapshot, nil
}

func copyAdapterStageInputs(root, checkoutRoot, work string) (map[string]publicationContent, error) {
	snapshot := map[string]publicationContent{}
	for _, tree := range adapterRegenerationInputTrees {
		source := filepath.Join(checkoutRoot, tree)
		if tree == adapterRegenerationInputTrees[0] {
			source = root
		}
		copied, err := copyAdapterStageTree(source, filepath.Join(work, tree))
		if err != nil {
			return nil, err
		}
		for relative, content := range copied {
			snapshot[filepath.Join(tree, relative)] = content
		}
	}
	return snapshot, nil
}

func collectAdapterStageChanges(stage, prefix string, snapshot map[string]publicationContent, files *[]adapterPublicationFile) error {
	err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if entry.IsDir() {
			if filepath.Base(path) == ".toolchain" || filepath.Base(path) == ".bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("staged adapter output is not a regular file: %s", path)
		}
		local, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		relative := filepath.Join(prefix, local)
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		newContent := publicationContent{Present: true, Bytes: contents}
		old := snapshot[relative]
		if !samePublicationContent(old, newContent) {
			*files = append(*files, adapterPublicationFile{Path: relative, Old: old, New: newContent})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for relative, old := range snapshot {
		if !strings.HasPrefix(relative, prefix+string(filepath.Separator)) {
			continue
		}
		local := strings.TrimPrefix(relative, prefix+string(filepath.Separator))
		if _, err := os.Stat(filepath.Join(stage, local)); errors.Is(err, os.ErrNotExist) {
			*files = append(*files, adapterPublicationFile{Path: relative, Old: old})
		} else if err != nil {
			return err
		}
	}
	return nil
}

func walkAdapterStageInputs(source string, visit func(path, relative string, entry fs.DirEntry) error) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if entry.IsDir() && (relative == ".toolchain" || relative == ".bin") {
			return filepath.SkipDir
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("adapter stage input is not a regular file: %s", relative)
		}
		return visit(path, relative, entry)
	})
}

func snapshotAdapterStageTree(source string) (map[string]publicationContent, error) {
	snapshot := map[string]publicationContent{}
	err := walkAdapterStageInputs(source, func(path, relative string, entry fs.DirEntry) error {
		if entry.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[relative] = publicationContent{Present: true, Bytes: contents}
		return nil
	})
	return snapshot, err
}

func copyAdapterStageTree(source, destination string) (map[string]publicationContent, error) {
	snapshot := map[string]publicationContent{}
	err := walkAdapterStageInputs(source, func(path, relative string, entry fs.DirEntry) error {
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(destination, relative), 0o700)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destination, relative), contents, info.Mode().Perm()); err != nil {
			return err
		}
		snapshot[relative] = publicationContent{Present: true, Bytes: contents}
		return nil
	})
	return snapshot, err
}

func editAdapterDescriptor(path, module, oldVersion, oldSum, version, sum string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var descriptor gomadversion.Descriptor
	if err := json.Unmarshal(contents, &descriptor); err != nil {
		return err
	}
	hits := 0
	for index := range descriptor.Adapters {
		adapter := &descriptor.Adapters[index]
		if adapter.Module != module {
			continue
		}
		if adapter.Version != oldVersion || adapter.Sum != oldSum {
			return errors.New("adapter descriptor changed since planning")
		}
		adapter.Version, adapter.Sum = version, sum
		hits++
	}
	if hits != 1 {
		return fmt.Errorf("adapter descriptor has %d entries for %s", hits, module)
	}
	updated, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(updated, '\n'), 0o644)
}

func editAdapterFixtures(root, module, oldVersion, oldSum, oldGoModSum, version, sum, newGoModSum, privateCache string) error {
	var changedModules []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "go.mod" && entry.Name() != "go.sum" {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated := string(contents)
		if entry.Name() == "go.mod" {
			updated = strings.ReplaceAll(updated, module+" "+oldVersion, module+" "+version)
		} else {
			updated = strings.ReplaceAll(updated, module+" "+oldVersion+" "+oldSum, module+" "+version+" "+sum)
			updated = strings.ReplaceAll(updated, module+" "+oldVersion+"/go.mod "+oldGoModSum, module+" "+version+"/go.mod "+newGoModSum)
		}
		if updated == string(contents) {
			return nil
		}
		if entry.Name() == "go.mod" {
			changedModules = append(changedModules, filepath.Dir(path))
		}
		return os.WriteFile(path, []byte(updated), 0o644)
	})
	if err != nil {
		return err
	}
	for _, directory := range changedModules {
		command := exec.CommandContext(context.Background(), "go", "mod", "tidy")
		command.Dir = directory
		command.Env = append(os.Environ(), "GOMODCACHE="+privateCache, "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local")
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("refresh staged adapter fixture %s: %w\n%s", directory, err, output)
		}
	}
	return nil
}
