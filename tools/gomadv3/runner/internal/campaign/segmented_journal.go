package campaign

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

	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
)

const (
	executionJournalSchema            = "gomadv3.execution-journal/v1"
	defaultExecutionSegmentBytes      = 1 << 20
	defaultExecutionSegmentRecords    = 1024
	maximumExecutionJournalExecutions = 1_000_000
	maximumExecutionJournalBytes      = 1 << 50
	maximumExecutionSegmentBytes      = 64 << 20
	maximumExecutionJournalIndexBytes = 16 << 20
)

type ExecutionJournalLimits struct {
	MaximumExecutions        uint64 `json:"maximum_executions"`
	MaximumBytes             uint64 `json:"maximum_bytes"`
	SegmentBytes             uint64 `json:"segment_bytes"`
	SegmentRecords           uint64 `json:"segment_records"`
	MaximumSegments          uint64 `json:"maximum_segments"`
	MaximumPartialExecutions uint64 `json:"maximum_partial_executions"`
}

type ExecutionJournalPlan struct {
	MaximumExecutions        record.Uint64String    `json:"maximum_executions"`
	MaximumBytes             record.Uint64String    `json:"maximum_bytes"`
	SegmentBytes             record.Uint64String    `json:"segment_bytes"`
	SegmentRecords           record.Uint64String    `json:"segment_records"`
	MaximumSegments          record.Uint64String    `json:"maximum_segments"`
	MaximumPartialExecutions record.Uint64String    `json:"maximum_partial_executions"`
	CapacityOutcome          JournalCapacityOutcome `json:"capacity_outcome"`
}

type executionJournalSegment struct {
	File    string              `json:"file"`
	Records record.Uint64String `json:"records"`
	Bytes   record.Uint64String `json:"bytes"`
	SHA256  record.SHA256       `json:"sha256"`
}

type executionJournalIndex struct {
	Schema   string                    `json:"schema"`
	Limits   ExecutionJournalPlan      `json:"limits"`
	Segments []executionJournalSegment `json:"segments"`
	Records  record.Uint64String       `json:"records"`
	Bytes    record.Uint64String       `json:"bytes"`
}

type ExecutionJournalReference struct {
	Schema      string              `json:"schema"`
	IndexFile   string              `json:"index_file"`
	IndexSHA256 record.SHA256       `json:"index_sha256"`
	Segments    record.Uint64String `json:"segments"`
	Records     record.Uint64String `json:"records"`
	Bytes       record.Uint64String `json:"bytes"`
}

type segmentedExecutionJournal struct {
	ctx           context.Context
	campaignPath  string
	limits        ExecutionJournalLimits
	segments      []executionJournalSegment
	totalRecords  uint64
	totalBytes    uint64
	active        *os.File
	activePath    string
	activeHasher  hash.Hash
	activeRecords uint64
	activeBytes   uint64
}

func normalizeExecutionJournalLimits(config CampaignConfig) (ExecutionJournalLimits, error) {
	limits := config.Journal
	if limits == (ExecutionJournalLimits{}) {
		maximumRuns := config.SelectionCount
		if (config.Strategy == "choice-exploration" || config.Strategy == "simulation-exploration") && config.MaxExecutions != 0 {
			maximumRuns = config.MaxExecutions
		}
		maximumPartialRuns := config.Parallel
		if maximumPartialRuns == 0 {
			maximumPartialRuns = 1
		}
		if maximumPartialRuns > maximumRuns {
			maximumPartialRuns = maximumRuns
		}
		if maximumRuns > ^uint64(0)/defaultExecutionSegmentBytes {
			return ExecutionJournalLimits{}, errors.New("execution journal byte capacity overflows")
		}
		limits = ExecutionJournalLimits{
			MaximumExecutions: maximumRuns, MaximumBytes: maximumRuns * defaultExecutionSegmentBytes,
			SegmentBytes: defaultExecutionSegmentBytes, SegmentRecords: defaultExecutionSegmentRecords,
			MaximumSegments: maximumRuns, MaximumPartialExecutions: maximumPartialRuns,
		}
	}
	if limits.MaximumExecutions == 0 || limits.MaximumExecutions > maximumExecutionJournalExecutions || limits.MaximumBytes == 0 || limits.MaximumBytes > maximumExecutionJournalBytes || limits.SegmentBytes == 0 || limits.SegmentBytes > maximumExecutionSegmentBytes || limits.SegmentBytes > limits.MaximumBytes ||
		limits.SegmentRecords == 0 || limits.MaximumSegments == 0 || limits.MaximumSegments > limits.MaximumExecutions || limits.MaximumPartialExecutions == 0 || limits.MaximumPartialExecutions > limits.MaximumExecutions {
		return ExecutionJournalLimits{}, errors.New("execution journal limits are invalid")
	}
	return limits, nil
}

