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
	"math/bits"
)

var gomadEnabled bool
var gomadSeed uint64
var gomadExternal bool
var gomadIOProfile bool
var gomadConfigPresent bool
var gomadConfig [gomadBootstrapFrameBytes]byte
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
var gomadChoicePeakGoroutines uint32
var gomadRuntimeGoroutineOrdinal atomic.Uint64
var gomadChoiceNext atomic.Uint64
var gomadChoiceRecords atomic.Uint64
var gomadChoiceOverflow atomic.Uint32
var gomadChoiceFinalized atomic.Uint32
var gomadChoiceHookRegistered bool
var gomadChoiceRunqRandom chacha8rand.State
var gomadChoiceSchedulerRandom chacha8rand.State
var gomadChoiceSelectRandom uint64

var gomadDiagnosticEnabled bool
var gomadDiagnosticMapping unsafe.Pointer
var gomadDiagnosticMappingBytes uint64
var gomadDiagnosticPerturb bool
var gomadDiagnosticPerturbOrdinal uint64

// gomadDiagnosticDraws counts the draws taken from each seeded stream since
// the process started. The counts are kept whether or not a diagnostic trace
// is recorded, so a traced run executes the same draw paths as an untraced one.
var gomadDiagnosticDraws gomadDiagnosticDrawCounts

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

// The response buffer is static because some runtime read implementations
// retain their pointer, while quiescence can run on an allocation-forbidden path.
var gomadSimulationTimeResponse [gomadSimulationTimeResponseBytes]byte

func gomadInit() {
	var seed uint64
	choiceConfigured := gomadChoiceConfigured()
	if _, profile := gomadEnv("GOMAD3_IO_PROFILE="); profile || choiceConfigured {
		gomadDisableASLR()
	} else if _, present := gomadSeedEnv(); present {
		gomadDisableASLR()
	}
	if choiceConfigured {
		gomadChoiceInit()
	}
	gomadDiagnosticInit()
	_, profile := gomadEnv("GOMAD3_IO_PROFILE=")
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

	gomadClockTickInit(seed)
	gomadEnabled = true
	gomadSeed = seed
	gomadChoiceSeedRandom()
	faketime = gomadInitialTime
	gomadSimulationTimeInit()
	debug.asyncpreemptoff = 1
	haveSysmon = false
	randomizeScheduler = true
}

// gomadClockForward advances what time.Now reports at every read so that two
// reads never share an instant, the way a real clock moves between them. The
// advance accumulates in its own offset rather than in faketime: timers, the
// scheduler, and the simulation time transport keep the idle-driven clock, and
// only time.Now runs ahead of it. The draw comes from its own stream derived
// from the seed, so it neither consumes nor perturbs the scheduling choices,
// and replay derives the same draws.
var (
	gomadClockForward    bool
	gomadClockTickState  uint64
	gomadClockTickOffset int64
)

// gomadClockTickMask bounds each forward draw to 1 through 1024 nanoseconds:
// enough to separate timestamps, far below any timer a test would set.
const gomadClockTickMask = 1023

func gomadClockTickInit(seed uint64) {
	value, present := gomadEnv("GOMAD3_CLOCK_TICK=")
	if !present {
		return
	}
	if value != "forward" {
		print("runtime: invalid GOMAD3_CLOCK_TICK\n")
		exit(2)
	}
	gomadClockForward = true
	gomadClockTickState = seed ^ 0x6c62272e07bb0142
}

func gomadClockTickDraw() int64 {
	gomadDiagnosticDraws.clockTick++
	gomadClockTickState += 0x9e3779b97f4a7c15
	value := gomadClockTickState
	value = (value ^ value>>30) * 0xbf58476d1ce4e5b9
	value = (value ^ value>>27) * 0x94d049bb133111eb
	value ^= value >> 31
	return int64(1 + value&gomadClockTickMask)
}

