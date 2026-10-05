package golden

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// declaredCase is a compact Case, as the lowering writes it, with a controller, an activity and a
// workflow entrypoint, whose Program carries program before its end and whose activity carries activity
// after its own members.
func declaredCase(program, activity, contract string) string {
	return `{"caseId":"c","version":{"major":1},"provenance":{"producerId":"p"},"program":{"programId":"p","entrypoints":[` +
		`{"entrypointId":"controller","controller":{"roleId":"r"},"instructions":[]},` +
		`{"entrypointId":"attempt","activity":{"activityType":"a","workerRoleId":"w"` + activity + `},"instructions":[]},` +
		`{"entrypointId":"flow","workflow":{"workflowType":"f"},"instructions":[]}]` + program + `},` +
		`"contract":{"contractId":"k","state":"` + contract + `"}}` + "\n"
}

const (
	declaredProgram  = `,"instructionDefaults":{"timeoutMilliseconds":"10000","maxAttempts":"1"},"runOrderIsCausal":true`
	declaredActivity = `,"attemptNumbering":{"first":"1","oneRun":true}`
)

func originalDeclared(t *testing.T) Declared {
	t.Helper()
	delta, err := OriginalDelta()
	require.NoError(t, err)
	return delta.Declared()
}

// TestDeclaredMembersProjectOnlyTheDeclaredMembers drops the members an API behavior declares from a
// current Case, whatever their values, and keeps every other byte: the Contract's, another member's and
// a member of the same name elsewhere. A baseline Case is kept as it is, and refused if it carries one.
func TestDeclaredMembersProjectOnlyTheDeclaredMembers(t *testing.T) {
	declared := originalDeclared(t)
	require.Len(t, declared, 3)
	base := declaredCase("", "", "s")
	current := declaredCase(declaredProgram, declaredActivity, "s")
	projected, err := declared.Current([]byte(current))
	require.NoError(t, err)
	require.Equal(t, base, string(projected), "exactly the declared members are dropped")
	carried, err := declared.Carried([]byte(current))
	require.NoError(t, err)
	require.Equal(t, []string(declared), carried)

	compare := func(baseline, current string) error {
		w, err := declared.Baseline([]byte(baseline))
		if err != nil {
			return err
		}
		g, err := declared.Current([]byte(current))
		if err != nil {
			return err
		}
		return Compare(map[string][]byte{"case": w}, map[string][]byte{"case": g})
	}
	require.NoError(t, compare(base, current))
	require.NoError(t, compare(base, base), "a Case of a realization that declares no behavior")
	require.NoError(t, compare(base, declaredCase(`,"instructionDefaults":{"timeoutMilliseconds":"5000"}`, `,"attemptNumbering":{"first":"0"}`, "s")),
		"a declared member is dropped whatever its value")
	for name, changed := range map[string]string{
		"Contract byte":                 declaredCase(declaredProgram, declaredActivity, "t"),
		"provenance":                    strings.Replace(current, `"producerId":"p"`, `"producerId":"q"`, 1),
		"another Program member":        strings.Replace(current, `"programId":"p"`, `"programId":"q"`, 1),
		"activity member beside it":     strings.Replace(current, `"activityType":"a"`, `"activityType":"b"`, 1),
		"unlisted Program member":       declaredCase(declaredProgram+`,"runOrderIsStrict":true`, declaredActivity, "s"),
		"unlisted activity member":      declaredCase(declaredProgram, declaredActivity+`,"attemptLimit":"3"`, "s"),
		"declared name in the workflow": strings.Replace(current, `"workflowType":"f"`, `"workflowType":"f"`+declaredActivity, 1),
		"declared name in the Contract": strings.Replace(current, `"state":"s"`, `"state":"s","runOrderIsCausal":true`, 1),
		"declared name at the Case":     strings.Replace(current, `"caseId":"c",`, `"caseId":"c","runOrderIsCausal":true,`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, current, changed)
			require.Error(t, compare(base, changed))
		})
	}
	for _, path := range declared {
		t.Run("baseline carries "+path, func(t *testing.T) {
			program, activity := "", ""
			if strings.Contains(path, "activity") {
				activity = declaredActivity
			} else {
				program = declaredProgram[:strings.Index(declaredProgram, `,"runOrderIsCausal"`)]
				if strings.HasSuffix(path, "runOrderIsCausal") {
					program = `,"runOrderIsCausal":true`
				}
			}
			_, err := declared.Baseline([]byte(declaredCase(program, activity, "s")))
			require.ErrorContains(t, err, path)
		})
	}

	const indented = "{\n  \"indented\": true\n}"
	kept, err := Declared(nil).Current([]byte(indented))
	require.NoError(t, err, "with no members listed a Case is kept as it is")
	require.Equal(t, indented, string(kept))
	_, err = declared.Current([]byte(indented))
	require.ErrorContains(t, err, "not compact JSON", "a Case that cannot be rewritten member by member")
	programOf := func(c string) string {
		return c[strings.Index(c, `"program":`)+len(`"program":`) : strings.Index(c, `,"contract":`)]
	}
	got, err := declared.CurrentProgram([]byte(programOf(current)))
	require.NoError(t, err)
	require.Equal(t, programOf(base), string(got), "a Program alone is read as within its Case")
	_, err = declared.BaselineProgram([]byte(programOf(current)))
	require.Error(t, err)
	kept, err = declared.BaselineProgram([]byte(programOf(base)))
	require.NoError(t, err)
	require.Equal(t, programOf(base), string(kept))
}

