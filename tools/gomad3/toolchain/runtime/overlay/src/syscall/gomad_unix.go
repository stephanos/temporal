// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The build constraint matches syscall_unix.go, whose Write uses these
// declarations.

//go:build unix

package syscall

import _ "unsafe"

//go:linkname gomadCapabilityGuard runtime.gomadCapabilityGuard
func gomadCapabilityGuard()

//go:linkname gomadWrite runtime.gomadSyscallWrite
func gomadWrite(fd int, p []byte) (int, uintptr, bool)
