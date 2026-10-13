// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadio

import (
	"sync"
	"time"
)

type simulationListener struct {
	address  Address
	owner    simulationEndpoint
	network  *simulationNetwork
	mu       sync.Mutex
	pending  []*simulationConn
	closed   bool
	deadline time.Time
	changed  chan struct{}
}
type simulationConn struct{ pairedConn }

func (listener *simulationListener) Accept() (*Conn, error) {
	connection, err := listener.network.accept(listener)
	if err != nil {
		record("net.accept", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(err), 0, 0)
		return nil, err
	}
	record("net.accept", networkArguments("tcp", listener.address.Port, connection.remote.Port), nil, 0, 0, 0, 0)
	return &Conn{implementation: connection}, nil
}
func (listener *simulationListener) Close() error {
	err := listener.network.closeListener(listener)
	record("net.listener.close", networkArguments("tcp", listener.address.Port), nil, 0, resultClass(err), 0, 0)
	return err
}
func (listener *simulationListener) Address() Address { return listener.address }
func (listener *simulationListener) SetDeadline(deadline time.Time) error {
	if err := validateSimulationEndpoint(listener.network, listener.owner); err != nil {
		return err
	}
	listener.mu.Lock()
	listener.deadline = deadline
	listener.signal()
	listener.mu.Unlock()
	return nil
}
func (listener *simulationListener) signal() {
	close(listener.changed)
	listener.changed = make(chan struct{})
}
