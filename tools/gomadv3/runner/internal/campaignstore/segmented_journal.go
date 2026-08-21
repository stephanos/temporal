package campaignstore

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
	"strconv"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

const (
	runJournalSchema            = "gomadv3.run-journal/v1"
	defaultRunSegmentBytes      = 1 << 20
	defaultRunSegmentRecords    = 1024
	maximumRunJournalRuns       = 1_000_000
	maximumRunJournalBytes      = 1 << 50
	maximumRunSegmentBytes      = 64 << 20
	maximumRunJournalIndexBytes = 16 << 20
)

type RunJournalLimits struct {
	MaximumRuns        uint64 `json:"maximum_runs"`
	MaximumBytes       uint64 `json:"maximum_bytes"`
	SegmentBytes       uint64 `json:"segment_bytes"`
	SegmentRecords     uint64 `json:"segment_records"`
	MaximumSegments    uint64 `json:"maximum_segments"`
	MaximumPartialRuns uint64 `json:"maximum_partial_runs"`
}

type RunJournalPlan struct {
	MaximumRuns        evidence.Uint64String  `json:"maximum_runs"`
	MaximumBytes       evidence.Uint64String  `json:"maximum_bytes"`
	SegmentBytes       evidence.Uint64String  `json:"segment_bytes"`
	SegmentRecords     evidence.Uint64String  `json:"segment_records"`
	MaximumSegments    evidence.Uint64String  `json:"maximum_segments"`
	MaximumPartialRuns evidence.Uint64String  `json:"maximum_partial_runs"`
	CapacityOutcome    JournalCapacityOutcome `json:"capacity_outcome"`
}

type runJournalSegment struct {
	File    string                `json:"file"`
	Records evidence.Uint64String `json:"records"`
	Bytes   evidence.Uint64String `json:"bytes"`
	SHA256  evidence.SHA256       `json:"sha256"`
}

type runJournalIndex struct {
	Schema   string                `json:"schema"`
	Limits   RunJournalPlan        `json:"limits"`
	Segments []runJournalSegment   `json:"segments"`
	Records  evidence.Uint64String `json:"records"`
	Bytes    evidence.Uint64String `json:"bytes"`
}

type RunJournalReference struct {
	Schema      string                `json:"schema"`
	IndexFile   string                `json:"index_file"`
	IndexSHA256 evidence.SHA256       `json:"index_sha256"`
	Segments    evidence.Uint64String `json:"segments"`
	Records     evidence.Uint64String `json:"records"`
	Bytes       evidence.Uint64String `json:"bytes"`
}

type segmentedRunJournal struct {
	ctx           context.Context
	batchPath     string
	limits        RunJournalLimits
	segments      []runJournalSegment
	totalRecords  uint64
	totalBytes    uint64
	active        *os.File
	activePath    string
	activeHasher  hash.Hash
	activeRecords uint64
	activeBytes   uint64
}

func normalizeRunJournalLimits(config CampaignConfig) (RunJournalLimits, error) {
	limits := config.Journal
	if limits == (RunJournalLimits{}) {
		maximumRuns := config.SelectionCount
		if (config.Strategy == "choice-frontier" || config.Strategy == "combined-frontier") && config.MaxRuns != 0 {
			maximumRuns = config.MaxRuns
		}
		maximumPartialRuns := config.Parallel
		if maximumPartialRuns == 0 {
			maximumPartialRuns = 1
		}
		if maximumPartialRuns > maximumRuns {
			maximumPartialRuns = maximumRuns
		}
		if maximumRuns > ^uint64(0)/defaultRunSegmentBytes {
			return RunJournalLimits{}, errors.New("run journal byte capacity overflows")
		}
		limits = RunJournalLimits{
			MaximumRuns: maximumRuns, MaximumBytes: maximumRuns * defaultRunSegmentBytes,
			SegmentBytes: defaultRunSegmentBytes, SegmentRecords: defaultRunSegmentRecords,
			MaximumSegments: maximumRuns, MaximumPartialRuns: maximumPartialRuns,
		}
	}
	if limits.MaximumRuns == 0 || limits.MaximumRuns > maximumRunJournalRuns || limits.MaximumBytes == 0 || limits.MaximumBytes > maximumRunJournalBytes || limits.SegmentBytes == 0 || limits.SegmentBytes > maximumRunSegmentBytes || limits.SegmentBytes > limits.MaximumBytes ||
		limits.SegmentRecords == 0 || limits.MaximumSegments == 0 || limits.MaximumSegments > limits.MaximumRuns || limits.MaximumPartialRuns == 0 || limits.MaximumPartialRuns > limits.MaximumRuns {
		return RunJournalLimits{}, errors.New("run journal limits are invalid")
	}
	return limits, nil
}

