package choice

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
)

var (
	ErrInvalidDecision   = errors.New("invalid canonical choice decision")
	ErrInvalidReplayPlan = errors.New("invalid choice decision tape")
	ErrReplayUnavailable = errors.New("exact choice replay unavailable")
)

type ExecutionIdentity struct {
	TargetSHA256         [sha256.Size]byte
	ToolchainBuildKey    string
	GOOS                 string
	GOARCH               string
	ImplementationSHA256 [sha256.Size]byte
}

type Decision struct {
	Ordinal              uint64
	Kind                 Kind
	SiteOffset           uint64
	SiteMissing          bool
	RankOverride         bool
	Alternatives         uint32
	Selected             uint32
	Data                 uint32
	SelectedIdentity     [sha256.Size]byte
	AlternativeSetDigest [sha256.Size]byte
}

func (decision Decision) Record() Record {
	flags := FlagDecision
	if decision.SiteMissing {
		flags |= FlagSiteMissing
	}
	if decision.RankOverride {
		flags |= FlagRankOverride
	}
	return Record{
		Ordinal: decision.Ordinal, Kind: decision.Kind, Flags: flags, Alternatives: decision.Alternatives,
		Selected: decision.Selected, Data: decision.Data, SiteOffset: decision.SiteOffset,
		SelectedIdentity: decision.SelectedIdentity, AlternativeSetDigest: decision.AlternativeSetDigest,
	}
}

func decisionFromRecord(record Record) (Decision, error) {
	if record.Flags&FlagDecision == 0 {
		return Decision{}, errors.Join(ErrInvalidDecision, errors.New("choice record is not a decision"))
	}
	return Decision{
		Ordinal: record.Ordinal, Kind: record.Kind, SiteOffset: record.SiteOffset,
		SiteMissing: record.Flags&FlagSiteMissing != 0, RankOverride: record.Flags&FlagRankOverride != 0, Alternatives: record.Alternatives,
		Selected: record.Selected, Data: record.Data, SelectedIdentity: record.SelectedIdentity,
		AlternativeSetDigest: record.AlternativeSetDigest,
	}, nil
}

func AlternativeSetDigest(alternatives [][sha256.Size]byte) ([sha256.Size]byte, error) {
	if len(alternatives) == 0 || uint64(len(alternatives)) > uint64(^uint32(0)) {
		return [sha256.Size]byte{}, errors.Join(ErrInvalidDecision, errors.New("choice alternatives are empty or exceed the protocol bound"))
	}
	ordered := append([][sha256.Size]byte(nil), alternatives...)
	slices.SortFunc(ordered, func(left, right [sha256.Size]byte) int { return bytes.Compare(left[:], right[:]) })
	for index, identity := range ordered {
		if identity == ([sha256.Size]byte{}) {
			return [sha256.Size]byte{}, errors.Join(ErrInvalidDecision, errors.New("choice alternative identity is missing"))
		}
		if index != 0 && identity == ordered[index-1] {
			return [sha256.Size]byte{}, errors.Join(ErrInvalidDecision, errors.New("choice alternative identity is duplicated"))
		}
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad3-choice-alternative-set/v1"))
	_, _ = hasher.Write([]byte{0})
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(ordered)))
	_, _ = hasher.Write(count[:])
	for _, identity := range ordered {
		_, _ = hasher.Write(identity[:])
	}
	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))
	return digest, nil
}

func CanonicalDecision(
	ordinal uint64,
	kind Kind,
	siteOffset uint64,
	siteMissing bool,
	alternatives [][sha256.Size]byte,
	selectedIdentity [sha256.Size]byte,
	data uint32,
) (Decision, error) {
	digest, err := AlternativeSetDigest(alternatives)
	if err != nil {
		return Decision{}, err
	}
	ordered := append([][sha256.Size]byte(nil), alternatives...)
	slices.SortFunc(ordered, func(left, right [sha256.Size]byte) int { return bytes.Compare(left[:], right[:]) })
	selected := slices.Index(ordered, selectedIdentity)
	if selected < 0 {
		return Decision{}, errors.Join(ErrInvalidDecision, errors.New("selected choice identity is not an alternative"))
	}
	decision := Decision{
		Ordinal: ordinal, Kind: kind, SiteOffset: siteOffset, SiteMissing: siteMissing,
		Alternatives: uint32(len(ordered)), Selected: uint32(selected), Data: data,
		SelectedIdentity: selectedIdentity, AlternativeSetDigest: digest,
	}
	if _, err := encodeRecord(decision.Record()); err != nil {
		return Decision{}, errors.Join(ErrInvalidDecision, err)
	}
	return decision, nil
}

