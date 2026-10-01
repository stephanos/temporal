package compatibility_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
)

// staleProfileBindings names every adapter binding of a pack governing
// platform that does not carry the given profile identity. Selection requires
// that identity to be equal, so a stale pack is never selected and its target
// analyzes as unsupported while the generated artifacts still agree with
// their requests.
func staleProfileBindings(packs []compatibility.Pack, platform, profileName, implementationSHA256 string) (stale []string, checked int) {
	for _, pack := range packs {
		if !slices.Contains(pack.Governance.Platforms, platform) {
			continue
		}
		check := func(location string, module compatibility.PackModule) {
			adapter := module.Replacement.Adapter
			if adapter == nil {
				return
			}
			checked++
			if adapter.ProfileName != profileName || adapter.ProfileImplementationSHA256 != implementationSHA256 {
				stale = append(stale, pack.ID+": "+location+" binds "+adapter.ProfileName+" "+adapter.ProfileImplementationSHA256)
			}
		}
		for _, module := range pack.Activation {
			check("activation "+module.Path, module)
		}
		for _, rule := range pack.Rules {
			check("rule "+rule.ImportPath, rule.Module)
		}
	}
	slices.Sort(stale)
	return stale, checked
}

func TestHostPacksBindCurrentProfile(t *testing.T) {
	host := runtime.GOOS + "/" + runtime.GOARCH
	if !slices.Contains(deterministicio.BoundaryPlatforms(), host) {
		t.Skipf("no deterministic profile for %s", host)
	}
	entries, err := os.ReadDir("packs")
	if err != nil {
		t.Fatal(err)
	}
	packs := make([]compatibility.Pack, 0, len(entries))
	for _, entry := range entries {
		contents, err := os.ReadFile(filepath.Join("packs", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		pack, err := compatibility.DecodePack(contents)
		if err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		packs = append(packs, pack)
	}
	profile := deterministicio.Default().Identity()
	stale, checked := staleProfileBindings(packs, host, profile.Name, string(profile.ImplementationSHA256))
	if len(stale) != 0 {
		t.Fatalf("%s packs do not bind the current profile %s %s; rediscover, review, and regenerate them:\n%s", host, profile.Name, profile.ImplementationSHA256, strings.Join(stale, "\n"))
	}
	t.Logf("%d adapter bindings of %s packs carry %s", checked, host, profile.ImplementationSHA256)
}

func TestStaleProfileBindingsAreReported(t *testing.T) {
	const (
		host    = "darwin/arm64"
		name    = "gomad3-deterministic/v1"
		current = "sha256:current"
		stale   = "sha256:stale"
	)
	bound := func(path, profileName, digest string) compatibility.PackModule {
		return compatibility.PackModule{Path: path, Replacement: compatibility.PackReplacement{
			Kind:    compatibility.ReplacementAdapter,
			Adapter: &compatibility.PackAdapter{ProfileName: profileName, ProfileImplementationSHA256: digest},
		}}
	}
	unbound := compatibility.PackModule{Path: "example.com/plain", Replacement: compatibility.PackReplacement{Kind: compatibility.ReplacementNone}}
	pack := func(id, platform string, activation compatibility.PackModule, rule compatibility.PackModule) compatibility.Pack {
		return compatibility.Pack{
			ID: id, Governance: compatibility.PackGovernance{Platforms: []string{platform}},
			Activation: []compatibility.PackModule{unbound, activation},
			Rules:      []compatibility.PackRule{{ImportPath: "example.com/rule", Module: rule}},
		}
	}
	// Activations and rules may repeat a module path or import path under
	// another module identity, so a current entry must not mask a stale one.
	repeated := pack("repeated", host, bound("example.com/a", name, stale), bound("example.com/a", name, stale))
	repeated.Activation = append(repeated.Activation, bound("example.com/a", name, current))
	repeated.Rules = append(repeated.Rules, compatibility.PackRule{ImportPath: "example.com/rule", Module: bound("example.com/a", name, current)})
	for testName, test := range map[string]struct {
		pack        compatibility.Pack
		wantStale   []string
		wantChecked int
	}{
		"current bindings": {
			pack:        pack("current", host, bound("example.com/a", name, current), bound("example.com/a", name, current)),
			wantChecked: 2,
		},
		"stale activation": {
			pack:        pack("activation", host, bound("example.com/a", name, stale), unbound),
			wantStale:   []string{"activation: activation example.com/a binds " + name + " " + stale},
			wantChecked: 1,
		},
		"stale rule module": {
			pack:        pack("rule", host, bound("example.com/a", name, current), bound("example.com/a", name, stale)),
			wantStale:   []string{"rule: rule example.com/rule binds " + name + " " + stale},
			wantChecked: 2,
		},
		"other profile": {
			pack:        pack("profile", host, bound("example.com/a", "other/v1", current), unbound),
			wantStale:   []string{"profile: activation example.com/a binds other/v1 " + current},
			wantChecked: 1,
		},
		"repeated keys": {
			pack: repeated,
			wantStale: []string{
				"repeated: activation example.com/a binds " + name + " " + stale,
				"repeated: rule example.com/rule binds " + name + " " + stale,
			},
			wantChecked: 4,
		},
		"other platform": {
			pack: pack("platform", "linux/amd64", bound("example.com/a", name, stale), bound("example.com/a", name, stale)),
		},
	} {
		t.Run(testName, func(t *testing.T) {
			gotStale, gotChecked := staleProfileBindings([]compatibility.Pack{test.pack}, host, name, current)
			if !reflect.DeepEqual(gotStale, test.wantStale) || gotChecked != test.wantChecked {
				t.Fatalf("staleProfileBindings() = %q, %d; want %q, %d", gotStale, gotChecked, test.wantStale, test.wantChecked)
			}
		})
	}
}
