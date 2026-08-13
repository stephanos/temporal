package filelock

import "errors"

var ErrContended = errors.New("lock is already held")
var ErrSymbolicLink = errors.New("lock path is a symbolic link")
var ErrUnsupported = errors.New("advisory file locking is unsupported")
