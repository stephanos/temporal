package hostfs

import "errors"

var ErrContended = errors.New("lock is already held")
var ErrUnsupported = errors.New("advisory file locking is unsupported")
