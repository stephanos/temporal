package wasi

import (
	"bytes"
	"fmt"
)

func helperProfileStderr(stderr []byte) error {
	for _, diagnostic := range []string{"LLVM Profile Error", "LLVM Profile Warning"} {
		if bytes.Contains(stderr, []byte(diagnostic)) {
			return fmt.Errorf("LLVM profile diagnostic in helper stderr: %s", stderr)
		}
	}
	return nil
}
