package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio/volume"
	netmodel "go.temporal.io/server/tools/gomad3/simulation/network"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

func TestTemporalHostDiagnosticInputs(t *testing.T) {
	for _, invalid := range []string{"not-a-duration", "0s", "-1s", "16m"} {
		if _, err := temporalPairBudget(invalid); err == nil {
			t.Fatalf("accepted invalid diagnostic budget %q", invalid)
		}
	}
	for _, value := range []struct {
		input string
		want  time.Duration
	}{{"", 15 * time.Minute}, {"3m", 3 * time.Minute}} {
		actual, err := temporalPairBudget(value.input)
		if err != nil || actual != value.want {
			t.Fatalf("budget %q: got %v, %v", value.input, actual, err)
		}
	}
	root := t.TempDir()
	if actual, err := temporalCompileCachePath(root, ""); err != nil || actual != filepath.Join(root, "compile-cache") {
		t.Fatalf("default cache changed: %s, %v", actual, err)
	}
	for _, invalid := range []string{"relative/cache", string(filepath.Separator), root + "/../"} {
		if _, err := temporalCompileCachePath(root, invalid); err == nil {
			t.Fatalf("accepted unsafe cache path %q", invalid)
		}
	}
	cache := filepath.Join(root, "owned-cache")
	if actual, err := temporalCompileCachePath(root, cache); err != nil || actual != cache {
		t.Fatalf("explicit cache changed: %s, %v", actual, err)
	}
}

func TestTemporalBrokerBoundedWorldAdmission(t *testing.T) {
	config := temporalBrokerConfiguration(&NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Deliveries: 32768, Bytes: 128 << 20, Transitions: 1 << 20}})
	if config.WorldProfile != world.CompressedSnapshotSchema || config.Limits != world.CompressedSnapshotProfile().Limits {
		t.Fatal("Temporal producer lost its explicit bounded World profile")
	}
	if _, err := NewBroker(config); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BrokerConfig){
		func(c *BrokerConfig) { c.Limits.MaxRequests++ },
		func(c *BrokerConfig) { c.Limits.MaxEvents++ },
		func(c *BrokerConfig) { c.Limits.MaxQueuedEvents++ },
		func(c *BrokerConfig) { c.Limits.MaxTransitions++ },
		func(c *BrokerConfig) { c.Limits.MaxPayloadBytes++ },
		func(c *BrokerConfig) { c.Limits.MaxStringBytes++ },
		func(c *BrokerConfig) { c.Limits.MaxRequests = 0 },
		func(c *BrokerConfig) { c.WorldProfile = "unknown" },
	} {
		invalid := config
		mutate(&invalid)
		if _, err := NewBroker(invalid); err == nil {
			t.Fatal("Temporal producer admitted an oversized, implicit or unknown World profile")
		}
	}
}