// SelectReadiness is what a select observed once it had locked its channels:
// how many of its cases could proceed, and the shape properties that tell the
// fixture shapes apart. The projection carries a select's readiness onto each
// of its select-poll decisions. Known is false for a runnable decision and for
// a select-poll decision whose select recorded no result because it blocked
// and never resumed.
type SelectReadiness struct {
	Known bool `json:"known"`
	// Ready counts the cases that could proceed; the default is not a case.
	Ready           uint32 `json:"ready"`
	Default         bool   `json:"default"`
	NilChannel      bool   `json:"nil_channel"`
	TimerChannel    bool   `json:"timer_channel"`
	ClosedChannel   bool   `json:"closed_channel"`
	RepeatedChannel bool   `json:"repeated_channel"`
}

func selectReadinessFromWord(readiness Readiness) SelectReadiness {
	if !readiness.Known() {
		return SelectReadiness{}
	}
	return SelectReadiness{
		Known: true, Ready: readiness.Ready(),
		Default: readiness&ReadinessDefault != 0, NilChannel: readiness&ReadinessNilChannel != 0, TimerChannel: readiness&ReadinessTimerChannel != 0,
		ClosedChannel: readiness&ReadinessClosedChannel != 0, RepeatedChannel: readiness&ReadinessRepeatedChannel != 0,
	}
}

// Word packs the readiness for a record; an unknown readiness is the zero word.
func (readiness SelectReadiness) Word() (Readiness, error) {
	if !readiness.Known {
		if readiness != (SelectReadiness{}) {
			return 0, errors.Join(ErrInvalidDecision, errors.New("unknown select readiness carries evidence"))
		}
		return 0, nil
	}
	var flags Readiness
	for _, flag := range []struct {
		set  bool
		flag Readiness
	}{{readiness.Default, ReadinessDefault}, {readiness.NilChannel, ReadinessNilChannel}, {readiness.TimerChannel, ReadinessTimerChannel}, {readiness.ClosedChannel, ReadinessClosedChannel}, {readiness.RepeatedChannel, ReadinessRepeatedChannel}} {
		if flag.set {
			flags |= flag.flag
		}
	}
	word, err := NewReadiness(readiness.Ready, flags)
	if err != nil {
		return 0, errors.Join(ErrInvalidDecision, err)
	}
	return word, nil
}

// ReplayPlan is a decision tape. Readiness has one entry per decision and is
// encoded in each decision's record, so it survives the tape's byte form; it is
// kept beside Decisions rather than inside Decision because a decision's
// identity and replay comparison exclude it: a child whose select never
// resumes still forced the same decision its parent recorded.
type ReplayPlan struct {
	Identity          ExecutionIdentity
	SourceTraceSHA256 [sha256.Size]byte
	Decisions         []Decision
	Readiness         []SelectReadiness
	Bytes             []byte
	SHA256            [sha256.Size]byte
}

func ProjectReplayPlan(trace Trace, identity ExecutionIdentity) (ReplayPlan, error) {
	if trace.Version == Version1 {
		return ReplayPlan{}, ErrReplayUnavailable
	}
	if trace.Version != Version3 || trace.Summary.Terminal != TerminalComplete {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice trace is not complete v3 evidence"))
	}
	if trace.SHA256 != sha256.Sum256(trace.Bytes) || trace.Summary.Records != uint64(len(trace.Records)) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice trace identity is inconsistent"))
	}
	decisions := make([]Decision, 0, len(trace.Records))
	planIndex := make([]int, len(trace.Records))
	for index, record := range trace.Records {
		planIndex[index] = -1
		if record.Flags&FlagObservation != 0 || record.Alternatives < 2 {
			continue
		}
		decision, err := decisionFromRecord(record)
		if err != nil {
			return ReplayPlan{}, err
		}
		decision.Ordinal = uint64(len(decisions))
		planIndex[index] = len(decisions)
		decisions = append(decisions, decision)
	}
	readiness, err := projectSelectReadiness(trace.Records, planIndex, len(decisions))
	if err != nil {
		return ReplayPlan{}, err
	}
	return encodeTape(identity, trace.SHA256, decisions, readiness)
}