// gomadTimeNow serves time.Now while Gomad is enabled.
func gomadTimeNow() (sec int64, nsec int32, mono int64) {
	now := faketime
	if gomadClockForward {
		gomadClockTickOffset += gomadClockTickDraw()
		now += gomadClockTickOffset
	}
	return now / 1e9, int32(now % 1e9), now
}

//go:linkname gomadCapabilityGuard
//go:noinline
func gomadCapabilityGuard() {
	if gomadEnabled {
		runExitHooks(2)
		throw("GOMAD_CAPABILITY_DENIED")
	}
}

func gomadSimulationTimeInit() {
	requestValue, requestPresent := gomadEnvEarly("GOMAD3_SIMULATION_TIME_REQUEST_FD=")
	responseValue, responsePresent := gomadEnvEarly("GOMAD3_SIMULATION_TIME_RESPONSE_FD=")
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
	_, configured := gomadEnvEarly("GOMAD3_CHOICE_TRACE_FD=")
	return configured
}

func gomadChoiceInit() {
	descriptorValue, enabled := gomadEnvEarly("GOMAD3_CHOICE_TRACE_FD=")
	if !enabled {
		return
	}
	terminalValue, terminalPresent := gomadEnvEarly("GOMAD3_CHOICE_TERMINAL_FD=")
	bytesValue, bytesPresent := gomadEnvEarly("GOMAD3_CHOICE_TRACE_BYTES=")
	modeValue, modePresent := gomadEnvEarly("GOMAD3_CHOICE_MODE=")
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
		if _, present := gomadEnvEarly("GOMAD3_CHOICE_TAPE_FD="); present {
			print("runtime: choice record mode cannot use a tape\n")
			exit(2)
		}
		return
	}
	gomadChoiceInitTape()
}

