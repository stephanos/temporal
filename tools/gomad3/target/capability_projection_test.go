package target

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
)

func TestCompatibilityPackProjectionPreservesCompleteEvidence(t *testing.T) {
	adapter := &compatibility.PackAdapter{ProfileName: "profile", ProfileImplementationSHA256: "profile-sha", Module: "adapter-module", Version: "adapter-version", Sum: "adapter-sum", OriginalSourceInventorySHA256: "original-sha", ReplacementSourceInventorySHA256: "replacement-sha", PreparedSourceSetSHA256: "prepared-sha"}
	module := compatibility.ModuleEvidence{Path: "module", Version: "version", Sum: "sum", Replacement: "adapter", Adapter: adapter}
	value := compatibility.PackEvidence{ID: "id", SHA256: "pack-sha", RequestSHA256: "request-sha", Governance: &compatibility.PackGovernance{Owner: "owner", ReviewedAt: "reviewed", Justification: "reason", Workloads: []string{"workload"}, Platforms: []string{"platform"}, ApprovalSHA256: "approval-sha"}, Activation: []compatibility.ModuleEvidence{module}, Rules: []compatibility.PackageRuleEvidence{{ImportPath: "package", Module: module, SourceSetSHA256: "source-set", GoSources: []compatibility.PackSource{{Name: "source.go", SHA256: "source-sha"}}, ForeignSources: []compatibility.PackForeignSource{{Kind: "asm", Name: "source.s", SHA256: "foreign-sha"}}, Capabilities: []string{"capability"}, Linknames: []compatibility.LinknameEvidence{{Source: "link.go", SHA256: "link-sha", Directives: []string{"directive"}}}}}}
	want := `{"id":"id","sha256":"pack-sha","request_sha256":"request-sha","governance":{"owner":"owner","reviewed_at":"reviewed","justification":"reason","workloads":["workload"],"platforms":["platform"],"approval_sha256":"approval-sha"},"activation":[{"path":"module","version":"version","sum":"sum","replacement":"adapter","adapter":{"profile_name":"profile","profile_implementation_sha256":"profile-sha","module":"adapter-module","version":"adapter-version","sum":"adapter-sum","original_source_inventory_sha256":"original-sha","replacement_source_inventory_sha256":"replacement-sha","prepared_source_set_sha256":"prepared-sha"}}],"rules":[{"import_path":"package","module":{"path":"module","version":"version","sum":"sum","replacement":"adapter","adapter":{"profile_name":"profile","profile_implementation_sha256":"profile-sha","module":"adapter-module","version":"adapter-version","sum":"adapter-sum","original_source_inventory_sha256":"original-sha","replacement_source_inventory_sha256":"replacement-sha","prepared_source_set_sha256":"prepared-sha"}},"source_set_sha256":"source-set","go_sources":[{"name":"source.go","sha256":"source-sha"}],"foreign_sources":[{"kind":"asm","name":"source.s","sha256":"foreign-sha"}],"capabilities":["capability"],"linknames":[{"source":"link.go","sha256":"link-sha","directives":["directive"]}]}]}`
	projected := projectCompatibilityPackEvidence([]compatibility.PackEvidence{value})[0]
	for _, graph := range []any{value, projected} {
		data, err := json.Marshal(graph)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("complete evidence changed: %s", data)
		}
	}
	oldCanonical, err := canonicaljson.CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	newCanonical, err := canonicaljson.CanonicalJSON(projected)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(oldCanonical, newCanonical) {
		t.Fatalf("canonical report changed: %s -> %s", oldCanonical, newCanonical)
	}
}

