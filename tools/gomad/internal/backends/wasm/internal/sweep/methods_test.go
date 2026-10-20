package sweep

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuiteInventoryRegistrations(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := `package example
import (
 "testing"
 ts "github.com/stretchr/testify/suite"
 ps "go.temporal.io/server/common/testing/parallelsuite"
)
type Base struct{}
func (*Base) TestInherited() {}
type S struct{ Base }
func (*S) TestOwn() {}
func TestFirst(t *testing.T) { ps.Run(t, &S{}, "argument") }
func TestSecond(t *testing.T) { ts.Run(t, new(S)) }
func TestNested(t *testing.T) { t.Run("variant", func(t *testing.T) { s := &S{}; alias := s; ts.Run(t, alias) }) }
func TestPlain(t *testing.T) {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
	inventory, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"TestFirst/TestInherited", "TestFirst/TestOwn", "TestNested/.*/TestInherited", "TestNested/.*/TestOwn", "TestPlain", "TestSecond/TestInherited", "TestSecond/TestOwn"}, inventory.Tests)
	require.Len(t, inventory.Methods, 2)
	require.Len(t, inventory.Registrations, 3)
	require.Equal(t, []string{"TestFirst/TestInherited", "TestNested/.*/TestInherited", "TestSecond/TestInherited"}, inventory.Methods[0].Units)
}

func TestSuiteInventoryNestedSuite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := `package example
import "testing"
import ps "go.temporal.io/server/common/testing/parallelsuite"
type Outer struct{}
type Inner struct{}
func TestOuter(t *testing.T) { ps.Run(t, &Outer{}) }
func (s *Outer) TestNested() { s.T().Run("variant", func(t *testing.T) { ps.RunLegacySequential(t, &Inner{}) }) }
func (*Inner) TestLeaf() {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
	inventory, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"TestOuter/TestNested/.*/TestLeaf"}, inventory.Tests)
	for _, method := range inventory.Methods {
		require.Equal(t, inventory.Tests, method.Units)
	}
}

func TestSuiteInventoryArguments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, setup, argument string }{
		{"literal", "", "&S{}"},
		{"new", "", "new(S)"},
		{"variable", "s := &S{};", "s"},
		{"alias", "s := &S{}; alias := s;", "alias"},
		{"var declaration", "var s = new(S);", "s"},
		{"typed variable", "var s *S; s = &S{};", "s"},
		{"type alias", "", "&Alias{}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			source := `package example
import "testing"
import "github.com/stretchr/testify/suite"
type S struct{}
type Alias = S
func (*S) Testlower() {}
func TestWrapper(t *testing.T) { ` + tc.setup + ` suite.Run(t, ` + tc.argument + `) }
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
			inventory, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, nil)
			require.NoError(t, err)
			require.Equal(t, []string{"TestWrapper/Testlower"}, inventory.Tests)
		})
	}
}

func TestSuiteInventoryVariableScopes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := `package example
import "testing"
import "github.com/stretchr/testify/suite"
type S struct{}
func (*S) TestOwn() {}
func TestWrapper(t *testing.T) {
 s := makeSuite()
 t.Run("variant", func(t *testing.T) { s := &S{}; alias := s; suite.Run(t, alias) })
 _ = s
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
	inventory, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"TestWrapper/.*/TestOwn"}, inventory.Tests)
}

func TestSuiteInventoryExternalEmbedding(t *testing.T) {
	t.Parallel()
	for _, promoted := range []bool{false, true} {
		t.Run(map[bool]string{false: "no tests", true: "promoted tests"}[promoted], func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			source := `package example
import "testing"
import "github.com/stretchr/testify/suite"
import "example.com/base"
type S struct{ base.Base }
func (*S) TestOwn() {}
func TestWrapper(t *testing.T) { suite.Run(t, &S{}) }
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
			external := "package base\ntype Base struct{}\n"
			if promoted {
				external += "func (*Base) TestPromoted() {}\n"
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "base.go"), []byte(external), 0o600))
			base, err := parseSuiteSource(dir, []string{"base.go"})
			require.NoError(t, err)
			inventory, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, func(path string) (*suiteSource, error) {
				require.Equal(t, "example.com/base", path)
				return base, nil
			})
			if promoted {
				require.ErrorContains(t, err, "cross-package promoted test method")
			} else {
				require.NoError(t, err)
				require.Equal(t, []string{"TestWrapper/TestOwn"}, inventory.Tests)
			}
		})
	}
}

