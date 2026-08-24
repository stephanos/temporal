//go:build !unix

package store

func processAlive(_ int) bool {
	return true
}
