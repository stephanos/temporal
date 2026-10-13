package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedNetworkIsCurrent(t *testing.T) {
	if err := run(true); err != nil {
		t.Fatal(err)
	}
}

func TestProjectionPreservesNativeOwnerBodies(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	generated, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "simulation/network/generated.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	functions := make(map[string]*ast.FuncDecl)
	for _, declaration := range generated.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			functions[functionKey(function)] = function
		}
	}
	adapted := map[string]bool{
		"simulationNetwork.listen": true, "simulationNetwork.dial": true,
		"simulationNetwork.accept": true, "simulationNetwork.revoke": true,
		".validateSimulationEndpoint": true,
		"simulationListener.Accept":   true, "simulationListener.Close": true,
		"pairedConn.Read": true, "pairedConn.Write": true, "pairedConn.Close": true,
		"pairedConn.CloseRead": true, "pairedConn.CloseWrite": true, "pairedConn.nowTime": true,
		"simulationNetwork.armStallTimerLocked": true,
		"simulationNetwork.armDialTimerLocked":  true,
	}
	compared := 0
	for _, name := range []string{"network.go", "network_connection.go", "simulation_network.go", "simulation_handles.go", "simulation_wire.go"} {
		native, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "backends/native/toolchain/runtime/overlay/src/internal/gomadio", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range native.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			key := functionKey(function)
			projection := functions[key]
			if projection == nil || adapted[key] {
				continue
			}
			var want, got bytes.Buffer
			if err := format.Node(&want, token.NewFileSet(), function); err != nil {
				t.Fatal(err)
			}
			if err := format.Node(&got, token.NewFileSet(), projection); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(want.Bytes(), got.Bytes()) {
				t.Errorf("native owner body changed in projection: %s", key)
			}
			compared++
		}
	}
	if compared < 60 {
		t.Fatalf("native preservation coverage shrank: only %d owner bodies compared", compared)
	}
}

func TestProjectionRejectsWaitBeforeUnlock(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "drift.go", `package network
func (network *simulationNetwork) accept() {
	waitForChange(changed, deadline)
	network.Unlock()
}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapt(file); err == nil {
		t.Fatal("generation accepted a wait before the native unlock")
	}
}

func TestProjectionRejectsChangedExpiryTimer(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(root, "backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []struct{ name, from, to string }{
		{"scheduler", "time.AfterFunc(", "time.NewTimer("},
		{"deadline", "time.Until(due)", "time.Duration(1)"},
		{"epoch capture", "state.stallEpoch, state.stallDue", "state.stallEpoch + 1, state.stallDue"},
		{"owned callback", "network.expireStallLocked(pair, state, epoch, due)", "network.expireStallLocked(pair, state, epoch + 1, due)"},
		{"extra callback mutation", "err := network.expireStallLocked", "network.pendingBytes++\nerr := network.expireStallLocked"},
		{"dial callback", "network.expireDialLocked(pending, epoch, due)", "network.expireDialLocked(pending, epoch, due.Add(1))"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			modified := strings.Replace(string(source), mutation.from, mutation.to, 1)
			if modified == string(source) {
				t.Fatal("timer mutation did not change fixture")
			}
			file, err := parser.ParseFile(token.NewFileSet(), "timer.go", modified, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := adapt(file); err == nil {
				t.Fatal("projection silently removed changed native timer semantics")
			}
		})
	}
}

func functionKey(function *ast.FuncDecl) string {
	return receiverType(function) + "." + function.Name.Name
}
