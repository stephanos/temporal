package process

import "go.temporal.io/server/tools/gomadv3/internal/outputcapture"

type OutputCapture = outputcapture.Capture

type Output = outputcapture.Output

func NewOutputCapture(limit uint64) (*OutputCapture, error) {
	return outputcapture.New(limit)
}
