// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rand

import (
	"crypto/internal/rand"
	"internal/gomadio"
)

func init() {
	if gomadio.Enabled() {
		Reader = gomadio.RandomReader()
		// Key generation and other internal consumers read the FIPS DRBG,
		// which ignores Reader and otherwise draws from the host.
		rand.SetTestingReader(Reader)
	}
}