func TestCompatibilityPackProjectionPreservesNilAndEmpty(t *testing.T) {
	for _, empty := range []bool{false, true} {
		value := compatibility.PackEvidence{Governance: &compatibility.PackGovernance{}, Activation: []compatibility.ModuleEvidence{{Adapter: &compatibility.PackAdapter{}}}, Rules: []compatibility.PackageRuleEvidence{{Linknames: []compatibility.LinknameEvidence{{}}}}}
		if empty {
			value.Governance.Workloads = []string{}
			value.Governance.Platforms = []string{}
			value.Rules[0].GoSources = []compatibility.PackSource{}
			value.Rules[0].ForeignSources = []compatibility.PackForeignSource{}
			value.Rules[0].Capabilities = []string{}
			value.Rules[0].Linknames[0].Directives = []string{}
		}
		projected := projectCompatibilityPackEvidence([]compatibility.PackEvidence{value})[0]
		for _, slice := range []any{projected.Governance.Workloads, projected.Governance.Platforms, projected.Rules[0].GoSources, projected.Rules[0].ForeignSources, projected.Rules[0].Capabilities, projected.Rules[0].Linknames[0].Directives} {
			if reflect.ValueOf(slice).IsNil() == empty {
				t.Fatalf("nil/empty changed (%t): %#v", empty, slice)
			}
		}
		if projected.Governance == nil || projected.Activation[0].Adapter == nil {
			t.Fatal("present zero pointers lost")
		}
	}
	projected := projectCompatibilityPackEvidence(nil)
	if projected == nil || len(projected) != 0 {
		t.Fatalf("outer nil projection changed: %#v", projected)
	}
	zero := projectCompatibilityPackEvidence([]compatibility.PackEvidence{{}})[0]
	if zero.Governance != nil || zero.Activation != nil || zero.Rules != nil {
		t.Fatalf("inner nil changed: %#v", zero)
	}
	for _, empty := range []bool{false, true} {
		value := compatibility.PackEvidence{}
		if empty {
			value.Activation = []compatibility.ModuleEvidence{}
			value.Rules = []compatibility.PackageRuleEvidence{}
		}
		projected := projectCompatibilityPackEvidence([]compatibility.PackEvidence{value})[0]
		if (projected.Activation == nil) == empty || (projected.Rules == nil) == empty {
			t.Fatalf("outer containers changed: %#v", projected)
		}
		value.Activation = []compatibility.ModuleEvidence{{Path: "populated", Replacement: "none"}}
		value.Rules = []compatibility.PackageRuleEvidence{{ImportPath: "package"}}
		if empty {
			value.Rules[0].Linknames = []compatibility.LinknameEvidence{}
		}
		projected = projectCompatibilityPackEvidence([]compatibility.PackEvidence{value})[0]
		if projected.Activation[0].Adapter != nil || (projected.Rules[0].Linknames == nil) == empty {
			t.Fatalf("absent adapter/linknames changed: %#v", projected)
		}
		oldJSON, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		newJSON, err := json.Marshal(projected)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(oldJSON, newJSON) {
			t.Fatalf("container bytes changed: %s -> %s", oldJSON, newJSON)
		}
	}
}

func TestCompatibilityPackProjectionPreservesOrderAndDetachesNestedStorage(t *testing.T) {
	values := []compatibility.PackEvidence{{ID: "first", Governance: &compatibility.PackGovernance{Workloads: []string{"w1", "w2"}, Platforms: []string{"p1", "p2"}}, Activation: []compatibility.ModuleEvidence{{Path: "a1", Adapter: &compatibility.PackAdapter{ProfileName: "adapter"}}, {Path: "a2"}}, Rules: []compatibility.PackageRuleEvidence{{ImportPath: "r1", Module: compatibility.ModuleEvidence{Adapter: &compatibility.PackAdapter{ProfileName: "rule-adapter"}}, GoSources: []compatibility.PackSource{{Name: "g1"}, {Name: "g2"}}, ForeignSources: []compatibility.PackForeignSource{{Name: "f1"}, {Name: "f2"}}, Capabilities: []string{"c1", "c2"}, Linknames: []compatibility.LinknameEvidence{{Source: "l1", Directives: []string{"d1", "d2"}}, {Source: "l2"}}}, {ImportPath: "r2"}}}, {ID: "second"}}
	projected := projectCompatibilityPackEvidence(values)
	want, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("ordered graph changed: %s -> %s", want, actual)
	}
	values[0].Governance.Workloads[0] = "changed"
	values[0].Governance.Platforms[0] = "changed"
	values[0].Activation[0].Adapter.ProfileName = "changed"
	values[0].Rules[0].Module.Adapter.ProfileName = "changed"
	values[0].Rules[0].GoSources[0].Name = "changed"
	values[0].Rules[0].ForeignSources[0].Name = "changed"
	values[0].Rules[0].Capabilities[0] = "changed"
	values[0].Rules[0].Linknames[0].Directives[0] = "changed"
	actual, err = json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatal("projection aliases source nested storage")
	}
	before, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	projected[0].Governance.Owner = "projected"
	projected[0].Governance.Workloads[1] = "projected"
	projected[0].Activation[0].Adapter.Module = "projected"
	projected[0].Rules[0].GoSources[1].SHA256 = "projected"
	projected[0].Rules[0].Linknames[0].Directives[1] = "projected"
	after, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("source aliases projection nested storage")
	}
}
