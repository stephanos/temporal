//go:build !unix

package campaign

import "os/exec"

// detach does nothing where process groups are not a thing.
func detach(*exec.Cmd) {}
