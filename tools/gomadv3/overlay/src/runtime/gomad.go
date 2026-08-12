// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "unsafe"

var gomadEnabled bool
var gomadSeed uint64
var gomadExternal bool
var gomadIOProfile bool
var gomadConfigPresent bool
var gomadConfig [212]byte

const gomadInitialTime = 946684800000000000

func gomadInit() {
	var seed uint64
	_, profile := gomadEnv("GOMADV3_IO_PROFILE=")
	if profile {
		if !gomadReadConfig() {
			print("runtime: missing Gomad bootstrap configuration\n")
			exit(2)
		}
		seed = gomadConfigSeed()
	} else {
		value, present := gomadSeedEnv()
		if !present {
			return
		}
		var ok bool
		seed, ok = gomadParseSeed(value)
		if !ok {
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

func gomadStartUserCode(mp *m) {
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
