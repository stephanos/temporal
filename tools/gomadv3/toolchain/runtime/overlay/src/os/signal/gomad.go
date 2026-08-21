// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package signal

import (
	"internal/gomadio"
	"internal/gomadtrace"
	"os"
)

func gomadObserveBoundary(id uint64) {
	gomadtrace.ObserveBoundary(id)
}

func gomadInterceptStop(_ chan<- os.Signal) bool {
	return gomadio.Enabled()
}
