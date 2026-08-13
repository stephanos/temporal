package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const maximumGateOutput = 16 << 20

type Options struct {
	Root             string
	Output           string
	BaselineManifest []byte
	CorpusReport     string
	Gates            []Gate
	Writer           io.Writer
}

type Gate struct {
	Name    string
	Command []string
}

type Dossier struct {
	Schema               string                   `json:"schema"`
	Qualified            bool                     `json:"qualified"`
	Version              VersionEvidence          `json:"version"`
	Host                 HostEvidence             `json:"host"`
	UpstreamPatch        PatchEvidence            `json:"upstream_patch"`
	BoundaryDiff         BoundaryDiff             `json:"boundary_manifest_diff"`
	InterceptionReport   InterceptionEvidence     `json:"interception_report"`
	OverlayCollision     OverlayCollisionEvidence `json:"overlay_collision_report"`
	Gates                []GateResult             `json:"gates"`
	RetainedCorpus       CorpusEvidence           `json:"retained_corpus"`
	MandatoryProbePolicy string                   `json:"mandatory_probe_policy"`
}

type VersionEvidence struct {
	GoVersion               string   `json:"go_version"`
	ArchiveSHA256           string   `json:"archive_sha256"`
	BoundaryManifestVersion string   `json:"boundary_manifest_version"`
	Patch                   string   `json:"patch"`
	SupportedPlatforms      []string `json:"supported_platforms"`
}

type HostEvidence struct {
	Platform  string `json:"platform"`
	Supported bool   `json:"supported"`
}

type PatchEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Diff   string `json:"diff"`
}

type BoundaryDiff struct {
	Status          string   `json:"status"`
	BaselineVersion string   `json:"baseline_version,omitempty"`
	CurrentVersion  string   `json:"current_version"`
	Added           []string `json:"added"`
	Removed         []string `json:"removed"`
	Changed         []string `json:"changed"`
}

type InterceptionEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Report string `json:"report"`
}

type OverlayCollisionEvidence struct {
	Checked       bool     `json:"checked"`
	ArchiveSHA256 string   `json:"archive_sha256"`
	OverlayPaths  []string `json:"overlay_paths"`
	Collisions    []string `json:"collisions"`
}

type GateResult struct {
	Name            string   `json:"name"`
	Command         []string `json:"command"`
	Status          string   `json:"status"`
	ExitCode        int      `json:"exit_code"`
	Output          string   `json:"output"`
	OutputSHA256    string   `json:"output_sha256"`
	OutputTruncated bool     `json:"output_truncated"`
}

type CorpusEvidence struct {
	Status string          `json:"status"`
	Path   string          `json:"path,omitempty"`
	SHA256 string          `json:"sha256,omitempty"`
	Report json.RawMessage `json:"report,omitempty"`
}

type boundaryManifest struct {
	ManifestVersion string              `json:"manifest_version"`
	Intercepts      []boundaryIntercept `json:"intercepts"`
}

type boundaryIntercept struct {
	Package             string            `json:"package"`
	Receiver            *boundaryReceiver `json:"receiver,omitempty"`
	Symbol              string            `json:"symbol"`
	Signature           string            `json:"signature"`
	Source              string            `json:"source,omitempty"`
	DeclarationSHA256   string            `json:"declaration_sha256,omitempty"`
	PackageSHA256       string            `json:"package_sha256,omitempty"`
	Operation           string            `json:"operation,omitempty"`
	Probe               string            `json:"probe,omitempty"`
	Disposition         string            `json:"disposition,omitempty"`
	Hook                string            `json:"hook"`
	DelegatedBoundary   string            `json:"delegated_boundary,omitempty"`
	Adapters            []string          `json:"adapters,omitempty"`
	ConformanceFixtures []string          `json:"conformance_fixtures,omitempty"`
	NegativeFixtures    []string          `json:"negative_fixtures,omitempty"`
	EscapeFixtures      []string          `json:"escape_fixtures,omitempty"`
}

type boundaryReceiver struct {
	Name    string `json:"name"`
	Pointer bool   `json:"pointer"`
}

