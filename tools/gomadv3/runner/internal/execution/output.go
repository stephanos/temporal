package execution

import "go.temporal.io/server/tools/gomadv3/internal/hostexec"

type OutputCapture = hostexec.Capture

type Output = hostexec.Output

func NewOutputCapture(limit uint64) (*OutputCapture, error) {
	return hostexec.New(limit)
}