func DeriveRunJournalPlan(strategy string, selectionCount, maxRuns, parallel uint64) (RunJournalPlan, error) {
	limits, err := normalizeRunJournalLimits(CampaignConfig{
		Strategy: strategy, SelectionCount: selectionCount, MaxRuns: maxRuns, Parallel: parallel,
	})
	if err != nil {
		return RunJournalPlan{}, err
	}
	return recordRunJournalLimits(limits), nil
}

func recordRunJournalLimits(limits RunJournalLimits) RunJournalPlan {
	return RunJournalPlan{
		MaximumRuns: evidence.Uint64String(limits.MaximumRuns), MaximumBytes: evidence.Uint64String(limits.MaximumBytes),
		SegmentBytes: evidence.Uint64String(limits.SegmentBytes), SegmentRecords: evidence.Uint64String(limits.SegmentRecords),
		MaximumSegments: evidence.Uint64String(limits.MaximumSegments), MaximumPartialRuns: evidence.Uint64String(limits.MaximumPartialRuns),
		CapacityOutcome: CapacityInfrastructureFailure,
	}
}

func validateRunJournalPlan(recorded RunJournalPlan, plan CampaignPlan) error {
	if recorded.CapacityOutcome != CapacityInfrastructureFailure {
		return errors.New("run journal capacity outcome is invalid")
	}
	limits := RunJournalLimits{
		MaximumRuns: uint64(recorded.MaximumRuns), MaximumBytes: uint64(recorded.MaximumBytes),
		SegmentBytes: uint64(recorded.SegmentBytes), SegmentRecords: uint64(recorded.SegmentRecords),
		MaximumSegments: uint64(recorded.MaximumSegments), MaximumPartialRuns: uint64(recorded.MaximumPartialRuns),
	}
	if _, err := normalizeRunJournalLimits(CampaignConfig{Journal: limits}); err != nil {
		return err
	}
	maximumRuns := uint64(plan.SelectionCount)
	if plan.Strategy == "choice-frontier" || plan.Strategy == "combined-frontier" {
		maximumRuns = uint64(plan.MaxRuns)
	}
	maximumPartialRuns := uint64(plan.Parallel)
	if maximumPartialRuns > maximumRuns {
		maximumPartialRuns = maximumRuns
	}
	if limits.MaximumRuns != maximumRuns || limits.MaximumPartialRuns != maximumPartialRuns {
		return errors.New("run journal limits do not match campaign execution limits")
	}
	return nil
}

func runJournalLimitsFromPlan(recorded RunJournalPlan) RunJournalLimits {
	return RunJournalLimits{
		MaximumRuns: uint64(recorded.MaximumRuns), MaximumBytes: uint64(recorded.MaximumBytes),
		SegmentBytes: uint64(recorded.SegmentBytes), SegmentRecords: uint64(recorded.SegmentRecords),
		MaximumSegments: uint64(recorded.MaximumSegments), MaximumPartialRuns: uint64(recorded.MaximumPartialRuns),
	}
}

func newSegmentedRunJournal(ctx context.Context, batchPath string, limits RunJournalLimits) (*segmentedRunJournal, error) {
	journal := &segmentedRunJournal{ctx: ctx, batchPath: batchPath, limits: limits, segments: []runJournalSegment{}}
	if err := makePrivateDirectoriesContext(ctx, journal.runsPath()); err != nil {
		return nil, err
	}
	if err := makePrivateDirectoriesContext(ctx, journal.partialPath()); err != nil {
		return nil, err
	}
	if err := journal.writeIndex(); err != nil {
		return nil, err
	}
	return journal, nil
}