// projectSelectReadiness carries each select_result's readiness onto the
// select-poll decisions of the same select. A result names its select's
// decisions exactly: they are the polled-case count minus one consecutive
// records from the result's origin, each a select-poll decision at the
// result's site whose alternatives count the poll steps 2, 3, ... in order.
// Nothing else records between the poll steps of one select, because the
// poll loop neither parks nor schedules, so a record that breaks the pattern
// is corrupt evidence and fails the projection rather than being guessed at.
// A decision no result names keeps the unknown readiness.
func projectSelectReadiness(records []Record, planIndex []int, decisions int) ([]SelectReadiness, error) {
	readiness := make([]SelectReadiness, decisions)
	for _, result := range records {
		if result.Kind != KindSelectResult {
			continue
		}
		polled := uint64(result.Data)
		if polled < 2 {
			continue
		}
		if polled-1 > result.Ordinal-result.Origin {
			return nil, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("select result %d names %d poll decisions before its origin %d", result.Ordinal, polled-1, result.Origin))
		}
		for step := uint64(0); step < polled-1; step++ {
			ordinal := result.Origin + step
			decision := records[ordinal]
			if decision.Kind != KindSelectPoll || decision.Flags&FlagDecision == 0 || decision.SiteOffset != result.SiteOffset || decision.Flags&FlagSiteMissing != result.Flags&FlagSiteMissing || uint64(decision.Alternatives) != step+2 {
				return nil, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("select result %d does not match the poll decision at %d", result.Ordinal, ordinal))
			}
			index := planIndex[ordinal]
			if readiness[index].Known {
				return nil, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("select results %d and another both name the poll decision at %d", result.Ordinal, ordinal))
			}
			readiness[index] = selectReadinessFromWord(result.Readiness)
		}
	}
	return readiness, nil
}

func ValidateReplayPlan(tape ReplayPlan, identity ExecutionIdentity) (ReplayPlan, error) {
	if tape.SHA256 != sha256.Sum256(tape.Bytes) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape digest mismatch"))
	}
	validated, err := decodeTape(tape.Bytes, identity)
	if err != nil {
		return ReplayPlan{}, err
	}
	for _, decision := range validated.Decisions {
		if decision.Alternatives < 2 {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("exact choice tape contains a non-branching decision"))
		}
		if decision.RankOverride {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("exact choice tape contains a rank override"))
		}
	}
	return validated, nil
}

func ValidatePrefixReplayPlan(tape ReplayPlan, identity ExecutionIdentity) (ReplayPlan, error) {
	if tape.SHA256 != sha256.Sum256(tape.Bytes) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape digest mismatch"))
	}
	validated, err := decodeTape(tape.Bytes, identity)
	if err != nil {
		return ReplayPlan{}, err
	}
	for index, decision := range validated.Decisions {
		if decision.Alternatives < 2 {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice prefix contains a non-branching decision"))
		}
		if !decision.RankOverride {
			continue
		}
		if index != len(validated.Decisions)-1 || decision.Kind == KindSelectResult || decision.SelectedIdentity != ([sha256.Size]byte{}) {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice rank override must be the final prefix decision"))
		}
	}
	return validated, nil
}

func BuildRankPrefix(source ReplayPlan, decisionOrdinal uint64, rank uint32) (ReplayPlan, error) {
	return buildRankPrefix(source, decisionOrdinal, rank, false)
}

func BuildForcedRankPrefix(source ReplayPlan, decisionOrdinal uint64, rank uint32) (ReplayPlan, error) {
	return buildRankPrefix(source, decisionOrdinal, rank, true)
}

