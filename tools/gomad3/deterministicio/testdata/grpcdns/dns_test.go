package dns

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/grpc/resolver"
)

func TestGomadDNSRefusesHostResolver(t *testing.T) {
	for _, authority := range []string{"", "127.0.0.1:53"} {
		network, err := newNetResolver(authority)
		if err == nil || !strings.Contains(err.Error(), "gomad: DNS resolution is unavailable") || network != nil {
			t.Fatalf("host resolver %q = %v, %v", authority, network, err)
		}
	}
}

func TestGomadDNSLiteralTarget(t *testing.T) {
	for _, address := range []string{"127.0.0.1:1234", "[::1]:1234"} {
		uri, err := url.Parse("dns:///" + address)
		if err != nil {
			t.Fatal(err)
		}
		cc := &gomadDNSClientConn{}
		connection, err := NewBuilder().Build(resolver.Target{URL: *uri}, cc, resolver.BuildOptions{})
		if err != nil {
			t.Fatal(err)
		}
		connection.Close()
		addresses := []resolver.Address{{Addr: address}}
		want := resolver.State{Addresses: addresses, Endpoints: []resolver.Endpoint{{Addresses: addresses}}}
		if !reflect.DeepEqual(cc.state, want) {
			t.Fatalf("literal target state = %#v, want %#v", cc.state, want)
		}
	}
}

func TestGomadDNSHostnameTarget(t *testing.T) {
	uri, err := url.Parse("dns:///localhost:1234")
	if err != nil {
		t.Fatal(err)
	}
	connection, err := NewBuilder().Build(resolver.Target{URL: *uri}, &gomadDNSClientConn{}, resolver.BuildOptions{})
	if connection != nil {
		connection.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "gomad: DNS resolution is unavailable") || connection != nil {
		t.Fatalf("hostname target = %v, %v", connection, err)
	}
}

type gomadDNSClientConn struct {
	resolver.ClientConn
	state resolver.State
}

func (cc *gomadDNSClientConn) UpdateState(state resolver.State) error {
	cc.state = state
	return nil
}
