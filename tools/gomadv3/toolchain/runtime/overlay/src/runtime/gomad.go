// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	"internal/chacha8rand"
	"internal/goarch"
	"internal/runtime/atomic"
	"internal/runtime/exithook"
	"internal/runtime/math"
)

var gomadEnabled bool
var gomadSeed uint64
var gomadExternal bool
var gomadIOProfile bool
var gomadConfigPresent bool
var gomadConfig [212]byte
var gomadSimulationTimeEnabled bool
var gomadSimulationTimeRequestDescriptor int32
var gomadSimulationTimeResponseDescriptor int32
var gomadSimulationTimeGeneration uint64
var gomadSimulationTimeQuiescing bool
var gomadSimulationTimeAwaitingExternal atomic.Bool
var gomadSimulationTimeArrivals atomic.Uint32
var gomadSimulationTimeArrivalEpoch atomic.Uint64
var gomadSimulationExternalRequests atomic.Int32
var gomadSimulationTransportSyscalls atomic.Int32

var gomadChoiceEnabled bool
var gomadChoiceMode uint8
var gomadChoiceMapping unsafe.Pointer
var gomadChoiceMappingBytes uint64
var gomadChoiceTerminalDescriptor int32
var gomadChoiceTape unsafe.Pointer
var gomadChoiceTapeBytes uint64
var gomadChoiceTapeRecords uint64
var gomadChoiceTapeCursor uint64
var gomadChoiceDecisionRecords uint64
var gomadRuntimeGoroutineOrdinal atomic.Uint64
var gomadChoiceNext atomic.Uint64
var gomadChoiceRecords atomic.Uint64
var gomadChoiceOverflow atomic.Uint32
var gomadChoiceFinalized atomic.Uint32
var gomadChoiceHookRegistered bool
var gomadChoiceRunqRandom chacha8rand.State
var gomadChoiceSchedulerRandom chacha8rand.State
var gomadChoiceSelectRandom uint64

const gomadInitialTime = 946684800000000000
const gomadMapShared = 1
const gomadChoiceMaximumAlternatives = 256
const gomadSimulationTimeRequestBytes = 40
const gomadSimulationTimeResponseBytes = 32

const (
	gomadSimulationTimeResponseAdvance = iota + 1
	gomadSimulationTimeResponseRetry
	gomadSimulationTimeResponseDeadlock
	gomadSimulationTimeResponseExternal
)

var gomadSimulationTimeRequestMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'Q', 1}
var gomadSimulationTimeResponseMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'R', 1}

func gomadInit() {
	var seed uint64
	choiceConfigured := gomadChoiceConfigured()
	if choiceConfigured {
		gomadChoiceInit()
	}
	_, profile := gomadEnv("GOMADV3_IO_PROFILE=")
	if profile {
		if !gomadReadConfig() {
			print("runtime: missing Gomad bootstrap configuration\n")
			exit(2)
		}
		seed = gomadConfigSeed()
	} else {
		value, present := gomadSeedEnv()
		if !present && !choiceConfigured {
			return
		}
		var ok bool
		if present {
			seed, ok = gomadParseSeed(value)
		}
		if present && !ok {
			print("runtime: invalid GOMADSEED\n")
			exit(2)
		}
	}
	if iscgo || gomadExternal {
		print("runtime: GOMADSEED does not support cgo or external linking\n")
		exit(2)
	}

	gomadEnabled = true
	gomadSeed = seed
	faketime = gomadInitialTime
	gomadSimulationTimeInit()
	debug.asyncpreemptoff = 1
	haveSysmon = false
	randomizeScheduler = true
}

//go:noinline
func gomadCapabilityGuard() {
	if gomadEnabled {
		throw("GOMAD_CAPABILITY_DENIED")
	}
}

func gomadSimulationTimeInit() {
	requestValue, requestPresent := gomadEnvEarly("GOMADV3_SIMULATION_TIME_REQUEST_FD=")
	responseValue, responsePresent := gomadEnvEarly("GOMADV3_SIMULATION_TIME_RESPONSE_FD=")
	if !requestPresent && !responsePresent {
		return
	}
	request, requestOK := gomadParseSeed(requestValue)
	response, responseOK := gomadParseSeed(responseValue)
	if !requestPresent || !responsePresent || !requestOK || !responseOK || request < 3 || response < 3 || request > 1<<31-1 || response > 1<<31-1 || request == response {
		print("runtime: invalid Gomad simulation time configuration\n")
		exit(2)
	}
	gomadSimulationTimeEnabled = true
	gomadSimulationTimeRequestDescriptor = int32(request)
	gomadSimulationTimeResponseDescriptor = int32(response)
}

func gomadChoiceConfigured() bool {
	_, configured := gomadEnvEarly("GOMADV3_CHOICE_TRACE_FD=")
	return configured
}

