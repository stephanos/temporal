package hostfs

import internal "go.temporal.io/server/tools/gomad3/internal/hostfs"

func ReadBounded(path string, maximum uint64) ([]byte, error) {
	return internal.ReadBounded(path, maximum)
}
