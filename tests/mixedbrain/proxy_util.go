package mixedbrain

import (
	"io"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// frontendProxy is a TCP proxy that distributes connections randomly across
// multiple frontend backends, ensuring omes exercises both server versions.
type frontendProxy struct {
	listener  net.Listener
	backends  []string
	connCount []atomic.Int64
	wg        sync.WaitGroup
}

func startFrontendProxy(t *testing.T, backends ...string) *frontendProxy {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	p := &frontendProxy{
		listener:  listener,
		backends:  backends,
		connCount: make([]atomic.Int64, len(backends)),
	}
	go p.serve()
	return p
}

func (p *frontendProxy) addr() string {
	return p.listener.Addr().String()
}

func (p *frontendProxy) stop() {
	_ = p.listener.Close()
	p.wg.Wait()
}

func (p *frontendProxy) serve() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			return
		}
		idx := rand.IntN(len(p.backends))
		p.connCount[idx].Add(1)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.proxyConn(conn, p.backends[idx])
		}()
	}
}

func (p *frontendProxy) proxyConn(client net.Conn, backend string) {
	server, err := net.DialTimeout("tcp", backend, 5*time.Second)
	if err != nil {
		_ = client.Close()
		return
	}
	go func() {
		_, _ = io.Copy(server, client)
		_ = server.Close()
	}()
	_, _ = io.Copy(client, server)
	_ = client.Close()
}
