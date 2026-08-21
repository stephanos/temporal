package livecap

import "testing"

func TestProjectFindingsUsesOnlyNonstandardLiveCapabilities(t *testing.T) {
	packages := []ClosurePackage{
		{ImportPath: "example.com/main", Root: true},
		{ImportPath: "example.com/dependency"},
		{ImportPath: "time", Standard: true},
	}
	findings := []ClosureFinding{
		{Kind: "forbidden_import", Package: "example.com/dependency", Capability: "import:syscall"},
		{Kind: "forbidden_import", Package: "example.com/main", Capability: "import:os/exec"},
		{Kind: "foreign_source", Package: "example.com/dependency", Capability: "foreign:assembly:bridge.s"},
	}
	manifest := Manifest{Facts: []Fact{
		{Kind: FactKindCapability, OwnerPackage: "time", OwnerSymbol: "time.initLocal", Capability: "import:syscall"},
		{Kind: FactKindCapability, OwnerPackage: "main", OwnerSymbol: "main.main", Capability: "import:os/exec"},
	}}
	projection := ProjectFindings(manifest, packages, findings)
	if projection.Active[0] || !projection.Active[1] || !projection.Active[2] {
		t.Fatalf("active = %v, want [false true true]", projection.Active)
	}
	if len(projection.Eliminated) != 1 || projection.Eliminated[0] != 0 {
		t.Fatalf("eliminated = %v, want [0]", projection.Eliminated)
	}
}

func TestProjectFindingsTreatsUnknownOwnersAsLive(t *testing.T) {
	projection := ProjectFindings(
		Manifest{Facts: []Fact{{Kind: FactKindCapability, OwnerPackage: "generated/owner", OwnerSymbol: "generated/owner.call", Capability: "import:syscall"}}},
		[]ClosurePackage{{ImportPath: "example.com/dependency"}},
		[]ClosureFinding{{Kind: "forbidden_import", Package: "example.com/dependency", Capability: "import:syscall"}},
	)
	if !projection.Active[0] {
		t.Fatal("unknown live owner eliminated a closure finding")
	}
}

func TestProjectFindingsReturnsNonstandardDeniedBoundaries(t *testing.T) {
	projection := ProjectFindings(
		Manifest{Facts: []Fact{
			{Kind: FactKindBoundary, Disposition: DispositionDenied, OwnerPackage: "time", OwnerSymbol: "time.read", Capability: "filesystem.readlink"},
			{Kind: FactKindBoundary, Disposition: DispositionDenied, OwnerPackage: "main", OwnerSymbol: "main.main", Capability: "filesystem.readlink"},
			{Kind: FactKindGuard, Disposition: DispositionGuarded, OwnerPackage: "main", OwnerSymbol: "main.interfaces", Capability: "network.interfaces", ReferencedSymbol: "net.Interfaces"},
		}},
		[]ClosurePackage{{ImportPath: "example.com/main", Root: true}, {ImportPath: "time", Standard: true}},
		nil,
	)
	if len(projection.Denied) != 1 || projection.Denied[0].OwnerPackage != "main" || projection.Denied[0].Capability != "filesystem.readlink" {
		t.Fatalf("denied = %#v", projection.Denied)
	}
	if len(projection.GuardedDenied) != 1 || projection.GuardedDenied[0].Capability != "network.interfaces" {
		t.Fatalf("guarded denied = %#v", projection.GuardedDenied)
	}
}

func TestProjectFindingsSeparatesExactGuardedAndActiveOwners(t *testing.T) {
	packages := []ClosurePackage{
		{ImportPath: "example.com/main", Root: true},
		{ImportPath: "example.com/dependency"},
		{ImportPath: "time", Standard: true},
	}
	findings := []ClosureFinding{
		{Kind: "forbidden_import", Package: "example.com/dependency", Capability: "import:syscall"},
		{Kind: "forbidden_import", Package: "example.com/main", Capability: "import:syscall"},
		{Kind: "forbidden_import", Package: "example.com/main", Capability: "import:os/exec"},
	}
	manifest := Manifest{Facts: []Fact{
		{Kind: FactKindGuard, Disposition: DispositionGuarded, OwnerPackage: "example.com/dependency", OwnerSymbol: "example.com/dependency.Call", Capability: "import:syscall", ReferencedSymbol: "syscall.Syscall"},
		{Kind: FactKindCapability, OwnerPackage: "example.com/main", OwnerSymbol: "main.main", Capability: "import:syscall", ReferencedSymbol: "syscall.Syscall"},
		{Kind: FactKindGuard, Disposition: DispositionGuarded, OwnerPackage: "time", OwnerSymbol: "time.loadLocation", Capability: "import:os/exec", ReferencedSymbol: "os/exec.Command"},
	}}
	projection := ProjectFindings(manifest, packages, findings)
	if projection.Active[0] || !projection.Active[1] || projection.Active[2] {
		t.Fatalf("active = %v, want [false true false]", projection.Active)
	}
	if len(projection.Guarded) != 1 || projection.Guarded[0] != 0 {
		t.Fatalf("guarded = %v, want [0]", projection.Guarded)
	}
	if len(projection.Eliminated) != 1 || projection.Eliminated[0] != 2 {
		t.Fatalf("eliminated = %v, want [2]", projection.Eliminated)
	}
}