func gomadChoiceInit() {
	descriptorValue, enabled := gomadEnvEarly("GOMADV3_CHOICE_TRACE_FD=")
	if !enabled {
		return
	}
	terminalValue, terminalPresent := gomadEnvEarly("GOMADV3_CHOICE_TERMINAL_FD=")
	bytesValue, bytesPresent := gomadEnvEarly("GOMADV3_CHOICE_TRACE_BYTES=")
	modeValue, modePresent := gomadEnvEarly("GOMADV3_CHOICE_MODE=")
	descriptor, descriptorOK := gomadParseSeed(descriptorValue)
	terminal, terminalOK := gomadParseSeed(terminalValue)
	mappingBytes, bytesOK := gomadParseSeed(bytesValue)
	mode, modeOK := gomadParseSeed(modeValue)
	if !terminalPresent || !bytesPresent || !modePresent || !descriptorOK || !terminalOK || !bytesOK || !modeOK || descriptor > 1<<31-1 || terminal > 1<<31-1 || mappingBytes < gomadChoiceHeaderBytes+gomadChoiceRecordBytes || mappingBytes > 64<<20 || mode < gomadChoiceModeRecord || mode > gomadChoiceModePrefix {
		print("runtime: invalid Gomad choice trace configuration\n")
		exit(2)
	}
	mapped, errno := mmap(nil, uintptr(mappingBytes), _PROT_READ|_PROT_WRITE, gomadMapShared, int32(descriptor), 0)
	if errno != 0 {
		print("runtime: could not map Gomad choice trace\n")
		exit(2)
	}
	bytes := unsafe.Slice((*byte)(mapped), int(mappingBytes))
	for index := range gomadChoiceTraceMagic {
		if bytes[index] != gomadChoiceTraceMagic[index] {
			print("runtime: invalid Gomad choice trace backing\n")
			exit(2)
		}
	}
	if gomadChoiceRead32(bytes[8:12]) != gomadChoiceWireVersion || gomadChoiceRead64(bytes[16:24]) != mappingBytes || gomadChoiceRead64(bytes[24:32]) != gomadChoiceHeaderBytes || gomadChoiceRead64(bytes[32:40]) != 0 {
		print("runtime: invalid Gomad choice trace header\n")
		exit(2)
	}
	gomadChoiceEnabled = true
	gomadChoiceMode = uint8(mode)
	gomadChoiceMapping = mapped
	gomadChoiceMappingBytes = mappingBytes
	gomadChoiceTerminalDescriptor = int32(terminal)
	gomadChoiceNext.Store(gomadChoiceHeaderBytes)
	if gomadChoiceMode == gomadChoiceModeRecord {
		if _, present := gomadEnvEarly("GOMADV3_CHOICE_TAPE_FD="); present {
			print("runtime: choice record mode cannot use a tape\n")
			exit(2)
		}
		return
	}
	gomadChoiceInitTape()
}

func gomadChoiceInitTape() {
	descriptorValue, descriptorPresent := gomadEnvEarly("GOMADV3_CHOICE_TAPE_FD=")
	bytesValue, bytesPresent := gomadEnvEarly("GOMADV3_CHOICE_TAPE_BYTES=")
	descriptor, descriptorOK := gomadParseSeed(descriptorValue)
	tapeBytes, bytesOK := gomadParseSeed(bytesValue)
	if !descriptorPresent || !bytesPresent || !descriptorOK || !bytesOK || descriptor > 1<<31-1 || tapeBytes < gomadChoiceTapeHeaderBytes || tapeBytes > 64<<20+gomadChoiceTapeHeaderBytes-gomadChoiceHeaderBytes {
		print("runtime: invalid Gomad choice tape configuration\n")
		exit(2)
	}
	mapped, errno := mmap(nil, uintptr(tapeBytes), _PROT_READ, gomadMapShared, int32(descriptor), 0)
	if errno != 0 {
		print("runtime: could not map Gomad choice tape\n")
		exit(2)
	}
	bytes := unsafe.Slice((*byte)(mapped), int(tapeBytes))
	for index := range gomadChoiceTapeMagic {
		if bytes[index] != gomadChoiceTapeMagic[index] {
			print("runtime: invalid Gomad choice tape magic\n")
			exit(2)
		}
	}
	records := gomadChoiceRead64(bytes[32:40])
	if gomadChoiceRead32(bytes[8:12]) != gomadChoiceWireVersion || gomadChoiceRead32(bytes[12:16]) != gomadChoiceTapeHeaderBytes || gomadChoiceRead32(bytes[16:20]) != gomadChoiceTapeRecordBytes || !gomadChoiceZero(bytes[20:24]) || gomadChoiceRead64(bytes[24:32]) != tapeBytes || records > (tapeBytes-gomadChoiceTapeHeaderBytes)/gomadChoiceTapeRecordBytes || gomadChoiceTapeHeaderBytes+records*gomadChoiceTapeRecordBytes != tapeBytes {
		print("runtime: invalid Gomad choice tape header\n")
		exit(2)
	}
	checksum := gomadChoiceHash(bytes[:gomadChoiceTapeChecksumOffset])
	if !gomadChoiceEqual(checksum[:], bytes[gomadChoiceTapeChecksumOffset:gomadChoiceTapeHeaderBytes]) {
		print("runtime: invalid Gomad choice tape checksum\n")
		exit(2)
	}
	payloadHash := gomadChoiceHash(bytes[gomadChoiceTapeHeaderBytes:])
	if !gomadChoiceEqual(payloadHash[:], bytes[200:232]) {
		print("runtime: invalid Gomad choice tape payload\n")
		exit(2)
	}
	for ordinal := uint64(0); ordinal < records; ordinal++ {
		record := bytes[gomadChoiceTapeHeaderBytes+ordinal*gomadChoiceTapeRecordBytes : gomadChoiceTapeHeaderBytes+(ordinal+1)*gomadChoiceTapeRecordBytes]
		if !gomadChoiceValidDecisionRecord(record, ordinal, records) {
			print("runtime: invalid Gomad choice tape record\n")
			exit(2)
		}
	}
	gomadChoiceTape = mapped
	gomadChoiceTapeBytes = tapeBytes
	gomadChoiceTapeRecords = records
}

