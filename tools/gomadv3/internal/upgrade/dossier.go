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
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const maximumGateOutput = 16 << 20
const maximumGateDuration = 30 * time.Minute
const gateTerminationGrace = 2 * time.Second

type Options struct {
	Root                       string
	Output                     string
	BaselineManifest           []byte
	ApprovedBoundaryDiffSHA256 string
	CorpusReport               string
	Gates                      []Gate
	Writer                     io.Writer
}

type Gate struct {
	Name    string
	Command []string
}

type Dossier struct {
	Schema                 string                   `json:"schema"`
	Qualified              bool                     `json:"qualified"`
	BoundaryApproved       bool                     `json:"boundary_changes_approved"`
	BoundaryApprovalSHA256 string                   `json:"boundary_approval_sha256,omitempty"`
	Version                VersionEvidence          `json:"version"`
	Host                   HostEvidence             `json:"host"`
	UpstreamPatch          PatchEvidence            `json:"upstream_patch"`
	BoundaryDiff           BoundaryDiff             `json:"boundary_manifest_diff"`
	InterceptionReport     InterceptionEvidence     `json:"interception_report"`
	OverlayCollision       OverlayCollisionEvidence `json:"overlay_collision_report"`
	Gates                  []GateResult             `json:"gates"`
	RetainedCorpus         CorpusEvidence           `json:"retained_corpus"`
	MandatoryProbePolicy   string                   `json:"mandatory_probe_policy"`
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
	SHA256          string   `json:"sha256"`
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
	ManifestVersion string
	Metadata        json.RawMessage
	Entries         map[string]json.RawMessage
}

type boundaryDocument struct {
	ManifestVersion string            `json:"manifest_version"`
	HookPolicies    []json.RawMessage `json:"hook_policies"`
	Intercepts      []json.RawMessage `json:"intercepts"`
}

