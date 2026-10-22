package choice

type Request struct {
	Mode              Mode
	ExecutionIdentity ExecutionIdentity
	Limit             uint64
	Trace             []byte
	TraceSHA256       [32]byte
	TapeSHA256        [32]byte
	Decisions         uint64
	PrefixControl     *PrefixControl
}

type PrefixControl struct {
	Cutoff   uint64        `json:"cutoff"`
	Override *RankOverride `json:"override,omitempty"`
}

type RankOverride struct {
	Ordinal     uint64 `json:"ordinal"`
	Rank        uint32 `json:"rank"`
	Reconstruct bool   `json:"reconstruct"`
}
