package cactusstatsd_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
)

func TestDefaultSenderRefusesUDP(t *testing.T) {
	sender, err := statsd.NewSimpleSender("127.0.0.1:8125")
	if sender != nil {
		t.Cleanup(func() {
			if err := sender.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	if sender != nil || err == nil || !strings.Contains(err.Error(), "gomad: StatsD UDP sender is unsupported") {
		t.Fatalf("default UDP sender = %T, %v; want explicit refusal before host resources", sender, err)
	}
}

func TestResolvingSenderRefusesUDP(t *testing.T) {
	sender, err := statsd.NewResolvingSimpleSender("127.0.0.1:8125", time.Hour)
	if sender != nil {
		t.Cleanup(func() {
			if err := sender.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	if sender != nil || err == nil || !strings.Contains(err.Error(), "gomad: StatsD UDP sender is unsupported") {
		t.Fatalf("resolving UDP sender = %T, %v; want explicit refusal before host resources", sender, err)
	}
}

func TestReconnectRefusesUDP(t *testing.T) {
	defer func() {
		failure := recover()
		if message, ok := failure.(string); !ok || !strings.Contains(message, "gomad: StatsD UDP sender is unsupported") {
			t.Fatalf("Reconnect panic = %v; want explicit unsupported service", failure)
		}
	}()
	new(statsd.ResolvingSimpleSender).Reconnect()
}

type recordingSender struct {
	messages []string
	closed   bool
}

func (s *recordingSender) Send(data []byte) (int, error) {
	s.messages = append(s.messages, string(data))
	return len(data), nil
}

func (s *recordingSender) Close() error {
	s.closed = true
	return nil
}

func TestSuppliedSenderEmitsMetrics(t *testing.T) {
	sender := new(recordingSender)
	client, err := statsd.NewClientWithSender(sender, "cell", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Inc("requests", 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := client.Gauge("queued", 5, 1); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"cell.requests:3|c", "cell.queued:5|g"}; !reflect.DeepEqual(sender.messages, want) || !sender.closed {
		t.Fatalf("supplied Sender messages = %#v, closed = %v; want %#v, true", sender.messages, sender.closed, want)
	}
}
