// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadsim

import (
	"errors"
	"math"
	"sync"
	"syscall"
	_ "unsafe"

	"internal/gomadchoicewire"
)

const maximumDomains = 1 << 20
const maximumRuns = 1024

const (
	OutputStdout uint8 = iota + 1
	OutputStderr
)

type domain struct {
	mu          sync.Mutex
	token       uint64
	node        string
	address     string
	incarnation uint64
	revoked     bool
	finished    bool
	run         *run
	stdout      capture
	stderr      capture
}

type run struct {
	token          uint64
	maximumDomains uint64
	streamLimit    uint64
	domains        []*domain
	finished       bool
}

type capture struct {
	seen      bool
	headLimit uint64
	tailLimit uint64
	head      []byte
	tail      []byte
	total     uint64
	hasher    *gomadchoicewire.Hasher
}

type output struct {
	Node          string
	Incarnation   uint64
	Stream        uint8
	Bytes         []byte
	FullSHA256    [gomadchoicewire.DigestBytes]byte
	TotalBytes    uint64
	RetainedBytes uint64
}

type NetworkDomain struct {
	Run         uint64
	Token       uint64
	Node        string
	Address     string
	Incarnation uint64
}

var outputMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'O', '1', 0}

var domains = struct {
	sync.Mutex
	nextDomain uint64
	nextRun    uint64
	values     map[uint64]*domain
	runs       map[uint64]*run
}{values: make(map[uint64]*domain), runs: make(map[uint64]*run)}

//go:linkname runtimeSimulationDomain runtime.gomadSimulationDomain
func runtimeSimulationDomain() uint64

//go:linkname runtimeSimulationSetDomain runtime.gomadSimulationSetDomain
func runtimeSimulationSetDomain(uint64) uint64

//go:linkname Begin
func Begin(observationBytes, maximumRunDomains uint64) uint64 {
	if observationBytes == 0 || maximumRunDomains == 0 || maximumRunDomains > maximumDomains {
		return 0
	}
	domains.Lock()
	defer domains.Unlock()
	if len(domains.runs) >= maximumRuns {
		return 0
	}
	domains.nextRun++
	if domains.nextRun == 0 {
		return 0
	}
	token := domains.nextRun
	domains.runs[token] = &run{
		token:          token,
		maximumDomains: maximumRunDomains,
		streamLimit:    observationBytes / (maximumRunDomains * 2),
	}
	return token
}

//go:linkname Register
func Register(runToken uint64, node, address string, incarnation uint64) uint64 {
	domains.Lock()
	defer domains.Unlock()
	if len(domains.values) >= maximumDomains {
		return 0
	}
	run, ok := domains.runs[runToken]
	if !ok || run.finished || uint64(len(run.domains)) >= run.maximumDomains {
		return 0
	}
	domains.nextDomain++
	if domains.nextDomain == 0 {
		return 0
	}
	token := domains.nextDomain
	domain := &domain{
		token:       token,
		node:        node,
		address:     address,
		incarnation: incarnation,
		run:         run,
		stdout:      newCapture(run.streamLimit),
		stderr:      newCapture(run.streamLimit),
	}
	domains.values[token] = domain
	run.domains = append(run.domains, domain)
	return token
}

//go:linkname Enter
func Enter(token uint64) (uint64, bool) {
	domains.Lock()
	domain, ok := domains.values[token]
	if !ok {
		domains.Unlock()
		return 0, false
	}
	domain.mu.Lock()
	domains.Unlock()
	defer domain.mu.Unlock()
	if domain.revoked || domain.finished {
		return 0, false
	}
	return runtimeSimulationSetDomain(token), true
}

//go:linkname Leave
func Leave(previous uint64) {
	runtimeSimulationSetDomain(previous)
}

//go:linkname Revoke
func Revoke(token uint64) bool {
	domains.Lock()
	domain, ok := domains.values[token]
	if !ok {
		domains.Unlock()
		return false
	}
	domain.mu.Lock()
	domains.Unlock()
	defer domain.mu.Unlock()
	if domain.revoked || domain.finished {
		return false
	}
	domain.revoked = true
	return true
}

func Hostname() (string, error, bool) {
	domain, err, handled := currentDomain()
	if !handled || err != nil {
		return "", err, handled
	}
	defer domain.mu.Unlock()
	return domain.node, nil, true
}

func ObserveOutput(stream uint8, source []byte) (error, bool) {
	domain, err, handled := currentDomain()
	if !handled || err != nil {
		return err, handled
	}
	defer domain.mu.Unlock()
	var destination *capture
	switch stream {
	case OutputStdout:
		destination = &domain.stdout
	case OutputStderr:
		destination = &domain.stderr
	default:
		return syscall.EINVAL, true
	}
	return destination.write(source), true
}

func ObserveWrite(stream uint8, source []byte, write func([]byte) (int, error)) (int, error, bool) {
	domain, err, handled := currentDomain()
	if !handled || err != nil {
		return 0, err, handled
	}
	defer domain.mu.Unlock()
	var destination *capture
	switch stream {
	case OutputStdout:
		destination = &domain.stdout
	case OutputStderr:
		destination = &domain.stderr
	default:
		return 0, syscall.EINVAL, true
	}
	written, writeErr := write(source)
	if written < 0 || written > len(source) {
		return 0, syscall.EIO, true
	}
	if written != 0 {
		writeErr = errors.Join(writeErr, destination.write(source[:written]))
	}
	return written, writeErr, true
}

