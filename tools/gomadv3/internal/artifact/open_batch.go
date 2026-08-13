package artifact

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const maximumRunsBytes = 64 << 20

type Batch struct {
	Path   string
	Record BatchRecord
	Runs   []RunRecord
}

func OpenBatch(path string) (Batch, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return Batch{}, fmt.Errorf("open batch directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return Batch{}, fmt.Errorf("batch path is not a directory")
	}
	if rootInfo.Mode().Perm() != 0o700 {
		return Batch{}, fmt.Errorf("batch directory mode is %#o, want 0700", rootInfo.Mode().Perm())
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return Batch{}, fmt.Errorf("pin batch directory: %w", err)
	}
	defer root.Close()
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return Batch{}, errors.Join(fmt.Errorf("batch directory changed while opening"), err)
	}
	batchBytes, err := readValidatedFile(root, "batch.json", 0o600, maximumManifestBytes)
	if err != nil {
		return Batch{}, fmt.Errorf("read batch record: %w", err)
	}
	var batch BatchRecord
	if err := record.StrictDecode(batchBytes, &batch); err != nil {
		return Batch{}, fmt.Errorf("decode batch record: %w", err)
	}
	canonicalBatch, err := record.CanonicalJSON(batch)
	if err != nil || !bytes.Equal(canonicalBatch, batchBytes) {
		return Batch{}, errors.Join(fmt.Errorf("batch record is not canonical"), err)
	}
	runsBytes, err := readValidatedFile(root, "runs.jsonl", 0o600, maximumRunsBytes)
	if err != nil {
		return Batch{}, fmt.Errorf("read batch runs: %w", err)
	}
	digest := record.SHA256(fmt.Sprintf("sha256:%x", sha256.Sum256(runsBytes)))
	if digest != batch.RunsSHA256 {
		return Batch{}, fmt.Errorf("batch runs digest is %s, want %s", digest, batch.RunsSHA256)
	}
	runs, err := decodeRuns(runsBytes)
	if err != nil {
		return Batch{}, err
	}
	if err := validateBatch(batch, runs); err != nil {
		return Batch{}, err
	}
	return Batch{Path: path, Record: batch, Runs: runs}, nil
}

func decodeRuns(contents []byte) ([]RunRecord, error) {
	if len(contents) == 0 {
		return []RunRecord{}, nil
	}
	if contents[len(contents)-1] != '\n' {
		return nil, fmt.Errorf("batch runs journal is not newline terminated")
	}
	lines := bytes.Split(contents[:len(contents)-1], []byte{'\n'})
	runs := make([]RunRecord, len(lines))
	for index, line := range lines {
		if len(line) == 0 {
			return nil, fmt.Errorf("batch runs journal has an empty record at line %d", index+1)
		}
		if err := record.StrictDecode(line, &runs[index]); err != nil {
			return nil, fmt.Errorf("decode batch run %d: %w", index+1, err)
		}
		canonical, err := record.CanonicalJSON(runs[index])
		if err != nil || !bytes.Equal(canonical, line) {
			return nil, errors.Join(fmt.Errorf("batch run %d is not canonical", index+1), err)
		}
	}
	return runs, nil
}

