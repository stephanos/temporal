// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(darwin && arm64)

package runtime

// gomadDisableASLR is a no-op where the target links at a fixed address:
// linux/amd64 targets are not position independent.
func gomadDisableASLR() {}
