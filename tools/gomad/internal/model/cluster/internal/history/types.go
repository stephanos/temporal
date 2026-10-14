package history

type BootID string

type NodeID string

type VolumeID string

type OutputStream string

type NetworkTransitionKind string

type NetworkOutcome string

type VolumeTransitionKind string

type VolumeOutcome string

type VolumeCrashCapacity string

type NodeHandle struct {
	Node        NodeID `json:"node"`
	Incarnation uint64 `json:"incarnation"`
}

type OutputObservation struct {
	Handle         NodeHandle   `json:"handle"`
	Stream         OutputStream `json:"stream"`
	Bytes          []byte       `json:"bytes"`
	FullSHA256     string       `json:"full_sha256"`
	TotalBytes     uint64       `json:"total_bytes"`
	RetainedBytes  uint64       `json:"retained_bytes"`
	DiscardedBytes uint64       `json:"discarded_bytes"`
	Truncated      bool         `json:"truncated"`
}

type NetworkEndpoint struct {
	Node        NodeID `json:"node,omitempty"`
	Incarnation uint64 `json:"incarnation,omitempty"`
	Address     string `json:"address,omitempty"`
	Port        uint64 `json:"port,omitempty"`
}

type NetworkTransition struct {
	Ordinal     uint64                `json:"ordinal"`
	Kind        NetworkTransitionKind `json:"kind"`
	Source      NetworkEndpoint       `json:"source,omitempty"`
	Destination NetworkEndpoint       `json:"destination,omitempty"`
	Connection  uint64                `json:"connection,omitempty"`
	Bytes       uint64                `json:"bytes,omitempty"`
	Count       uint64                `json:"count,omitempty"`
	DelayNanos  uint64                `json:"delay_nanos,omitempty"`
	Outcome     NetworkOutcome        `json:"outcome"`
}

type NetworkNodeSnapshot struct {
	Node             NodeID `json:"node"`
	Address          string `json:"address"`
	LastIncarnation  uint64 `json:"last_incarnation"`
	NextListenerPort uint64 `json:"next_listener_port"`
	NextClientPort   uint64 `json:"next_client_port"`
}

type NetworkLinkSnapshot struct {
	From       NodeID `json:"from"`
	To         NodeID `json:"to"`
	Enabled    bool   `json:"enabled"`
	DelayNanos uint64 `json:"delay_nanos"`
}

type NetworkListenerSnapshot struct {
	Endpoint NetworkEndpoint `json:"endpoint"`
	Closed   bool            `json:"closed"`
}

type NetworkConnectionSnapshot struct {
	Identity    uint64          `json:"identity"`
	Client      NetworkEndpoint `json:"client"`
	Server      NetworkEndpoint `json:"server"`
	Closed      bool            `json:"closed"`
	Reset       bool            `json:"reset"`
	ClientBytes uint64          `json:"client_bytes"`
	ServerBytes uint64          `json:"server_bytes"`
}

type NetworkSnapshot struct {
	Nodes            []NetworkNodeSnapshot       `json:"nodes"`
	Links            []NetworkLinkSnapshot       `json:"links"`
	Listeners        []NetworkListenerSnapshot   `json:"listeners"`
	Connections      []NetworkConnectionSnapshot `json:"connections"`
	NextConnection   uint64                      `json:"next_connection"`
	TransitionSHA256 string                      `json:"transition_sha256"`
	Identity         string                      `json:"identity"`
}

type NetworkRecord struct {
	Transitions []NetworkTransition `json:"transitions"`
	Snapshot    NetworkSnapshot     `json:"snapshot"`
}

type VolumeTransition struct {
	Ordinal            uint64               `json:"ordinal"`
	Kind               VolumeTransitionKind `json:"kind"`
	Handle             NodeHandle           `json:"handle"`
	Volume             VolumeID             `json:"volume"`
	Operation          uint64               `json:"operation,omitempty"`
	Dependencies       []uint64             `json:"dependencies,omitempty"`
	SelectedOperations []uint64             `json:"selected_operations,omitempty"`
	Path               string               `json:"path,omitempty"`
	Destination        string               `json:"destination,omitempty"`
	Inode              uint64               `json:"inode,omitempty"`
	Offset             uint64               `json:"offset,omitempty"`
	Bytes              uint64               `json:"bytes,omitempty"`
	PayloadSHA256      string               `json:"payload_sha256,omitempty"`
	EffectSHA256       string               `json:"effect_sha256,omitempty"`
	Outcome            VolumeOutcome        `json:"outcome"`
}

type VolumeEntrySnapshot struct {
	Path       string `json:"path"`
	Mode       uint32 `json:"mode"`
	Kind       string `json:"kind"`
	ModTime    int64  `json:"mod_time"`
	Size       uint64 `json:"size"`
	DataSHA256 string `json:"data_sha256,omitempty"`
}

type VolumeStateSnapshot struct {
	Volume            VolumeID              `json:"volume"`
	Node              NodeID                `json:"node"`
	Mount             string                `json:"mount"`
	CapacityBytes     uint64                `json:"capacity_bytes"`
	Persisted         []VolumeEntrySnapshot `json:"persisted"`
	Volatile          []VolumeEntrySnapshot `json:"volatile"`
	PendingOperations uint64                `json:"pending_operations"`
	PendingSHA256     string                `json:"pending_sha256"`
	NextOperation     uint64                `json:"next_operation"`
	Identity          string                `json:"identity"`
}

