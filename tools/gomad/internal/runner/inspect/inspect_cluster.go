package inspect

// ClusterInspection projects the retained WASM cluster evidence a backend
// hook authenticated (fn-235): each incarnation's partial streams, the abort
// virtual time and an exhausted limit, which the generic record omits.
type ClusterInspection struct {
	EvidenceSchema      string                      `json:"evidence_schema"`
	Outcome             string                      `json:"outcome"`
	Reason              string                      `json:"reason,omitempty"`
	VirtualElapsedNanos uint64                      `json:"virtual_elapsed_nanos"`
	Exhausted           *ClusterLimitInspection     `json:"exhausted,omitempty"`
	Instances           []ClusterInstanceInspection `json:"instances"`
}

// ClusterLimitInspection is an exhausted cluster limit and its accounting.
type ClusterLimitInspection struct {
	Dimension string `json:"dimension"`
	Limit     uint64 `json:"limit"`
	Used      uint64 `json:"used"`
	Delta     uint64 `json:"delta"`
}

// ClusterInstanceInspection is one incarnation's retained terminal facts.
type ClusterInstanceInspection struct {
	Node        string                  `json:"node"`
	Incarnation uint64                  `json:"incarnation"`
	State       string                  `json:"state,omitempty"`
	Reason      string                  `json:"reason,omitempty"`
	Interrupted bool                    `json:"interrupted"`
	Stdout      ClusterStreamInspection `json:"stdout"`
	Stderr      ClusterStreamInspection `json:"stderr"`
}

// ClusterStreamInspection is a retained stream's size and bounded tail.
type ClusterStreamInspection struct {
	Bytes uint64 `json:"bytes"`
	Tail  string `json:"tail"`
}
