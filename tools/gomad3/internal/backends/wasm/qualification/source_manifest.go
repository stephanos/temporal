package qualification

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
)

// The conductor's manifest uses sha256sum's plain, non-escaped line format:
// lowercase hex, two spaces, absolute path. Paths may contain spaces.
func validateSourceManifest(encoded []byte, checkFiles bool) error {
	if len(encoded) == 0 || len(encoded) > 16<<20 {
		return errors.New("bounded nonempty integrated source manifest required")
	}
	lines := bytes.Split(bytes.TrimSuffix(encoded, []byte{'\n'}), []byte{'\n'})
	if len(lines) > 20000 {
		return errors.New("integrated source manifest entry limit")
	}
	seen := map[string]bool{}
	var total uint64
	for _, line := range lines {
		if len(line) < 67 || string(line[64:66]) != "  " {
			return errors.New("malformed integrated source manifest line")
		}
		digest, err := record.ParseSHA256("sha256:" + string(line[:64]))
		if err != nil {
			return err
		}
		path := string(line[66:])
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") || seen[path] {
			return errors.New("duplicate or malformed integrated source manifest path")
		}
		seen[path] = true
		if checkFiles {
			data, err := hostfs.ReadBounded(path, 128<<20)
			if err != nil {
				return err
			}
			total += uint64(len(data))
			if total > 512<<20 {
				return errors.New("integrated source inventory byte limit")
			}
			if record.HashBytes(data) != digest {
				return errors.New("integrated source manifest file identity changed")
			}
		}
	}
	return nil
}
