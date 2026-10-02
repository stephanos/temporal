package replay

import (
	"fmt"
	"io"
)

// writeLine reports one line. A report the caller cannot receive is not a failure worth changing
// the exit code for, so the write error is deliberately dropped.
func writeLine(destination io.Writer, format string, arguments ...any) {
	_, _ = fmt.Fprintf(destination, format+"\n", arguments...)
}