func DeriveExecutionJournalPlan(strategy string, selectionCount, maxExecutions, parallel uint64) (ExecutionJournalPlan, error) {
	limits, err := normalizeExecutionJournalLimits(CampaignConfig{
		Strategy: strategy, SelectionCount: selectionCount, MaxExecutions: maxExecutions, Parallel: parallel,
	})
	if err != nil {
		return ExecutionJournalPlan{}, err
	}
	return recordExecutionJournalLimits(limits), nil
}

func recordExecutionJournalLimits(limits ExecutionJournalLimits) ExecutionJournalPlan {
	return ExecutionJournalPlan{
		MaximumExecutions: record.Uint64String(limits.MaximumExecutions), MaximumBytes: record.Uint64String(limits.MaximumBytes),
		SegmentBytes: record.Uint64String(limits.SegmentBytes), SegmentRecords: record.Uint64String(limits.SegmentRecords),
		MaximumSegments: record.Uint64String(limits.MaximumSegments), MaximumPartialExecutions: record.Uint64String(limits.MaximumPartialExecutions),
		CapacityOutcome: CapacityInfrastructureFailure,
	}
}

func validateExecutionJournalPlan(recorded ExecutionJournalPlan, plan CampaignPlan) error {
	if recorded.CapacityOutcome != CapacityInfrastructureFailure {
		return errors.New("execution journal capacity outcome is invalid")
	}
	limits := ExecutionJournalLimits{
		MaximumExecutions: uint64(recorded.MaximumExecutions), MaximumBytes: uint64(recorded.MaximumBytes),
		SegmentBytes: uint64(recorded.SegmentBytes), SegmentRecords: uint64(recorded.SegmentRecords),
		MaximumSegments: uint64(recorded.MaximumSegments), MaximumPartialExecutions: uint64(recorded.MaximumPartialExecutions),
	}
	if _, err := normalizeExecutionJournalLimits(CampaignConfig{Journal: limits}); err != nil {
		return err
	}
	maximumRuns := uint64(plan.SelectionCount)
	if plan.Strategy == "choice-exploration" || plan.Strategy == "simulation-exploration" {
		maximumRuns = uint64(plan.MaxExecutions)
	}
	maximumPartialRuns := uint64(plan.Parallel)
	if maximumPartialRuns > maximumRuns {
		maximumPartialRuns = maximumRuns
	}
	if limits.MaximumExecutions != maximumRuns || limits.MaximumPartialExecutions != maximumPartialRuns {
		return errors.New("execution journal limits do not match campaign execution limits")
	}
	return nil
}

func executionJournalLimitsFromPlan(recorded ExecutionJournalPlan) ExecutionJournalLimits {
	return ExecutionJournalLimits{
		MaximumExecutions: uint64(recorded.MaximumExecutions), MaximumBytes: uint64(recorded.MaximumBytes),
		SegmentBytes: uint64(recorded.SegmentBytes), SegmentRecords: uint64(recorded.SegmentRecords),
		MaximumSegments: uint64(recorded.MaximumSegments), MaximumPartialExecutions: uint64(recorded.MaximumPartialExecutions),
	}
}