func buildRankPrefix(source ReplayPlan, decisionOrdinal uint64, rank uint32, reconstruct bool) (ReplayPlan, error) {
	validated, err := ValidateReplayPlan(source, source.Identity)
	if err != nil {
		return ReplayPlan{}, err
	}
	if decisionOrdinal >= uint64(len(validated.Decisions)) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice rank override exceeds its source tape"))
	}
	target := validated.Decisions[decisionOrdinal]
	if rank >= target.Alternatives {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice rank override is outside its alternative set"))
	}
	if reconstruct && rank != target.Selected {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("reconstructed choice rank override must select the observed alternative"))
	}
	if !reconstruct && rank == target.Selected {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice rank override must select a non-selected alternative"))
	}
	decisions := append([]Decision(nil), validated.Decisions[:decisionOrdinal+1]...)
	decisions[decisionOrdinal].Selected = rank
	decisions[decisionOrdinal].SelectedIdentity = [sha256.Size]byte{}
	decisions[decisionOrdinal].RankOverride = true
	sourceHash, err := rankPrefixSourceHash(decisions)
	if err != nil {
		return ReplayPlan{}, err
	}
	return encodeTape(validated.Identity, sourceHash, decisions, validated.Readiness[:decisionOrdinal+1])
}

func (tape ReplayPlan) Prefix(records uint64) (ReplayPlan, error) {
	if records > uint64(len(tape.Decisions)) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice prefix exceeds its source tape"))
	}
	var readiness []SelectReadiness
	if tape.Readiness != nil {
		readiness = tape.Readiness[:records]
	}
	return encodeTape(tape.Identity, tape.SourceTraceSHA256, tape.Decisions[:records], readiness)
}

func (tape ReplayPlan) Branching() []Decision {
	result := make([]Decision, 0)
	for _, decision := range tape.Decisions {
		if decision.Alternatives > 1 {
			result = append(result, decision)
		}
	}
	return result
}

// encodeTape writes one record per decision; readiness is nil when every
// decision's readiness is unknown, otherwise it has one entry per decision.
func encodeTape(identity ExecutionIdentity, sourceTrace [sha256.Size]byte, decisions []Decision, readiness []SelectReadiness) (ReplayPlan, error) {
	headerIdentity, err := tapeHeaderIdentity(identity)
	if err != nil {
		return ReplayPlan{}, err
	}
	if readiness != nil && len(readiness) != len(decisions) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape readiness does not cover its decisions"))
	}
	payload := make([]byte, len(decisions)*replayPlanRecordBytes)
	cloned := make([]Decision, len(decisions))
	clonedReadiness := make([]SelectReadiness, len(decisions))
	for index, decision := range decisions {
		decision.Ordinal = uint64(index)
		value := decision.Record()
		if readiness != nil {
			value.Readiness, err = readiness[index].Word()
			if err != nil {
				return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("encode decision %d readiness: %w", index, err))
			}
			clonedReadiness[index] = readiness[index]
		}
		record, err := encodeRecord(value)
		if err != nil {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("encode decision %d: %w", index, err))
		}
		copy(payload[index*replayPlanRecordBytes:], record[:])
		cloned[index] = decision
	}
	payloadHash := sha256.Sum256(payload)
	header, err := encodeReplayPlanHeader(replayPlanHeader{
		TotalBytes: uint64(replayPlanHeaderBytes + len(payload)), Records: uint64(len(cloned)),
		SourceTraceHash: sourceTrace, TargetHash: identity.TargetSHA256,
		ImplementationHash: identity.ImplementationSHA256, ToolchainBuildKey: headerIdentity.toolchain,
		PlatformHash: headerIdentity.platform, PayloadHash: payloadHash,
	})
	if err != nil {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, err)
	}
	encoded := make([]byte, 0, len(header)+len(payload))
	encoded = append(encoded, header[:]...)
	encoded = append(encoded, payload...)
	return ReplayPlan{
		Identity: identity, SourceTraceSHA256: sourceTrace, Decisions: cloned, Readiness: clonedReadiness,
		Bytes: encoded, SHA256: sha256.Sum256(encoded),
	}, nil
}

func rankPrefixSourceHash(decisions []Decision) ([sha256.Size]byte, error) {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad3-choice-rank-prefix/v1"))
	_, _ = hasher.Write([]byte{0})
	for index, decision := range decisions {
		decision.Ordinal = uint64(index)
		record, err := encodeRecord(decision.Record())
		if err != nil {
			return [sha256.Size]byte{}, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("encode rank prefix decision %d: %w", index, err))
		}
		_, _ = hasher.Write(record[:])
	}
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result, nil
}

