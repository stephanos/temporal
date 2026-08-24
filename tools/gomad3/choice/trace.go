package choice

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

const (
	MissingSiteFingerprint     = "missing"
	MaximumAdjacentChoicePairs = 4096
)

var (
	ErrMalformed    = errors.New("malformed choice trace")
	ErrOverflow     = errors.New("choice trace overflow")
	ErrDiverged     = errors.New("choice replay divergence")
	ErrUnterminated = errors.New("unterminated choice trace")
)

type Summary struct {
	Records      uint64
	Branching    uint64
	Runnable     uint64
	SelectPoll   uint64
	SelectResult uint64
	Terminal     TerminalState
}

type Trace struct {
	Version int
	Bytes   []byte
	SHA256  [sha256.Size]byte
	Records []Record
	Summary Summary
}

type CompleteMetadata struct {
	Limit   uint64
	Records uint64
	SHA256  [sha256.Size]byte
}

type TerminalMetadata struct {
	State   TerminalState
	Limit   uint64
	Records uint64
	SHA256  [sha256.Size]byte
}

type Site struct {
	Fingerprint         string
	Kind                Kind
	Count               uint64
	MaximumAlternatives uint32
}

type FeatureKind string

const (
	FeatureRecordKind          FeatureKind = "record_kind"
	FeatureSite                FeatureKind = "site"
	FeatureBranchingSite       FeatureKind = "branching_site"
	FeatureSelectedAlternative FeatureKind = "selected_alternative"
	FeatureAdjacentPair        FeatureKind = "adjacent_pair"
	FeatureTerminal            FeatureKind = "terminal"
)

type Feature struct {
	Kind  FeatureKind `json:"kind"`
	Value string      `json:"value"`
}

