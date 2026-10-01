package deterministicio

import gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"

const (
	memberlistModulePath                       = "github.com/hashicorp/memberlist"
	memberlistVersion                          = "v0.5.4"
	memberlistSum                              = "h1:40YY+3qq2tAUhZIMEK8kqusKZBBjdwJ3NUjvYkcxh74="
	memberlistOriginalSourceInventorySHA256    = "sha256:f33d5f03cc17648417c6100b58f46e7ca7376d8120080616f0fa17ba4356d23b"
	memberlistReplacementSourceInventorySHA256 = "sha256:7cb0e5713d95a57ecb5803cc4205f917b8c6ad972ba5c52cef4ddd6d95643eec"
)

var memberlistPreparedSourceSetSHA256 = hostPin(map[string]string{
	"darwin/arm64": "sha256:d1b8958261401506ac8134d4ceec0d937d19d971046c5184d33493f4743ed7aa",
	"linux/amd64":  "sha256:d1b8958261401506ac8134d4ceec0d937d19d971046c5184d33493f4743ed7aa",
})

// memberlistRewrites refuse the native UDP transport before resource creation
// while leaving caller-supplied Transport implementations intact.
var memberlistRewrites = []sourceRewrite{
	{
		path: "net_transport.go", sourceSHA256: "sha256:c49122ee6355d9b94ea705a56bf8653c2fe845ff269a29c6b27c0df1c2115dc5", replacementSHA256: "sha256:c61e112c5847cfbb6c25114abbf9fc2ec20f15e1e52b435050ec0eba88edc53c",
		rewrites: []anchorRewrite{
			{anchor: []byte(`func NewNetTransport(config *NetTransportConfig) (*NetTransport, error) {
	// If we reject the empty list outright we can assume that there's at
	// least one listener of each type later during operation.
	if len(config.BindAddrs) == 0 {
		return nil, fmt.Errorf("at least one bind address is required")
	}

	// Build out the new transport.
	var ok bool
	t := NetTransport{
		config:       config,
		packetCh:     make(chan *Packet),
		streamCh:     make(chan net.Conn),
		logger:       config.Logger,
		metricLabels: config.MetricLabels,
	}

	// Clean up listeners if there's an error.
	defer func() {
		if !ok {
			_ = t.Shutdown()
		}
	}()

	// Build all the TCP and UDP listeners.
	port := config.BindPort
	for _, addr := range config.BindAddrs {
		ip := net.ParseIP(addr)

		tcpAddr := &net.TCPAddr{IP: ip, Port: port}
		tcpLn, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to start TCP listener on %q port %d: %v", addr, port, err)
		}
		t.tcpListeners = append(t.tcpListeners, tcpLn)

		// If the config port given was zero, use the first TCP listener
		// to pick an available port and then apply that to everything
		// else.
		if port == 0 {
			port = tcpLn.Addr().(*net.TCPAddr).Port
		}

		udpAddr := &net.UDPAddr{IP: ip, Port: port}
		udpLn, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to start UDP listener on %q port %d: %v", addr, port, err)
		}
		if err := setUDPRecvBuf(udpLn); err != nil {
			return nil, fmt.Errorf("failed to resize UDP buffer: %v", err)
		}
		t.udpListeners = append(t.udpListeners, udpLn)
	}

	// Fire them up now that we've been able to create them all.
	for i := 0; i < len(config.BindAddrs); i++ {
		t.wg.Add(2)
		go t.tcpListen(t.tcpListeners[i])
		go t.udpListen(t.udpListeners[i])
	}

	ok = true
	return &t, nil
}
`), replacement: []byte(`func NewNetTransport(config *NetTransportConfig) (*NetTransport, error) {
	// If we reject the empty list outright we can assume that there's at
	// least one listener of each type later during operation.
	// Build out the new transport.
	// Clean up listeners if there's an error.
	// Build all the TCP and UDP listeners.
	// If the config port given was zero, use the first TCP listener
	// to pick an available port and then apply that to everything
	// else.
	// Fire them up now that we've been able to create them all.
	return nil, fmt.Errorf("gomad: memberlist UDP transport is unsupported; supply a Transport")
}
`)},
			{anchor: []byte(`func (t *NetTransport) WriteToAddress(b []byte, a Address) (time.Time, error) {
	addr := a.Addr

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return time.Time{}, err
	}

	// We made sure there's at least one UDP listener, so just use the
	// packet sending interface on the first one. Take the time after the
	// write call comes back, which will underestimate the time a little,
	// but help account for any delays before the write occurs.
	_, err = t.udpListeners[0].WriteTo(b, udpAddr)
	return time.Now(), err
}
`), replacement: []byte(`func (t *NetTransport) WriteToAddress(b []byte, a Address) (time.Time, error) {
	// We made sure there's at least one UDP listener, so just use the
	// packet sending interface on the first one. Take the time after the
	// write call comes back, which will underestimate the time a little,
	// but help account for any delays before the write occurs.
	return time.Time{}, fmt.Errorf("gomad: memberlist UDP transport is unsupported; supply a Transport")
}
`)},
		},
	},
}

func prepareMemberlist(moduleCache, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
	return prepareRewrittenModule(moduleCache, root, identity, rewrittenModule{
		module: memberlistModulePath, version: memberlistVersion, sum: memberlistSum,
		cacheElements:              []string{"github.com", "hashicorp", "memberlist@" + memberlistVersion},
		replacementDirectory:       "memberlist",
		originalInventorySHA256:    memberlistOriginalSourceInventorySHA256,
		replacementInventorySHA256: memberlistReplacementSourceInventorySHA256,
		preparedPackage:            memberlistModulePath,
		preparedSourceSetSHA256:    memberlistPreparedSourceSetSHA256,
		rewrites:                   memberlistRewrites,
	})
}
