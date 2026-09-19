// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package user

import (
	"internal/gomadio"
	"internal/gomadtrace"
)

func gomadObserveBoundary(id uint64) {
	gomadtrace.ObserveBoundary(id)
}

func gomadInterceptCurrent() (*User, error, bool) {
	if !gomadio.Enabled() {
		return nil, nil, false
	}
	return nil, gomadio.ErrUnsupported, true
}