type VolumeSnapshot struct {
	Volumes          []VolumeStateSnapshot `json:"volumes"`
	TransitionSHA256 string                `json:"transition_sha256"`
	Identity         string                `json:"identity"`
}

type VolumeRecord struct {
	Transitions []VolumeTransition `json:"transitions"`
	Snapshot    VolumeSnapshot     `json:"snapshot"`
}

type VolumeCrashEnumerationLimits struct {
	States     uint64 `json:"states"`
	Operations uint64 `json:"operations"`
	Depth      uint64 `json:"depth"`
	Bytes      uint64 `json:"bytes"`
	WallNanos  uint64 `json:"wall_nanos"`
}

type VolumeCrashFrontier struct {
	Volume        VolumeID `json:"volume"`
	PendingSHA256 string   `json:"pending_sha256"`
	Cursor        []byte   `json:"cursor"`
	Seen          []string `json:"seen"`
	Identity      string   `json:"identity"`
}

type VolumeCrashEntry struct {
	Path    string `json:"path"`
	Mode    uint32 `json:"mode"`
	Kind    string `json:"kind"`
	ModTime int64  `json:"mod_time"`
	Data    []byte `json:"data"`
}

type VolumeCrashState struct {
	Volume             VolumeID           `json:"volume"`
	PendingSHA256      string             `json:"pending_sha256"`
	SelectedOperations []uint64           `json:"selected_operations"`
	Entries            []VolumeCrashEntry `json:"entries"`
	Identity           string             `json:"identity"`
}

type VolumeCrashEnumeration struct {
	States   []VolumeCrashState   `json:"states"`
	Frontier *VolumeCrashFrontier `json:"frontier,omitempty"`
	Complete bool                 `json:"complete"`
	Capacity VolumeCrashCapacity  `json:"capacity,omitempty"`
}

type ClusterStaticIdentities struct {
	TargetSHA256         string `json:"target_sha256"`
	PlatformBundleSHA256 string `json:"platform_bundle_sha256"`
}

type ClusterModelIdentities struct {
	RuntimeDomainSHA256 string `json:"runtime_domain_sha256"`
	ProcessSHA256       string `json:"process_sha256"`
	NetworkSHA256       string `json:"network_sha256"`
	VolumeSHA256        string `json:"volume_sha256"`
	FaultSHA256         string `json:"fault_sha256"`
	ScenarioSHA256      string `json:"scenario_sha256"`
	OracleSHA256        string `json:"oracle_sha256"`
}

type NodeContext struct {
	NodeHandle
	Address string `json:"address"`
	Config  []byte `json:"config,omitempty"`
}

type HistoryOperation struct {
	ID         string `json:"id"`
	Actor      string `json:"actor"`
	Kind       string `json:"kind"`
	Invocation uint64 `json:"invocation"`
	Completion uint64 `json:"completion"`
	Input      []byte `json:"input,omitempty"`
	Output     []byte `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

type OracleEvidence struct {
	Label      string `json:"label"`
	Value      []byte `json:"value"`
	FullSHA256 string `json:"full_sha256"`
}

type OracleResult struct {
	SoftAssertion   bool             `json:"soft_assertion,omitempty"`
	ReportOnly      bool             `json:"report_only,omitempty"`
	Inconclusive    bool             `json:"inconclusive,omitempty"`
	Name            string           `json:"name"`
	Passed          bool             `json:"passed"`
	Evidence        []OracleEvidence `json:"evidence"`
	FailureIdentity string           `json:"failure_identity,omitempty"`
	Identity        string           `json:"identity"`
}

type RuntimeNetworkError struct {
	Kind           string             `json:"kind"`
	Message        string             `json:"message"`
	Ordinal        uint64             `json:"ordinal,omitempty"`
	ExpectedSHA256 string             `json:"expected_sha256,omitempty"`
	ActualSHA256   string             `json:"actual_sha256,omitempty"`
	Expected       *NetworkTransition `json:"expected,omitempty"`
	Actual         *NetworkTransition `json:"actual,omitempty"`
}

type RuntimeVolumeError struct {
	Kind           string
	Message        string
	Ordinal        uint64
	ExpectedSHA256 string
	ActualSHA256   string
	Expected       *VolumeTransition
	Actual         *VolumeTransition
}

type RuntimeConfig struct {
	Seed    uint64
	Limits  RuntimeLimits
	Nodes   []RuntimeNode
	Links   []RuntimeLink
	Volumes []RuntimeVolume
	Replay  *RuntimeReplay
}

type RuntimeLimits struct {
	NetworkListeners, NetworkConnections, NetworkBytes            uint64
	NetworkConnectionBytes, NetworkStallNanos, NetworkTransitions uint64
	VolumeOperations, VolumeTransitions                           uint64
}

type RuntimeNode struct {
	ID      NodeID
	Address string
	Volumes []RuntimeMount
}

type RuntimeLink struct {
	From, To   NodeID
	Enabled    bool
	DelayNanos uint64
}

type RuntimeVolume struct {
	ID            VolumeID
	CapacityBytes uint64
}

type RuntimeMount struct {
	Volume VolumeID
	Path   string
}

type RuntimeReplay struct {
	Network NetworkRecord
	Volumes VolumeRecord
}

func OutputBefore(left, right OutputObservation) bool {
	if left.Handle.Node != right.Handle.Node {
		return left.Handle.Node < right.Handle.Node
	}
	if left.Handle.Incarnation != right.Handle.Incarnation {
		return left.Handle.Incarnation < right.Handle.Incarnation
	}
	if left.Stream != right.Stream {
		return left.Stream == "stdout"
	}
	return false
}
