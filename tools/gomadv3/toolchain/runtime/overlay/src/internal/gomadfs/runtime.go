// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"sync"
	"syscall"
	_ "unsafe"

	"internal/gomadsim"
)

type simulationFilesystemBinding struct {
	filesystem *FS
	active     bool
}

var simulationFilesystems = struct {
	sync.Mutex
	domains map[uint64]simulationFilesystemBinding
}{domains: make(map[uint64]simulationFilesystemBinding)}

var unavailableFilesystem = &FS{unavailable: syscall.ESTALE}

//go:linkname runtimeSimulationDomain runtime.gomadSimulationDomain
func runtimeSimulationDomain() uint64

//go:linkname runtimeWallNanotime runtime.nanotime1
func runtimeWallNanotime() int64

func Current() *FS {
	if gomadsim.ProcessRole() == 2 {
		return processFilesystem
	}
	domain := runtimeSimulationDomain()
	if domain == 0 {
		return Default
	}
	simulationFilesystems.Lock()
	binding, ok := simulationFilesystems.domains[domain]
	simulationFilesystems.Unlock()
	if !ok || !binding.active {
		return unavailableFilesystem
	}
	return binding.filesystem
}

func registerSimulationFilesystem(domain uint64, filesystem *FS) bool {
	if domain == 0 || filesystem == nil {
		return false
	}
	simulationFilesystems.Lock()
	defer simulationFilesystems.Unlock()
	if _, exists := simulationFilesystems.domains[domain]; exists {
		return false
	}
	simulationFilesystems.domains[domain] = simulationFilesystemBinding{filesystem: filesystem, active: true}
	return true
}

func revokeSimulationFilesystem(domain uint64) (*FS, bool) {
	simulationFilesystems.Lock()
	defer simulationFilesystems.Unlock()
	binding, ok := simulationFilesystems.domains[domain]
	if !ok || !binding.active {
		return nil, false
	}
	binding.active = false
	simulationFilesystems.domains[domain] = binding
	return binding.filesystem, true
}

func restoreSimulationFilesystem(domain uint64, filesystem *FS) bool {
	simulationFilesystems.Lock()
	defer simulationFilesystems.Unlock()
	binding, ok := simulationFilesystems.domains[domain]
	if !ok || binding.active || binding.filesystem != filesystem {
		return false
	}
	binding.active = true
	simulationFilesystems.domains[domain] = binding
	return true
}

func removeSimulationFilesystems(domains []uint64) {
	simulationFilesystems.Lock()
	defer simulationFilesystems.Unlock()
	for _, domain := range domains {
		delete(simulationFilesystems.domains, domain)
	}
}