func gomadEnvEarly(prefix string) (string, bool) {
	n := int32(0)
	for argv_index(argv, argc+1+n) != nil {
		n++
	}
	for i := int32(0); i < n; i++ {
		value := gostringnocopy(argv_index(argv, argc+1+i))
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return value[len(prefix):], true
		}
	}
	return "", false
}

type gomadChoiceRecordValue struct {
	ordinal          uint64
	kind             uint8
	flags            uint8
	alternatives     uint32
	selected         uint32
	data             uint32
	siteOffset       uint64
	selectedIdentity [32]byte
	alternativeSet   [32]byte
}

func gomadChoiceAppendRecord(value gomadChoiceRecordValue) {
	if !gomadChoiceEnabled || gomadChoiceOverflow.Load() != 0 || gomadChoiceFinalized.Load() != 0 {
		return
	}
	offset := gomadChoiceNext.Add(gomadChoiceRecordBytes) - gomadChoiceRecordBytes
	if offset > gomadChoiceMappingBytes-gomadChoiceRecordBytes {
		gomadChoiceOverflow.Store(1)
		return
	}
	ordinal := (offset - gomadChoiceHeaderBytes) / gomadChoiceRecordBytes
	bytes := unsafe.Slice((*byte)(gomadChoiceMapping), int(gomadChoiceMappingBytes))
	record := bytes[offset : offset+gomadChoiceRecordBytes]
	for index := range record {
		record[index] = 0
	}
	value.ordinal = ordinal
	gomadChoiceEncodeRecord(record, value)
	gomadChoiceRecords.Store(ordinal + 1)
	gomadChoicePut64(bytes[24:32], offset+gomadChoiceRecordBytes)
	gomadChoicePut64(bytes[32:40], ordinal+1)
}

func gomadChoiceRecord(kind, flags uint8, siteOffset uint64, alternatives, selected, data uint32) {
	gomadChoiceAppendRecord(gomadChoiceRecordValue{kind: kind, flags: flags, siteOffset: siteOffset, alternatives: alternatives, selected: selected, data: data})
}

func gomadChoiceRunqSeeded(n uint32) uint32 {
	if !gomadChoiceEnabled {
		return randn(n)
	}
	return gomadChoiceRandom(&gomadChoiceRunqRandom, n)
}

func gomadChoiceRunnextSeeded(n uint32) uint32 {
	if !gomadChoiceEnabled {
		return randn(n)
	}
	return gomadChoiceRandom(&gomadChoiceSchedulerRandom, n)
}

func gomadChoiceShuffleSeeded(n uint32) uint32 {
	if !gomadChoiceEnabled {
		return cheaprandn(n)
	}
	return gomadChoiceRandom(&gomadChoiceSchedulerRandom, n)
}

func gomadChoiceRandom(random *chacha8rand.State, n uint32) uint32 {
	for {
		value, ok := random.Next()
		if ok {
			return uint32((uint64(uint32(value)) * uint64(n)) >> 32)
		}
		random.Refill()
	}
}

func gomadChoiceSelectSeeded(n uint32) uint32 {
	if !gomadChoiceEnabled {
		return cheaprandn(n)
	}
	gomadChoiceSelectRandom += 0xa0761d6478bd642f
	if goarch.IsAmd64|goarch.IsArm64|goarch.IsPpc64|
		goarch.IsPpc64le|goarch.IsMips64|goarch.IsMips64le|
		goarch.IsS390x|goarch.IsRiscv64|goarch.IsLoong64 == 1 {
		hi, lo := math.Mul64(gomadChoiceSelectRandom, gomadChoiceSelectRandom^0xe7037ed1a0b428db)
		return uint32((uint64(uint32(hi^lo)) * uint64(n)) >> 32)
	}
	t := (*[2]uint32)(unsafe.Pointer(&gomadChoiceSelectRandom))
	s1, s0 := t[0], t[1]
	s1 ^= s1 << 17
	s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16
	t[0], t[1] = s0, s1
	return uint32((uint64(s0+s1) * uint64(n)) >> 32)
}

