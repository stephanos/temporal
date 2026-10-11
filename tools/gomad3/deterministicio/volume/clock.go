package volume

// Clock is the host owner's explicit model time. Reading it has no host effect.
type Clock struct{ Value int64 }

func (clock *Clock) Now() int64 { return clock.Value }