func Run(ctx context.Context, options Options) error {
	if options.Root == "" || options.Output == "" {
		return errors.New("upgrade dossier requires root and output paths")
	}
	descriptor, err := gomadversion.Load(options.Root)
	if err != nil {
		return err
	}
	patchPath := filepath.Join(options.Root, filepath.FromSlash(descriptor.Patch))
	patch, err := os.ReadFile(patchPath)
	if err != nil {
		return fmt.Errorf("read upstream source diff: %w", err)
	}
	manifestPath := filepath.Join(options.Root, "boundary", "manifest.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read boundary manifest: %w", err)
	}
	difference, err := compareBoundaries(options.BaselineManifest, manifest)
	if err != nil {
		return err
	}
	reportName := "expected-intercepts-" + descriptor.GoVersion + ".txt"
	report, err := os.ReadFile(filepath.Join(options.Root, reportName))
	if err != nil {
		return fmt.Errorf("read interception report: %w", err)
	}
	collision, err := inspectOverlayCollision(options.Root, descriptor)
	if err != nil {
		return err
	}
	if len(collision.Collisions) != 0 {
		return fmt.Errorf("overlay collides with upstream source: %s", strings.Join(collision.Collisions, ", "))
	}
	corpus, err := loadCorpus(options.Root, options.CorpusReport)
	if err != nil {
		return err
	}
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	dossier := Dossier{
		Schema:    "gomadv3.upgrade-dossier/v1",
		Qualified: false,
		Version: VersionEvidence{
			GoVersion: descriptor.GoVersion, ArchiveSHA256: descriptor.Archive.SHA256,
			BoundaryManifestVersion: descriptor.BoundaryManifestVersion, Patch: descriptor.Patch,
			SupportedPlatforms: append([]string(nil), descriptor.SupportedPlatforms...),
		},
		Host:               HostEvidence{Platform: hostPlatform, Supported: slices.Contains(descriptor.SupportedPlatforms, hostPlatform)},
		UpstreamPatch:      PatchEvidence{Path: descriptor.Patch, SHA256: digest(patch), Diff: string(patch)},
		BoundaryDiff:       difference,
		InterceptionReport: InterceptionEvidence{Path: reportName, SHA256: digest(report), Report: string(report)},
		OverlayCollision:   collision,
		Gates:              []GateResult{}, RetainedCorpus: corpus,
		MandatoryProbePolicy: "runner-test and runtime gates must reject absent required semantic probes",
	}
	var gateFailure error
	seenGates := make(map[string]struct{}, len(options.Gates))
	for _, gate := range options.Gates {
		if gate.Name == "" || len(gate.Command) == 0 {
			return errors.New("upgrade dossier gate is incomplete")
		}
		if _, duplicate := seenGates[gate.Name]; duplicate {
			return fmt.Errorf("upgrade dossier gate is duplicated: %s", gate.Name)
		}
		seenGates[gate.Name] = struct{}{}
		if options.Writer != nil {
			fmt.Fprintf(options.Writer, "gomadv3 qualification gate: %s\n", gate.Name)
		}
		result, runErr := runGate(ctx, options.Root, gate, options.Writer)
		dossier.Gates = append(dossier.Gates, result)
		if runErr != nil {
			gateFailure = fmt.Errorf("qualification gate %s: %w", gate.Name, runErr)
			break
		}
	}
	dossier.Qualified = dossier.Host.Supported && gateFailure == nil
	if err := publish(options.Output, dossier); err != nil {
		return err
	}
	if gateFailure != nil {
		return gateFailure
	}
	if !dossier.Host.Supported {
		return fmt.Errorf("qualification host %s is unsupported", hostPlatform)
	}
	return nil
}

func runGate(ctx context.Context, root string, gate Gate, output io.Writer) (GateResult, error) {
	command := exec.CommandContext(ctx, gate.Command[0], gate.Command[1:]...)
	command.Dir = root
	buffer := &boundedBuffer{maximum: maximumGateOutput}
	command.Stdout = buffer
	command.Stderr = buffer
	err := command.Run()
	if output != nil && buffer.Len() != 0 {
		_, _ = output.Write(buffer.Bytes())
	}
	result := GateResult{
		Name: gate.Name, Command: append([]string(nil), gate.Command...), Output: buffer.String(),
		OutputSHA256: digest(buffer.Bytes()), OutputTruncated: buffer.truncated,
	}
	if err == nil {
		result.Status = "passed"
		return result, nil
	}
	result.Status = "failed"
	result.ExitCode = -1
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}

func compareBoundaries(baseline, current []byte) (BoundaryDiff, error) {
	currentManifest, err := decodeBoundary("current", current)
	if err != nil {
		return BoundaryDiff{}, err
	}
	result := BoundaryDiff{Status: "not-requested", CurrentVersion: currentManifest.ManifestVersion, Added: []string{}, Removed: []string{}, Changed: []string{}}
	if len(baseline) == 0 {
		return result, nil
	}
	baselineManifest, err := decodeBoundary("baseline", baseline)
	if err != nil {
		return BoundaryDiff{}, err
	}
	result.Status = "compared"
	result.BaselineVersion = baselineManifest.ManifestVersion
	oldEntries := indexBoundaryEntries(baselineManifest.Intercepts)
	newEntries := indexBoundaryEntries(currentManifest.Intercepts)
	for target, oldEntry := range oldEntries {
		newEntry, found := newEntries[target]
		if !found {
			result.Removed = append(result.Removed, target)
			continue
		}
		oldJSON, _ := json.Marshal(oldEntry)
		newJSON, _ := json.Marshal(newEntry)
		if !bytes.Equal(oldJSON, newJSON) {
			result.Changed = append(result.Changed, target)
		}
	}
	for target := range newEntries {
		if _, found := oldEntries[target]; !found {
			result.Added = append(result.Added, target)
		}
	}
	slices.Sort(result.Added)
	slices.Sort(result.Removed)
	slices.Sort(result.Changed)
	return result, nil
}