func gomadChoiceDecision(kind, flags uint8, siteOffset uint64, alternatives [][32]byte, seeded, data uint32) uint32 {
	if !gomadChoiceEnabled {
		return seeded
	}
	if len(alternatives) == 0 || len(alternatives) > gomadChoiceMaximumAlternatives || seeded >= uint32(len(alternatives)) {
		gomadChoiceDivergeCurrent(gomadChoiceDivergenceAlternativeCapacity)
	}
	var ordered [gomadChoiceMaximumAlternatives][32]byte
	for index := range alternatives {
		if gomadChoiceZero(alternatives[index][:]) {
			gomadChoiceDivergeCurrent(gomadChoiceDivergenceIdentityMissing)
		}
		ordered[index] = alternatives[index]
		for previous := 0; previous < index; previous++ {
			if gomadChoiceEqual(alternatives[index][:], alternatives[previous][:]) {
				gomadChoiceDivergeCurrent(gomadChoiceDivergenceIdentityDuplicate)
			}
		}
	}
	for index := 1; index < len(alternatives); index++ {
		for current := index; current > 0 && gomadChoiceCompare(ordered[current][:], ordered[current-1][:]) < 0; current-- {
			ordered[current], ordered[current-1] = ordered[current-1], ordered[current]
		}
	}
	setDigest := gomadChoiceAlternativeSet(ordered[:len(alternatives)])
	selectedIdentity := alternatives[seeded]
	selectedRank := uint32(0)
	for index := range alternatives {
		if gomadChoiceEqual(ordered[index][:], selectedIdentity[:]) {
			selectedRank = uint32(index)
			break
		}
	}
	observed := gomadChoiceRecordValue{
		ordinal: gomadChoiceDecisionRecords, kind: kind, flags: flags, siteOffset: siteOffset,
		alternatives: uint32(len(alternatives)), selected: selectedRank, data: data,
		selectedIdentity: selectedIdentity, alternativeSet: setDigest,
	}
	physical := seeded
	if gomadChoiceMode == gomadChoiceModeReplay || gomadChoiceMode == gomadChoiceModePrefix && gomadChoiceTapeCursor < gomadChoiceTapeRecords {
		if gomadChoiceTapeCursor >= gomadChoiceTapeRecords {
			gomadChoiceDiverge(gomadChoiceDivergenceTapeExhausted, nil, &observed)
		}
		expected := gomadChoiceTapeRecord(gomadChoiceTapeCursor)
		reason := gomadChoiceCompareDecision(expected, observed)
		rankOverride := expected.flags&gomadChoiceFlagRankOverride != 0
		if reason == 0 {
			if expected.selected >= uint32(len(alternatives)) || !rankOverride && !gomadChoiceEqual(expected.selectedIdentity[:], ordered[expected.selected][:]) {
				reason = gomadChoiceDivergenceSelected
			}
		}
		if reason != 0 {
			gomadChoiceDiverge(reason, &expected, &observed)
		}
		selectedIdentity := expected.selectedIdentity
		if rankOverride {
			selectedIdentity = ordered[expected.selected]
		}
		for index := range alternatives {
			if gomadChoiceEqual(alternatives[index][:], selectedIdentity[:]) {
				physical = uint32(index)
				break
			}
		}
		observed.selected = expected.selected
		observed.selectedIdentity = selectedIdentity
		gomadChoiceTapeCursor++
	}
	gomadChoiceDecisionRecords++
	gomadChoiceAppendRecord(observed)
	return physical
}

func gomadChoiceCompareDecision(expected, observed gomadChoiceRecordValue) uint8 {
	if expected.kind != observed.kind {
		return gomadChoiceDivergenceKind
	}
	if expected.siteOffset != observed.siteOffset || expected.flags&^gomadChoiceFlagRankOverride != observed.flags {
		return gomadChoiceDivergenceSite
	}
	if expected.alternatives != observed.alternatives {
		return gomadChoiceDivergenceAlternatives
	}
	if !gomadChoiceEqual(expected.alternativeSet[:], observed.alternativeSet[:]) {
		return gomadChoiceDivergenceAlternativeSet
	}
	return 0
}

func gomadChoiceTapeRecord(ordinal uint64) gomadChoiceRecordValue {
	bytes := unsafe.Slice((*byte)(gomadChoiceTape), int(gomadChoiceTapeBytes))
	record := bytes[gomadChoiceTapeHeaderBytes+ordinal*gomadChoiceTapeRecordBytes : gomadChoiceTapeHeaderBytes+(ordinal+1)*gomadChoiceTapeRecordBytes]
	return gomadChoiceDecodeRecord(record)
}

func gomadChoiceEncodeRecord(record []byte, value gomadChoiceRecordValue) {
	for index := range record {
		record[index] = 0
	}
	gomadChoicePut64(record[:8], value.ordinal)
	record[8] = value.kind
	record[9] = value.flags
	gomadChoicePut32(record[12:16], value.alternatives)
	gomadChoicePut32(record[16:20], value.selected)
	gomadChoicePut32(record[20:24], value.data)
	gomadChoicePut64(record[24:32], value.siteOffset)
	copy(record[32:64], value.selectedIdentity[:])
	copy(record[64:96], value.alternativeSet[:])
}

