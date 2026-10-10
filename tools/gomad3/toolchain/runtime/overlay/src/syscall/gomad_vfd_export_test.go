// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (darwin && arm64) || (linux && amd64)

package syscall

var (
	GomadGenericRead   = gomadGenericRead
	GomadGenericWrite  = gomadGenericWrite
	GomadGenericWritev = gomadGenericWritev
	GomadVirtualBind   = gomadVirtualBind
	GomadVirtualSocket = gomadVirtualSocket
	GomadGeneric       = gomadGeneric
)