func newSegmentedExecutionJournal(ctx context.Context, campaignPath string, limits ExecutionJournalLimits) (*segmentedExecutionJournal, error) {
	journal := &segmentedExecutionJournal{ctx: ctx, campaignPath: campaignPath, limits: limits, segments: []executionJournalSegment{}}
	if err := makePrivateDirectoriesContext(ctx, journal.executionsPath()); err != nil {
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

func (journal *segmentedExecutionJournal) executionsPath() string {
	return filepath.Join(journal.campaignPath, "executions")
}

func (journal *segmentedExecutionJournal) partialPath() string {
	return filepath.Join(journal.campaignPath, ".partial", "executions")
}

func (journal *segmentedExecutionJournal) index() executionJournalIndex {
	return executionJournalIndex{
		Schema: executionJournalSchema, Limits: recordExecutionJournalLimits(journal.limits), Segments: append([]executionJournalSegment(nil), journal.segments...),
		Records: record.Uint64String(journal.totalRecords), Bytes: record.Uint64String(journal.totalBytes),
	}
}

func (journal *segmentedExecutionJournal) writeIndex() error {
	encoded, err := canonicaljson.CanonicalJSON(journal.index())
	if err != nil {
		return err
	}
	if len(encoded) > maximumExecutionJournalIndexBytes {
		return &JournalCapacityError{Limit: JournalLimitIndexBytes, Required: uint64(len(encoded)), Maximum: maximumExecutionJournalIndexBytes, Outcome: CapacityInfrastructureFailure}
	}
	return atomicWriteContext(journal.ctx, filepath.Join(journal.executionsPath(), "index.json"), encoded)
}

func (journal *segmentedExecutionJournal) append(record []byte) error {
	required := uint64(len(record))
	if required > journal.limits.SegmentBytes {
		return &JournalCapacityError{Limit: JournalLimitSegmentBytes, Required: required, Maximum: journal.limits.SegmentBytes, Outcome: CapacityInfrastructureFailure}
	}
	if journal.totalRecords >= journal.limits.MaximumExecutions {
		return &JournalCapacityError{Limit: JournalLimitExecutions, Required: journal.totalRecords + 1, Maximum: journal.limits.MaximumExecutions, Outcome: CapacityInfrastructureFailure}
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
	return syncFileContext(journal.ctx, journal.active, "execution-segment")
}

func (journal *segmentedExecutionJournal) openActive() error {
	name := fmt.Sprintf("%020d.jsonl", len(journal.segments))
	path := filepath.Join(journal.partialPath(), name)
	if err := observeMutation(journal.ctx, mutationCreate, "execution-segment"); err != nil {
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

func (journal *segmentedExecutionJournal) seal() error {
	if journal.active == nil {
		return nil
	}
	if journal.activeRecords == 0 {
		return journal.discardActive()
	}
	if err := syncFileContext(journal.ctx, journal.active, "execution-segment"); err != nil {
		return err
	}
	if err := journal.active.Close(); err != nil {
		return err
	}
	name := filepath.Base(journal.activePath)
	destination := filepath.Join(journal.executionsPath(), name)
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("execution segment %s already exists", name)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameContext(journal.ctx, journal.activePath, destination, "execution-segment-publish"); err != nil {
		return err
	}
	if err := syncDirectoryContext(journal.ctx, journal.partialPath()); err != nil {
		return err
	}
	if err := syncDirectoryContext(journal.ctx, journal.executionsPath()); err != nil {
		return err
	}
	journal.segments = append(journal.segments, executionJournalSegment{
		File: name, Records: record.Uint64String(journal.activeRecords), Bytes: record.Uint64String(journal.activeBytes),
		SHA256: record.SHA256("sha256:" + hex.EncodeToString(journal.activeHasher.Sum(nil))),
	})
	journal.active = nil
	journal.activePath = ""
	journal.activeHasher = nil
	journal.activeRecords = 0
	journal.activeBytes = 0
	return journal.writeIndex()
}

func (journal *segmentedExecutionJournal) discardActive() error {
	path := journal.activePath
	closeErr := journal.active.Close()
	journal.active = nil
	journal.activePath = ""
	journal.activeHasher = nil
	journal.activeRecords = 0
	journal.activeBytes = 0
	if err := observeMutation(journal.ctx, mutationDelete, "execution-segment"); err != nil {
		return errors.Join(closeErr, err)
	}
	removeErr := os.Remove(path)
	return errors.Join(closeErr, removeErr)
}

func (journal *segmentedExecutionJournal) reference() (ExecutionJournalReference, error) {
	if err := journal.seal(); err != nil {
		return ExecutionJournalReference{}, err
	}
	encoded, err := canonicaljson.CanonicalJSON(journal.index())
	if err != nil {
		return ExecutionJournalReference{}, err
	}
	return ExecutionJournalReference{
		Schema: executionJournalSchema, IndexFile: "executions/index.json", IndexSHA256: record.HashBytes(encoded),
		Segments: record.Uint64String(len(journal.segments)), Records: record.Uint64String(journal.totalRecords), Bytes: record.Uint64String(journal.totalBytes),
	}, nil
}

func (journal *segmentedExecutionJournal) close() error {
	if journal == nil || journal.active == nil {
		return nil
	}
	err := journal.active.Close()
	journal.active = nil
	return err
}

func readPublishedExecutionJournal(root *os.Root, campaign CampaignRecord) ([]ExecutionRecord, *ExecutionJournalInfo, error) {
	reference := campaign.Journal
	if reference == nil || reference.Schema != executionJournalSchema || reference.IndexFile != "executions/index.json" || !validRecordSHA256(reference.IndexSHA256) {
		return nil, nil, errors.New("campaign execution journal reference is invalid")
	}
	indexBytes, err := readValidatedFile(root, filepath.FromSlash(reference.IndexFile), 0o600, maximumExecutionJournalIndexBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("read execution journal index: %w", err)
	}
	if digest := record.HashBytes(indexBytes); digest != reference.IndexSHA256 {
		return nil, nil, fmt.Errorf("execution journal index digest is %s, want %s", digest, reference.IndexSHA256)
	}
	var index executionJournalIndex
	if err := canonicaljson.DecodeCanonicalJSON(indexBytes, &index); err != nil {
		return nil, nil, fmt.Errorf("decode execution journal index: %w", err)
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
			return nil, nil, fmt.Errorf("execution segment sequence has a gap at %s", name)
		}
		contents, err := readValidatedFile(root, filepath.Join("executions", name), 0o600, uint64(segment.Bytes))
		if err != nil {
			return nil, nil, fmt.Errorf("read execution segment %s: %w", name, err)
		}
		if uint64(len(contents)) != uint64(segment.Bytes) || record.HashBytes(contents) != segment.SHA256 {
			return nil, nil, fmt.Errorf("execution segment %s identity changed", name)
		}
		decoded, err := decodeExecutions(contents)
		if err != nil {
			return nil, nil, fmt.Errorf("decode execution segment %s: %w", name, err)
		}
		if uint64(len(decoded)) != uint64(segment.Records) {
			return nil, nil, fmt.Errorf("execution segment %s record count changed", name)
		}
		runs = append(runs, decoded...)
	}
	return runs, &ExecutionJournalInfo{
		Schema: reference.Schema, IndexSHA256: reference.IndexSHA256, Segments: uint64(reference.Segments),
		Records: uint64(reference.Records), Bytes: uint64(reference.Bytes), Limits: index.Limits,
	}, nil
}

func readResumableExecutionJournal(batchPath string, limits ExecutionJournalLimits) (_ executionJournalIndex, _ []ExecutionRecord, _ []ExecutionRecord, retErr error) {
	root, err := os.OpenRoot(batchPath)
	if err != nil {
		return executionJournalIndex{}, nil, nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, root.Close())
	}()
	indexBytes, err := readValidatedFile(root, filepath.Join("executions", "index.json"), 0o600, maximumExecutionJournalIndexBytes)
	if err != nil {
		return executionJournalIndex{}, nil, nil, classifyIntegrityError(fmt.Errorf("read resumable execution journal index: %w", err))
	}
	var index executionJournalIndex
	if err := canonicaljson.DecodeCanonicalJSON(indexBytes, &index); err != nil {
		return executionJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("decode resumable execution journal index: %w", err))
	}
	reference := ExecutionJournalReference{
		Schema: executionJournalSchema, IndexFile: "executions/index.json", IndexSHA256: record.HashBytes(indexBytes),
		Segments: record.Uint64String(len(index.Segments)), Records: index.Records, Bytes: index.Bytes,
	}
	if err := validateRunJournalIndex(index, reference); err != nil {
		return executionJournalIndex{}, nil, nil, newIntegrityError(err)
	}
	if index.Limits != recordExecutionJournalLimits(limits) {
		return executionJournalIndex{}, nil, nil, newIntegrityError(errors.New("resumable execution journal limits changed"))
	}
	orphan, err := validateRunJournalInventory(root, index, true)
	if err != nil {
		return executionJournalIndex{}, nil, nil, err
	}
	closedRuns, err := readIndexedRunSegments(root, index)
	if err != nil {
		return executionJournalIndex{}, nil, nil, classifyIntegrityError(err)
	}
	if orphan != "" {
		contents, err := readValidatedFile(root, filepath.Join("executions", orphan), 0o600, limits.SegmentBytes)
		if err != nil {
			return executionJournalIndex{}, nil, nil, classifyIntegrityError(fmt.Errorf("read orphan execution segment %s: %w", orphan, err))
		}
		decoded, err := decodeExecutions(contents)
		if err != nil {
			return executionJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("decode orphan execution segment %s: %w", orphan, err))
		}
		if len(decoded) == 0 || uint64(len(decoded)) > limits.SegmentRecords || uint64(len(index.Segments)) == limits.MaximumSegments || uint64(len(contents)) > limits.MaximumBytes-uint64(index.Bytes) || uint64(len(decoded)) > limits.MaximumExecutions-uint64(index.Records) {
			return executionJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("orphan execution segment %s exceeds journal limits", orphan))
		}
		index.Segments = append(index.Segments, executionJournalSegment{
			File: orphan, Records: record.Uint64String(len(decoded)), Bytes: record.Uint64String(len(contents)), SHA256: record.HashBytes(contents),
		})
		index.Records += record.Uint64String(len(decoded))
		index.Bytes += record.Uint64String(len(contents))
		closedRuns = append(closedRuns, decoded...)
	}
	name := fmt.Sprintf("%020d.jsonl", len(index.Segments))
	activeBytes, err := readRecoverableActiveSegment(root, batchPath, name, limits.SegmentBytes)
	if err != nil {
		return executionJournalIndex{}, nil, nil, classifyIntegrityError(err)
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
		return executionJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("decode active execution segment %s: %w", name, err))
	}
	if uint64(len(activeRuns)) > limits.SegmentRecords || uint64(len(activeRuns)) > limits.MaximumExecutions-uint64(index.Records) || uint64(len(activeBytes)) > limits.MaximumBytes-uint64(index.Bytes) {
		return executionJournalIndex{}, nil, nil, newIntegrityError(fmt.Errorf("active execution segment %s exceeds journal limits", name))
	}
	return index, closedRuns, activeRuns, nil
}