func gomadChoiceDecodeRecord(record []byte) gomadChoiceRecordValue {
	value := gomadChoiceRecordValue{
		ordinal: gomadChoiceRead64(record[:8]), kind: record[8], flags: record[9], alternatives: gomadChoiceRead32(record[12:16]),
		selected: gomadChoiceRead32(record[16:20]), data: gomadChoiceRead32(record[20:24]), siteOffset: gomadChoiceRead64(record[24:32]),
	}
	copy(value.selectedIdentity[:], record[32:64])
	copy(value.alternativeSet[:], record[64:96])
	return value
}

func gomadChoiceValidDecisionRecord(record []byte, ordinal, records uint64) bool {
	value := gomadChoiceDecodeRecord(record)
	rankOverride := value.flags&gomadChoiceFlagRankOverride != 0
	return len(record) == gomadChoiceTapeRecordBytes && value.ordinal == ordinal && value.kind >= gomadChoiceKindRunnable && value.kind <= gomadChoiceKindSelectPoll && value.flags&gomadChoiceFlagDecision != 0 && value.flags&gomadChoiceFlagObservation == 0 && value.flags & ^uint8(gomadChoiceFlagDecision|gomadChoiceFlagSiteMissing|gomadChoiceFlagRankOverride) == 0 && (!rankOverride || gomadChoiceMode == gomadChoiceModePrefix && ordinal+1 == records) && gomadChoiceZero(record[10:12]) && value.alternatives != 0 && value.selected < value.alternatives && (value.flags&gomadChoiceFlagSiteMissing == 0 || value.siteOffset == 0) && (rankOverride && gomadChoiceZero(value.selectedIdentity[:]) || !rankOverride && !gomadChoiceZero(value.selectedIdentity[:])) && !gomadChoiceZero(value.alternativeSet[:])
}

func gomadChoiceAlternativeSet(ordered [][32]byte) [32]byte {
	var hasher gomadChoiceHasher
	hasher.init()
	hasher.write([]byte("gomadv3-choice-alternative-set/v1"))
	hasher.write([]byte{0})
	var count [8]byte
	gomadChoicePut64(count[:], uint64(len(ordered)))
	hasher.write(count[:])
	for index := range ordered {
		hasher.write(ordered[index][:])
	}
	return hasher.sum()
}

func gomadChoiceZero(value []byte) bool {
	for _, item := range value {
		if item != 0 {
			return false
		}
	}
	return true
}

func gomadChoiceEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func gomadChoiceCompare(left, right []byte) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

func gomadChoiceSite(pc uintptr) (uint64, uint8) {
	offset, ok := firstmoduledata.textOff(pc)
	if !ok {
		return 0, gomadChoiceFlagSiteMissing
	}
	return uint64(offset), 0
}

func gomadChoiceFinalize() {
	if !gomadChoiceEnabled || !gomadChoiceFinalized.CompareAndSwap(0, 1) {
		return
	}
	if (gomadChoiceMode == gomadChoiceModeReplay || gomadChoiceMode == gomadChoiceModePrefix) && gomadChoiceTapeCursor != gomadChoiceTapeRecords {
		expected := gomadChoiceTapeRecord(gomadChoiceTapeCursor)
		gomadChoicePublishTerminal(gomadChoiceTerminalDiverged, gomadChoiceDivergenceTapeUnconsumed, &expected, nil)
		return
	}
	gomadChoicePublishTerminal(gomadChoiceTerminalComplete, 0, nil, nil)
}

func gomadChoiceDiverge(reason uint8, expected, observed *gomadChoiceRecordValue) {
	if gomadChoiceFinalized.CompareAndSwap(0, 1) {
		gomadChoicePublishTerminal(gomadChoiceTerminalDiverged, reason, expected, observed)
	}
	exit(125)
}

func gomadChoiceDivergeCurrent(reason uint8) {
	if (gomadChoiceMode == gomadChoiceModeReplay || gomadChoiceMode == gomadChoiceModePrefix) && gomadChoiceTapeCursor < gomadChoiceTapeRecords {
		expected := gomadChoiceTapeRecord(gomadChoiceTapeCursor)
		gomadChoiceDiverge(reason, &expected, nil)
	}
	gomadChoiceDiverge(reason, nil, nil)
}

