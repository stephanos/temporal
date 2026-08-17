package gomadv3sim

import (
	"fmt"
	"sync/atomic"
)

var testBootSequence atomic.Uint64

func uniqueBootID(prefix string) BootID {
	return BootID(fmt.Sprintf("sim0-%s-%d", prefix, testBootSequence.Add(1)))
}