type boundaryInterceptIdentity struct {
	Package  string            `json:"package"`
	Receiver *boundaryReceiver `json:"receiver,omitempty"`
	Symbol   string            `json:"symbol"`
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
	boundaryApprovalSHA256 := ""
	boundaryApproved := difference.Status == "compared" && boundaryDiffEmpty(difference)
	if difference.Status == "compared" && !boundaryDiffEmpty(difference) && options.ApprovedBoundaryDiffSHA256 == difference.SHA256 {
		boundaryApproved = true
		boundaryApprovalSHA256 = difference.SHA256
	}
	dossier := Dossier{
		Schema:                 "gomadv3.upgrade-dossier/v2",
		Qualified:              false,
		BoundaryApproved:       boundaryApproved,
		BoundaryApprovalSHA256: boundaryApprovalSHA256,
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
	dossier.Qualified = dossier.Host.Supported && gateFailure == nil && dossier.RetainedCorpus.Status == "checked" && dossier.BoundaryApproved
	if err := publish(options.Output, dossier); err != nil {
		return err
	}
	if gateFailure != nil {
		return gateFailure
	}
	if !dossier.Host.Supported {
		return fmt.Errorf("qualification host %s is unsupported", hostPlatform)
	}
	if dossier.RetainedCorpus.Status != "checked" {
		return errors.New("a checked core qualification corpus is required")
	}
	if difference.Status != "compared" {
		return errors.New("a baseline boundary manifest is required")
	}
	if !dossier.BoundaryApproved {
		return fmt.Errorf("boundary changes require explicit approval for %s", difference.SHA256)
	}
	return nil
}

func boundaryDiffEmpty(difference BoundaryDiff) bool {
	return len(difference.Added) == 0 && len(difference.Removed) == 0 && len(difference.Changed) == 0
}

func runGate(ctx context.Context, root string, gate Gate, output io.Writer) (GateResult, error) {
	executed, err := commandrun.Run(ctx, commandrun.Request{
		Command: gate.Command, Dir: root, Env: os.Environ(), Timeout: maximumGateDuration,
		TerminateGrace: gateTerminationGrace, OutputLimit: maximumGateOutput,
	})
	combined := make([]byte, 0, len(executed.Stdout.Bytes)+len(executed.Stderr.Bytes))
	combined = append(combined, executed.Stdout.Bytes...)
	combined = append(combined, executed.Stderr.Bytes...)
	if output != nil && len(combined) != 0 {
		_, _ = output.Write(combined)
	}
	result := GateResult{
		Name: gate.Name, Command: append([]string(nil), gate.Command...), Output: string(combined),
		OutputSHA256: digest(combined), OutputTruncated: executed.Stdout.Truncated || executed.Stderr.Truncated,
	}
	if err == nil && !executed.WatchdogTimeout && !executed.Cancelled && executed.Termination == commandrun.TerminationExit && executed.ExitCode == 0 {
		result.Status = "passed"
		return result, nil
	}
	result.Status = "failed"
	result.ExitCode = -1
	if executed.Termination == commandrun.TerminationExit {
		result.ExitCode = executed.ExitCode
	}
	switch {
	case err != nil:
		return result, err
	case executed.WatchdogTimeout:
		return result, context.DeadlineExceeded
	case executed.Cancelled:
		return result, context.Canceled
	case executed.Termination == commandrun.TerminationSignal:
		return result, fmt.Errorf("command terminated by signal %s", executed.Signal)
	default:
		return result, fmt.Errorf("command exited with status %d", executed.ExitCode)
	}
}

func compareBoundaries(baseline, current []byte) (BoundaryDiff, error) {
	currentManifest, err := decodeBoundary("current", current)
	if err != nil {
		return BoundaryDiff{}, err
	}
	result := BoundaryDiff{Status: "not-requested", CurrentVersion: currentManifest.ManifestVersion, Added: []string{}, Removed: []string{}, Changed: []string{}}
	if len(baseline) == 0 {
		return finalizeBoundaryDiff(result, nil, current)
	}
	baselineManifest, err := decodeBoundary("baseline", baseline)
	if err != nil {
		return BoundaryDiff{}, err
	}
	result.Status = "compared"
	result.BaselineVersion = baselineManifest.ManifestVersion
	oldEntries := baselineManifest.Entries
	newEntries := currentManifest.Entries
	for target, oldEntry := range oldEntries {
		newEntry, found := newEntries[target]
		if !found {
			result.Removed = append(result.Removed, target)
			continue
		}
		if !bytes.Equal(oldEntry, newEntry) {
			result.Changed = append(result.Changed, target)
		}
	}
	for target := range newEntries {
		if _, found := oldEntries[target]; !found {
			result.Added = append(result.Added, target)
		}
	}
	if !bytes.Equal(baselineManifest.Metadata, currentManifest.Metadata) {
		result.Changed = append(result.Changed, "manifest")
	}
	slices.Sort(result.Added)
	slices.Sort(result.Removed)
	slices.Sort(result.Changed)
	return finalizeBoundaryDiff(result, baseline, current)
}

func finalizeBoundaryDiff(difference BoundaryDiff, baseline, current []byte) (BoundaryDiff, error) {
	canonicalCurrent, err := canonicalBoundaryJSON(json.RawMessage(current))
	if err != nil {
		return BoundaryDiff{}, fmt.Errorf("canonicalize current boundary diff input: %w", err)
	}
	var canonicalBaseline json.RawMessage
	if len(baseline) != 0 {
		canonicalBaseline, err = canonicalBoundaryJSON(json.RawMessage(baseline))
		if err != nil {
			return BoundaryDiff{}, fmt.Errorf("canonicalize baseline boundary diff input: %w", err)
		}
	}
	encoded, err := json.Marshal(struct {
		Schema   string          `json:"schema"`
		Status   string          `json:"status"`
		Baseline json.RawMessage `json:"baseline,omitempty"`
		Current  json.RawMessage `json:"current"`
	}{
		Schema: "gomadv3.boundary-diff/v1", Status: difference.Status,
		Baseline: canonicalBaseline, Current: canonicalCurrent,
	})
	if err != nil {
		return BoundaryDiff{}, fmt.Errorf("encode boundary diff identity: %w", err)
	}
	difference.SHA256 = digest(encoded)
	return difference, nil
}

func decodeBoundary(name string, contents []byte) (boundaryManifest, error) {
	var document boundaryDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		return boundaryManifest{}, fmt.Errorf("decode %s boundary manifest: %w", name, err)
	}
	if document.ManifestVersion == "" {
		return boundaryManifest{}, fmt.Errorf("%s boundary manifest has no version", name)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(contents, &fields); err != nil {
		return boundaryManifest{}, fmt.Errorf("decode %s boundary manifest fields: %w", name, err)
	}
	delete(fields, "manifest_version")
	delete(fields, "hook_policies")
	delete(fields, "intercepts")
	metadata, err := canonicalBoundaryJSON(fields)
	if err != nil {
		return boundaryManifest{}, fmt.Errorf("canonicalize %s boundary manifest: %w", name, err)
	}
	entries := make(map[string]json.RawMessage, len(document.HookPolicies)+len(document.Intercepts))
	for _, raw := range document.Intercepts {
		var identity boundaryInterceptIdentity
		if err := json.Unmarshal(raw, &identity); err != nil {
			return boundaryManifest{}, fmt.Errorf("decode %s boundary intercept identity: %w", name, err)
		}
		if identity.Package == "" || identity.Symbol == "" {
			return boundaryManifest{}, fmt.Errorf("%s boundary manifest has an incomplete intercept identity", name)
		}
		key := identity.Package + "." + boundaryTarget(identity)
		if err := addBoundaryEntry(entries, key, raw); err != nil {
			return boundaryManifest{}, fmt.Errorf("%s boundary manifest: %w", name, err)
		}
	}
	for _, raw := range document.HookPolicies {
		var identity struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &identity); err != nil {
			return boundaryManifest{}, fmt.Errorf("decode %s hook policy identity: %w", name, err)
		}
		if identity.ID == "" {
			return boundaryManifest{}, fmt.Errorf("%s boundary manifest has an incomplete hook policy identity", name)
		}
		if err := addBoundaryEntry(entries, "hook-policy:"+identity.ID, raw); err != nil {
			return boundaryManifest{}, fmt.Errorf("%s boundary manifest: %w", name, err)
		}
	}
	return boundaryManifest{ManifestVersion: document.ManifestVersion, Metadata: metadata, Entries: entries}, nil
}

func addBoundaryEntry(entries map[string]json.RawMessage, key string, raw json.RawMessage) error {
	if _, duplicate := entries[key]; duplicate {
		return fmt.Errorf("duplicate boundary entry %s", key)
	}
	canonical, err := canonicalBoundaryJSON(raw)
	if err != nil {
		return fmt.Errorf("canonicalize boundary entry %s: %w", key, err)
	}
	entries[key] = canonical
	return nil
}

func canonicalBoundaryJSON(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func boundaryTarget(entry boundaryInterceptIdentity) string {
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
	if !report.ExpectationsMet {
		return CorpusEvidence{}, errors.New("retained qualification set did not meet its expectations")
	}
	if report.Name != "gomadv3-core" {
		return CorpusEvidence{}, fmt.Errorf("retained qualification set %q is not the required gomadv3-core corpus", report.Name)
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