func gomadChoicePublishTerminal(state, reason uint8, expected, observed *gomadChoiceRecordValue) {
	records := gomadChoiceRecords.Load()
	mappingBytes := uint64(gomadChoiceHeaderBytes) + records*gomadChoiceRecordBytes
	bytes := unsafe.Slice((*byte)(gomadChoiceMapping), int(gomadChoiceMappingBytes))
	digest := gomadChoiceHash(bytes[gomadChoiceHeaderBytes:mappingBytes])
	var terminal [gomadChoiceTerminalBytes]byte
	copy(terminal[:8], gomadChoiceTerminalMagic[:])
	gomadChoicePut32(terminal[8:12], gomadChoiceWireVersion)
	terminal[12] = state
	terminal[13] = reason
	if gomadChoiceOverflow.Load() != 0 {
		terminal[12] = gomadChoiceTerminalOverflow
		terminal[13] = 0
	}
	gomadChoicePut64(terminal[16:24], records)
	gomadChoicePut64(terminal[24:32], mappingBytes)
	copy(terminal[32:64], digest[:])
	gomadChoicePut64(terminal[80:88], gomadChoiceTapeRecords)
	if terminal[12] == gomadChoiceTerminalDiverged {
		gomadChoicePut64(terminal[72:80], gomadChoiceDecisionRecords)
	}
	if terminal[12] == gomadChoiceTerminalDiverged && expected != nil {
		terminal[64] = 1
		gomadChoiceEncodeRecord(terminal[88:184], *expected)
	}
	if terminal[12] == gomadChoiceTerminalDiverged && observed != nil {
		terminal[65] = 1
		gomadChoiceEncodeRecord(terminal[184:280], *observed)
	}
	checksum := gomadChoiceHash(terminal[:gomadChoiceTerminalChecksumOffset])
	copy(terminal[gomadChoiceTerminalChecksumOffset:], checksum[:])
	if write1(uintptr(gomadChoiceTerminalDescriptor), unsafe.Pointer(&terminal[0]), gomadChoiceTerminalBytes) != gomadChoiceTerminalBytes {
		exit(125)
	}
}

func gomadChoiceRootIdentity(gp *g) {
	gp.gomadChildOrdinal = 0
	gp.gomadIdentity = gomadChoiceHash([]byte("gomadv3-choice-goroutine-root/v1"))
}

func gomadChoiceAssignGoroutineIdentity(newg, parent *g, pc uintptr) {
	newg.gomadChildOrdinal = 0
	var hasher gomadChoiceHasher
	hasher.init()
	if parent != nil && !gomadChoiceZero(parent.gomadIdentity[:]) {
		hasher.write([]byte("gomadv3-choice-goroutine-child/v1"))
		hasher.write(parent.gomadIdentity[:])
		parent.gomadChildOrdinal++
		var ordinal [8]byte
		gomadChoicePut64(ordinal[:], parent.gomadChildOrdinal)
		hasher.write(ordinal[:])
		site, flags := gomadChoiceSite(pc)
		var encoded [9]byte
		encoded[0] = flags
		gomadChoicePut64(encoded[1:], site)
		hasher.write(encoded[:])
	} else {
		hasher.write([]byte("gomadv3-choice-goroutine-runtime/v1"))
		var ordinal [8]byte
		gomadChoicePut64(ordinal[:], gomadRuntimeGoroutineOrdinal.Add(1))
		hasher.write(ordinal[:])
	}
	newg.gomadIdentity = hasher.sum()
}

func gomadChoiceRunqIndex(pp *p, head, tail, seeded uint32) uint32 {
	if !gomadChoiceEnabled {
		return seeded
	}
	count := tail - head
	var alternatives [gomadChoiceMaximumAlternatives][32]byte
	if count > uint32(len(alternatives)) {
		gomadChoiceDivergeCurrent(gomadChoiceDivergenceAlternativeCapacity)
	}
	for offset := uint32(0); offset < count; offset++ {
		gp := pp.runq[(head+offset)%uint32(len(pp.runq))].ptr()
		if gp == nil {
			gomadChoiceDivergeCurrent(gomadChoiceDivergenceIdentityMissing)
		}
		alternatives[offset] = gp.gomadIdentity
	}
	return gomadChoiceDecision(gomadChoiceKindRunnable, gomadChoiceFlagDecision|gomadChoiceFlagSiteMissing, 0, alternatives[:count], seeded, 0)
}

func gomadChoiceSelectPollIndex(pollorder []uint16, norder, current, nsends int, site uint64, siteFlags uint8, seeded uint32) uint32 {
	if !gomadChoiceEnabled {
		return seeded
	}
	count := norder + 1
	var alternatives [gomadChoiceMaximumAlternatives][32]byte
	if count > len(alternatives) {
		gomadChoiceDivergeCurrent(gomadChoiceDivergenceAlternativeCapacity)
	}
	for index := 0; index < norder; index++ {
		alternatives[index] = gomadChoiceSelectIdentity(site, siteFlags, int(pollorder[index]), nsends)
	}
	alternatives[norder] = gomadChoiceSelectIdentity(site, siteFlags, current, nsends)
	return gomadChoiceDecision(gomadChoiceKindSelectPoll, gomadChoiceFlagDecision|siteFlags, site, alternatives[:count], seeded, uint32(current))
}

func gomadChoiceSelectIdentity(site uint64, siteFlags uint8, ordinal, nsends int) [32]byte {
	var hasher gomadChoiceHasher
	hasher.init()
	hasher.write([]byte("gomadv3-choice-select-case/v1"))
	var encoded [18]byte
	encoded[0] = siteFlags
	gomadChoicePut64(encoded[1:9], site)
	gomadChoicePut64(encoded[9:17], uint64(ordinal))
	if ordinal >= nsends {
		encoded[17] = 1
	}
	hasher.write(encoded[:])
	return hasher.sum()
}

