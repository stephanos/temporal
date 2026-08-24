// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadsim

import "testing"

func TestProcessRequestBlocksSimulationTime(t *testing.T) {
	tests := map[string]struct {
		request []byte
		want    bool
	}{
		"start": {request: []byte(`{"profile":"gomad3.simulation-process/v3","kind":"start","request":1}`), want: true},
		"wait":  {request: []byte(`{"profile":"gomad3.simulation-process/v3","kind":"wait","request":1}`), want: false},
		"other": {request: []byte(`{"kind":"wait","profile":"gomad3.simulation-process/v3","request":1}`), want: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := processRequestBlocksSimulationTime(test.request); got != test.want {
				t.Fatalf("processRequestBlocksSimulationTime() = %t, want %t", got, test.want)
			}
		})
	}
}
