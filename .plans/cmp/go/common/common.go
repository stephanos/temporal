// Package common holds the input domains both Models share. Go has one package per directory and no
// nested namespaces, so a type two Models reuse lives in a third package rather than in either.
package common

//go:generate go run ../umpire/cmd/finite -type=Timeout,Delivery

// Timeout is whether a deadline the caller may set fires while the operation runs.
type Timeout uint8

const (
	Unset Timeout = iota
	Expires
)

// Delivery is the result enum of an action an implementation may answer "not found" to.
type Delivery uint8

const (
	Accepted Delivery = iota
	NotFound
)