func TestSuiteInventoryPromotionUsesShallowestMethod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := `package example
import "testing"
import "github.com/stretchr/testify/suite"
type Base struct{}
func (*Base) TestOne() {}
type A struct{ Base }
type B struct{}
func (*B) TestOne() {}
type S struct{ A; B }
func TestWrapper(t *testing.T) { suite.Run(t, &S{}) }
func TestBase(t *testing.T) { suite.Run(t, &Base{}) }
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
	inventory, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"TestBase/TestOne", "TestWrapper/TestOne"}, inventory.Tests)
	require.Equal(t, []SuiteMethod{
		{Suite: "example.B", Name: "TestOne", File: "suite_test.go", Units: []string{"TestWrapper/TestOne"}},
		{Suite: "example.Base", Name: "TestOne", File: "suite_test.go", Units: []string{"TestBase/TestOne"}},
	}, inventory.Methods)
}

func TestSuiteInventoryRefusesMissingCoverage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, source, reason string }{
		{"unregistered", `type S struct{}; func (*S) TestLost() {}`, "unregistered test method"},
		{"unresolved", `func TestWrapper(t *testing.T) { suite.Run(t, makeSuite()) }`, "unresolved suite registration"},
		{"reassigned", `type S struct{}; type Other struct{}; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { s := &S{}; s = &Other{}; suite.Run(t, s) }`, "unresolved suite registration"},
		{"range shadow", `type S struct{}; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { s := &S{}; for _, s := range unknown { suite.Run(t, s) }; _ = s }`, "unresolved suite registration"},
		{"colliding registrations", `type S struct{}; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { suite.Run(t, &S{}); suite.Run(t, &S{}) }`, "overlapping suite registrations"},
		{"ambiguous promotion", `type A struct{}; type B struct{}; type S struct { A; B }; func (*A) TestSame() {}; func (*B) TestSame() {}; func TestWrapper(t *testing.T) { suite.Run(t, &S{}) }`, "ambiguous promoted method"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			source := "package example\nimport \"testing\"\nimport \"github.com/stretchr/testify/suite\"\n" + tc.source
			require.NoError(t, os.WriteFile(filepath.Join(dir, "suite_test.go"), []byte(source), 0o600))
			_, err := parseSuiteInventory(dir, []string{"suite_test.go"}, nil, nil)
			require.ErrorContains(t, err, tc.reason)
		})
	}
}

func TestSuiteInventoryGuestExclusionsIncludeWrapperAndMethod(t *testing.T) {
	t.Parallel()
	for _, wrapperExcluded := range []bool{false, true} {
		t.Run(map[bool]string{false: "method", true: "wrapper"}[wrapperExcluded], func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			dir := filepath.Join(f.options.Checkout, "tests", "alpha")
			wrapper, method := "alpha_test.go", "unix_test.go"
			if wrapperExcluded {
				wrapper, method = method, wrapper
			}
			sources := map[string]string{
				"alpha_test.go": "package alpha\nimport \"testing\"\nfunc TestAlpha(t *testing.T) {}\nfunc TestBeta(t *testing.T) {}\n",
				"unix_test.go":  "package alpha\nimport \"testing\"\nfunc TestUnixOnly(t *testing.T) {}\n",
			}
			sources[wrapper] = "package alpha\nimport \"testing\"\nimport ps \"go.temporal.io/server/common/testing/parallelsuite\"\ntype S struct{}\n"
			if wrapperExcluded {
				sources[wrapper] += "func TestUnixOnly(t *testing.T) { ps.Run(t, &S{}) }\n"
			} else {
				sources[wrapper] += "func TestAlpha(t *testing.T) { ps.Run(t, &S{}) }\nfunc TestBeta(t *testing.T) {}\n"
			}
			sources[method] += "func (*S) TestMethod() {}\n"
			for name, source := range sources {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(source), 0o600))
			}
			packages, err := Enumerate(t.Context(), f, f.options, GuestTags(), nil)
			require.NoError(t, err)
			unit := "TestAlpha/TestMethod"
			if wrapperExcluded {
				unit = "TestUnixOnly/TestMethod"
			}
			require.Equal(t, "unix_test.go", packages[0].GuestExcluded[unit])
		})
	}
}

func TestSuiteTestSelection(t *testing.T) {
	t.Parallel()
	require.Equal(t, `^TestWrapper$/^TestLeaf\+$`, testRunPattern("TestWrapper/TestLeaf+"))
	require.Equal(t, "^TestWrapper$/.*/^TestLeaf$", testRunPattern("TestWrapper/.*/TestLeaf"))
	require.True(t, selectedTest("TestWrapper/.*/TestLeaf", "TestWrapper/variant/TestLeaf"))
	require.False(t, selectedTest("TestWrapper/.*/TestLeaf", "TestWrapper/variant/TestLeaf/subtest"))
	require.False(t, selectedTest("TestWrapper/.*/TestLeaf", "TestWrapper/variant/TestOther"))
}
