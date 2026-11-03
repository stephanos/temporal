package temporal

import _ "embed"

//go:embed umpire_temporal_guest.wasm
var guest string

// Guest returns the committed Temporal Model guest.
func Guest() []byte { return []byte(guest) }
