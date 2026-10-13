// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The build constraint matches env_unix.go, whose copyenv uses these
// declarations.

//go:build unix || (js && wasm) || plan9 || wasip1

package syscall

import _ "unsafe"

//go:linkname gomadDeterministicEnabled runtime.gomadDeterministicEnabled
func gomadDeterministicEnabled() bool

//go:linkname gomadIOProfileEnabled runtime.gomadIOProfileEnabled
func gomadIOProfileEnabled() bool
