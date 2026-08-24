//go:build !unix

package execution

import "fmt"

func BootstrapMain() error {
	return fmt.Errorf("target bootstrap is unsupported on this platform")
}
