package choicewire

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

const MissingSiteFingerprint = "missing"

var (
	ErrMalformed    = errors.New("malformed choice trace")
	ErrOverflow     = errors.New("choice trace overflow")
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

type Projection struct {
	Profile      string
	Limit        uint64
	PayloadBytes uint64
	SHA256       [sha256.Size]byte
	Summary      Summary
	Sites        []Site
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
	_, _ = hasher.Write([]byte("gomadv3-choice-implementation-v1"))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(ImplementationSourceSHA256[:])
	_, _ = hasher.Write(buildKey[:])
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result, nil
}

func ProjectComplete(payload []byte, metadata CompleteMetadata, targetIdentity [sha256.Size]byte) (Projection, error) {
	return Project(payload, TerminalMetadata{State: TerminalComplete, Limit: metadata.Limit, Records: metadata.Records, SHA256: metadata.SHA256}, targetIdentity)
}

func Project(payload []byte, metadata TerminalMetadata, targetIdentity [sha256.Size]byte) (Projection, error) {
	terminal := EncodeTerminal(Terminal{State: metadata.State, Records: metadata.Records, MappingBytes: HeaderBytes + uint64(len(payload)), PayloadHash: metadata.SHA256})
	trace, err := DecodeTrace(payload, terminal[:], metadata.Limit)
	if errors.Is(err, ErrOverflow) && metadata.State == TerminalOverflow {
		err = nil
	}
	if err != nil {
		return Projection{}, err
	}
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
		if key.missing {
			site.Fingerprint = MissingSiteFingerprint
		} else {
			var material [sha256.Size + 1 + 8]byte
			copy(material[:sha256.Size], targetIdentity[:])
			material[sha256.Size] = byte(record.Kind)
			binary.BigEndian.PutUint64(material[sha256.Size+1:], record.SiteOffset)
			digest := sha256.Sum256(material[:])
			site.Fingerprint = hex.EncodeToString(digest[:])
		}
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
		Profile: Profile, Limit: metadata.Limit, PayloadBytes: uint64(len(payload)), SHA256: metadata.SHA256, Summary: trace.Summary, Sites: sites,
	}, nil
}

func DecodeTrace(payload, terminalFrame []byte, mappingLimit uint64) (Trace, error) {
	if len(terminalFrame) == 0 {
		return Trace{}, ErrUnterminated
	}
	terminal, err := DecodeTerminal(terminalFrame)
	if err != nil {
		return Trace{}, errors.Join(ErrMalformed, err)
	}
	if terminal.MappingBytes > mappingLimit || terminal.MappingBytes != HeaderBytes+uint64(len(payload)) || len(payload)%RecordBytes != 0 || terminal.Records != uint64(len(payload))/RecordBytes {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace terminal bounds do not match payload"))
	}
	digest := sha256.Sum256(payload)
	if digest != terminal.PayloadHash {
		return Trace{}, errors.Join(ErrMalformed, errors.New("choice trace digest mismatch"))
	}
	result := Trace{
		Bytes:   append([]byte(nil), payload...),
		SHA256:  digest,
		Records: make([]Record, 0, terminal.Records),
		Summary: Summary{Records: terminal.Records, Terminal: terminal.State},
	}
	for offset := 0; offset < len(payload); offset += RecordBytes {
		record, decodeErr := DecodeRecord(payload[offset : offset+RecordBytes])
		if decodeErr != nil {
			return Trace{}, errors.Join(ErrMalformed, decodeErr)
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
	return result, nil
}
