/*
Package gomadruntime simulates the go runtime for programs tested using gomad.

The gomadruntime API is internal to gomad and programs should not directly use
it. This package is exported only because translated code must be able to
import it. Instead, use the public API in the gomad package.
*/
package gomadruntime
