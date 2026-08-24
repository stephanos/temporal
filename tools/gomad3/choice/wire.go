package choice

import (
	"crypto/sha256"

	"go.temporal.io/server/tools/gomad3/choice/internal/wire"
)

const (
	Profile = wire.Profile

	traceHeaderBytes      = wire.HeaderBytes
	traceRecordBytes      = wire.RecordBytes
	replayPlanHeaderBytes = wire.TapeHeaderBytes
	replayPlanRecordBytes = wire.TapeRecordBytes
	terminalFrameBytes    = wire.TerminalFrameBytes

	Version1 = wire.Version1
	Version2 = wire.Version2
)

type Kind uint8

const (
	KindRunnable     Kind = Kind(wire.KindRunnable)
	KindSelectPoll   Kind = Kind(wire.KindSelectPoll)
	KindSelectResult Kind = Kind(wire.KindSelectResult)
)

type Flags uint8

const (
	FlagDecision     Flags = Flags(wire.FlagDecision)
	FlagObservation  Flags = Flags(wire.FlagObservation)
	FlagSiteMissing  Flags = Flags(wire.FlagSiteMissing)
	FlagRankOverride Flags = Flags(wire.FlagRankOverride)
)

type Mode uint8

const (
	ModeSeed   Mode = Mode(wire.ModeSeed)
	ModeRecord Mode = Mode(wire.ModeRecord)
	ModeReplay Mode = Mode(wire.ModeReplay)
	ModePrefix Mode = Mode(wire.ModePrefix)
)

type DivergenceReason uint8

const (
	DivergenceKind                DivergenceReason = DivergenceReason(wire.DivergenceKind)
	DivergenceSite                DivergenceReason = DivergenceReason(wire.DivergenceSite)
	DivergenceAlternatives        DivergenceReason = DivergenceReason(wire.DivergenceAlternatives)
	DivergenceSelected            DivergenceReason = DivergenceReason(wire.DivergenceSelected)
	DivergenceAlternativeSet      DivergenceReason = DivergenceReason(wire.DivergenceAlternativeSet)
	DivergenceTapeExhausted       DivergenceReason = DivergenceReason(wire.DivergenceTapeExhausted)
	DivergenceTapeUnconsumed      DivergenceReason = DivergenceReason(wire.DivergenceTapeUnconsumed)
	DivergenceIdentityMissing     DivergenceReason = DivergenceReason(wire.DivergenceIdentityMissing)
	DivergenceIdentityDuplicate   DivergenceReason = DivergenceReason(wire.DivergenceIdentityDuplicate)
	DivergenceAlternativeCapacity DivergenceReason = DivergenceReason(wire.DivergenceAlternativeCapacity)
	DivergenceObservation         DivergenceReason = DivergenceReason(wire.DivergenceObservation)
)

type TerminalState uint8

const (
	TerminalComplete TerminalState = TerminalState(wire.TerminalComplete)
	TerminalOverflow TerminalState = TerminalState(wire.TerminalOverflow)
	TerminalDiverged TerminalState = TerminalState(wire.TerminalDiverged)
)

var implementationSourceSHA256 = wire.ImplementationSourceSHA256

type Record struct {
	Ordinal              uint64
	Kind                 Kind
	Flags                Flags
	Alternatives         uint32
	Selected             uint32
	Data                 uint32
	SiteOffset           uint64
	SelectedIdentity     [sha256.Size]byte
	AlternativeSetDigest [sha256.Size]byte
}

type replayPlanHeader struct {
	TotalBytes         uint64
	Records            uint64
	SourceTraceHash    [sha256.Size]byte
	TargetHash         [sha256.Size]byte
	ImplementationHash [sha256.Size]byte
	ToolchainBuildKey  [sha256.Size]byte
	PlatformHash       [sha256.Size]byte
	PayloadHash        [sha256.Size]byte
}

type terminal struct {
	State            TerminalState
	Records          uint64
	MappingBytes     uint64
	PayloadHash      [sha256.Size]byte
	DivergenceReason DivergenceReason
	DivergentOrdinal uint64
	TapeRecords      uint64
	ExpectedPresent  bool
	ObservedPresent  bool
	Expected         Record
	Observed         Record
}

