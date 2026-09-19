package testpilot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/protobuf/proto"
)

// checkedInFixtures enumerates what is actually stored rather than a list maintained here, so a
// Case added to a Model file is covered the moment its fixture is generated.
func checkedInFixtures(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("testdata")
	require.NoError(t, err)
	var fixtures []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-case.json") {
			continue
		}
		fixtures = append(fixtures, entry.Name())
	}
	require.NotEmpty(t, fixtures)
	return fixtures
}

// TestEveryCheckedInFixtureDecodesPreparesAndCarriesItsIdentity is the one shared admission test.
// Every fixture must decode strictly and carry a complete identity; every fixture whose Profile
// `DeriveProfile` can read must also prepare over unchanged bytes. The per-Case semantic assertions
// -- the outage Deadline, the typed tenfold load, Run isolation, checked Provenance -- stay in
// their own tests, because they are about what one Case means rather than about admission.
func TestEveryCheckedInFixtureDecodesPreparesAndCarriesItsIdentity(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	seenCaseIDs := map[string]string{}
	derivable := 0

	for _, fixture := range checkedInFixtures(t) {
		t.Run(fixture, func(t *testing.T) {
			encoded, err := os.ReadFile(filepath.Join("testdata", fixture))
			require.NoError(t, err)
			source, err := testpilot.DecodeCaseProtoJSON(encoded)
			require.NoError(t, err, "every checked-in fixture decodes strictly")

			require.NotEmpty(t, source.GetCaseId())
			require.NotEmpty(t, source.GetProgram().GetProgramId())
			require.NotEmpty(t, source.GetContract().GetContractId())
			require.Equal(t, int32(1), source.GetVersion().GetMajor())
			previous, duplicate := seenCaseIDs[source.GetCaseId()]
			require.False(t, duplicate,
				"Case ID %q is carried by both %s and %s", source.GetCaseId(), previous, fixture)
			seenCaseIDs[source.GetCaseId()] = fixture

			profile, deriveErr := temporal.DeriveProfile(source, catalog, temporal.Environment{
				Identity:      strings.TrimSuffix(fixture, "-case.json") + "-profile",
				Namespace:     "namespace",
				TaskQueue:     "task-queue",
				NexusEndpoint: "nexus-endpoint",
			})
			if deriveErr != nil {
				// A typed Case's Profile is hand-built beside it; those Cases are prepared by their
				// own tests, which is where the hand-written Profile lives.
				t.Logf("%s carries a Profile DeriveProfile does not read: %v", fixture, deriveErr)
				return
			}
			derivable++

			prepared, err := testpilot.Prepare(source, profile)
			require.NoError(t, err, "a derived Profile prepares the Case it was derived from")
			require.True(t, proto.Equal(source, prepared.Snapshot()),
				"preparation carries the Case bytes unchanged")
		})
	}

	require.GreaterOrEqual(t, derivable, 3,
		"the derivable fixtures are the ones a black-box consumer can run; losing one is a regression")
}

// TestEveryCheckedInFixtureRejectsAMutatedRole pins that admission is over the Case the Profile was
// derived from, not over anything shaped like it.
func TestEveryCheckedInFixtureRejectsAMutatedRole(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	for _, fixture := range checkedInFixtures(t) {
		t.Run(fixture, func(t *testing.T) {
			source, err := testpilot.DecodeCaseProtoJSON(readFixture(t, fixture))
			require.NoError(t, err)
			profile, deriveErr := temporal.DeriveProfile(source, catalog, temporal.Environment{
				Identity: "mutation-probe", Namespace: "namespace", TaskQueue: "task-queue",
				NexusEndpoint: "nexus-endpoint",
			})
			if deriveErr != nil {
				t.Skipf("%s carries a Profile DeriveProfile does not read", fixture)
			}
			require.NotEmpty(t, source.GetProgram().GetRoles())

			mutated := proto.CloneOf(source)
			mutated.Program.Roles[0].RoleId = "temporal.unauthorized-role"

			_, err = testpilot.Prepare(mutated, profile)
			require.Error(t, err)
		})
	}
}

func readFixture(t *testing.T, fixture string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", fixture))
	require.NoError(t, err)
	return encoded
}
