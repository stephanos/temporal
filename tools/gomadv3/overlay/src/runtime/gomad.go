// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

var gomadEnabled bool
var gomadSeed uint64

const gomadInitialTime = 946684800000000000

func gomadInit() {
	value, present := gomadSeedEnv()
	if !present {
		return
	}

	seed, ok := gomadParseSeed(value)
	if !ok {
		print("runtime: invalid GOMADSEED\n")
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
	switch GOOS {
	case "aix", "darwin", "ios", "dragonfly", "freebsd", "netbsd", "openbsd", "illumos", "solaris", "linux":
	default:
		return "", false
	}

	const prefix = "GOMADSEED="
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