func (journal *segmentedRunJournal) runsPath() string {
	return filepath.Join(journal.batchPath, "runs")
}

func (journal *segmentedRunJournal) partialPath() string {
	return filepath.Join(journal.batchPath, ".partial", "runs")
}

func (journal *segmentedRunJournal) index() runJournalIndex {
	return runJournalIndex{
		Schema: runJournalSchema, Limits: recordRunJournalLimits(journal.limits), Segments: append([]runJournalSegment(nil), journal.segments...),
		Records: evidence.Uint64String(journal.totalRecords), Bytes: evidence.Uint64String(journal.totalBytes),
	}
}

func (journal *segmentedRunJournal) writeIndex() error {
	encoded, err := evidence.CanonicalJSON(journal.index())
	if err != nil {
		return err
	}
	if len(encoded) > maximumRunJournalIndexBytes {
		return &JournalCapacityError{Limit: JournalLimitIndexBytes, Required: uint64(len(encoded)), Maximum: maximumRunJournalIndexBytes, Outcome: CapacityInfrastructureFailure}
	}
	return atomicWriteContext(journal.ctx, filepath.Join(journal.runsPath(), "index.json"), encoded)
}

func (journal *segmentedRunJournal) append(record []byte) error {
	required := uint64(len(record))
	if required > journal.limits.SegmentBytes {
		return &JournalCapacityError{Limit: JournalLimitSegmentBytes, Required: required, Maximum: journal.limits.SegmentBytes, Outcome: CapacityInfrastructureFailure}
	}
	if journal.totalRecords >= journal.limits.MaximumRuns {
		return &JournalCapacityError{Limit: JournalLimitRuns, Required: journal.totalRecords + 1, Maximum: journal.limits.MaximumRuns, Outcome: CapacityInfrastructureFailure}
	}
	if required > journal.limits.MaximumBytes-journal.totalBytes {
		return &JournalCapacityError{Limit: JournalLimitBytes, Required: journal.totalBytes + required, Maximum: journal.limits.MaximumBytes, Outcome: CapacityInfrastructureFailure}
	}
	if journal.active != nil && journal.activeRecords != 0 && (journal.activeRecords == journal.limits.SegmentRecords || required > journal.limits.SegmentBytes-journal.activeBytes) {
		if err := journal.seal(); err != nil {
			return err
		}
	}
	if journal.active == nil {
		if uint64(len(journal.segments)) >= journal.limits.MaximumSegments {
			return &JournalCapacityError{Limit: JournalLimitSegments, Required: uint64(len(journal.segments)) + 1, Maximum: journal.limits.MaximumSegments, Outcome: CapacityInfrastructureFailure}
		}
		if err := journal.openActive(); err != nil {
			return err
		}
	}
	written, err := journal.active.Write(record)
	journal.activeBytes += uint64(written)
	journal.totalBytes += uint64(written)
	if _, hashErr := journal.activeHasher.Write(record[:written]); hashErr != nil {
		err = errors.Join(err, hashErr)
	}
	if err != nil {
		return err
	}
	if written != len(record) {
		return io.ErrShortWrite
	}
	journal.activeRecords++
	journal.totalRecords++
	return syncFileContext(journal.ctx, journal.active, "runs-segment")
}

func (journal *segmentedRunJournal) openActive() error {
	name := fmt.Sprintf("%020d.jsonl", len(journal.segments))
	path := filepath.Join(journal.partialPath(), name)
	if err := observeMutation(journal.ctx, mutationCreate, "runs-segment"); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		return errors.Join(err, file.Close())
	}
	journal.active = file
	journal.activePath = path
	journal.activeHasher = sha256.New()
	journal.activeRecords = 0
	journal.activeBytes = 0
	return nil
}