func validateBatch(batch BatchRecord, runs []RunRecord) error {
	if batch.SchemaVersion != record.SchemaVersion || batch.Schema != "gomadv3.batch/v1" || batch.RunID == "" || batch.Selection == "" || batch.SelectionCount == 0 {
		return fmt.Errorf("batch record identity is invalid")
	}
	if uint64(batch.Attempted) != uint64(len(runs)) || uint64(batch.Succeeded)+uint64(batch.Failures)+uint64(batch.Cancelled) != uint64(batch.Attempted) || batch.Watchdogs > batch.Failures || batch.RetainedSuccesses > batch.Succeeded || batch.RetainedSuccesses == 0 && batch.RetainedSuccessBytes != 0 {
		return fmt.Errorf("batch summary counts are inconsistent")
	}
	if batch.StopReason != "seeds_exhausted" && batch.StopReason != "first_failure" && batch.StopReason != "failure_budget" {
		return fmt.Errorf("batch stop reason is invalid: %s", batch.StopReason)
	}
	ordinals := make(map[uint64]struct{}, len(runs))
	failures := make(map[record.SHA256]struct{})
	var succeeded, failed, watchdogs, cancelled, retainedSuccesses, retainedSuccessBytes uint64
	for index, run := range runs {
		ordinal := uint64(run.SelectionOrdinal)
		if ordinal >= uint64(batch.SelectionCount) {
			return fmt.Errorf("batch run %d selection ordinal is out of range", index+1)
		}
		if _, duplicate := ordinals[ordinal]; duplicate {
			return fmt.Errorf("batch selection ordinal is duplicated: %d", ordinal)
		}
		ordinals[ordinal] = struct{}{}
		if run.Reason == "" {
			return fmt.Errorf("batch run %d identity is invalid", index+1)
		}
		if (run.IOTranscriptSHA256 == nil) != (run.IOTranscriptRecords == nil) {
			return fmt.Errorf("batch run %d transcript identity is incomplete", index+1)
		}
		if run.IOTranscriptSHA256 != nil && !validRecordSHA256(*run.IOTranscriptSHA256) {
			return fmt.Errorf("batch run %d transcript digest is invalid", index+1)
		}
		if err := validateSemanticProbeLists(run.SemanticProbes, run.NovelSemanticProbes); err != nil {
			return fmt.Errorf("batch run %d: %w", index+1, err)
		}
		switch run.Domain {
		case "success":
			succeeded++
			if run.Termination != "exit" || run.FailureSignature != nil || run.Artifact != nil {
				return fmt.Errorf("successful batch run %d has failure evidence", index+1)
			}
			if (run.SuccessArtifact == nil) != (run.SuccessArtifactBytes == nil) {
				return fmt.Errorf("successful batch run %d has incomplete retained evidence", index+1)
			}
			if run.SuccessArtifact == nil {
				if len(run.NovelSemanticProbes) != 0 {
					return fmt.Errorf("unretained successful batch run %d has novelty reasons", index+1)
				}
			} else if !validSuccessArtifactReference(*run.SuccessArtifact) || *run.SuccessArtifactBytes == 0 {
				return fmt.Errorf("successful batch run %d has invalid retained evidence", index+1)
			} else {
				retainedSuccesses++
				if uint64(*run.SuccessArtifactBytes) > ^uint64(0)-retainedSuccessBytes {
					return fmt.Errorf("retained success byte count overflows")
				}
				retainedSuccessBytes += uint64(*run.SuccessArtifactBytes)
			}
		case "target", "watchdog":
			failed++
			if run.Domain == "watchdog" {
				watchdogs++
			}
			if run.FailureSignature == nil || !validRecordSHA256(*run.FailureSignature) || run.Artifact == nil || !validArtifactReference(*run.Artifact) {
				return fmt.Errorf("failed batch run %d has invalid artifact evidence", index+1)
			}
			failures[*run.FailureSignature] = struct{}{}
			if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 {
				return fmt.Errorf("failed batch run %d has successful-run evidence", index+1)
			}
		case "runner":
			cancelled++
			if run.Reason != "runner_cancelled" || run.Termination != "none" || run.FailureSignature != nil || run.Artifact != nil {
				return fmt.Errorf("cancelled batch run %d is invalid", index+1)
			}
			if run.SuccessArtifact != nil || run.SuccessArtifactBytes != nil || len(run.NovelSemanticProbes) != 0 {
				return fmt.Errorf("cancelled batch run %d has successful-run evidence", index+1)
			}
		default:
			return fmt.Errorf("batch run %d domain is invalid: %s", index+1, run.Domain)
		}
	}
	if succeeded != uint64(batch.Succeeded) || failed != uint64(batch.Failures) || watchdogs != uint64(batch.Watchdogs) || cancelled != uint64(batch.Cancelled) {
		return fmt.Errorf("batch run counts do not match the summary")
	}
	if retainedSuccesses != uint64(batch.RetainedSuccesses) || retainedSuccessBytes != uint64(batch.RetainedSuccessBytes) {
		return fmt.Errorf("batch retained success counts do not match the summary")
	}
	if uint64(len(failures)) != uint64(batch.DistinctFailures) || len(batch.FailureSignatures) != len(failures) {
		return fmt.Errorf("batch distinct failure count is inconsistent")
	}
	if !sort.SliceIsSorted(batch.FailureSignatures, func(i, j int) bool { return batch.FailureSignatures[i] < batch.FailureSignatures[j] }) {
		return fmt.Errorf("batch failure signatures are not sorted")
	}
	for index, signature := range batch.FailureSignatures {
		if !validRecordSHA256(signature) || index > 0 && batch.FailureSignatures[index-1] == signature {
			return fmt.Errorf("batch failure signatures are invalid")
		}
		if _, found := failures[signature]; !found {
			return fmt.Errorf("batch failure signature has no run: %s", signature)
		}
	}
	return nil
}

func validArtifactReference(reference string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(reference)))
	return clean == reference && strings.HasPrefix(reference, "failures/sha256-") && !strings.Contains(reference, "..")
}

func validSuccessArtifactReference(reference string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(reference)))
	return clean == reference && strings.HasPrefix(reference, "successes/sha256-") && !strings.Contains(reference, "..")
}

func validateSemanticProbeLists(probes, novel []string) error {
	if !sort.StringsAreSorted(probes) || !sort.StringsAreSorted(novel) {
		return fmt.Errorf("semantic probes are not sorted")
	}
	probeSet := make(map[string]struct{}, len(probes))
	for index, probe := range probes {
		if probe == "" || index > 0 && probes[index-1] == probe {
			return fmt.Errorf("semantic probes are invalid")
		}
		probeSet[probe] = struct{}{}
	}
	for index, probe := range novel {
		if probe == "" || index > 0 && novel[index-1] == probe {
			return fmt.Errorf("novel semantic probes are invalid")
		}
		if _, found := probeSet[probe]; !found {
			return fmt.Errorf("novel semantic probe %q was not observed by the run", probe)
		}
	}
	return nil
}

func validRecordSHA256(value record.SHA256) bool {
	text := string(value)
	if len(text) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(text, "sha256:") {
		return false
	}
	for _, character := range strings.TrimPrefix(text, "sha256:") {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}
