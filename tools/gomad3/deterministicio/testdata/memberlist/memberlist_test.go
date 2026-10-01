package memberlist_test

import (
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
)

func TestDefaultTransportRefusesUDP(t *testing.T) {
	transport, err := memberlist.NewNetTransport(&memberlist.NetTransportConfig{
		BindAddrs: []string{"127.0.0.1"}, BindPort: 0, Logger: log.New(io.Discard, "", 0),
	})
	if transport != nil {
		t.Cleanup(func() {
			if err := transport.Shutdown(); err != nil {
				t.Error(err)
			}
		})
	}
	if transport != nil || err == nil || !strings.Contains(err.Error(), "gomad: memberlist UDP transport is unsupported") {
		t.Fatalf("default UDP transport = %T, %v; want explicit refusal before host resources", transport, err)
	}
}

func TestDefaultMemberlistRefusesUDP(t *testing.T) {
	config := memberlist.DefaultLocalConfig()
	config.Name = "udp-refusal"
	config.BindAddr = "127.0.0.1"
	config.BindPort = 0
	config.LogOutput = io.Discard
	cluster, err := memberlist.Create(config)
	if cluster != nil {
		t.Cleanup(func() {
			if err := cluster.Shutdown(); err != nil {
				t.Error(err)
			}
		})
	}
	if cluster != nil || err == nil || !strings.Contains(err.Error(), "gomad: memberlist UDP transport is unsupported") {
		t.Fatalf("default memberlist = %T, %v; want supplied Transport requirement", cluster, err)
	}
}

func TestNativeTransportPacketWritesRefuseUDP(t *testing.T) {
	transport := new(memberlist.NetTransport)
	for _, write := range []func() (time.Time, error){
		func() (time.Time, error) { return transport.WriteTo([]byte("packet"), "127.0.0.1:8125") },
		func() (time.Time, error) {
			return transport.WriteToAddress([]byte("packet"), memberlist.Address{Addr: "127.0.0.1:8125", Name: "peer"})
		},
	} {
		stamp, err := write()
		if !stamp.IsZero() || err == nil || !strings.Contains(err.Error(), "gomad: memberlist UDP transport is unsupported") {
			t.Fatalf("native packet write = %v, %v; want explicit refusal before UDP resolution", stamp, err)
		}
	}
}