func (journal *segmentedRunJournal) seal() error {
	if journal.active == nil {
		return nil
	}
	if journal.activeRecords == 0 {
		return journal.discardActive()
	}
	if err := syncFileContext(journal.ctx, journal.active, "runs-segment"); err != nil {
		return err
	}
	if err := journal.active.Close(); err != nil {
		return err
	}
	name := filepath.Base(journal.activePath)
	destination := filepath.Join(journal.runsPath(), name)
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("run segment %s already exists", name)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameContext(journal.ctx, journal.activePath, destination, "runs-segment-publish"); err != nil {
		return err
	}
	if err := syncDirectoryContext(journal.ctx, journal.partialPath()); err != nil {
		return err
	}
	if err := syncDirectoryContext(journal.ctx, journal.runsPath()); err != nil {
		return err
	}
	journal.segments = append(journal.segments, runJournalSegment{
		File: name, Records: evidence.Uint64String(journal.activeRecords), Bytes: evidence.Uint64String(journal.activeBytes),
		SHA256: evidence.SHA256("sha256:" + hex.EncodeToString(journal.activeHasher.Sum(nil))),
	})
	journal.active = nil
	journal.activePath = ""
	journal.activeHasher = nil
	journal.activeRecords = 0
	journal.activeBytes = 0
	return journal.writeIndex()
}

func (journal *segmentedRunJournal) discardActive() error {
	path := journal.activePath
	closeErr := journal.active.Close()
	journal.active = nil
	journal.activePath = ""
	journal.activeHasher = nil
	journal.activeRecords = 0
	journal.activeBytes = 0
	if err := observeMutation(journal.ctx, mutationDelete, "runs-segment"); err != nil {
		return errors.Join(closeErr, err)
	}
	removeErr := os.Remove(path)
	return errors.Join(closeErr, removeErr)
}

func (journal *segmentedRunJournal) reference() (RunJournalReference, error) {
	if err := journal.seal(); err != nil {
		return RunJournalReference{}, err
	}
	encoded, err := evidence.CanonicalJSON(journal.index())
	if err != nil {
		return RunJournalReference{}, err
	}
	return RunJournalReference{
		Schema: runJournalSchema, IndexFile: "runs/index.json", IndexSHA256: evidence.HashBytes(encoded),
		Segments: evidence.Uint64String(len(journal.segments)), Records: evidence.Uint64String(journal.totalRecords), Bytes: evidence.Uint64String(journal.totalBytes),
	}, nil
}

func (journal *segmentedRunJournal) close() error {
	if journal == nil || journal.active == nil {
		return nil
	}
	err := journal.active.Close()
	journal.active = nil
	return err
}

func readLegacyPublishedRuns(root *os.Root, batch CampaignRecord) ([]ExecutionRecord, error) {
	runsBytes, err := readValidatedFile(root, "runs.jsonl", 0o600, maximumRunsBytes)
	if err != nil {
		return nil, fmt.Errorf("read batch runs: %w", err)
	}
	digest := evidence.HashBytes(runsBytes)
	if digest != batch.RunsSHA256 {
		return nil, fmt.Errorf("batch runs digest is %s, want %s", digest, batch.RunsSHA256)
	}
	return decodeExecutions(runsBytes)
}

