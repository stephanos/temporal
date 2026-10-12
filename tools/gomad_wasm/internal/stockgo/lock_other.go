//go:build !unix

package stockgo

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

// lock polls for exclusive creation of path. Without advisory locks a crashed
// holder leaves the file behind; it is treated as stale after ten minutes.
func lock(path string) (unlock func(), err error) {
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = file.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > 10*time.Minute {
			_ = os.Remove(path)
			continue
		}
		time.Sleep(200 * time.Millisecond)
	}
}