func decodeBoundary(name string, contents []byte) (boundaryManifest, error) {
	var manifest boundaryManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return boundaryManifest{}, fmt.Errorf("decode %s boundary manifest: %w", name, err)
	}
	if manifest.ManifestVersion == "" {
		return boundaryManifest{}, fmt.Errorf("%s boundary manifest has no version", name)
	}
	return manifest, nil
}

func indexBoundaryEntries(entries []boundaryIntercept) map[string]boundaryIntercept {
	result := make(map[string]boundaryIntercept, len(entries))
	for _, entry := range entries {
		result[entry.Package+"."+boundaryTarget(entry)] = entry
	}
	return result
}

func boundaryTarget(entry boundaryIntercept) string {
	if entry.Receiver == nil {
		return entry.Symbol
	}
	prefix := ""
	if entry.Receiver.Pointer {
		prefix = "*"
	}
	return "(" + prefix + entry.Receiver.Name + ")." + entry.Symbol
}

func inspectOverlayCollision(root string, descriptor gomadversion.Descriptor) (OverlayCollisionEvidence, error) {
	archivePath := filepath.Join(root, ".toolchain", "downloads", filepath.FromSlash(descriptor.Archive.Name))
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		return OverlayCollisionEvidence{}, fmt.Errorf("read qualified Go archive: %w", err)
	}
	if actual := strings.TrimPrefix(digest(archive), "sha256:"); actual != descriptor.Archive.SHA256 {
		return OverlayCollisionEvidence{}, fmt.Errorf("qualified Go archive digest is %s, want %s", actual, descriptor.Archive.SHA256)
	}
	zipper, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return OverlayCollisionEvidence{}, fmt.Errorf("open qualified Go archive: %w", err)
	}
	defer zipper.Close()
	upstream := make(map[string]struct{})
	reader := tar.NewReader(zipper)
	for {
		header, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return OverlayCollisionEvidence{}, fmt.Errorf("read qualified Go archive: %w", nextErr)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		name := strings.TrimPrefix(filepath.ToSlash(header.Name), "go/")
		upstream[name] = struct{}{}
	}
	result := OverlayCollisionEvidence{
		Checked: true, ArchiveSHA256: "sha256:" + descriptor.Archive.SHA256,
		OverlayPaths: append([]string(nil), descriptor.OverlayAllowlist...), Collisions: []string{},
	}
	for _, path := range descriptor.OverlayAllowlist {
		if _, found := upstream[path]; found {
			result.Collisions = append(result.Collisions, path)
		}
	}
	return result, nil
}

func loadCorpus(root, path string) (CorpusEvidence, error) {
	if path == "" {
		return CorpusEvidence{Status: "not-configured"}, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	report, err := qualificationset.OpenReport(path)
	if err != nil {
		return CorpusEvidence{}, fmt.Errorf("validate retained qualification set: %w", err)
	}
	if !report.Qualified {
		return CorpusEvidence{}, errors.New("retained qualification set is not qualified")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return CorpusEvidence{}, fmt.Errorf("read retained corpus report: %w", err)
	}
	return CorpusEvidence{Status: "checked", Path: path, SHA256: digest(contents), Report: append(json.RawMessage(nil), contents...)}, nil
}

func publish(path string, dossier Dossier) error {
	contents, err := json.MarshalIndent(dossier, "", "  ")
	if err != nil {
		return fmt.Errorf("encode upgrade dossier: %w", err)
	}
	contents = append(contents, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create upgrade dossier directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".upgrade-dossier-*")
	if err != nil {
		return fmt.Errorf("create upgrade dossier: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("chmod upgrade dossier: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write upgrade dossier: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close upgrade dossier: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish upgrade dossier: %w", err)
	}
	return nil
}

func digest(contents []byte) string {
	value := sha256.Sum256(contents)
	return fmt.Sprintf("sha256:%x", value)
}

type boundedBuffer struct {
	bytes.Buffer
	maximum   int
	truncated bool
}

func (buffer *boundedBuffer) Write(contents []byte) (int, error) {
	accepted := contents
	remaining := buffer.maximum - buffer.Len()
	if remaining <= 0 {
		buffer.truncated = true
		return len(contents), nil
	}
	if len(accepted) > remaining {
		accepted = accepted[:remaining]
		buffer.truncated = true
	}
	_, err := buffer.Buffer.Write(accepted)
	return len(contents), err
}