// TestOriginalDeclaredMembersAreClosed refuses an entry that is no one member of the Program.
func TestOriginalDeclaredMembersAreClosed(t *testing.T) {
	require.NoError(t, Delta{DeclaredMembers: []string{"program.a", "program.b[*].c.d"}}.check())
	for name, path := range map[string][]string{
		"the Program":         {"program"},
		"a Contract member":   {"contract.state"},
		"a Case member":       {"caseId"},
		"an array's elements": {"program.entrypoints[*]"},
		"an empty member":     {"program..a"},
		"not lowerCamel":      {"program.Run"},
		"listed twice":        {"program.a", "program.a"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, Delta{DeclaredMembers: path}.check())
		})
	}
}

// TestOriginalDeclaredMembersAreWhatTheCasesCarry holds the recorded list to the Cases: each listed
// member is carried by some current Case, so the list without it leaves that Case differing, and no
// archived Case carries one.
func TestOriginalDeclaredMembersAreWhatTheCasesCarry(t *testing.T) {
	root, err := Root()
	require.NoError(t, err)
	declared := originalDeclared(t)
	archived, err := OriginalArchive(root)
	require.NoError(t, err)
	current, err := OriginalCurrent(root)
	require.NoError(t, err)
	isCase := func(key string) bool {
		return strings.HasPrefix(key, OriginalCases) && strings.HasSuffix(key, "-case.json")
	}
	carriers := map[string]int{}
	for _, key := range slices.Sorted(maps.Keys(current)) {
		if !isCase(key) {
			continue
		}
		carried, err := declared.Carried(current[key])
		require.NoError(t, err, key)
		for _, path := range carried {
			carriers[path]++
		}
		projected, err := declared.Current(current[key])
		require.NoError(t, err, key)
		left, err := declared.Carried(projected)
		require.NoError(t, err, key)
		require.Empty(t, left, key)
	}
	for _, path := range declared {
		require.Positive(t, carriers[path], "no current Case carries %s", path)
	}
	baselines := 0
	for _, key := range slices.Sorted(maps.Keys(archived)) {
		if !isCase(key) {
			continue
		}
		baselines++
		kept, err := declared.Baseline(archived[key])
		require.NoError(t, err, key)
		require.Equal(t, archived[key], kept, key)
	}
	require.Positive(t, baselines)
}
