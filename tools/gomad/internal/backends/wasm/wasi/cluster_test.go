package wasi

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClusterFinishPreservesSessionWithLiveIncarnations(t *testing.T) {
	session := &ClusterSession{states: map[BrokerNode]*clusterState{{ID: "a", Incarnation: 1}: {done: make(chan struct{})}}}
	if _, err := session.Finish(context.Background()); err == nil {
		t.Fatal("finish accepted a live incarnation")
	}
	if session.finishing {
		t.Fatal("refused finish prevented subsequent lifecycle control")
	}
}

func TestClusterPreservesWholeRunInfrastructureFailure(t *testing.T) {
	stream := &clusterStream{session: &ClusterSession{}}
	err := stream.frame([]byte(`{"type":"cluster_result","termination":"infrastructure","message":"helper panic","instances":[],"virtual_elapsed_nanos":0}`))
	var boundary *BoundaryError
	if !errors.As(err, &boundary) || boundary.Kind != "infrastructure" {
		t.Fatalf("helper failure lost infrastructure classification: %v", err)
	}
}

func TestClusterRejectsGuestCallbacks(t *testing.T) {
	for _, frame := range []string{
		`{"instance":"a","type":"call","id":1,"op":"runtime_idle","input":{}}`,
		`{"instance":"a","type":"sync","id":1}`,
	} {
		session := &ClusterSession{states: map[BrokerNode]*clusterState{}}
		stream := &clusterStream{session: session}
		if err := stream.frame([]byte(frame)); err == nil {
			t.Fatal("cluster accepted a guest callback to Go")
		}
		if session.FrameCounts().GuestCallbacks != 1 {
			t.Fatal("guest callback refusal was absent from the transport counter")
		}
	}
}

func TestClusterRejectsResultWithoutModelEvidence(t *testing.T) {
	config := testConfig()
	e, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	state := &clusterState{sink: &frameSink{request: Request{Config: config, Fuel: 100, MemoryBytes: 65536}, environment: e, started: &Started{}, authorizedAt: time.Now()}}
	frame, err := newValidatedFrame([]byte(`{"type":"result","termination":"capacity","message":"memory limit","fuel_remaining":100,"peak_memory_bytes":0,"output_bytes":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := state.collect(frame, BrokerConfig{}); err == nil {
		t.Fatal("accepted an incarnation result with truncated model evidence")
	}
}

func TestClusterRejectsUnattributableFrames(t *testing.T) {
	for _, frame := range []string{
		`{"instance":"unknown","type":"result"}`,
		`{"instance":"a","incarnation":0,"type":"evidence"}`,
		`{"instance":"a","instance":"b","type":"result"}`,
		`{"type":"evidence","stream":"world","data_base64":"!"}`,
	} {
		stream := &clusterStream{session: &ClusterSession{states: map[BrokerNode]*clusterState{}}}
		if err := stream.frame([]byte(frame)); err == nil {
			t.Fatalf("accepted corrupt cluster frame %s", frame)
		}
	}
}
