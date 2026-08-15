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
		}},
		[]ClosurePackage{{ImportPath: "example.com/main", Root: true}, {ImportPath: "time", Standard: true}},
		nil,
	)
	if len(projection.Denied) != 1 || projection.Denied[0].OwnerPackage != "main" || projection.Denied[0].Capability != "filesystem.readlink" {
		t.Fatalf("denied = %#v", projection.Denied)
	}
}