func TestTemporalBrokerReadinessSource(t *testing.T) {
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if !filepath.IsAbs(compiler) {
		t.Skip("set absolute GOMAD3_STOCK_GO for the ordinary fixture-source readiness unit test")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	var body []byte
	for _, name := range []string{"temporal_storage/recovery_test.go", "temporal_broker/replication_test.go"} {
		data, err := os.ReadFile(filepath.Join(root, "tools/gomad_wasm/testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		body = append(body, data...)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "temporal_broker-"+digest(body)[:16])
	temporalStageBrokerFixture(t, root, staged)
	environment := append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0", "GOENV=off", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOROOT="+filepath.Dir(filepath.Dir(compiler)))
	temporalCommand(t, t.TempDir(), "readiness-source", root, environment, []string{compiler, "test", "-mod=readonly", "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "-run", "^TestReplicationReadinessRequiresRemoteAcknowledgment$", "-count=1", "./" + filepath.ToSlash(staged)})
}

func temporalPairBudget(value string) (time.Duration, error) {
	if value == "" {
		return 15 * time.Minute, nil
	}
	budget, err := time.ParseDuration(value)
	if err != nil || budget <= 0 || budget > 15*time.Minute {
		return 0, fmt.Errorf("diagnostic budget must be positive and at most 15m: %q", value)
	}
	return budget, nil
}

func temporalStageBrokerFixture(t *testing.T, root, staged string) {
	t.Helper()
	destination := filepath.Join(root, staged)
	if err := os.MkdirAll(destination, 0700); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{
		"recovery_test.go":    filepath.Join(root, "tools/gomad_wasm/testdata/temporal_storage/recovery_test.go"),
		"replication_test.go": filepath.Join(root, "tools/gomad_wasm/testdata/temporal_broker/replication_test.go"),
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if sources[entry.Name()] == "" || !entry.Type().IsRegular() {
			t.Fatalf("unexpected staged broker fixture %q", entry.Name())
		}
	}
	for name, source := range sources {
		expected, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(destination, name)
		actual, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			if err := os.WriteFile(path, expected, 0600); err != nil {
				t.Fatal(err)
			}
		} else if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("staged broker source body differs from %s", source)
		}
	}
}

func temporalBrokerConfiguration(network *NetworkModelV2) BrokerConfig {
	return BrokerConfig{
		Nodes: []BrokerNode{{ID: "active", Incarnation: 1}, {ID: "standby", Incarnation: 1}}, Seed: 7,
		Limits: world.CompressedSnapshotProfile().Limits, WorldProfile: world.CompressedSnapshotSchema,
		Network: network, Addresses: map[string]string{"active": "10.0.0.1", "standby": "10.0.0.2"}, Diagnostics: true,
		Faults: []BrokerNetworkFault{
			{Name: "temporal-active-to-standby-partition", Action: "partition", From: "active", To: "standby", AtNanos: 1},
			{Name: "temporal-active-to-standby-heal", Action: "heal", From: "active", To: "standby", AtNanos: uint64(30 * time.Second)},
		},
	}
}

func TestTemporalIndependentClustersPartitionHealWholeClusterCrash(t *testing.T) {
	request, evidence := temporalRecoveryRequest(t, "temporal_broker")
	request.Config.Args = []string{"replication.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	network := &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Deliveries: 32768, Bytes: 128 << 20, Transitions: 1 << 20}}
	request.Config.NetworkV2 = network
	brokerConfig := temporalBrokerConfiguration(network)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-broker-config.json"), brokerConfig)
	b, err := NewBroker(brokerConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		temporalWrite(t, filepath.Join(evidence, "two-cluster-broker-idle.jsonl"), b.Diagnostics())
		snapshot, err := b.NetworkSnapshot()
		if err != nil {
			t.Error(err)
			return
		}
		temporalJSON(t, filepath.Join(evidence, "two-cluster-network.json"), snapshot)
	})
	progressContext, stopProgress := context.WithCancel(t.Context())
	var progress sync.WaitGroup
	progress.Go(func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-progressContext.Done():
				return
			case <-ticker.C:
				// Host-only diagnostic observation never advances modeled time.
				if err := os.WriteFile(filepath.Join(evidence, "two-cluster-broker-live.jsonl"), b.Diagnostics(), 0600); err != nil {
					t.Error(err)
				}
			}
		}
	})
	t.Cleanup(func() { stopProgress(); progress.Wait() })
	names := []string{"active", "standby"}
	if err := b.CrashAfterOutputs(names, []byte(temporalObservationPrefix)); err != nil {
		t.Fatal(err)
	}
	requests := make([]Request, 2)
	commitRequests := make([]Request, 2)
	for i, name := range names {
		requests[i] = request
		storage := *request.Config.Storage
		storage.Volumes = append([]volume.VolumeConfig(nil), storage.Volumes...)
		storage.Volumes[0].ID = name
		requests[i].Config.Storage = &storage
		requests[i].Config.EntropyKey = sha256.Sum256([]byte("gomad-wasm-temporal-replication/v1/" + name))
		requests[i].Broker = b.Session(name)
		requests[i].Config.Environment = append(append([]string(nil), request.Config.Environment...), "GOMAD_TEMPORAL_CLUSTER="+name, "GOMAD_TEMPORAL_PHASE=commit", "GOMAD_TEMPORAL_REMOTE_FRONTEND="+b.Address(names[1-i])+":7233", "GOMAD_TEMPORAL_REMOTE_HTTP="+b.Address(names[1-i])+":7243", "GOMAD_TEMPORAL_REMOTE_HISTORY="+b.Address(names[1-i])+":7234")
		temporalRetainRequest(t, evidence, name+"-cluster-commit", requests[i])
		commitRequests[i] = requests[i]
		commitRequests[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
	}
	results, errors := temporalRunPair(t, evidence, "cluster-crash", requests, true)
	crashed := append([]Result(nil), results...)
	var expected []byte
	identities := make([]string, 2)
	original := make([]temporalRecoveryObservation, 2)
	converged := map[string][]byte{}
	for i, name := range names {
		if errors[i] != nil || results[i].Termination != "crash" || !results[i].Reaped || results[i].StorageImage == nil {
			t.Fatalf("independent %s crash: termination=%s message=%s reaped=%v image=%v err=%v stdout=%s stderr=%s", name, results[i].Termination, results[i].Message, results[i].Reaped, results[i].StorageImage != nil, errors[i], results[i].Stdout, results[i].Stderr)
		}
		value := temporalRecoveryValue(t, results[i].Stdout)
		original[i] = value
		converged[name] = value.History
		identities[i] = value.ClusterID
		if i == 0 {
			expected = value.History
		} else if !bytes.Equal(expected, value.History) {
			t.Fatal("independently observed histories did not converge after directional heal")
		}
		requests[i].StorageImage = results[i].StorageImage
		requests[i].StdoutMirror = nil
		requests[i].Config.Environment[len(request.Config.Environment)+1] = "GOMAD_TEMPORAL_PHASE=reopen"
		requests[i].Broker, err = b.Restart(name)
		if err != nil {
			t.Fatal(err)
		}
		temporalRetainRequest(t, evidence, name+"-cluster-reopen", requests[i])
	}
	if identities[0] == identities[1] {
		t.Fatal("independent Temporal clusters have the same identity")
	}
	convergence, err := gomad3sim.EventualConvergence("temporal.replication.healed", converged, 1<<20)
	if err != nil || !convergence.Passed {
		t.Fatalf("detached Temporal convergence oracle: passed=%v err=%v", convergence.Passed, err)
	}
	temporalJSON(t, filepath.Join(evidence, "temporal-replication-healed-oracle.json"), convergence)
	temporalJSON(t, filepath.Join(evidence, "temporal-replication-healed-oracle-inputs.json"), converged)
	results, errors = temporalRunPair(t, evidence, "cluster-reopened", requests, false)
	for i, name := range names {
		if errors[i] != nil || results[i].Termination != "exit" || results[i].ExitCode == nil || *results[i].ExitCode != 0 || !results[i].Reaped {
			t.Fatalf("independent %s reopen: termination=%s message=%s err=%v stdout=%s stderr=%s", name, results[i].Termination, results[i].Message, errors[i], results[i].Stdout, results[i].Stderr)
		}
		value := temporalRecoveryValue(t, results[i].Stdout)
		temporalHistoryOracles(t, evidence, "temporal.replication."+name+"-recovered", original[i], value)
		if !reflect.DeepEqual(expected, value.History) || value.ClusterID != identities[i] {
			t.Fatalf("committed %s history changed after whole-cluster crash", name)
		}
	}
	encodedWorld, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, filepath.Join(evidence, "two-cluster-world.json"), encodedWorld)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-world-cardinality.json"), b.DiagnosticCardinality())
	temporalReplayClusters(t, evidence, brokerConfig, commitRequests, crashed, results, encodedWorld)
	fmt.Printf("independent Temporal clusters recovered after directional partition/heal\n")
}

