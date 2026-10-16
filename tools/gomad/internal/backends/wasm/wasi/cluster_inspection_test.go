package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestClusterInspectionMetadataDoesNotReplaceWorldEvidence(t *testing.T) {
	session := &ClusterSession{inspectionWire: []byte(`{"schema":"retained"}`)}
	if err := session.finalizeEvidence(); err == nil {
		t.Fatal("ordinary execution accepted missing World evidence")
	}
}

func TestClusterInspectedPrefixRequiresRequestedSealedInstance(t *testing.T) {
	for _, requested := range []bool{false, true} {
		t.Run(map[bool]string{false: "unsolicited", true: "requested"}[requested], func(t *testing.T) {
			done := make(chan struct{})
			close(done)
			session := &ClusterSession{
				states: map[BrokerNode]*clusterState{{ID: "a", Incarnation: 1}: {
					done: done, result: Result{Termination: "inspected_prefix"}, inspection: requested,
				}},
				inspectionWire: []byte(`{"schema":"retained"}`),
				authorizedAt:   time.Now(),
			}
			stream := &clusterStream{session: session}
			err := stream.frame([]byte(`{"type":"cluster_result","termination":"inspected_prefix","message":"","instances":[{"id":"a","termination":"inspected_prefix"}],"virtual_elapsed_nanos":7}`))
			if !requested {
				if err == nil {
					t.Fatal("accepted unsolicited inspection terminal")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := session.finalizeEvidence(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(session.result.InspectionMetadata, session.inspectionWire) || session.result.VirtualElapsedNanos != 7 || len(session.result.World) != 0 {
				t.Fatalf("inspection metadata was lost or fabricated final evidence: %+v", session.result)
			}
		})
	}
}

func TestClusterInspectionAbortsNeighborsWithoutCompletingTheirTapes(t *testing.T) {
	for _, requested := range []bool{false, true} {
		config := testConfig()
		environment, err := NewEnvironment(config)
		if err != nil {
			t.Fatal(err)
		}
		neighbor := &clusterState{done: make(chan struct{}), sink: &frameSink{
			request: Request{Config: config, Fuel: 100, MemoryBytes: 65536}, environment: environment,
		}}
		session := &ClusterSession{states: map[BrokerNode]*clusterState{
			{ID: "a", Incarnation: 1}: {inspection: requested},
			{ID: "b", Incarnation: 1}: neighbor,
		}}
		stream := &clusterStream{session: session}
		err = stream.frame([]byte(`{"instance":"b","type":"result","termination":"inspection_aborted","message":"","fuel_remaining":100,"peak_memory_bytes":0,"output_bytes":0}`))
		if !requested {
			if err == nil {
				t.Fatal("ordinary execution accepted an inspection abort")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-neighbor.done:
			if neighbor.result.Termination != "inspection_aborted" || !neighbor.result.Reaped || len(neighbor.result.ChoiceTrace.Bytes) != 0 {
				t.Fatal("aborted neighbor reported a completed replay")
			}
		default:
			t.Fatal("aborted neighbor was not released")
		}
	}
}

func TestClusterFinishWaitsForInspectionCleanup(t *testing.T) {
	session, controls := clusterControlSession(t)
	selectedDone := make(chan struct{})
	close(selectedDone)
	session.states = map[BrokerNode]*clusterState{
		{ID: "a", Incarnation: 1}: {done: selectedDone, inspection: true, result: Result{Termination: "inspected_prefix"}},
		{ID: "b", Incarnation: 1}: {done: make(chan struct{})},
	}
	finished := make(chan error, 1)
	go func() {
		result, err := session.Finish(context.Background())
		if err == nil && (!result.GroupGone || result.Termination != "inspected_prefix") {
			t.Error("inspection cleanup lost terminal or reaping")
		}
		finished <- err
	}()
	if control := nextClusterControl(t, controls); control.Type != "finish" {
		t.Fatalf("inspection cleanup failed to request helper finish: %+v", control)
	}
	session.mu.Lock()
	session.result = ClusterResult{Termination: "inspected_prefix", GroupGone: true}
	close(session.done)
	session.mu.Unlock()
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}

func TestHelperRejectsUnsolicitedInspectionTerminal(t *testing.T) {
	sink := &frameSink{terminal: &resultFrame{LocalModel: &helperModelResult{InspectionSealed: true}}}
	if err := sink.collectHelper(&Result{Termination: "inspected_prefix"}); err == nil {
		t.Fatal("accepted inspected prefix without a runtime request")
	}
}

func TestHelperPreservesInspectionFailuresWithoutFullTapeFinish(t *testing.T) {
	for _, termination := range []string{"instrumentation_divergence", "debug_error", "probe_fuel_limit"} {
		t.Run(termination, func(t *testing.T) {
			sink := &frameSink{
				request:        Request{Runtime: &RuntimeControl{Inspection: json.RawMessage(`{}`)}},
				terminal:       &resultFrame{LocalModel: &helperModelResult{Now: 7}},
				helperEvidence: helperEvidence{choice: []byte{1}, transcript: []byte{2}},
			}
			result := Result{Termination: termination}
			if err := sink.collectHelper(&result); err != nil {
				t.Fatal(err)
			}
			if result.Termination != termination || !bytes.Equal(result.PartialHelperChoice, []byte{1}) || !bytes.Equal(result.PartialHelperTranscript, []byte{2}) {
				t.Fatal("inspection failure lost its diagnostic prefix")
			}
		})
	}
}