func gomadStartUserCode(mp *m) {
	if gomadChoiceEnabled {
		gomadChoiceRunqRandom.Init64([4]uint64{gomadSeed})
		gomadChoiceSchedulerRandom.Init64([4]uint64{gomadSeed, 0x676f6d6164736368})
		gomadChoiceSelectRandom = gomadSeed
		gomadChoiceRootIdentity(mp.curg)
		if !gomadChoiceHookRegistered {
			exithook.Add(exithook.Hook{F: gomadChoiceFinalize, RunOnFailure: true})
			gomadChoiceHookRegistered = true
		}
	}
	mrandinit(mp)
	mp.p.ptr().schedtick = 0
}

func gomadSeedEnv() (string, bool) {
	return gomadEnv("GOMADSEED=")
}

//go:linkname gomadIOProfileEnabled
func gomadIOProfileEnabled() bool {
	return gomadIOProfile
}

//go:linkname gomadDeterministicEnabled
func gomadDeterministicEnabled() bool {
	return gomadEnabled
}

//go:linkname gomadSimulationDomain
func gomadSimulationDomain() uint64 {
	return getg().gomadSimulationDomain
}

//go:linkname gomadSimulationSetDomain
func gomadSimulationSetDomain(domain uint64) uint64 {
	gp := getg()
	previous := gp.gomadSimulationDomain
	gp.gomadSimulationDomain = domain
	return previous
}

//go:linkname gomadSimulationTimeAdvance
func gomadSimulationTimeAdvance(current int64) bool {
	if !gomadSimulationTimeEnabled || current < faketime {
		return false
	}
	faketime = current
	return true
}

//go:linkname gomadSimulationTimeCurrent
func gomadSimulationTimeCurrent() int64 {
	if !gomadSimulationTimeEnabled {
		return 0
	}
	return faketime
}

//go:linkname gomadSimulationTimeObserve
func gomadSimulationTimeObserve(current int64) bool {
	if !gomadSimulationTimeEnabled {
		return current == 0
	}
	return gomadSimulationTimeAdvance(current)
}

//go:linkname gomadSimulationExternalBegin
func gomadSimulationExternalBegin() {
	if gomadSimulationTimeEnabled {
		gomadSimulationExternalRequests.Add(1)
	}
}

//go:linkname gomadSimulationExternalEnd
func gomadSimulationExternalEnd() {
	if gomadSimulationTimeEnabled {
		gomadSimulationExternalRequests.Add(-1)
	}
}

//go:linkname gomadSimulationExternalArrive
func gomadSimulationExternalArrive() {
	if gomadSimulationTimeEnabled {
		gomadSimulationTimeArrivals.Add(1)
		gomadSimulationTimeArrivalEpoch.Add(1)
	}
}

//go:linkname gomadSimulationTimeTakeArrivals
func gomadSimulationTimeTakeArrivals() uint32 {
	if !gomadSimulationTimeEnabled {
		return 0
	}
	arrivals := gomadSimulationTimeArrivals.Swap(0)
	if arrivals != 0 {
		gomadSimulationTimeAwaitingExternal.Store(false)
	}
	return arrivals
}

//go:nosplit
func gomadSimulationTimeQuiesce(deadline int64) (int64, uint8, bool) {
	if !gomadSimulationTimeEnabled {
		return 0, 0, false
	}
	gomadSimulationTimeGeneration++
	if gomadSimulationTimeGeneration == 0 {
		return 0, 0, false
	}
	var request [gomadSimulationTimeRequestBytes]byte
	for index := range gomadSimulationTimeRequestMagic {
		request[index] = gomadSimulationTimeRequestMagic[index]
	}
	gomadSimulationTimePut64(request[8:16], gomadSimulationTimeGeneration)
	gomadSimulationTimePut64(request[16:24], uint64(faketime))
	gomadSimulationTimePut64(request[24:32], uint64(deadline))
	arrivalEpoch := gomadSimulationTimeArrivalEpoch.Load()
	arrivals := gomadSimulationTimeArrivals.Swap(0)
	gomadSimulationTimePut32(request[32:36], arrivals)
	if !gomadSimulationTimeWrite(gomadSimulationTimeRequestDescriptor, request[:]) {
		return 0, 0, false
	}
	var response [gomadSimulationTimeResponseBytes]byte
	if !gomadSimulationTimeRead(gomadSimulationTimeResponseDescriptor, response[:]) {
		return 0, 0, false
	}
	for index := range gomadSimulationTimeResponseMagic {
		if response[index] != gomadSimulationTimeResponseMagic[index] {
			return 0, 0, false
		}
	}
	if gomadSimulationTimeGet64(response[8:16]) != gomadSimulationTimeGeneration {
		return 0, 0, false
	}
	for _, value := range response[25:] {
		if value != 0 {
			return 0, 0, false
		}
	}
	current := int64(gomadSimulationTimeGet64(response[16:24]))
	kind := response[24]
	if current < faketime || kind < gomadSimulationTimeResponseAdvance || kind > gomadSimulationTimeResponseExternal {
		return 0, 0, false
	}
	if arrivals != 0 {
		gomadSimulationTimeAwaitingExternal.Store(false)
	}
	if kind == gomadSimulationTimeResponseExternal {
		gomadSimulationTimeAwaitingExternal.Store(true)
		if arrivalEpoch != gomadSimulationTimeArrivalEpoch.Load() || gomadSimulationTimeArrivals.Load() != 0 {
			gomadSimulationTimeAwaitingExternal.Store(false)
		}
	}
	return current, kind, true
}