func temporalRunPair(t *testing.T, evidence, label string, requests []Request, warmFirst bool) ([]Result, []error) {
	t.Helper()
	budget, err := temporalPairBudget(os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET"))
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(t.Context(), budget)
	defer cancel()
	results, errors := make([]Result, len(requests)), make([]error, len(requests))
	warmed, firstFinished := make(chan struct{}), make(chan struct{})
	var group sync.WaitGroup
	for i := range requests {
		observer := &temporalGuestWarm{test: t, node: requests[i].Broker.Node.ID}
		if i == 0 && warmFirst {
			observer.ready = warmed
		}
		requests[i].StdoutMirror = observer
		if i == 1 && warmFirst {
			select {
			case <-warmed:
			case <-firstFinished:
			case <-ctx.Done():
				// The bounded host budget is not an application terminal.
				cancel()
			}
		}
		group.Go(func() {
			results[i], errors[i] = Run(ctx, requests[i])
			name := requests[i].Broker.Node.ID
			temporalRetainRecoveryRun(t, evidence, name+"-"+label, results[i])
			t.Logf("%s %s terminated: %s %s", name, label, results[i].Termination, results[i].Message)
			if i == 0 {
				close(firstFinished)
			}
			if errors[i] != nil || results[i].Termination != "exit" && results[i].Termination != "crash" ||
				results[i].Termination == "exit" && (results[i].ExitCode == nil || *results[i].ExitCode != 0) {
				cancel()
			}
		})
	}
	group.Wait()
	temporalJSON(t, filepath.Join(evidence, label+"-host-budget.json"), struct {
		BudgetNanos, ElapsedNanos int64
		Expired                   bool
	}{int64(budget), time.Since(started).Nanoseconds(), ctx.Err() == context.DeadlineExceeded})
	return results, errors
}

func temporalReplayClusters(t *testing.T, evidence string, configuration BrokerConfig, requests []Request, crashed, recovered []Result, expected []byte) {
	t.Helper()
	b, err := NewBroker(configuration)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"active", "standby"}
	if err := b.CrashAfterOutputs(names, []byte(temporalObservationPrefix)); err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		if !crashed[i].RuntimeInterrupted {
			t.Fatalf("%s crash lost its interrupted choice evidence", name)
		}
		tape, err := choice.ProjectInterruptedReplayPlan(crashed[i].InterruptedChoiceBytes, requests[i].Runtime.Choice.Limit, requests[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *requests[i].Runtime, *requests[i].Runtime.Choice
		choices.Mode, choices.Tape = choice.ModePrefix, &tape
		control.Choice, requests[i].Runtime = &choices, &control
		requests[i].Broker, requests[i].StdoutMirror = b.Session(name), nil
		requests[i].Config.Environment = append([]string(nil), requests[i].Config.Environment...)
	}
	actual, errors := temporalRunPair(t, evidence, "cluster-prefix-replay", requests, true)
	for i, name := range names {
		if errors[i] != nil || actual[i].Termination != "crash" || !actual[i].Reaped || actual[i].StorageImage == nil ||
			!bytes.Equal(crashed[i].Stdout, actual[i].Stdout) || !bytes.Equal(crashed[i].StorageTranscript, actual[i].StorageTranscript) ||
			!bytes.Equal(crashed[i].NetworkTranscript, actual[i].NetworkTranscript) || !bytes.Equal(crashed[i].InterruptedChoiceBytes, actual[i].InterruptedChoiceBytes) {
			t.Fatalf("%s interrupted cluster replay changed evidence: termination=%s message=%s err=%v", name, actual[i].Termination, actual[i].Message, errors[i])
		}
		tape, err := choice.ProjectReplayPlan(recovered[i].ChoiceTrace, requests[i].Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *requests[i].Runtime, *requests[i].Runtime.Choice
		choices.Mode, choices.Tape = choice.ModeReplay, &tape
		control.Choice, requests[i].Runtime = &choices, &control
		requests[i].StorageImage, requests[i].StdoutMirror = actual[i].StorageImage, nil
		for j, value := range requests[i].Config.Environment {
			if value == "GOMAD_TEMPORAL_PHASE=commit" {
				requests[i].Config.Environment[j] = "GOMAD_TEMPORAL_PHASE=reopen"
			}
		}
		requests[i].Broker, err = b.Restart(name)
		if err != nil {
			t.Fatal(err)
		}
	}
	actual, errors = temporalRunPair(t, evidence, "cluster-full-replay", requests, false)
	for i, name := range names {
		if errors[i] != nil || actual[i].Termination != "exit" || actual[i].ExitCode == nil || *actual[i].ExitCode != 0 || !actual[i].Reaped ||
			!bytes.Equal(recovered[i].Stdout, actual[i].Stdout) || !bytes.Equal(recovered[i].StorageTranscript, actual[i].StorageTranscript) ||
			!bytes.Equal(recovered[i].NetworkTranscript, actual[i].NetworkTranscript) || !bytes.Equal(recovered[i].ChoiceTrace.Bytes, actual[i].ChoiceTrace.Bytes) {
			t.Fatalf("%s full cluster replay changed evidence: termination=%s message=%s err=%v", name, actual[i].Termination, actual[i].Message, errors[i])
		}
		temporalHistoryOracles(t, evidence, "temporal.replication."+name+"-replayed", temporalRecoveryValue(t, recovered[i].Stdout), temporalRecoveryValue(t, actual[i].Stdout))
	}
	encodedWorld, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, encodedWorld) {
		t.Fatal("forced independent cluster replay changed shared World delivery order")
	}
	temporalWrite(t, filepath.Join(evidence, "two-cluster-replayed-world.json"), encodedWorld)
	temporalJSON(t, filepath.Join(evidence, "two-cluster-replayed-world-cardinality.json"), b.DiagnosticCardinality())
}

type temporalGuestWarm struct {
	ready chan struct{}
	once  sync.Once
	ack   sync.Once
	test  *testing.T
	node  string
}

func (w *temporalGuestWarm) Write(data []byte) (int, error) {
	if w.ready != nil {
		w.once.Do(func() { close(w.ready) })
	}
	if bytes.Contains(data, []byte(temporalObservationPrefix)) {
		w.ack.Do(func() { w.test.Logf("%s acknowledged committed Temporal history", w.node) })
	}
	return len(data), nil
}