func decodeTape(encoded []byte, identity ExecutionIdentity) (ReplayPlan, error) {
	if len(encoded) < replayPlanHeaderBytes {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape is shorter than its header"))
	}
	header, err := decodeReplayPlanHeader(encoded[:replayPlanHeaderBytes])
	if err != nil {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, err)
	}
	if header.TotalBytes != uint64(len(encoded)) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape byte length is inconsistent"))
	}
	payload := encoded[replayPlanHeaderBytes:]
	if len(payload)%replayPlanRecordBytes != 0 || header.Records != uint64(len(payload)/replayPlanRecordBytes) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape record count is inconsistent"))
	}
	headerIdentity, err := tapeHeaderIdentity(identity)
	if err != nil {
		return ReplayPlan{}, err
	}
	if header.TargetHash != identity.TargetSHA256 || header.ImplementationHash != identity.ImplementationSHA256 || header.ToolchainBuildKey != headerIdentity.toolchain || header.PlatformHash != headerIdentity.platform {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape execution identity does not match"))
	}
	if header.PayloadHash != sha256.Sum256(payload) {
		return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice tape payload digest mismatch"))
	}
	decisions := make([]Decision, header.Records)
	readiness := make([]SelectReadiness, header.Records)
	for index := range decisions {
		record, err := decodeRecord(payload[index*replayPlanRecordBytes : (index+1)*replayPlanRecordBytes])
		if err != nil {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("decode decision %d: %w", index, err))
		}
		if record.Ordinal != uint64(index) || record.Flags&FlagDecision == 0 {
			return ReplayPlan{}, errors.Join(ErrInvalidReplayPlan, fmt.Errorf("decision %d ordinal or role is invalid", index))
		}
		decisions[index], err = decisionFromRecord(record)
		if err != nil {
			return ReplayPlan{}, err
		}
		readiness[index] = selectReadinessFromWord(record.Readiness)
	}
	copyBytes := append([]byte(nil), encoded...)
	return ReplayPlan{
		Identity: identity, SourceTraceSHA256: header.SourceTraceHash, Decisions: decisions, Readiness: readiness,
		Bytes: copyBytes, SHA256: sha256.Sum256(copyBytes),
	}, nil
}

type encodedExecutionIdentity struct {
	toolchain [sha256.Size]byte
	platform  [sha256.Size]byte
}

func ValidateExecutionIdentity(identity ExecutionIdentity) error {
	_, err := tapeHeaderIdentity(identity)
	return err
}