//go:nosplit
func gomadSimulationTimeWrite(descriptor int32, source []byte) bool {
	for len(source) != 0 {
		count := write1(uintptr(descriptor), unsafe.Pointer(&source[0]), int32(len(source)))
		if count <= 0 || count > int32(len(source)) {
			return false
		}
		source = source[count:]
	}
	return true
}

//go:nosplit
func gomadSimulationTimeRead(descriptor int32, destination []byte) bool {
	for len(destination) != 0 {
		count := read(descriptor, unsafe.Pointer(&destination[0]), int32(len(destination)))
		if count <= 0 || count > int32(len(destination)) {
			return false
		}
		destination = destination[count:]
	}
	return true
}

//go:nosplit
func gomadSimulationTimePut64(destination []byte, value uint64) {
	for index := 7; index >= 0; index-- {
		destination[index] = byte(value)
		value >>= 8
	}
}

//go:nosplit
func gomadSimulationTimePut32(destination []byte, value uint32) {
	for index := 3; index >= 0; index-- {
		destination[index] = byte(value)
		value >>= 8
	}
}

//go:nosplit
func gomadSimulationTimeGet64(source []byte) uint64 {
	var value uint64
	for _, current := range source {
		value = value<<8 | uint64(current)
	}
	return value
}

//go:linkname gomadBlockingRead
//go:nosplit
func gomadBlockingRead(fd int32, destination unsafe.Pointer, bytes int32) int32 {
	gp := getg()
	if gomadSimulationTimeEnabled {
		gp.gomadSimulationTransport = true
		gomadSimulationTransportSyscalls.Add(1)
	}
	entersyscallblock()
	count := read(fd, destination, bytes)
	if gomadSimulationTimeEnabled {
		gomadSimulationTransportSyscalls.Add(-1)
	}
	exitsyscall()
	gp.gomadSimulationTransport = false
	return count
}

//go:linkname gomadBlockingWrite
//go:nosplit
func gomadBlockingWrite(fd uintptr, source unsafe.Pointer, bytes int32) int32 {
	entersyscallblock()
	count := write1(fd, source, bytes)
	exitsyscall()
	return count
}

//go:linkname gomadIOConfigFrame
func gomadIOConfigFrame() *[212]byte {
	return &gomadConfig
}

func gomadReadConfig() bool {
	offset := int32(0)
	for offset < int32(len(gomadConfig)) {
		count := read(5, unsafe.Pointer(&gomadConfig[offset]), int32(len(gomadConfig))-offset)
		if count <= 0 {
			break
		}
		offset += count
	}
	closefd(5)
	if offset == 0 {
		return false
	}
	if offset != int32(len(gomadConfig)) || gomadConfig[0] != 'G' || gomadConfig[1] != 'O' || gomadConfig[2] != 'M' || gomadConfig[3] != 'A' || gomadConfig[4] != 'D' || gomadConfig[5] != 'I' || gomadConfig[6] != 'O' || gomadConfig[7] != 1 || gomadConfig[8] != 0 || gomadConfig[9] != 1 || gomadConfig[10] != 0 || gomadConfig[11] != 1 {
		print("runtime: invalid Gomad bootstrap configuration\n")
		exit(2)
	}
	gomadConfigPresent = true
	gomadIOProfile = true
	return true
}

func gomadConfigSeed() uint64 {
	const offset = 172
	value := uint64(0)
	for i := 0; i < 8; i++ {
		value = value<<8 | uint64(gomadConfig[offset+i])
	}
	return value
}

func gomadEnv(prefix string) (string, bool) {
	switch GOOS {
	case "aix", "darwin", "ios", "dragonfly", "freebsd", "netbsd", "openbsd", "illumos", "solaris", "linux":
	default:
		return "", false
	}

	n := int32(0)
	for argv_index(argv, argc+1+n) != nil {
		n++
	}
	for i := int32(0); i < n; i++ {
		value := gostringnocopy(argv_index(argv, argc+1+i))
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return value[len(prefix):], true
		}
	}
	return "", false
}

func gomadParseSeed(value string) (uint64, bool) {
	if value == "" {
		return 0, false
	}

	var seed uint64
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		digit := uint64(value[i] - '0')
		if seed > (^uint64(0)-digit)/10 {
			return 0, false
		}
		seed = seed*10 + digit
	}
	return seed, true
}
