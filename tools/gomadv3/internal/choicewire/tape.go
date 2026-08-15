package choicewire

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
	ErrInvalidTape       = errors.New("invalid choice decision tape")
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
	_, _ = hasher.Write([]byte("gomadv3-choice-alternative-set/v1"))
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
	if _, err := EncodeRecord(decision.Record()); err != nil {
		return Decision{}, errors.Join(ErrInvalidDecision, err)
	}
	return decision, nil
}

type Tape struct {
	Identity          ExecutionIdentity
	SourceTraceSHA256 [sha256.Size]byte
	Decisions         []Decision
	Bytes             []byte
	SHA256            [sha256.Size]byte
}

func ProjectDecisionTape(trace Trace, identity ExecutionIdentity) (Tape, error) {
	if trace.Version == Version1 {
		return Tape{}, ErrReplayUnavailable
	}
	if trace.Version != Version2 || trace.Summary.Terminal != TerminalComplete {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice trace is not complete v2 evidence"))
	}
	if trace.SHA256 != sha256.Sum256(trace.Bytes) || trace.Summary.Records != uint64(len(trace.Records)) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice trace identity is inconsistent"))
	}
	decisions := make([]Decision, 0, len(trace.Records))
	for _, record := range trace.Records {
		if record.Flags&FlagObservation != 0 {
			continue
		}
		decision, err := decisionFromRecord(record)
		if err != nil {
			return Tape{}, err
		}
		decision.Ordinal = uint64(len(decisions))
		decisions = append(decisions, decision)
	}
	return encodeTape(identity, trace.SHA256, decisions)
}

func ValidateDecisionTape(tape Tape, identity ExecutionIdentity) (Tape, error) {
	if tape.SHA256 != sha256.Sum256(tape.Bytes) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape digest mismatch"))
	}
	validated, err := decodeTape(tape.Bytes, identity)
	if err != nil {
		return Tape{}, err
	}
	for _, decision := range validated.Decisions {
		if decision.RankOverride {
			return Tape{}, errors.Join(ErrInvalidTape, errors.New("exact choice tape contains a rank override"))
		}
	}
	return validated, nil
}

func ValidatePrefixTape(tape Tape, identity ExecutionIdentity) (Tape, error) {
	if tape.SHA256 != sha256.Sum256(tape.Bytes) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape digest mismatch"))
	}
	validated, err := decodeTape(tape.Bytes, identity)
	if err != nil {
		return Tape{}, err
	}
	for index, decision := range validated.Decisions {
		if !decision.RankOverride {
			continue
		}
		if index != len(validated.Decisions)-1 || decision.Kind == KindSelectResult || decision.SelectedIdentity != ([sha256.Size]byte{}) {
			return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice rank override must be the final prefix decision"))
		}
	}
	return validated, nil
}

func BuildRankPrefix(source Tape, decisionOrdinal uint64, rank uint32) (Tape, error) {
	validated, err := ValidateDecisionTape(source, source.Identity)
	if err != nil {
		return Tape{}, err
	}
	if decisionOrdinal >= uint64(len(validated.Decisions)) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice rank override exceeds its source tape"))
	}
	target := validated.Decisions[decisionOrdinal]
	if rank >= target.Alternatives {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice rank override is outside its alternative set"))
	}
	if rank == target.Selected {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice rank override must select a non-selected alternative"))
	}
	decisions := append([]Decision(nil), validated.Decisions[:decisionOrdinal+1]...)
	decisions[decisionOrdinal].Selected = rank
	decisions[decisionOrdinal].SelectedIdentity = [sha256.Size]byte{}
	decisions[decisionOrdinal].RankOverride = true
	sourceHash, err := rankPrefixSourceHash(decisions)
	if err != nil {
		return Tape{}, err
	}
	return encodeTape(validated.Identity, sourceHash, decisions)
}

func (tape Tape) Prefix(records uint64) (Tape, error) {
	if records > uint64(len(tape.Decisions)) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice prefix exceeds its source tape"))
	}
	return encodeTape(tape.Identity, tape.SourceTraceSHA256, tape.Decisions[:records])
}

func (tape Tape) Branching() []Decision {
	result := make([]Decision, 0)
	for _, decision := range tape.Decisions {
		if decision.Alternatives > 1 {
			result = append(result, decision)
		}
	}
	return result
}

