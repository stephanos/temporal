// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	"internal/runtime/atomic"
	"internal/runtime/exithook"
)

var gomadEnabled bool
var gomadSeed uint64
var gomadExternal bool
var gomadIOProfile bool
var gomadConfigPresent bool
var gomadConfig [212]byte

var gomadChoiceEnabled bool
var gomadChoiceMapping unsafe.Pointer
var gomadChoiceMappingBytes uint64
var gomadChoiceTerminalDescriptor int32
var gomadChoiceNext atomic.Uint64
var gomadChoiceRecords atomic.Uint64
var gomadChoiceOverflow atomic.Uint32
var gomadChoiceFinalized atomic.Uint32
var gomadChoiceHookRegistered bool

const gomadInitialTime = 946684800000000000
const gomadMapShared = 1

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
	debug.asyncpreemptoff = 1
	haveSysmon = false
	randomizeScheduler = true
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
	descriptor, descriptorOK := gomadParseSeed(descriptorValue)
	terminal, terminalOK := gomadParseSeed(terminalValue)
	mappingBytes, bytesOK := gomadParseSeed(bytesValue)
	if !terminalPresent || !bytesPresent || !descriptorOK || !terminalOK || !bytesOK || descriptor > 1<<31-1 || terminal > 1<<31-1 || mappingBytes < gomadChoiceHeaderBytes+gomadChoiceRecordBytes || mappingBytes > 64<<20 {
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
	if gomadChoiceRead32(bytes[8:12]) != 1 || gomadChoiceRead64(bytes[16:24]) != mappingBytes || gomadChoiceRead64(bytes[24:32]) != gomadChoiceHeaderBytes || gomadChoiceRead64(bytes[32:40]) != 0 {
		print("runtime: invalid Gomad choice trace header\n")
		exit(2)
	}
	gomadChoiceEnabled = true
	gomadChoiceMapping = mapped
	gomadChoiceMappingBytes = mappingBytes
	gomadChoiceTerminalDescriptor = int32(terminal)
	gomadChoiceNext.Store(gomadChoiceHeaderBytes)
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

func gomadChoiceRecord(kind, flags uint8, siteOffset uint64, alternatives, selected, data uint32) {
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
	gomadChoicePut64(record[:8], ordinal)
	record[8] = kind
	record[9] = flags
	gomadChoicePut32(record[12:16], alternatives)
	gomadChoicePut32(record[16:20], selected)
	gomadChoicePut32(record[20:24], data)
	gomadChoicePut64(record[24:32], siteOffset)
	gomadChoiceRecords.Store(ordinal + 1)
	gomadChoicePut64(bytes[24:32], offset+gomadChoiceRecordBytes)
	gomadChoicePut64(bytes[32:40], ordinal+1)
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
	records := gomadChoiceRecords.Load()
	mappingBytes := uint64(gomadChoiceHeaderBytes) + records*gomadChoiceRecordBytes
	bytes := unsafe.Slice((*byte)(gomadChoiceMapping), int(gomadChoiceMappingBytes))
	digest := gomadChoiceHash(bytes[gomadChoiceHeaderBytes:mappingBytes])
	var terminal [gomadChoiceTerminalBytes]byte
	copy(terminal[:8], gomadChoiceTerminalMagic[:])
	gomadChoicePut32(terminal[8:12], 1)
	terminal[12] = gomadChoiceTerminalComplete
	if gomadChoiceOverflow.Load() != 0 {
		terminal[12] = gomadChoiceTerminalOverflow
	}
	gomadChoicePut64(terminal[16:24], records)
	gomadChoicePut64(terminal[24:32], mappingBytes)
	copy(terminal[32:64], digest[:])
	checksum := gomadChoiceHash(terminal[:gomadChoiceTerminalChecksumOffset])
	copy(terminal[gomadChoiceTerminalChecksumOffset:], checksum[:])
	if write1(uintptr(gomadChoiceTerminalDescriptor), unsafe.Pointer(&terminal[0]), gomadChoiceTerminalBytes) != gomadChoiceTerminalBytes {
		panic("gomadv3: write choice terminal frame")
	}
}

func gomadStartUserCode(mp *m) {
	if gomadChoiceEnabled && !gomadChoiceHookRegistered {
		exithook.Add(exithook.Hook{F: gomadChoiceFinalize, RunOnFailure: true})
		gomadChoiceHookRegistered = true
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