func encodeTraceHeader(capacity uint64) [wire.HeaderBytes]byte {
	return wire.EncodeHeader(capacity)
}

func decodeTraceHeader(encoded []byte) (capacity, nextOffset, records uint64, err error) {
	header, err := wire.DecodeHeader(encoded)
	return header.Capacity, header.NextOffset, header.RecordCount, err
}

func publishTraceHeader(encoded []byte, nextOffset, records uint64) error {
	return wire.PublishHeader(encoded, nextOffset, records)
}

func encodeRecord(value Record) ([wire.RecordBytes]byte, error) {
	return wire.EncodeRecord(toWireRecord(value))
}

func decodeRecord(encoded []byte) (Record, error) {
	value, err := wire.DecodeRecord(encoded)
	return fromWireRecord(value), err
}

func encodeReplayPlanHeader(value replayPlanHeader) ([wire.TapeHeaderBytes]byte, error) {
	return wire.EncodeTapeHeader(wire.TapeHeader{
		TotalBytes: value.TotalBytes, Records: value.Records, SourceTraceHash: value.SourceTraceHash,
		TargetHash: value.TargetHash, ImplementationHash: value.ImplementationHash,
		ToolchainBuildKey: value.ToolchainBuildKey, PlatformHash: value.PlatformHash, PayloadHash: value.PayloadHash,
	})
}

func decodeReplayPlanHeader(encoded []byte) (replayPlanHeader, error) {
	value, err := wire.DecodeTapeHeader(encoded)
	return replayPlanHeader{
		TotalBytes: value.TotalBytes, Records: value.Records, SourceTraceHash: value.SourceTraceHash,
		TargetHash: value.TargetHash, ImplementationHash: value.ImplementationHash,
		ToolchainBuildKey: value.ToolchainBuildKey, PlatformHash: value.PlatformHash, PayloadHash: value.PayloadHash,
	}, err
}

func encodeTerminal(value terminal) [wire.TerminalFrameBytes]byte {
	return wire.EncodeTerminal(wire.Terminal{
		State: wire.TerminalState(value.State), Records: value.Records, MappingBytes: value.MappingBytes,
		PayloadHash: value.PayloadHash, DivergenceReason: wire.DivergenceReason(value.DivergenceReason),
		DivergentOrdinal: value.DivergentOrdinal, TapeRecords: value.TapeRecords,
		ExpectedPresent: value.ExpectedPresent, ObservedPresent: value.ObservedPresent,
		Expected: toWireRecord(value.Expected), Observed: toWireRecord(value.Observed),
	})
}

func decodeTerminal(encoded []byte) (terminal, error) {
	value, err := wire.DecodeTerminal(encoded)
	return terminal{
		State: TerminalState(value.State), Records: value.Records, MappingBytes: value.MappingBytes,
		PayloadHash: value.PayloadHash, DivergenceReason: DivergenceReason(value.DivergenceReason),
		DivergentOrdinal: value.DivergentOrdinal, TapeRecords: value.TapeRecords,
		ExpectedPresent: value.ExpectedPresent, ObservedPresent: value.ObservedPresent,
		Expected: fromWireRecord(value.Expected), Observed: fromWireRecord(value.Observed),
	}, err
}

func toWireRecord(value Record) wire.Record {
	return wire.Record{
		Ordinal: value.Ordinal, Kind: wire.Kind(value.Kind), Flags: wire.Flags(value.Flags),
		Alternatives: value.Alternatives, Selected: value.Selected, Data: value.Data, SiteOffset: value.SiteOffset,
		SelectedIdentity: value.SelectedIdentity, AlternativeSetDigest: value.AlternativeSetDigest,
	}
}

func fromWireRecord(value wire.Record) Record {
	return Record{
		Ordinal: value.Ordinal, Kind: Kind(value.Kind), Flags: Flags(value.Flags),
		Alternatives: value.Alternatives, Selected: value.Selected, Data: value.Data, SiteOffset: value.SiteOffset,
		SelectedIdentity: value.SelectedIdentity, AlternativeSetDigest: value.AlternativeSetDigest,
	}
}
