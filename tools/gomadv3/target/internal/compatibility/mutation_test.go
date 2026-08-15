package compatibility

import (
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

type generatedPackMutation struct {
	PackID   string
	Mutation string
}

func TestGeneratedCompatibilityPackMutations(t *testing.T) {
	for _, test := range generatedPackMutationInventory {
		t.Run(test.PackID+"/"+test.Mutation, func(t *testing.T) {
			runGeneratedPackMutation(t, test)
		})
	}
}

func runGeneratedPackMutation(t *testing.T, test generatedPackMutation) {
	t.Helper()
	contents, err := packFiles.ReadFile("packs/" + test.PackID + ".json")
	if err != nil {
		t.Fatal(err)
	}
	validated, err := LoadPack(contents)
	if err != nil {
		t.Fatal(err)
	}
	packages := generatedExactPackages(validated.pack)
	platform := validated.pack.Governance.Platforms[0]

	switch test.Mutation {
	case "positive":
		generatedRequireAllowed(t, validated, packages, platform)
	case "availability":
		if err := VerifyPackIdentities(nil, []Identity{{ID: validated.pack.ID, SHA256: validated.digest}}); err == nil {
			t.Fatal("unavailable pack identity was accepted")
		}
	case "pack_digest":
		if err := VerifyPackIdentities([]ValidatedPack{validated}, []Identity{{ID: validated.pack.ID, SHA256: changedDigest(validated.digest)}}); err == nil {
			t.Fatal("changed pack identity was accepted")
		}
	case "platform":
		selection, err := SelectPacksForPlatform([]ValidatedPack{validated}, packages, "invalid/invalid")
		if err != nil {
			t.Fatal(err)
		}
		if len(selection.Identities()) != 0 {
			t.Fatal("wrong platform selected the pack")
		}
	case "arbitrary_local_replacement":
		activation := validated.pack.Activation[0]
		for index := range packages {
			if packages[index].Module.Path == activation.Path && packages[index].Module.Version == activation.Version {
				packages[index].Module.Replaced = true
				packages[index].Module.LocalReplacement = true
				packages[index].Module.Adapter = nil
			}
		}
		generatedRequireNotSelected(t, validated, packages, platform)
	case "source_set":
		packages[0].SourceSetSHA256 = changedDigest(packages[0].SourceSetSHA256)
		generatedRequireFirstFactDenied(t, validated, packages, platform)
	case "go_source":
		packages[0].GoSources[0].SHA256 = changedDigest(packages[0].GoSources[0].SHA256)
		generatedRequireFirstFactDenied(t, validated, packages, platform)
	case "foreign_source":
		index := generatedPackageWithForeignSource(packages)
		packages[index].ForeignSources[0].SHA256 = changedDigest(packages[index].ForeignSources[0].SHA256)
		generatedRequireRuleFactDenied(t, validated, packages, platform, index)
	case "module_sum":
		packages[0].Module.Sum = "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
		generatedRequireFirstFactDenied(t, validated, packages, platform)
	case "module_version":
		packages[0].Module.Version += "-changed"
		generatedRequireFirstFactDenied(t, validated, packages, platform)
	case "directive":
		generatedRequireChangedDirectiveDenied(t, validated, packages, platform)
	case "adapter_identity", "original_source_inventory", "prepared_source_set", "replacement_source_inventory":
		generatedMutateAdapter(packages, test.Mutation)
		generatedRequireNotSelected(t, validated, packages, platform)
	case "approval", "justification", "owner", "request_identity", "review_time", "workload":
		generatedRequirePackIdentityMutation(t, validated, test.Mutation)
	default:
		t.Fatalf("unknown generated compatibility-pack mutation %q", test.Mutation)
	}
}

func generatedExactPackages(pack Pack) []Package {
	packages := make([]Package, 0, len(pack.Rules)+len(pack.Activation))
	for _, rule := range pack.Rules {
		goSources := make([]Source, len(rule.GoSources))
		for index, source := range rule.GoSources {
			goSources[index] = Source{Name: source.Name, SHA256: source.SHA256}
		}
		foreignSources := make([]ForeignSource, len(rule.ForeignSources))
		for index, source := range rule.ForeignSources {
			foreignSources[index] = ForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256}
		}
		packages = append(packages, Package{
			ImportPath: rule.ImportPath, Module: generatedActualModule(rule.Module), SourceSetSHA256: rule.SourceSetSHA256,
			GoSources: goSources, ForeignSources: foreignSources,
		})
	}
	for index, activation := range pack.Activation {
		found := false
		for _, pkg := range packages {
			found = found || matchesPackModule(activation, pkg.Module)
		}
		if !found {
			packages = append(packages, Package{ImportPath: "generated.activation/" + string(rune('a'+index)), Module: generatedActualModule(activation)})
		}
	}
	return packages
}

func generatedActualModule(module PackModule) Module {
	actual := Module{Path: module.Path, Version: module.Version, Sum: module.Sum}
	if module.Replacement.Kind == ReplacementAdapter {
		adapter := module.Replacement.Adapter
		actual.Replaced = true
		actual.LocalReplacement = true
		actual.Adapter = &AdapterEvidence{
			ProfileName: adapter.ProfileName, ProfileImplementationSHA256: adapter.ProfileImplementationSHA256,
			Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum,
			OriginalSourceInventorySHA256:    adapter.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: adapter.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          adapter.PreparedSourceSetSHA256,
		}
	}
	return actual
}