func validateRunJournalInventory(root *os.Root, index executionJournalIndex, allowOrphan bool) (_ string, retErr error) {
	info, err := root.Lstat("executions")
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return "", newIntegrityError(errors.New("execution journal directory metadata is invalid"))
	}
	directory, err := root.Open("executions")
	if err != nil {
		return "", err
	}
	defer func() {
		retErr = errors.Join(retErr, directory.Close())
	}()
	pinned, err := directory.Stat()
	if err != nil || !os.SameFile(info, pinned) {
		return "", errors.Join(newIntegrityError(errors.New("execution journal directory changed while opening")), err)
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
		return "", newIntegrityError(fmt.Errorf("execution journal contains unexpected entry %q", entry.Name()))
	}
	if len(entries) == len(expected) {
		return "", nil
	}
	if allowOrphan && len(entries) == len(expected)+1 {
		return orphan, nil
	}
	return "", newIntegrityError(errors.New("execution journal inventory is incomplete"))
}

func readRecoverableActiveSegment(root *os.Root, batchPath, name string, maximumBytes uint64) ([]byte, error) {
	candidates := make([][]byte, 0, 2)
	partialPath := filepath.Join(batchPath, ".partial", "executions")
	entries, err := os.ReadDir(partialPath)
	if err == nil {
		if len(entries) > 1 || len(entries) == 1 && (entries[0].Name() != name || entries[0].IsDir()) {
			return nil, newIntegrityError(errors.New("active execution journal segment is ambiguous"))
		}
		if len(entries) == 1 {
			contents, err := readValidatedFile(root, filepath.Join(".partial", "executions", name), 0o600, maximumBytes)
			if err != nil {
				return nil, fmt.Errorf("read active execution segment %s: %w", name, err)
			}
			candidates = append(candidates, contents)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read active execution journal: %w", err)
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
		relative := filepath.Join(".partial", "resume", attempt.Name(), "partials", "executions", name)
		if _, err := root.Stat(relative); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect archived execution segment %s: %w", name, err)
		}
		contents, err := readValidatedFile(root, relative, 0o600, maximumBytes)
		if err != nil {
			return nil, fmt.Errorf("read archived execution segment %s: %w", name, err)
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
			return nil, newIntegrityError(fmt.Errorf("active execution segment %s diverges from its resume archive", name))
		}
	}
	return selected, nil
}