func gomadChoiceInitTape() {
	descriptorValue, descriptorPresent := gomadEnvEarly("GOMAD3_CHOICE_TAPE_FD=")
	bytesValue, bytesPresent := gomadEnvEarly("GOMAD3_CHOICE_TAPE_BYTES=")
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

// gomadDiagnosticInit maps the diagnostic trace, which holds one runtime-state
// digest per choice record. It has its own descriptor and byte bound so that
// it never takes space from the choice trace, and it is read from the control
// variables that gomadGoenvs hides, so requesting it leaves the early heap as
// it was.
func gomadDiagnosticInit() {
	descriptorValue, enabled := gomadEnvEarly("GOMAD3_DIAGNOSTIC_TRACE_FD=")
	perturbValue, perturbPresent := gomadEnvEarly("GOMAD3_DIAGNOSTIC_PERTURB_DRAW=")
	if !enabled && !perturbPresent {
		return
	}
	bytesValue, bytesPresent := gomadEnvEarly("GOMAD3_DIAGNOSTIC_TRACE_BYTES=")
	descriptor, descriptorOK := gomadParseSeed(descriptorValue)
	mappingBytes, bytesOK := gomadParseSeed(bytesValue)
	perturbOrdinal, perturbOK := gomadParseSeed(perturbValue)
	if !enabled || !gomadChoiceEnabled || !bytesPresent || !descriptorOK || !bytesOK || perturbPresent && !perturbOK || descriptor > 1<<31-1 || mappingBytes < gomadDiagnosticHeaderBytes+gomadDiagnosticRecordBytes || mappingBytes > gomadDiagnosticMaximumBytes {
		print("runtime: invalid Gomad diagnostic trace configuration\n")
		exit(2)
	}
	mapped, errno := mmap(nil, uintptr(mappingBytes), _PROT_READ|_PROT_WRITE, gomadMapShared, int32(descriptor), 0)
	if errno != 0 {
		print("runtime: invalid Gomad diagnostic trace mapping\n")
		exit(2)
	}
	bytes := unsafe.Slice((*byte)(mapped), int(mappingBytes))
	for index := range gomadDiagnosticMagic {
		if bytes[index] != gomadDiagnosticMagic[index] {
			print("runtime: invalid Gomad diagnostic trace backing\n")
			exit(2)
		}
	}
	if gomadChoiceRead32(bytes[8:12]) != gomadDiagnosticWireVersion || !gomadChoiceZero(bytes[12:16]) || gomadChoiceRead64(bytes[16:24]) != mappingBytes || gomadChoiceRead64(bytes[24:32]) != gomadDiagnosticHeaderBytes || !gomadChoiceZero(bytes[32:gomadDiagnosticHeaderBytes]) {
		print("runtime: invalid Gomad diagnostic trace header\n")
		exit(2)
	}
	gomadDiagnosticEnabled = true
	gomadDiagnosticMapping = mapped
	gomadDiagnosticMappingBytes = mappingBytes
	gomadDiagnosticPerturb = perturbPresent
	gomadDiagnosticPerturbOrdinal = perturbOrdinal
}

// gomadDiagnosticAppend writes the digest for the choice record just appended
// at ordinal, into the slot with the same ordinal. It reads runtime state and
// stores into the mapping only: an allocation or a seeded draw here would move
// the state it reports.
func gomadDiagnosticAppend(ordinal uint64) {
	if !gomadDiagnosticEnabled {
		return
	}
	// The perturbation stands in for a host-timed draw from the process-wide
	// stream, so that a fixture can show the differ naming this ordinal.
	if gomadDiagnosticPerturb && ordinal == gomadDiagnosticPerturbOrdinal {
		gomadRuntimeCheapRand()
	}
	bytes := unsafe.Slice((*byte)(gomadDiagnosticMapping), int(gomadDiagnosticMappingBytes))
	offset := gomadDiagnosticHeaderBytes + ordinal*gomadDiagnosticRecordBytes
	if offset > gomadDiagnosticMappingBytes-gomadDiagnosticRecordBytes {
		bytes[gomadDiagnosticStateOffset] = gomadDiagnosticStateOverflow
		print("runtime: Gomad diagnostic trace overflow\n")
		exit(125)
	}
	pp := getg().m.p.ptr()
	value := gomadDiagnosticRecordValue{
		ordinal: ordinal, virtualTime: faketime, allocations: gomadDiagnosticAllocations(pp),
		gcCycle: work.cycles.Load(), gcPhase: uint8(gcphase), runQueueLength: pp.runqtail - pp.runqhead,
		draws: gomadDiagnosticDraws,
	}
	if pp.runnext != 0 {
		value.runQueueLength++
	}
	gomadDiagnosticEncodeRecord(bytes[offset:offset+gomadDiagnosticRecordBytes], &value)
	gomadChoicePut64(bytes[24:32], offset+gomadDiagnosticRecordBytes)
	gomadChoicePut64(bytes[32:40], ordinal+1)
}

// gomadDiagnosticAllocations counts the objects allocated so far: the counts
// the allocator has already flushed to the heap statistics plus the slots used
// in the spans pp still caches. Only the M holding the P writes either, and it
// writes them between choice points, so the sum needs no synchronization and
// does not depend on when a span was last refilled.
func gomadDiagnosticAllocations(pp *p) uint64 {
	var count uint64
	for generation := range memstats.heapStats.stats {
		stats := &memstats.heapStats.stats[generation]
		count += stats.tinyAllocCount + stats.largeAllocCount
		for class := range stats.smallAllocCount {
			count += stats.smallAllocCount[class]
		}
	}
	cache := pp.mcache
	count += uint64(cache.tinyAllocs)
	for class := range cache.alloc {
		span := cache.alloc[class]
		count += uint64(span.allocCount - span.allocCountBeforeCache)
	}
	return count
}

// gomadDiagnosticComplete marks a trace the runtime closed in order, as
// opposed to one cut short by overflow or by a killed process.
func gomadDiagnosticComplete() {
	if gomadDiagnosticEnabled {
		*(*byte)(add(gomadDiagnosticMapping, gomadDiagnosticStateOffset)) = gomadDiagnosticStateComplete
	}
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
	gomadDiagnosticAppend(ordinal)
}

func gomadChoiceRecord(kind, flags uint8, siteOffset uint64, alternatives, selected, data uint32) {
	gomadChoiceAppendRecord(gomadChoiceRecordValue{kind: kind, flags: flags, siteOffset: siteOffset, alternatives: alternatives, selected: selected, data: data})
}

// The scheduler draws from process-wide seeded states rather than the per-M
// streams: the P changes hands between Ms during runtime initialization and
// around blocking syscalls, and which M picks it up is a host-timing race.
func gomadChoiceSeedRandom() {
	gomadChoiceRunqRandom.Init64([4]uint64{gomadSeed})
	gomadChoiceSchedulerRandom.Init64([4]uint64{gomadSeed, 0x676f6d6164736368})
	gomadChoiceSelectRandom = gomadSeed
	gomadRuntimeRandom.Init64([4]uint64{gomadSeed, 0x676f6d616472616e})
	gomadRuntimeCheapRandom = uint32(gomadSeed)
	gomadTimerRandom = uint32(gomadSeed) ^ 0x74696d65
}

// gomadTimerRandom breaks ties between timers due at the same instant. It is
// separate from cheaprand because the runtime also draws from cheaprand on
// contended lock hand-offs, which happen at host-timed moments.
var gomadTimerRandom uint32

//go:nosplit
func gomadTimerRand() uint32 {
	gomadDiagnosticDraws.timer++
	gomadTimerRandom += 0xa0761d65
	value := uint64(gomadTimerRandom) * 0xe7037ed1a0b428db
	return uint32(value>>32) ^ uint32(value)
}

// The runtime's rand and cheaprand streams are per M, and every M starts from
// the same seeded position. Which M holds the P after a hand-off is a
// host-timing race, so two same-seed runs that hand the P between Ms at
// different points read different map hash seeds, timer tie-breaks, and
// semaphore tickets from otherwise identical streams. The M holding the P
// therefore draws from these process-wide states instead; Ms without a P
// (lock backoff on idle threads) keep their own streams, which decide nothing
// the program observes.
var gomadRuntimeRandom chacha8rand.State
var gomadRuntimeCheapRandom uint32

//go:nosplit
func gomadRuntimeRand(mp *m) uint64 {
	gomadDiagnosticDraws.runtimeRand++
	for {
		x, ok := gomadRuntimeRandom.Next()
		if ok {
			return x
		}
		mp.locks++ // hold m even though Refill may do stack split checks
		gomadRuntimeRandom.Refill()
		mp.locks--
	}
}

//go:nosplit
func gomadRuntimeCheapRand() uint32 {
	gomadDiagnosticDraws.runtimeCheapRand++
	gomadRuntimeCheapRandom += 0x53c5ca59
	hi, lo := bits.Mul32(gomadRuntimeCheapRandom, gomadRuntimeCheapRandom^0x74743c1b)
	return hi ^ lo
}

// gomadHostCheapRand draws from the M's own stream for decisions that only
// shape host-side scheduling: lock hand-off fairness, work-steal order, and
// pcvalue-cache eviction. Those draws happen at host-timed moments (contended
// runtime locks, idle windows whose length the runner decides, stack walks on
// whichever M holds the P), so taking them from the process-wide stream moved
// every later type-assertion cache fill and semaphore ticket between same-seed
// runs.
//
//go:nosplit
func gomadHostCheapRand() uint32 {
	mp := getg().m
	mp.cheaprand += 0x53c5ca59
	hi, lo := bits.Mul32(mp.cheaprand, mp.cheaprand^0x74743c1b)
	return hi ^ lo
}

//go:nosplit
func gomadHostCheapRandN(n uint32) uint32 {
	return uint32(uint64(gomadHostCheapRand()) * uint64(n) >> 32)
}

// gomadLockProfileStart stands in for mLockProfile.start where lock2 is about
// to sleep on a contended runtime lock. Whether a lock outlasts the spin is
// host timing, and the waiting M can hold the P: wakep and findRunnable take
// sched.lock while Ms returning from runner syscalls queue their arrival and
// park under it. The wait-time sampling draw came from the process-wide
// stream there, so one contended acquisition shifted every later
// type-assertion cache fill, a cache grew on a different assertion, and a
// same-seed replay refilled two heap spans in the other order and printed
// different addresses.
func gomadLockProfileStart(prof *mLockProfile) int64 {
	if !gomadEnabled {
		return prof.start()
	}
	if gomadHostCheapRandN(gTrackingPeriod) == 0 {
		return nanotime()
	}
	return 0
}

func gomadChoiceRunqSeeded(n uint32) uint32 {
	if !gomadEnabled {
		return randn(n)
	}
	gomadDiagnosticDraws.runq++
	return gomadChoiceRandom(&gomadChoiceRunqRandom, n)
}

func gomadChoiceRunnextSeeded(n uint32) uint32 {
	if !gomadEnabled {
		return randn(n)
	}
	gomadDiagnosticDraws.scheduler++
	return gomadChoiceRandom(&gomadChoiceSchedulerRandom, n)
}

func gomadChoiceShuffleSeeded(n uint32) uint32 {
	if !gomadEnabled {
		return cheaprandn(n)
	}
	gomadDiagnosticDraws.scheduler++
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
	if !gomadEnabled {
		return cheaprandn(n)
	}
	gomadDiagnosticDraws.selectPoll++
	gomadChoiceSelectRandom += 0xa0761d6478bd642f
	if goarch.IsAmd64|goarch.IsArm64|goarch.IsPpc64|
		goarch.IsPpc64le|goarch.IsMips64|goarch.IsMips64le|
		goarch.IsS390x|goarch.IsRiscv64|goarch.IsLoong64 == 1 {
		hi, lo := bits.Mul64(gomadChoiceSelectRandom, gomadChoiceSelectRandom^0xe7037ed1a0b428db)
		return uint32((uint64(uint32(hi^lo)) * uint64(n)) >> 32)
	}
	t := (*[2]uint32)(unsafe.Pointer(&gomadChoiceSelectRandom))
	s1, s0 := t[0], t[1]
	s1 ^= s1 << 17
	s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16
	t[0], t[1] = s0, s1
	return uint32((uint64(s0+s1) * uint64(n)) >> 32)
}

// ordered is caller-provided scratch for the sorted alternative set so the
// decision itself carries no large frame onto the system stack.
func gomadChoiceDecision(kind, flags uint8, siteOffset uint64, alternatives [][32]byte, ordered *[gomadChoiceMaximumAlternatives][32]byte, seeded, data uint32) uint32 {
	if !gomadChoiceEnabled {
		return seeded
	}
	if len(alternatives) == 0 || len(alternatives) > gomadChoiceMaximumAlternatives || seeded >= uint32(len(alternatives)) {
		gomadChoiceDivergeCurrent(gomadChoiceDivergenceAlternativeCapacity)
	}
	if len(alternatives) == 1 {
		return seeded
	}
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
	hasher.write([]byte("gomad3-choice-alternative-set/v1"))
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
	gomadDiagnosticComplete()
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
	gomadChoicePut32(terminal[66:70], gomadChoicePeakGoroutines)
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
	gp.gomadIdentity = gomadChoiceHash([]byte("gomad3-choice-goroutine-root/v1"))
}

func gomadChoiceAssignGoroutineIdentity(newg, parent *g, pc uintptr) {
	newg.gomadChildOrdinal = 0
	var hasher gomadChoiceHasher
	hasher.init()
	if parent != nil && !gomadChoiceZero(parent.gomadIdentity[:]) {
		hasher.write([]byte("gomad3-choice-goroutine-child/v1"))
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
		hasher.write([]byte("gomad3-choice-goroutine-runtime/v1"))
		var ordinal [8]byte
		gomadChoicePut64(ordinal[:], gomadRuntimeGoroutineOrdinal.Add(1))
		hasher.write(ordinal[:])
	}
	newg.gomadIdentity = hasher.sum()
	if live := uint32(gcount(false)); live > gomadChoicePeakGoroutines {
		gomadChoicePeakGoroutines = live
	}
}

// The run-queue choice runs on the system stack, which Linux sizes at 16 KiB
// for non-main threads, so its 8 KiB candidate buffers live here instead of in
// the frame. The system stack is never preempted and there is one P, so the
// scheduler is the only user.
var gomadChoiceSchedulerAlternatives [gomadChoiceMaximumAlternatives][32]byte
var gomadChoiceSchedulerOrdered [gomadChoiceMaximumAlternatives][32]byte

func gomadChoiceRunqIndex(pp *p, head, tail, seeded uint32) uint32 {
	if !gomadChoiceEnabled {
		return seeded
	}
	count := tail - head
	alternatives := &gomadChoiceSchedulerAlternatives
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
	return gomadChoiceDecision(gomadChoiceKindRunnable, gomadChoiceFlagDecision|gomadChoiceFlagSiteMissing, 0, alternatives[:count], &gomadChoiceSchedulerOrdered, seeded, 0)
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
	var ordered [gomadChoiceMaximumAlternatives][32]byte
	return gomadChoiceDecision(gomadChoiceKindSelectPoll, gomadChoiceFlagDecision|siteFlags, site, alternatives[:count], &ordered, seeded, uint32(current))
}

func gomadChoiceSelectIdentity(site uint64, siteFlags uint8, ordinal, nsends int) [32]byte {
	var hasher gomadChoiceHasher
	hasher.init()
	hasher.write([]byte("gomad3-choice-select-case/v1"))
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

// gomadReservedMs is the number of idle Ms created before user code runs.
// A goroutine that hands its P off for a runner syscall needs another M to
// keep running the P, and one that returns without a P parks its M; creating
// those Ms on demand would allocate m structures and g0 stacks at host-timed
// moments and move every later heap address and collection trigger. The
// reserve covers the concurrent runner syscalls a target issues, so on-demand
// creation is the exception the choice trace then exposes.
const gomadReservedMs = 8

// gomadIdleM parks a reserved M until the scheduler hands it a P. User code
// starts only once every reserved M has parked, so the reserve is complete
// before the first hand-off can ask for it.
func gomadIdleM() {
	stopm()
	schedule()
}

// gomadGoenvs copies the process environment for the runtime without the
// Gomad control variables, which the runner varies between recording and
// replay (choice mode, tape descriptors) and the runtime reads from argv
// directly, so both runs allocate the same early heap.
func gomadGoenvs() {
	n := int32(0)
	kept := int32(0)
	for argv_index(argv, argc+1+n) != nil {
		if !gomadControlVariable(gostringnocopy(argv_index(argv, argc+1+n))) {
			kept++
		}
		n++
	}
	envs = make([]string, kept)
	index := 0
	for i := int32(0); i < n; i++ {
		// Filter on the C string before copying it: the control variables
		// differ between a recording and its replay (choice mode, tape
		// descriptor and size), and copying them first put a different
		// number of bytes on the heap before user code ran.
		if gomadControlVariable(gostringnocopy(argv_index(argv, argc+1+i))) {
			continue
		}
		envs[index] = gostring(argv_index(argv, argc+1+i))
		index++
	}
}

func gomadControlVariable(variable string) bool {
	for _, prefix := range [...]string{"GOMAD3_", "GOMADSEED="} {
		if len(variable) >= len(prefix) && variable[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// gomadMarkWorkerAllowed reports whether a background mark worker may take
// this scheduling slot. The worker runs only when the P has runnable
// goroutines or parked assists to interleave with; an otherwise idle P stays
// idle, because marking there would pace the collector by the host time that
// goroutines spend in runner syscalls instead of by the recorded schedule.
func gomadMarkWorkerAllowed(pp *p) bool {
	return !runqempty(pp) || !sched.runq.empty() || !work.assistQueue.q.empty()
}

// gomadGreyRuntimeStructures greys every m, its g0, gsignal and self handle,
// every p's oldm handle, and every g while the world is still stopped at mark
// start. Which M runs a goroutine after a syscall hand-off is host timing, and
// execute's gp.m = mp write shades that M through the write barrier; an M
// scanned early also greys its curg and g0 early. acquirep records the M that
// took the P in p.oldm, a copy of that M's self handle (an 8-byte heap object),
// so the scan of the p greyed whichever handle the host-timed hand-off had left
// there and moved 8 bytes of scan work between drain slices; the assist or
// worker that ends a slice stops on a work boundary, so the work buffers, the
// page-heap layout and every %p the target prints followed the hand-off. With
// the classic collector the total scan work of these objects is fixed, so
// greying them all here, in allm, allp and allgs order, puts the same work at
// the same point of every cycle.
func gomadGreyRuntimeStructures() {
	for mp := allm; mp != nil; mp = mp.alllink {
		shade(uintptr(unsafe.Pointer(mp)))
		if mp.g0 != nil {
			shade(uintptr(unsafe.Pointer(mp.g0)))
		}
		if mp.gsignal != nil {
			shade(uintptr(unsafe.Pointer(mp.gsignal)))
		}
		if mp.self.m != nil {
			shade(uintptr(unsafe.Pointer(mp.self.m)))
		}
	}
	for _, pp := range allp {
		if pp.oldm.m != nil {
			shade(uintptr(unsafe.Pointer(pp.oldm.m)))
		}
	}
	forEachG(func(gp *g) {
		shade(uintptr(unsafe.Pointer(gp)))
	})
}

// gomadArrivals holds goroutines whose host syscall returned while another
// goroutine held the P. The host decides when such a syscall returns, so a
// goroutine is admitted only once the P has nothing else to run, one per idle
// window, in the order the syscalls returned. Guarded by sched.lock.
var gomadArrivals gQueue

// gomadAdmit moves the queued goroutines onto pp's local run queue and picks
// the next one through the recorded run-queue choice, so goroutines that reach
// the scheduler through the global run queue never run unrecorded ahead of it.
func gomadAdmit(pp *p, queue *gQueue) (*g, bool) {
	if runqputbatch(pp, queue); !queue.empty() {
		throw("gomad: local run queue could not take the admitted goroutines")
	}
	gp, inheritTime := runqget(pp)
	if gp == nil {
		throw("gomad: local run queue empty after admission")
	}
	return gp, inheritTime
}

func gomadStartUserCode(mp *m) {
	if gomadEnabled {
		gomadChoiceSeedRandom()
		// Heap profile samples are drawn from cheaprand and allocate a
		// bucket each, so their placement would move the heap and every
		// later collection with the contended lock draws described at
		// gomadTimerRandom.
		MemProfileRate = 0
	}
	if gomadEnabled {
		lock(&sched.lock)
		idle := sched.nmidle
		unlock(&sched.lock)
		for count := 0; count < gomadReservedMs; count++ {
			newm(gomadIdleM, nil, -1)
		}
		for {
			lock(&sched.lock)
			parked := sched.nmidle >= idle+gomadReservedMs
			unlock(&sched.lock)
			if parked {
				break
			}
			osyield()
		}
	}
	if gomadChoiceEnabled {
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

// gomadWallNanotime exposes the host monotonic clock to the deterministic I/O
// packages; Go 1.27 no longer lets them pull the assembly nanotime1 directly.
//
//go:linkname gomadWallNanotime
func gomadWallNanotime() int64 {
	return nanotime1()
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
	response := gomadSimulationTimeResponse[:]
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

// gomadHostRead blocks on a host descriptor whose answer arrives in wall
// time, such as a read-only mount lookup. Unlike the simulation transport
// reader, the waiting goroutine stays a plain syscall, so checkdead does not
// quiesce and simulation time does not move while the answer is pending.
//
//go:linkname gomadHostRead
//go:nosplit
func gomadHostRead(fd int32, destination unsafe.Pointer, bytes int32) int32 {
	entersyscallblock()
	count := read(fd, destination, bytes)
	exitsyscall()
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

// gomadSyscallWrite carries the writes syscall.Write still allows under Gomad
// (stdout, stderr, and the trace transport) to the kernel without passing
// through syscall.Syscall: on Linux that trampoline has a Go body and is a
// guarded capability entry point, so guarded targets would trip the guard on
// their own output.
//
//go:linkname gomadSyscallWrite
func gomadSyscallWrite(fd int, source []byte) (int, uintptr, bool) {
	if !gomadEnabled {
		return 0, 0, false
	}
	if len(source) == 0 {
		return 0, 0, true
	}
	count := int32(1 << 30)
	if len(source) < int(count) {
		count = int32(len(source))
	}
	entersyscall()
	written := write1(uintptr(fd), unsafe.Pointer(&source[0]), count)
	exitsyscall()
	if written < 0 {
		return 0, uintptr(-written), true
	}
	return int(written), 0, true
}

//go:linkname gomadTraceExit
func gomadTraceExit(code int32) {
	exit(code)
}

//go:linkname gomadTraceMap
func gomadTraceMap(descriptor int32, size uintptr, writable bool) []byte {
	if descriptor < 3 || size == 0 {
		return nil
	}
	protection := int32(_PROT_READ)
	if writable {
		protection |= _PROT_WRITE
	}
	mapped, errno := mmap(nil, size, protection, gomadMapShared, descriptor, 0)
	if errno != 0 {
		return nil
	}
	return unsafe.Slice((*byte)(mapped), int(size))
}

//go:linkname gomadTraceWrite
func gomadTraceWrite(descriptor int32, source []byte) bool {
	if descriptor < 3 || len(source) == 0 || len(source) > 1<<31-1 {
		return false
	}
	return gomadBlockingWrite(uintptr(descriptor), unsafe.Pointer(&source[0]), int32(len(source))) == int32(len(source))
}

//go:linkname gomadIOConfigFrame
func gomadIOConfigFrame() *[gomadBootstrapFrameBytes]byte {
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
	if offset != int32(len(gomadConfig)) || !gomadBootstrapHeaderValid(&gomadConfig) {
		print("runtime: invalid Gomad bootstrap configuration\n")
		exit(2)
	}
	gomadConfigPresent = true
	gomadIOProfile = true
	return true
}

func gomadConfigSeed() uint64 {
	return gomadBootstrapSeed(&gomadConfig)
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

//go:linkname gomadControlEnvironment
func gomadControlEnvironment(prefix string) (string, bool) {
	return gomadEnv(prefix)
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

// gomadAwaitHostSyscallExit waits until gp, if it is inside a host syscall
// that the runner answers on its own schedule (a pipe write, a read-only
// mount lookup), has returned and queued itself as an arrival. A stack scan
// or goroutine profile otherwise sees either the frames inside the syscall
// or the frames after it depending on when the host answered, and the
// collector's view of live memory would follow host time. Simulation
// transport reads are excluded: they block until the simulation advances,
// which the scanning goroutine may itself be needed for.
func gomadAwaitHostSyscallExit(gp *g) {
	for !gp.gomadSimulationTransport && readgstatus(gp)&^_Gscan == _Gsyscall {
		osyield()
	}
}