type FeatureProjection struct {
	Values                 []Feature `json:"values"`
	AdjacentPairsObserved  uint64    `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool      `json:"adjacent_pairs_truncated"`
}

func (feature Feature) ID() string {
	return string(feature.Kind) + "/" + feature.Value
}

type Projection struct {
	Profile      string
	Limit        uint64
	PayloadBytes uint64
	SHA256       [sha256.Size]byte
	Summary      Summary
	Sites        []Site
	Features     FeatureProjection
}

func ImplementationIdentity(toolchainBuildKey string) ([sha256.Size]byte, error) {
	var buildKey [sha256.Size]byte
	if len(toolchainBuildKey) != hex.EncodedLen(len(buildKey)) {
		return [sha256.Size]byte{}, errors.New("choice implementation toolchain build key is malformed")
	}
	if _, err := hex.Decode(buildKey[:], []byte(toolchainBuildKey)); err != nil || hex.EncodeToString(buildKey[:]) != toolchainBuildKey {
		return [sha256.Size]byte{}, errors.New("choice implementation toolchain build key is malformed")
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad3-choice-implementation-v2"))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(implementationSourceSHA256[:])
	_, _ = hasher.Write(buildKey[:])
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result, nil
}

func ProjectComplete(payload []byte, metadata CompleteMetadata, targetIdentity [sha256.Size]byte) (Projection, error) {
	return Project(payload, TerminalMetadata{State: TerminalComplete, Limit: metadata.Limit, Records: metadata.Records, SHA256: metadata.SHA256}, targetIdentity)
}

func Project(payload []byte, metadata TerminalMetadata, targetIdentity [sha256.Size]byte) (Projection, error) {
	trace, err := DecodeStoredTrace(Profile, payload, metadata)
	if errors.Is(err, ErrOverflow) && metadata.State == TerminalOverflow {
		err = nil
	}
	if err != nil {
		return Projection{}, err
	}
	return projectTrace(trace, metadata.Limit, targetIdentity)
}

func DecodeStoredTrace(profile string, payload []byte, metadata TerminalMetadata) (Trace, error) {
	switch profile {
	case Profile:
		terminal := encodeTerminal(terminal{State: metadata.State, Records: metadata.Records, MappingBytes: traceHeaderBytes + uint64(len(payload)), PayloadHash: metadata.SHA256})
		return DecodeTrace(payload, terminal[:], metadata.Limit)
	case LegacyProfile:
		return decodeStoredLegacyV1Trace(payload, metadata)
	default:
		return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("unsupported choice trace profile %q", profile))
	}
}

func BuildTrace(records []Record, state TerminalState) (Trace, error) {
	if state != TerminalComplete && state != TerminalOverflow {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace fixture terminal is invalid"))
	}
	payload := make([]byte, 0, len(records)*traceRecordBytes)
	for index, record := range records {
		if record.Ordinal != uint64(index) {
			return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("choice trace ordinal %d at record %d", record.Ordinal, index))
		}
		encoded, err := encodeRecord(record)
		if err != nil {
			return Trace{}, errors.Join(ErrMalformed, err)
		}
		payload = append(payload, encoded[:]...)
	}
	limit, err := TraceBytes(uint64(len(records)))
	if err != nil {
		return Trace{}, err
	}
	completed := encodeTerminal(terminal{
		State: state, Records: uint64(len(records)), MappingBytes: limit, PayloadHash: sha256.Sum256(payload),
	})
	return DecodeTrace(payload, completed[:], limit)
}

func TraceRecordCount(encoded []byte) (uint64, error) {
	if len(encoded)%traceRecordBytes != 0 {
		return 0, errors.Join(ErrMalformed, fmt.Errorf("choice trace has invalid length %d", len(encoded)))
	}
	return uint64(len(encoded) / traceRecordBytes), nil
}

func ProjectTrace(trace Trace, limit uint64, targetIdentity [sha256.Size]byte) (Projection, error) {
	profile := Profile
	if trace.Version == Version1 {
		profile = LegacyProfile
	} else if trace.Version != Version2 {
		return Projection{}, errors.Join(ErrMalformed, fmt.Errorf("unsupported choice trace version %d", trace.Version))
	}
	validated, err := DecodeStoredTrace(profile, trace.Bytes, TerminalMetadata{State: trace.Summary.Terminal, Limit: limit, Records: trace.Summary.Records, SHA256: trace.SHA256})
	if errors.Is(err, ErrOverflow) && trace.Summary.Terminal == TerminalOverflow {
		err = nil
	}
	if err != nil {
		return Projection{}, err
	}
	return projectTrace(validated, limit, targetIdentity)
}

func projectTrace(trace Trace, limit uint64, targetIdentity [sha256.Size]byte) (Projection, error) {
	type siteKey struct {
		kind    Kind
		offset  uint64
		missing bool
	}
	bySite := make(map[siteKey]Site)
	for _, record := range trace.Records {
		key := siteKey{kind: record.Kind, offset: record.SiteOffset, missing: record.Flags&FlagSiteMissing != 0}
		site := bySite[key]
		site.Kind = record.Kind
		site.Count++
		if record.Alternatives > site.MaximumAlternatives {
			site.MaximumAlternatives = record.Alternatives
		}
		site.Fingerprint = siteFingerprint(record, targetIdentity)
		bySite[key] = site
	}
	sites := make([]Site, 0, len(bySite))
	for _, site := range bySite {
		sites = append(sites, site)
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].Kind != sites[j].Kind {
			return sites[i].Kind < sites[j].Kind
		}
		return sites[i].Fingerprint < sites[j].Fingerprint
	})
	return Projection{
		Profile: profileForVersion(trace.Version), Limit: limit, PayloadBytes: uint64(len(trace.Bytes)), SHA256: trace.SHA256, Summary: trace.Summary, Sites: sites,
		Features: projectFeatures(trace, sites, targetIdentity),
	}, nil
}

func profileForVersion(version int) string {
	if version == Version1 {
		return LegacyProfile
	}
	return Profile
}

func projectFeatures(trace Trace, sites []Site, targetIdentity [sha256.Size]byte) FeatureProjection {
	recordFeatures := projectRecordFeatures(trace.Records, targetIdentity)
	result := recordFeatures.projection
	appendUniqueFeatures(&result, FeatureRecordKind, recordFeatures.recordKinds)

	siteFeatures := make([]string, 0, len(sites))
	for _, site := range sites {
		siteFeatures = append(siteFeatures, kindName(site.Kind)+"/"+site.Fingerprint)
	}
	appendUniqueFeatures(&result, FeatureSite, siteFeatures)

	branchingSites := make([]string, 0, len(sites))
	for _, site := range sites {
		maximum := recordFeatures.decisionMaximums[kindName(site.Kind)+"/"+site.Fingerprint]
		if maximum > 1 {
			branchingSites = append(branchingSites, fmt.Sprintf("%s/%s/max-alternatives=%d", kindName(site.Kind), site.Fingerprint, maximum))
		}
	}
	appendUniqueFeatures(&result, FeatureBranchingSite, branchingSites)
	appendUniqueFeatures(&result, FeatureSelectedAlternative, recordFeatures.selectedAlternatives)
	appendUniqueFeatures(&result, FeatureAdjacentPair, recordFeatures.pairs)
	appendUniqueFeatures(&result, FeatureTerminal, []string{terminalName(trace.Summary.Terminal)})
	return result
}

type recordFeatureProjection struct {
	projection           FeatureProjection
	recordKinds          []string
	selectedAlternatives []string
	decisionMaximums     map[string]uint32
	pairs                []string
}

func projectRecordFeatures(records []Record, targetIdentity [sha256.Size]byte) recordFeatureProjection {
	result := recordFeatureProjection{
		projection:  FeatureProjection{Values: []Feature{}},
		recordKinds: make([]string, 0, 3), selectedAlternatives: make([]string, 0, len(records)),
		decisionMaximums: make(map[string]uint32), pairs: make([]string, 0, min(len(records), MaximumAdjacentChoicePairs)),
	}
	seenRecordKinds := make(map[string]struct{}, 3)
	seenSelectedAlternatives := make(map[string]struct{}, len(records))
	seenPairs := make(map[string]struct{}, min(len(records), MaximumAdjacentChoicePairs))
	previousIdentity := ""
	for _, record := range records {
		role := recordRole(record)
		kind := kindName(record.Kind)
		site := siteFingerprint(record, targetIdentity)
		selectedClass := alternativeClass(record.Alternatives, record.Selected)
		recordKind := role + "/" + kind
		if _, found := seenRecordKinds[recordKind]; !found {
			seenRecordKinds[recordKind] = struct{}{}
			result.recordKinds = append(result.recordKinds, recordKind)
		}
		recordIdentity := role + "/" + kind + "/" + site + "/" + selectedClass
		if _, found := seenSelectedAlternatives[recordIdentity]; !found {
			seenSelectedAlternatives[recordIdentity] = struct{}{}
			result.selectedAlternatives = append(result.selectedAlternatives, recordIdentity)
		}
		if previousIdentity != "" {
			result.projection.AdjacentPairsObserved++
			pair := previousIdentity + "->" + recordIdentity
			if _, duplicate := seenPairs[pair]; !duplicate {
				if len(result.pairs) == MaximumAdjacentChoicePairs {
					result.projection.AdjacentPairsTruncated = true
				} else {
					seenPairs[pair] = struct{}{}
					result.pairs = append(result.pairs, pair)
				}
			}
		}
		previousIdentity = recordIdentity
		key := kind + "/" + site
		if record.Flags&FlagDecision != 0 && record.Alternatives > result.decisionMaximums[key] {
			result.decisionMaximums[key] = record.Alternatives
		}
	}
	return result
}

func appendUniqueFeatures(projection *FeatureProjection, kind FeatureKind, values []string) {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		projection.Values = append(projection.Values, Feature{Kind: kind, Value: value})
	}
}

func siteFingerprint(record Record, targetIdentity [sha256.Size]byte) string {
	if record.Flags&FlagSiteMissing != 0 {
		return MissingSiteFingerprint
	}
	var material [sha256.Size + 1 + 8]byte
	copy(material[:sha256.Size], targetIdentity[:])
	material[sha256.Size] = byte(record.Kind)
	binary.BigEndian.PutUint64(material[sha256.Size+1:], record.SiteOffset)
	digest := sha256.Sum256(material[:])
	return hex.EncodeToString(digest[:])
}

func recordRole(record Record) string {
	if record.Flags&FlagDecision != 0 {
		return "decision"
	}
	return "observation"
}

func kindName(kind Kind) string {
	switch kind {
	case KindRunnable:
		return "runnable"
	case KindSelectPoll:
		return "select-poll"
	case KindSelectResult:
		return "select-result"
	default:
		return "unknown"
	}
}

func alternativeClass(alternatives, selected uint32) string {
	switch {
	case alternatives == 1:
		return "only"
	case selected == 0:
		return "first"
	case selected == alternatives-1:
		return "last"
	default:
		return "interior"
	}
}

func terminalName(terminal TerminalState) string {
	if terminal == TerminalOverflow {
		return "overflow"
	}
	if terminal == TerminalDiverged {
		return "diverged"
	}
	return "complete"
}

func DecodeTrace(payload, terminalFrame []byte, mappingLimit uint64) (Trace, error) {
	if len(terminalFrame) == 0 {
		return Trace{}, ErrUnterminated
	}
	terminal, err := decodeTerminal(terminalFrame)
	if err != nil {
		return Trace{}, errors.Join(ErrMalformed, err)
	}
	if terminal.MappingBytes > mappingLimit || terminal.MappingBytes != traceHeaderBytes+uint64(len(payload)) || len(payload)%traceRecordBytes != 0 || terminal.Records != uint64(len(payload))/traceRecordBytes {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace terminal bounds do not match payload"))
	}
	digest := sha256.Sum256(payload)
	if digest != terminal.PayloadHash {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace digest mismatch"))
	}
	result := Trace{
		Version: Version2,
		Bytes:   append([]byte(nil), payload...),
		SHA256:  digest,
		Records: make([]Record, 0, terminal.Records),
		Summary: Summary{Records: terminal.Records, Terminal: terminal.State},
	}
	for offset := 0; offset < len(payload); offset += traceRecordBytes {
		record, decodeErr := decodeRecord(payload[offset : offset+traceRecordBytes])
		if decodeErr != nil {
			return Trace{}, errors.Join(ErrMalformed, decodeErr)
		}
		if record.Flags&FlagRankOverride != 0 {
			return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace contains a controller-only rank override"))
		}
		if record.Ordinal != uint64(len(result.Records)) {
			return Trace{}, errors.Join(ErrMalformed, fmt.Errorf("choice trace ordinal %d at record %d", record.Ordinal, len(result.Records)))
		}
		result.Records = append(result.Records, record)
		if record.Flags&FlagDecision != 0 && record.Alternatives > 1 {
			result.Summary.Branching++
		}
		switch record.Kind {
		case KindRunnable:
			result.Summary.Runnable++
		case KindSelectPoll:
			result.Summary.SelectPoll++
		case KindSelectResult:
			result.Summary.SelectResult++
		default:
		}
	}
	if terminal.State == TerminalOverflow {
		return result, ErrOverflow
	}
	if terminal.State == TerminalDiverged {
		return result, ErrDiverged
	}
	return result, nil
}