func readIndexedRunSegments(root *os.Root, index executionJournalIndex) ([]ExecutionRecord, error) {
	runs := make([]ExecutionRecord, 0, uint64(index.Records))
	for segmentIndex, segment := range index.Segments {
		name := fmt.Sprintf("%020d.jsonl", segmentIndex)
		if segment.File != name {
			return nil, fmt.Errorf("execution segment sequence has a gap at %s", name)
		}
		contents, err := readValidatedFile(root, filepath.Join("executions", name), 0o600, uint64(segment.Bytes))
		if err != nil {
			return nil, fmt.Errorf("read execution segment %s: %w", name, err)
		}
		if uint64(len(contents)) != uint64(segment.Bytes) || record.HashBytes(contents) != segment.SHA256 {
			return nil, fmt.Errorf("execution segment %s identity changed", name)
		}
		decoded, err := decodeExecutions(contents)
		if err != nil {
			return nil, fmt.Errorf("decode execution segment %s: %w", name, err)
		}
		if uint64(len(decoded)) != uint64(segment.Records) {
			return nil, fmt.Errorf("execution segment %s record count changed", name)
		}
		runs = append(runs, decoded...)
	}
	return runs, nil
}

func validateRunJournalIndex(index executionJournalIndex, reference ExecutionJournalReference) error {
	limits := index.Limits
	if index.Schema != executionJournalSchema || limits.CapacityOutcome != CapacityInfrastructureFailure || limits.MaximumExecutions == 0 || limits.MaximumExecutions > maximumExecutionJournalExecutions || limits.MaximumBytes == 0 || limits.MaximumBytes > maximumExecutionJournalBytes || limits.SegmentBytes == 0 || limits.SegmentBytes > maximumExecutionSegmentBytes ||
		limits.SegmentBytes > limits.MaximumBytes || limits.SegmentRecords == 0 || limits.MaximumSegments == 0 || limits.MaximumSegments > limits.MaximumExecutions ||
		limits.MaximumPartialExecutions == 0 || limits.MaximumPartialExecutions > limits.MaximumExecutions || uint64(len(index.Segments)) > uint64(limits.MaximumSegments) {
		return errors.New("execution journal index limits are invalid")
	}
	var records, bytes uint64
	for segmentIndex, segment := range index.Segments {
		if segment.Records == 0 || segment.Records > limits.SegmentRecords || segment.Bytes == 0 || segment.Bytes > limits.SegmentBytes || !validRecordSHA256(segment.SHA256) {
			return fmt.Errorf("execution segment %020d.jsonl metadata is invalid", segmentIndex)
		}
		if uint64(segment.Records) > ^uint64(0)-records || uint64(segment.Bytes) > ^uint64(0)-bytes {
			return errors.New("execution journal aggregate overflows")
		}
		records += uint64(segment.Records)
		bytes += uint64(segment.Bytes)
	}
	if records != uint64(index.Records) || bytes != uint64(index.Bytes) || records > uint64(limits.MaximumExecutions) || bytes > uint64(limits.MaximumBytes) ||
		uint64(reference.Segments) != uint64(len(index.Segments)) || reference.Records != index.Records || reference.Bytes != index.Bytes {
		return errors.New("execution journal index aggregate is invalid")
	}
	return nil
}