func tapeHeaderIdentity(identity ExecutionIdentity) (encodedExecutionIdentity, error) {
	if identity.TargetSHA256 == ([sha256.Size]byte{}) || identity.ImplementationSHA256 == ([sha256.Size]byte{}) || identity.GOOS == "" || identity.GOARCH == "" {
		return encodedExecutionIdentity{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice execution identity is incomplete"))
	}
	var result encodedExecutionIdentity
	if len(identity.ToolchainBuildKey) != hex.EncodedLen(len(result.toolchain)) {
		return encodedExecutionIdentity{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice toolchain build key is malformed"))
	}
	if _, err := hex.Decode(result.toolchain[:], []byte(identity.ToolchainBuildKey)); err != nil || hex.EncodeToString(result.toolchain[:]) != identity.ToolchainBuildKey {
		return encodedExecutionIdentity{}, errors.Join(ErrInvalidReplayPlan, errors.New("choice toolchain build key is malformed"))
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad3-choice-platform/v1"))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(identity.GOOS))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(identity.GOARCH))
	copy(result.platform[:], hasher.Sum(nil))
	return result, nil
}

type Divergence struct {
	Ordinal     uint64
	Reason      DivergenceReason
	Expected    *Decision
	Observed    *Decision
	TapeRecords uint64
}

func divergenceFromTerminal(terminal terminal) (Divergence, error) {
	if terminal.State != TerminalDiverged {
		return Divergence{}, errors.New("choice terminal is not divergence evidence")
	}
	result := Divergence{Ordinal: terminal.DivergentOrdinal, Reason: terminal.DivergenceReason, TapeRecords: terminal.TapeRecords}
	if terminal.ExpectedPresent {
		expected, err := decisionFromRecord(terminal.Expected)
		if err != nil {
			return Divergence{}, err
		}
		result.Expected = &expected
	}
	if terminal.ObservedPresent {
		observed, err := decisionFromRecord(terminal.Observed)
		if err != nil {
			return Divergence{}, err
		}
		result.Observed = &observed
	}
	return result, nil
}

func validateDivergenceTerminal(tape ReplayPlan, mode Mode, terminal terminal) (Divergence, error) {
	if mode != ModeReplay && mode != ModePrefix {
		return Divergence{}, errors.New("choice divergence validation requires replay or prefix mode")
	}
	var validated ReplayPlan
	var err error
	if mode == ModePrefix {
		validated, err = ValidatePrefixReplayPlan(tape, tape.Identity)
	} else {
		validated, err = ValidateReplayPlan(tape, tape.Identity)
	}
	if err != nil {
		return Divergence{}, err
	}
	divergence, err := divergenceFromTerminal(terminal)
	if err != nil {
		return Divergence{}, err
	}
	if divergence.TapeRecords != uint64(len(validated.Decisions)) {
		return Divergence{}, errors.New("choice divergence tape count does not match")
	}
	if mode == ModeReplay && divergence.Ordinal > uint64(len(validated.Decisions)) {
		return Divergence{}, errors.New("choice replay divergence ordinal exceeds its tape")
	}
	if divergence.Expected != nil {
		if divergence.Ordinal >= uint64(len(validated.Decisions)) || *divergence.Expected != validated.Decisions[divergence.Ordinal] {
			return Divergence{}, errors.New("choice divergence expected decision does not match its tape")
		}
	} else if divergence.Ordinal < uint64(len(validated.Decisions)) {
		return Divergence{}, errors.New("choice divergence omitted its expected decision")
	}
	switch divergence.Reason {
	case DivergenceKind, DivergenceSite, DivergenceAlternatives, DivergenceSelected, DivergenceAlternativeSet:
		if divergence.Expected == nil || divergence.Observed == nil || compareDecisions(*divergence.Expected, *divergence.Observed) != divergence.Reason {
			return Divergence{}, errors.New("choice divergence reason does not match its decisions")
		}
	case DivergenceTapeExhausted:
		if divergence.Ordinal != uint64(len(validated.Decisions)) || divergence.Expected != nil || divergence.Observed == nil {
			return Divergence{}, errors.New("choice tape exhaustion evidence is inconsistent")
		}
	case DivergenceTapeUnconsumed:
		if divergence.Expected == nil || divergence.Observed != nil {
			return Divergence{}, errors.New("choice unconsumed tape evidence is inconsistent")
		}
	case DivergenceIdentityMissing, DivergenceIdentityDuplicate, DivergenceAlternativeCapacity:
		if divergence.Observed != nil {
			return Divergence{}, errors.New("unformed choice divergence contains an observed decision")
		}
	case DivergenceObservation:
		if divergence.Observed == nil {
			return Divergence{}, errors.New("choice observation divergence omitted observed evidence")
		}
	default:
		return Divergence{}, errors.New("choice divergence reason is invalid")
	}
	return divergence, nil
}

func compareDecisions(expected, observed Decision) DivergenceReason {
	switch {
	case expected.Kind != observed.Kind:
		return DivergenceKind
	case expected.SiteOffset != observed.SiteOffset || expected.SiteMissing != observed.SiteMissing:
		return DivergenceSite
	case expected.Alternatives != observed.Alternatives:
		return DivergenceAlternatives
	case expected.AlternativeSetDigest != observed.AlternativeSetDigest:
		return DivergenceAlternativeSet
	case expected.Selected != observed.Selected || expected.SelectedIdentity != observed.SelectedIdentity:
		return DivergenceSelected
	default:
		return 0
	}
}

func DivergenceReasonName(reason DivergenceReason) string {
	switch reason {
	case DivergenceKind:
		return "kind"
	case DivergenceSite:
		return "site"
	case DivergenceAlternatives:
		return "alternatives"
	case DivergenceSelected:
		return "selected"
	case DivergenceAlternativeSet:
		return "alternative_set"
	case DivergenceTapeExhausted:
		return "tape_exhausted"
	case DivergenceTapeUnconsumed:
		return "tape_unconsumed"
	case DivergenceIdentityMissing:
		return "identity_missing"
	case DivergenceIdentityDuplicate:
		return "identity_duplicate"
	case DivergenceAlternativeCapacity:
		return "alternative_capacity"
	case DivergenceObservation:
		return "observation"
	default:
		return "unknown"
	}
}
