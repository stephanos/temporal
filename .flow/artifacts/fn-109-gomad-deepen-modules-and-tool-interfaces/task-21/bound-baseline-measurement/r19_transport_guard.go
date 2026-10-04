//go:build linux

package execution

import "sync/atomic"

var r19TransportCalls atomic.Uint64

func r19GuardedDup2(oldfd, newfd int) error {
	r19TransportCalls.Add(1)
	panic("R19 measurement entered excluded descriptor transport")
}

func R19TransportCalls() uint64 { return r19TransportCalls.Load() }
