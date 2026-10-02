package leannames_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/leannames"
)

func TestExtractSpecNamesReadsOnlyBacktickedModelNames(t *testing.T) {
	t.Parallel()

	document := "" +
		"Prose naming Umpire.Property without backticks is not a citation.\n" +
		"The module is `Umpire.Property` and its check is `Umpire.Property.check`.\n" +
		"Paths such as `model/Umpire/Property.lean` and bare `Umpire` are not names.\n" +
		"Neither is `go.temporal.io/server/common/testing/testpilot`.\n" +
		"`Shared.CorrelatedObligation` and `Testpilot.Authoring` and `Temporal.Feature` count.\n"

	names := leannames.ExtractSpecNames(document)
	got := make([]string, 0, len(names))
	for _, name := range names {
		got = append(got, name.Name)
		require.Empty(t, name.PlannedSpec)
	}
	require.Equal(t, []string{
		"Umpire.Property",
		"Umpire.Property.check",
		"Shared.CorrelatedObligation",
		"Testpilot.Authoring",
		"Temporal.Feature",
	}, got)
	require.Equal(t, 2, names[0].Line)
}

func TestExtractSpecNamesScopesAPlannedMarkerToItsOwnBlock(t *testing.T) {
	t.Parallel()

	document := "" +
		"- **VER-01.** `Temporal.Verify` is the optional checker integration.\n" +
		"  *(planned: fn-24-lean-native-verification-receipts-and)*\n" +
		"- **VER-02.** `Umpire.Property` opts in explicitly.\n" +
		"\n" +
		"A following paragraph cites `Umpire.Query`.\n"

	planned := map[string]string{}
	for _, name := range leannames.ExtractSpecNames(document) {
		planned[name.Name] = name.PlannedSpec
	}
	require.Equal(t, map[string]string{
		"Temporal.Verify": "fn-24-lean-native-verification-receipts-and",
		"Umpire.Property": "",
		"Umpire.Query":    "",
	}, planned)
}

func TestSpecIsOpenReadsTheFlowRecord(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	write := func(id, status string) {
		require.NoError(t, os.WriteFile(
			filepath.Join(directory, id+".json"),
			[]byte(`{"id":"`+id+`","status":"`+status+`"}`+"\n"), 0o600))
	}
	write("fn-24-open-owner", "open")
	write("fn-25-closed-owner", "closed")

	open, err := leannames.SpecIsOpen(directory, "fn-24-open-owner")
	require.NoError(t, err)
	require.True(t, open)

	open, err = leannames.SpecIsOpen(directory, "fn-25-closed-owner")
	require.NoError(t, err)
	require.False(t, open)

	// A planned term whose owner does not exist has nobody delivering it.
	open, err = leannames.SpecIsOpen(directory, "fn-99-absent-owner")
	require.NoError(t, err)
	require.False(t, open)
}

func TestUnresolvedReportsBothRefusals(t *testing.T) {
	t.Parallel()

	modelRoot := t.TempDir()
	writeLean(t, modelRoot, "Umpire/Property.lean", "namespace Umpire\nstructure Property where\n  id : String\nend Umpire\n")
	index, err := leannames.Build(modelRoot)
	require.NoError(t, err)

	specsDirectory := t.TempDir()
	for id, status := range map[string]string{
		"fn-24-open-owner":   "open",
		"fn-25-closed-owner": "closed",
	} {
		require.NoError(t, os.WriteFile(
			filepath.Join(specsDirectory, id+".json"),
			[]byte(`{"id":"`+id+`","status":"`+status+`"}`+"\n"), 0o600))
	}

	document := "" +
		"`Umpire.Property` and `Umpire.Property.id` both exist.\n" +
		"`Umpire.Property.missing` does not.\n" +
		"- A rule citing `Umpire.Future` *(planned: fn-24-open-owner)*\n" +
		"- A rule citing `Umpire.Abandoned` *(planned: fn-25-closed-owner)*\n" +
		"- A rule citing `Umpire.Orphan` *(planned: fn-99-absent-owner)*\n" +
		"- A planned rule also citing `Umpire.Property` *(planned: fn-25-closed-owner)*\n" +
		"- A planned rule citing `Umpire.Property.absent` *(planned: fn-24-open-owner)*\n"

	unresolved, err := leannames.Unresolved(
		index, leannames.ExtractSpecNames(document), "DOC.md", specsDirectory)
	require.NoError(t, err)
	// Line 6 is silent because the name resolves, even though its block's owner is
	// closed: a planned marker exempts names the tree does not have, and must not start
	// failing the ones it does. Line 7 is silent because its owner is still open, which
	// is the whole point of the exemption.
	require.Equal(t, []string{
		"DOC.md:2: Umpire.Property.missing names no module, namespace, or declaration",
		"DOC.md:4: Umpire.Abandoned is marked planned under fn-25-closed-owner, which is not open",
		"DOC.md:5: Umpire.Orphan is marked planned under fn-99-absent-owner, which is not open",
	}, unresolved)
}