func generatedRequireAllowed(t *testing.T, pack ValidatedPack, packages []Package, platform string) {
	t.Helper()
	selection, err := SelectPacksForPlatform([]ValidatedPack{pack}, packages, platform)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Identities()) != 1 || selection.Identities()[0].ID != pack.pack.ID {
		t.Fatalf("selected identities = %#v", selection.Identities())
	}
	for index, rule := range pack.pack.Rules {
		for _, capability := range rule.Capabilities {
			if !selection.Evaluate(packages[index], Fact{Kind: FactCapability, Capability: capability}).Allowed {
				t.Fatalf("capability %s was not allowed", capability)
			}
		}
		for _, linkname := range rule.Linknames {
			if !selection.Evaluate(packages[index], Fact{Kind: FactLinkname, Source: linkname.Source, SHA256: linkname.SHA256, Directives: linkname.Directives}).Allowed {
				t.Fatalf("linkname %s was not allowed", linkname.Source)
			}
		}
	}
}

func generatedRequireNotSelected(t *testing.T, pack ValidatedPack, packages []Package, platform string) {
	t.Helper()
	selection, err := SelectPacksForPlatform([]ValidatedPack{pack}, packages, platform)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Identities()) != 0 {
		t.Fatalf("mutated packages selected identities %#v", selection.Identities())
	}
}

func generatedRequireFirstFactDenied(t *testing.T, pack ValidatedPack, packages []Package, platform string) {
	t.Helper()
	generatedRequireRuleFactDenied(t, pack, packages, platform, 0)
}

func generatedRequireRuleFactDenied(t *testing.T, pack ValidatedPack, packages []Package, platform string, index int) {
	t.Helper()
	selection, err := SelectPacksForPlatform([]ValidatedPack{pack}, packages, platform)
	if err != nil {
		t.Fatal(err)
	}
	fact := generatedFirstFact(pack.pack.Rules[index])
	if selection.Evaluate(packages[index], fact).Allowed {
		t.Fatalf("mutated package authorized %#v", fact)
	}
}

func generatedFirstFact(rule PackRule) Fact {
	if len(rule.Capabilities) != 0 {
		return Fact{Kind: FactCapability, Capability: rule.Capabilities[0]}
	}
	linkname := rule.Linknames[0]
	return Fact{Kind: FactLinkname, Source: linkname.Source, SHA256: linkname.SHA256, Directives: append([]string{}, linkname.Directives...)}
}

func generatedPackageWithForeignSource(packages []Package) int {
	for index, pkg := range packages {
		if len(pkg.ForeignSources) != 0 {
			return index
		}
	}
	return -1
}

func generatedRequireChangedDirectiveDenied(t *testing.T, pack ValidatedPack, packages []Package, platform string) {
	t.Helper()
	selection, err := SelectPacksForPlatform([]ValidatedPack{pack}, packages, platform)
	if err != nil {
		t.Fatal(err)
	}
	for index, rule := range pack.pack.Rules {
		if len(rule.Linknames) == 0 {
			continue
		}
		linkname := rule.Linknames[0]
		directives := append([]string{}, linkname.Directives...)
		directives[0] += "Changed"
		if selection.Evaluate(packages[index], Fact{Kind: FactLinkname, Source: linkname.Source, SHA256: linkname.SHA256, Directives: directives}).Allowed {
			t.Fatal("changed linkname directive was authorized")
		}
		return
	}
	t.Fatal("generated directive mutation has no linkname")
}

func generatedMutateAdapter(packages []Package, mutation string) {
	for index := range packages {
		adapter := packages[index].Module.Adapter
		if adapter == nil {
			continue
		}
		switch mutation {
		case "adapter_identity":
			adapter.ProfileImplementationSHA256 = changedDigest(adapter.ProfileImplementationSHA256)
		case "original_source_inventory":
			adapter.OriginalSourceInventorySHA256 = changedDigest(adapter.OriginalSourceInventorySHA256)
		case "prepared_source_set":
			adapter.PreparedSourceSetSHA256 = changedDigest(adapter.PreparedSourceSetSHA256)
		case "replacement_source_inventory":
			adapter.ReplacementSourceInventorySHA256 = changedDigest(adapter.ReplacementSourceInventorySHA256)
		}
	}
}

func generatedRequirePackIdentityMutation(t *testing.T, original ValidatedPack, mutation string) {
	t.Helper()
	pack := original.pack
	pack.Governance.Workloads = append([]string{}, pack.Governance.Workloads...)
	pack.Governance.Platforms = append([]string{}, pack.Governance.Platforms...)
	switch mutation {
	case "approval":
		pack.Governance.ApprovalSHA256 = changedDigest(pack.Governance.ApprovalSHA256)
	case "justification":
		pack.Governance.Justification += " changed"
	case "owner":
		pack.Governance.Owner += "-changed"
	case "request_identity":
		pack.RequestSHA256 = changedDigest(pack.RequestSHA256)
	case "review_time":
		if pack.Governance.ReviewedAt == "2026-08-16T00:00:00Z" {
			pack.Governance.ReviewedAt = "2026-08-17T00:00:00Z"
		} else {
			pack.Governance.ReviewedAt = "2026-08-16T00:00:00Z"
		}
	case "workload":
		pack.Governance.Workloads[0] += "-changed"
	}
	contents, err := evidence.CanonicalJSON(pack)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := LoadPack(append(contents, '\n'))
	if err != nil {
		t.Fatal(err)
	}
	if changed.digest == original.digest {
		t.Fatalf("%s mutation retained pack identity %s", mutation, original.digest)
	}
}

func changedDigest(value string) string {
	if strings.HasSuffix(value, "0") {
		return value[:len(value)-1] + "1"
	}
	return value[:len(value)-1] + "0"
}
