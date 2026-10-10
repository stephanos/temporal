// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix && !((darwin && arm64) || (linux && amd64))

package syscall

//go:nosplit
//go:norace
func gomadVirtualFD(int) bool { return false }

func gomadVirtualWrite(int, []byte) (int, error, bool) { return 0, nil, false }

//go:nosplit
//go:norace
func gomadGeneric(trap, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, uintptr, Errno, bool) {
	return 0, 0, 0, false
}