func encodeTape(identity ExecutionIdentity, sourceTrace [sha256.Size]byte, decisions []Decision) (Tape, error) {
	headerIdentity, err := tapeHeaderIdentity(identity)
	if err != nil {
		return Tape{}, err
	}
	payload := make([]byte, len(decisions)*TapeRecordBytes)
	cloned := make([]Decision, len(decisions))
	for index, decision := range decisions {
		decision.Ordinal = uint64(index)
		record, err := EncodeRecord(decision.Record())
		if err != nil {
			return Tape{}, errors.Join(ErrInvalidTape, fmt.Errorf("encode decision %d: %w", index, err))
		}
		copy(payload[index*TapeRecordBytes:], record[:])
		cloned[index] = decision
	}
	payloadHash := sha256.Sum256(payload)
	header, err := EncodeTapeHeader(TapeHeader{
		TotalBytes: uint64(TapeHeaderBytes + len(payload)), Records: uint64(len(cloned)),
		SourceTraceHash: sourceTrace, TargetHash: identity.TargetSHA256,
		ImplementationHash: identity.ImplementationSHA256, ToolchainBuildKey: headerIdentity.toolchain,
		PlatformHash: headerIdentity.platform, PayloadHash: payloadHash,
	})
	if err != nil {
		return Tape{}, errors.Join(ErrInvalidTape, err)
	}
	encoded := make([]byte, 0, len(header)+len(payload))
	encoded = append(encoded, header[:]...)
	encoded = append(encoded, payload...)
	return Tape{
		Identity: identity, SourceTraceSHA256: sourceTrace, Decisions: cloned,
		Bytes: encoded, SHA256: sha256.Sum256(encoded),
	}, nil
}

func rankPrefixSourceHash(decisions []Decision) ([sha256.Size]byte, error) {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomadv3-choice-rank-prefix/v1"))
	_, _ = hasher.Write([]byte{0})
	for index, decision := range decisions {
		decision.Ordinal = uint64(index)
		record, err := EncodeRecord(decision.Record())
		if err != nil {
			return [sha256.Size]byte{}, errors.Join(ErrInvalidTape, fmt.Errorf("encode rank prefix decision %d: %w", index, err))
		}
		_, _ = hasher.Write(record[:])
	}
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result, nil
}

func decodeTape(encoded []byte, identity ExecutionIdentity) (Tape, error) {
	if len(encoded) < TapeHeaderBytes {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape is shorter than its header"))
	}
	header, err := DecodeTapeHeader(encoded[:TapeHeaderBytes])
	if err != nil {
		return Tape{}, errors.Join(ErrInvalidTape, err)
	}
	if header.TotalBytes != uint64(len(encoded)) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape byte length is inconsistent"))
	}
	payload := encoded[TapeHeaderBytes:]
	if len(payload)%TapeRecordBytes != 0 || header.Records != uint64(len(payload)/TapeRecordBytes) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape record count is inconsistent"))
	}
	headerIdentity, err := tapeHeaderIdentity(identity)
	if err != nil {
		return Tape{}, err
	}
	if header.TargetHash != identity.TargetSHA256 || header.ImplementationHash != identity.ImplementationSHA256 || header.ToolchainBuildKey != headerIdentity.toolchain || header.PlatformHash != headerIdentity.platform {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape execution identity does not match"))
	}
	if header.PayloadHash != sha256.Sum256(payload) {
		return Tape{}, errors.Join(ErrInvalidTape, errors.New("choice tape payload digest mismatch"))
	}
	decisions := make([]Decision, header.Records)
	for index := range decisions {
		record, err := DecodeRecord(payload[index*TapeRecordBytes : (index+1)*TapeRecordBytes])
		if err != nil {
			return Tape{}, errors.Join(ErrInvalidTape, fmt.Errorf("decode decision %d: %w", index, err))
		}
		if record.Ordinal != uint64(index) || record.Flags&FlagDecision == 0 {
			return Tape{}, errors.Join(ErrInvalidTape, fmt.Errorf("decision %d ordinal or role is invalid", index))
		}
		decisions[index], err = decisionFromRecord(record)
		if err != nil {
			return Tape{}, err
		}
	}
	copyBytes := append([]byte(nil), encoded...)
	return Tape{
		Identity: identity, SourceTraceSHA256: header.SourceTraceHash, Decisions: decisions,
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
		return encodedExecutionIdentity{}, errors.Join(ErrInvalidTape, errors.New("choice execution identity is incomplete"))
	}
	var result encodedExecutionIdentity
	if len(identity.ToolchainBuildKey) != hex.EncodedLen(len(result.toolchain)) {
		return encodedExecutionIdentity{}, errors.Join(ErrInvalidTape, errors.New("choice toolchain build key is malformed"))
	}
	if _, err := hex.Decode(result.toolchain[:], []byte(identity.ToolchainBuildKey)); err != nil || hex.EncodeToString(result.toolchain[:]) != identity.ToolchainBuildKey {
		return encodedExecutionIdentity{}, errors.Join(ErrInvalidTape, errors.New("choice toolchain build key is malformed"))
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomadv3-choice-platform/v1"))
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

func DivergenceFromTerminal(terminal Terminal) (Divergence, error) {
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

func ValidateDivergenceTerminal(tape Tape, mode Mode, terminal Terminal) (Divergence, error) {
	if mode != ModeReplay && mode != ModePrefix {
		return Divergence{}, errors.New("choice divergence validation requires replay or prefix mode")
	}
	var validated Tape
	var err error
	if mode == ModePrefix {
		validated, err = ValidatePrefixTape(tape, tape.Identity)
	} else {
		validated, err = ValidateDecisionTape(tape, tape.Identity)
	}
	if err != nil {
		return Divergence{}, err
	}
	divergence, err := DivergenceFromTerminal(terminal)
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