func readPublishedRunJournal(root *os.Root, batch CampaignRecord) ([]ExecutionRecord, *RunJournalInfo, error) {
	reference := batch.Journal
	if reference == nil || reference.Schema != runJournalSchema || reference.IndexFile != "runs/index.json" || !validRecordSHA256(reference.IndexSHA256) {
		return nil, nil, errors.New("batch run journal reference is invalid")
	}
	indexBytes, err := readValidatedFile(root, filepath.FromSlash(reference.IndexFile), 0o600, maximumRunJournalIndexBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("read run journal index: %w", err)
	}
	if digest := evidence.HashBytes(indexBytes); digest != reference.IndexSHA256 {
		return nil, nil, fmt.Errorf("run journal index digest is %s, want %s", digest, reference.IndexSHA256)
	}
	var index runJournalIndex
	if err := evidence.DecodeCanonicalJSON(indexBytes, &index); err != nil {
		return nil, nil, fmt.Errorf("decode run journal index: %w", err)
	}
	if err := validateRunJournalIndex(index, *reference); err != nil {
		return nil, nil, err
	}
	if _, err := validateRunJournalInventory(root, index, false); err != nil {
		return nil, nil, err
	}
	runs := make([]ExecutionRecord, 0, uint64(index.Records))
	for segmentIndex, segment := range index.Segments {
		name := fmt.Sprintf("%020d.jsonl", segmentIndex)
		if segment.File != name {
			return nil, nil, fmt.Errorf("run segment sequence has a gap at %s", name)
		}
		contents, err := readValidatedFile(root, filepath.Join("runs", name), 0o600, uint64(segment.Bytes))
		if err != nil {
			return nil, nil, fmt.Errorf("read run segment %s: %w", name, err)
		}
		if uint64(len(contents)) != uint64(segment.Bytes) || evidence.HashBytes(contents) != segment.SHA256 {
			return nil, nil, fmt.Errorf("run segment %s identity changed", name)
		}
		decoded, err := decodeExecutions(contents)
		if err != nil {
			return nil, nil, fmt.Errorf("decode run segment %s: %w", name, err)
		}
		if uint64(len(decoded)) != uint64(segment.Records) {
			return nil, nil, fmt.Errorf("run segment %s record count changed", name)
		}
		runs = append(runs, decoded...)
	}
	return runs, &RunJournalInfo{
		Schema: reference.Schema, IndexSHA256: reference.IndexSHA256, Segments: uint64(reference.Segments),
		Records: uint64(reference.Records), Bytes: uint64(reference.Bytes), Limits: index.Limits,
	}, nil
}

func readResumableRunJournal(batchPath string, limits RunJournalLimits) (_ runJournalIndex, _ []ExecutionRecord, _ []ExecutionRecord, retErr error) {
	root, err := os.OpenRoot(batchPath)
	if err != nil {
		return runJournalIndex{}, nil, nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, root.Close())
	}()
	indexBytes, err := readValidatedFile(root, filepath.Join("runs", "index.json"), 0o600, maximumRunJournalIndexBytes)
	if err != nil {
		return runJournalIndex{}, nil, nil, classifyIntegrityError(fmt.Errorf("read resumable run journal index: %w", err))
	}
	var index runJournalIndex
	if err := evidence.DecodeCanonicalJSON(indexBytes, &index); err != nil {
		return runJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("decode resumable run journal index: %w", err))
	}
	reference := RunJournalReference{
		Schema: runJournalSchema, IndexFile: "runs/index.json", IndexSHA256: evidence.HashBytes(indexBytes),
		Segments: evidence.Uint64String(len(index.Segments)), Records: index.Records, Bytes: index.Bytes,
	}
	if err := validateRunJournalIndex(index, reference); err != nil {
		return runJournalIndex{}, nil, nil, newIntegrityError(err)
	}
	if index.Limits != recordRunJournalLimits(limits) {
		return runJournalIndex{}, nil, nil, newIntegrityError(errors.New("resumable run journal limits changed"))
	}
	orphan, err := validateRunJournalInventory(root, index, true)
	if err != nil {
		return runJournalIndex{}, nil, nil, err
	}
	closedRuns, err := readIndexedRunSegments(root, index)
	if err != nil {
		return runJournalIndex{}, nil, nil, classifyIntegrityError(err)
	}
	if orphan != "" {
		contents, err := readValidatedFile(root, filepath.Join("runs", orphan), 0o600, limits.SegmentBytes)
		if err != nil {
			return runJournalIndex{}, nil, nil, classifyIntegrityError(fmt.Errorf("read orphan run segment %s: %w", orphan, err))
		}
		decoded, err := decodeExecutions(contents)
		if err != nil {
			return runJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("decode orphan run segment %s: %w", orphan, err))
		}
		if len(decoded) == 0 || uint64(len(decoded)) > limits.SegmentRecords || uint64(len(index.Segments)) == limits.MaximumSegments || uint64(len(contents)) > limits.MaximumBytes-uint64(index.Bytes) || uint64(len(decoded)) > limits.MaximumRuns-uint64(index.Records) {
			return runJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("orphan run segment %s exceeds journal limits", orphan))
		}
		index.Segments = append(index.Segments, runJournalSegment{
			File: orphan, Records: evidence.Uint64String(len(decoded)), Bytes: evidence.Uint64String(len(contents)), SHA256: evidence.HashBytes(contents),
		})
		index.Records += evidence.Uint64String(len(decoded))
		index.Bytes += evidence.Uint64String(len(contents))
		closedRuns = append(closedRuns, decoded...)
	}
	name := fmt.Sprintf("%020d.jsonl", len(index.Segments))
	activeBytes, err := readRecoverableActiveSegment(root, batchPath, name, limits.SegmentBytes)
	if err != nil {
		return runJournalIndex{}, nil, nil, classifyIntegrityError(err)
	}
	if len(activeBytes) != 0 && activeBytes[len(activeBytes)-1] != '\n' {
		lastComplete := bytes.LastIndexByte(activeBytes, '\n')
		if lastComplete < 0 {
			activeBytes = nil
		} else {
			activeBytes = activeBytes[:lastComplete+1]
		}
	}
	activeRuns, err := decodeExecutions(activeBytes)
	if err != nil {
		return runJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("decode active run segment %s: %w", name, err))
	}
	if uint64(len(activeRuns)) > limits.SegmentRecords || uint64(len(activeRuns)) > limits.MaximumRuns-uint64(index.Records) || uint64(len(activeBytes)) > limits.MaximumBytes-uint64(index.Bytes) {
		return runJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("active run segment %s exceeds journal limits", name))
	}
	return index, closedRuns, activeRuns, nil
}