func currentDomain() (*domain, error, bool) {
	token := runtimeSimulationDomain()
	if token == 0 {
		return nil, nil, false
	}
	domains.Lock()
	domain, ok := domains.values[token]
	if !ok {
		domains.Unlock()
		return nil, syscall.ESTALE, true
	}
	domain.mu.Lock()
	domains.Unlock()
	if domain.revoked || domain.finished {
		domain.mu.Unlock()
		return nil, syscall.ESTALE, true
	}
	return domain, nil, true
}

func CurrentNetworkDomain() (NetworkDomain, error, bool) {
	domain, err, handled := currentDomain()
	if !handled || err != nil {
		return NetworkDomain{}, err, handled
	}
	defer domain.mu.Unlock()
	return NetworkDomain{
		Run:         domain.run.token,
		Token:       domain.token,
		Node:        domain.node,
		Address:     domain.address,
		Incarnation: domain.incarnation,
	}, nil, true
}

func DescribeNetworkDomain(token uint64) (NetworkDomain, bool) {
	domains.Lock()
	domain, ok := domains.values[token]
	if !ok {
		domains.Unlock()
		return NetworkDomain{}, false
	}
	domain.mu.Lock()
	domains.Unlock()
	defer domain.mu.Unlock()
	if domain.finished {
		return NetworkDomain{}, false
	}
	return NetworkDomain{
		Run:         domain.run.token,
		Token:       domain.token,
		Node:        domain.node,
		Address:     domain.address,
		Incarnation: domain.incarnation,
	}, true
}

//go:linkname Finish
func Finish(runToken uint64) ([]byte, bool) {
	domains.Lock()
	run, ok := domains.runs[runToken]
	if !ok || run.finished {
		domains.Unlock()
		return nil, false
	}
	run.finished = true
	outputs := make([]output, 0, len(run.domains)*2)
	for _, domain := range run.domains {
		domain.mu.Lock()
		domain.finished = true
		if domain.stdout.seen {
			outputs = append(outputs, domain.stdout.result(domain, OutputStdout))
		}
		if domain.stderr.seen {
			outputs = append(outputs, domain.stderr.result(domain, OutputStderr))
		}
		delete(domains.values, domain.token)
		domain.mu.Unlock()
	}
	delete(domains.runs, runToken)
	run.domains = nil
	domains.Unlock()
	return encodeOutputs(outputs), true
}

func newCapture(limit uint64) capture {
	tail := limit / 4
	return capture{headLimit: limit - tail, tailLimit: tail, hasher: gomadchoicewire.NewHasher()}
}

func (capture *capture) write(source []byte) error {
	if math.MaxUint64-capture.total < uint64(len(source)) {
		return syscall.EOVERFLOW
	}
	capture.hasher.Write(source)
	capture.seen = true
	capture.total += uint64(len(source))
	remainingHead := capture.headLimit - uint64(len(capture.head))
	if remainingHead != 0 {
		toHead := min(remainingHead, uint64(len(source)))
		capture.head = append(capture.head, source[:toHead]...)
		source = source[toHead:]
	}
	capture.appendTail(source)
	return nil
}

func (capture *capture) appendTail(source []byte) {
	if capture.tailLimit == 0 || len(source) == 0 {
		return
	}
	if uint64(len(source)) >= capture.tailLimit {
		capture.tail = append(capture.tail[:0], source[uint64(len(source))-capture.tailLimit:]...)
		return
	}
	if combined := uint64(len(capture.tail) + len(source)); combined > capture.tailLimit {
		excess := combined - capture.tailLimit
		copy(capture.tail, capture.tail[excess:])
		capture.tail = capture.tail[:uint64(len(capture.tail))-excess]
	}
	capture.tail = append(capture.tail, source...)
}

func (capture *capture) result(domain *domain, stream uint8) output {
	retained := make([]byte, 0, len(capture.head)+len(capture.tail))
	retained = append(retained, capture.head...)
	retained = append(retained, capture.tail...)
	retainedBytes := uint64(len(retained))
	result := output{Node: domain.node, Incarnation: domain.incarnation, Stream: stream, Bytes: retained, TotalBytes: capture.total, RetainedBytes: retainedBytes}
	result.FullSHA256 = capture.hasher.Sum()
	return result
}

func encodeOutputs(outputs []output) []byte {
	encoded := make([]byte, 0, 16)
	encoded = append(encoded, outputMagic[:]...)
	encoded = appendUint64(encoded, uint64(len(outputs)))
	for _, output := range outputs {
		encoded = append(encoded, output.Stream)
		encoded = append(encoded, 0, 0, 0, 0, 0, 0, 0)
		encoded = appendUint64(encoded, output.Incarnation)
		encoded = appendUint64(encoded, uint64(len(output.Node)))
		encoded = appendUint64(encoded, output.RetainedBytes)
		encoded = appendUint64(encoded, output.TotalBytes)
		encoded = append(encoded, output.FullSHA256[:]...)
		encoded = append(encoded, output.Node...)
		encoded = append(encoded, output.Bytes...)
	}
	return encoded
}

func appendUint64(destination []byte, value uint64) []byte {
	return append(destination,
		byte(value), byte(value>>8), byte(value>>16), byte(value>>24),
		byte(value>>32), byte(value>>40), byte(value>>48), byte(value>>56),
	)
}
