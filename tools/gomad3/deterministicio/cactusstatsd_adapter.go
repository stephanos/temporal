package deterministicio

import gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"

const (
	cactusStatsDModulePath                       = "github.com/cactus/go-statsd-client/v5"
	cactusStatsDVersion                          = "v5.1.0"
	cactusStatsDSum                              = "h1:sbbdfIl9PgisjEoXzvXI1lwUKWElngsjJKaZeC021P4="
	cactusStatsDOriginalSourceInventorySHA256    = "sha256:c90a044ece52d58fcbf4c3f62a2dc5c00509cb3be655428a9c45c74ce46d5e14"
	cactusStatsDReplacementSourceInventorySHA256 = "sha256:970b5cf407dcbbe2d9c3f8b3c95b0258ad95c86a216c7931407ed4a33f7ca8b7"
)

var cactusStatsDPreparedSourceSetSHA256ByHost = map[string]string{
	"darwin/arm64": "sha256:efcbc2fafa3680f9d3d6d245aaad39956446159c25d97e28eee9f1a2b1e376d0",
	"linux/amd64":  "sha256:efcbc2fafa3680f9d3d6d245aaad39956446159c25d97e28eee9f1a2b1e376d0",
}

var cactusStatsDPreparedSourceSetSHA256 = hostPin(cactusStatsDPreparedSourceSetSHA256ByHost)

// cactusStatsDRewrites refuse UDP senders before host resources while preserving
// metric formatting and caller-supplied Sender implementations.
var cactusStatsDRewrites = []sourceRewrite{
	{
		path: "statsd/sender.go", sourceSHA256: "sha256:10a117d75ffcbcae7779a5e0ec66d7a6c77a46d424d7c782ff0d0d90773b83bb", replacementSHA256: "sha256:0c3b30d8146cc6215bd68d8f5866c9c77e68bf4ff2b428b799424e49f8219eb5",
		rewrites: []anchorRewrite{
			{anchor: []byte(`func NewSimpleSender(addr string) (Sender, error) {
	c, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}

	ra, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		c.Close()
		return nil, err
	}

	sender := &SimpleSender{
		c:  c,
		ra: ra,
	}

	return sender, nil
}
`), replacement: []byte(`func NewSimpleSender(addr string) (Sender, error) {
	return nil, errors.New("gomad: StatsD UDP sender is unsupported; supply a Sender")
}
`)},
		},
	},
	{
		path: "statsd/sender_resolving.go", sourceSHA256: "sha256:19994cfe803a08aa0ff6d978be1b93fbf82753a3428fad4fec91d1c955132606", replacementSHA256: "sha256:c1b21262f87f311b8eba97b3d08d9fa419ac6fb3c0295188c0b1c6adb5bb2bbd",
		rewrites: []anchorRewrite{
			{anchor: []byte(`func NewResolvingSimpleSender(addr string, interval time.Duration) (Sender, error) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}

	addrResolved, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	sender := &ResolvingSimpleSender{
		conn:              conn,
		addrResolved:      addrResolved,
		addrUnresolved:    addr,
		reresolveInterval: interval,
		doneChan:          make(chan struct{}),
		running:           false,
	}

	sender.Start()
	return sender, nil
}
`), replacement: []byte(`func NewResolvingSimpleSender(addr string, interval time.Duration) (Sender, error) {
	return nil, errors.New("gomad: StatsD UDP sender is unsupported; supply a Sender")
}
`)},
			{anchor: []byte(`func (s *ResolvingSimpleSender) Reconnect() {
	// Note: use manual unlocking instead of defer unlocking.
	// This is done here because we use a read lock first,
	// read a value safely, then perform an action that doesn't require
	// locking, then acquire a write lock for safe updating.

	// lock to guard against s.running mutation
	s.mx.RLock()

	if !s.running {
		s.mx.RUnlock()
		return
	}

	// get old addr for comparison, then release lock (asap)
	oldAddr := s.addrResolved.String()

	// done with rlock for now
	s.mx.RUnlock()

	// s.addrUnresolved doesn't change, so no do this under read lock
	addrResolved, err := net.ResolveUDPAddr("udp", s.addrUnresolved)

	if err != nil {
		// no good new address.. so continue with old address
		return
	}

	if oldAddr == addrResolved.String() {
		// got same address.. so continue with old address
		return
	}

	// acquire write lock to both guard against s.running having been mutated in the
	// meantime, as well as for safely setting s.ra
	s.mx.Lock()

	// check running again, just to be sure nothing was terminated in the meantime...
	if s.running {
		s.addrResolved = addrResolved
	}
	s.mx.Unlock()
}
`), replacement: []byte(`func (s *ResolvingSimpleSender) Reconnect() {
	// Note: use manual unlocking instead of defer unlocking.
	// This is done here because we use a read lock first,
	// read a value safely, then perform an action that doesn't require
	// locking, then acquire a write lock for safe updating.
	// lock to guard against s.running mutation
	// get old addr for comparison, then release lock (asap)
	// done with rlock for now
	// s.addrUnresolved doesn't change, so no do this under read lock
	// no good new address.. so continue with old address
	// got same address.. so continue with old address
	// acquire write lock to both guard against s.running having been mutated in the
	// meantime, as well as for safely setting s.ra
	// check running again, just to be sure nothing was terminated in the meantime...
	panic("gomad: StatsD UDP sender is unsupported; supply a Sender")
}
`)},
		},
	},
}

var cactusStatsDAdapter = rewrittenModule{
	module: cactusStatsDModulePath, version: cactusStatsDVersion, sum: cactusStatsDSum,
	cacheElements:                 []string{"github.com", "cactus", "go-statsd-client", "v5@" + cactusStatsDVersion},
	replacementDirectory:          "cactus-statsd",
	originalInventorySHA256:       cactusStatsDOriginalSourceInventorySHA256,
	replacementInventorySHA256:    cactusStatsDReplacementSourceInventorySHA256,
	preparedPackage:               cactusStatsDModulePath + "/statsd",
	preparedSourceSetSHA256ByHost: cactusStatsDPreparedSourceSetSHA256ByHost,
	rewrites:                      cactusStatsDRewrites,
}

func prepareCactusStatsD(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, cactusStatsDAdapter)
}