func validateRunJournalInventory(root *os.Root, index runJournalIndex, allowOrphan bool) (_ string, retErr error) {
	info, err := root.Lstat("runs")
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return "", newIntegrityError(errors.New("run journal directory metadata is invalid"))
	}
	directory, err := root.Open("runs")
	if err != nil {
		return "", err
	}
	defer func() {
		retErr = errors.Join(retErr, directory.Close())
	}()
	pinned, err := directory.Stat()
	if err != nil || !os.SameFile(info, pinned) {
		return "", errors.Join(newIntegrityError(errors.New("run journal directory changed while opening")), err)
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return "", err
	}
	expected := make(map[string]struct{}, len(index.Segments)+1)
	expected["index.json"] = struct{}{}
	for _, segment := range index.Segments {
		expected[segment.File] = struct{}{}
	}
	orphan := fmt.Sprintf("%020d.jsonl", len(index.Segments))
	for _, entry := range entries {
		if _, found := expected[entry.Name()]; found {
			continue
		}
		if allowOrphan && entry.Name() == orphan && !entry.IsDir() {
			continue
		}
		return "", newIntegrityError(fmt.Errorf("run journal contains unexpected entry %q", entry.Name()))
	}
	if len(entries) == len(expected) {
		return "", nil
	}
	if allowOrphan && len(entries) == len(expected)+1 {
		return orphan, nil
	}
	return "", newIntegrityError(errors.New("run journal inventory is incomplete"))
}

func readRecoverableActiveSegment(root *os.Root, batchPath, name string, maximumBytes uint64) ([]byte, error) {
	candidates := make([][]byte, 0, 2)
	partialPath := filepath.Join(batchPath, ".partial", "runs")
	entries, err := os.ReadDir(partialPath)
	if err == nil {
		if len(entries) > 1 || len(entries) == 1 && (entries[0].Name() != name || entries[0].IsDir()) {
			return nil, newIntegrityError(errors.New("active run journal segment is ambiguous"))
		}
		if len(entries) == 1 {
			contents, err := readValidatedFile(root, filepath.Join(".partial", "runs", name), 0o600, maximumBytes)
			if err != nil {
				return nil, fmt.Errorf("read active run segment %s: %w", name, err)
			}
			candidates = append(candidates, contents)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read active run journal: %w", err)
	}

	resumeRoot := filepath.Join(batchPath, ".partial", "resume")
	attempts, err := os.ReadDir(resumeRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read resume archives: %w", err)
	}
	for _, attempt := range attempts {
		info, infoErr := attempt.Info()
		value, parseErr := strconv.ParseUint(attempt.Name(), 10, 64)
		if infoErr != nil || parseErr != nil || value == 0 || fmt.Sprintf("%06d", value) != attempt.Name() || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.Join(newIntegrityError(fmt.Errorf("resume archive entry %q is invalid", attempt.Name())), infoErr, parseErr)
		}
		relative := filepath.Join(".partial", "resume", attempt.Name(), "partials", "runs", name)
		if _, err := root.Stat(relative); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect archived run segment %s: %w", name, err)
		}
		contents, err := readValidatedFile(root, relative, 0o600, maximumBytes)
		if err != nil {
			return nil, fmt.Errorf("read archived run segment %s: %w", name, err)
		}
		candidates = append(candidates, contents)
	}

	selected := []byte{}
	for _, candidate := range candidates {
		switch {
		case bytes.HasPrefix(candidate, selected):
			selected = candidate
		case bytes.HasPrefix(selected, candidate):
		default:
			return nil, newIntegrityError(fmt.Errorf("active run segment %s diverges from its resume archive", name))
		}
	}
	return selected, nil
}

func readIndexedRunSegments(root *os.Root, index runJournalIndex) ([]ExecutionRecord, error) {
	runs := make([]ExecutionRecord, 0, uint64(index.Records))
	for segmentIndex, segment := range index.Segments {
		name := fmt.Sprintf("%020d.jsonl", segmentIndex)
		if segment.File != name {
			return nil, fmt.Errorf("run segment sequence has a gap at %s", name)
		}
		contents, err := readValidatedFile(root, filepath.Join("runs", name), 0o600, uint64(segment.Bytes))
		if err != nil {
			return nil, fmt.Errorf("read run segment %s: %w", name, err)
		}
		if uint64(len(contents)) != uint64(segment.Bytes) || evidence.HashBytes(contents) != segment.SHA256 {
			return nil, fmt.Errorf("run segment %s identity changed", name)
		}
		decoded, err := decodeExecutions(contents)
		if err != nil {
			return nil, fmt.Errorf("decode run segment %s: %w", name, err)
		}
		if uint64(len(decoded)) != uint64(segment.Records) {
			return nil, fmt.Errorf("run segment %s record count changed", name)
		}
		runs = append(runs, decoded...)
	}
	return runs, nil
}

func validateRunJournalIndex(index runJournalIndex, reference RunJournalReference) error {
	limits := index.Limits
	if index.Schema != runJournalSchema || limits.CapacityOutcome != CapacityInfrastructureFailure || limits.MaximumRuns == 0 || limits.MaximumRuns > maximumRunJournalRuns || limits.MaximumBytes == 0 || limits.MaximumBytes > maximumRunJournalBytes || limits.SegmentBytes == 0 || limits.SegmentBytes > maximumRunSegmentBytes ||
		limits.SegmentBytes > limits.MaximumBytes || limits.SegmentRecords == 0 || limits.MaximumSegments == 0 || limits.MaximumSegments > limits.MaximumRuns ||
		limits.MaximumPartialRuns == 0 || limits.MaximumPartialRuns > limits.MaximumRuns || uint64(len(index.Segments)) > uint64(limits.MaximumSegments) {
		return errors.New("run journal index limits are invalid")
	}
	var records, bytes uint64
	for segmentIndex, segment := range index.Segments {
		if segment.Records == 0 || segment.Records > limits.SegmentRecords || segment.Bytes == 0 || segment.Bytes > limits.SegmentBytes || !validRecordSHA256(segment.SHA256) {
			return fmt.Errorf("run segment %020d.jsonl metadata is invalid", segmentIndex)
		}
		if uint64(segment.Records) > ^uint64(0)-records || uint64(segment.Bytes) > ^uint64(0)-bytes {
			return errors.New("run journal aggregate overflows")
		}
		records += uint64(segment.Records)
		bytes += uint64(segment.Bytes)
	}
	if records != uint64(index.Records) || bytes != uint64(index.Bytes) || records > uint64(limits.MaximumRuns) || bytes > uint64(limits.MaximumBytes) ||
		uint64(reference.Segments) != uint64(len(index.Segments)) || reference.Records != index.Records || reference.Bytes != index.Bytes {
		return errors.New("run journal index aggregate is invalid")
	}
	return nil
}
