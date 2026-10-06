package check

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

// A realization is named by its id as well as its name: both are required, and neither is shared.
func TestARealizationIsNamedByItsIdAndItsName(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	again := proto.Clone(m.GetRealizations()[0]).(*umpirespb.Realization)
	again.Name = "another"
	m.Realizations = append(m.Realizations, again)
	err := ir.Validate(m)
	require.ErrorContains(t, err, "two realizations with id temporal.features.nexuscaller.NexusRealization.asyncNexus")
	require.ErrorContains(t, err, admRealizationAt)

	m = proto.Clone(load(t)).(*umpirespb.Model)
	m.GetRealizations()[0].Id = ""
	err = ir.Validate(m)
	require.ErrorContains(t, err, "realization asyncNexus has no id")
	require.ErrorContains(t, err, admRealizationAt)

	// A machine the Model does not declare leaves the classes unread, and reports no class for it.
	m = proto.Clone(load(t)).(*umpirespb.Model)
	m.GetRealizations()[0].Machine = "nope"
	err = ir.Validate(m)
	require.ErrorContains(t, err, "realization asyncNexus: no machine nope")
	require.NotContains(t, err.Error(), "binds no action")
	require.Len(t, problems(err), 1)
}

// The table a producer of Cases reads is the check table: the machine's rows with its hole rows as
// unknown pairs, never as disabled ones, and the same identity. It carries the state fields and the
// Abstraction Claims beside them, which enter no fingerprint.
func TestTheRealizersTableIsTheCheckTableWithFieldsAndClaims(t *testing.T) {
	m := lifted(t, "declarations")
	realizer, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	checking, realizing := bind(m, DefaultScope), realizer.b
	holed := 0
	for _, decl := range m.GetMachines() {
		mm := checking.machines[decl.GetName()]
		checked, err := checking.claimed(mm)
		require.NoError(t, err)
		realized, err := realizing.claimed(realizing.machines[decl.GetName()])
		require.NoError(t, err)
		var unknown, holes []string
		for _, u := range realized.Unknown {
			unknown = append(unknown, u.Row)
		}
		for _, h := range mm.Holes {
			holes = append(holes, h.Row)
			holed++
		}
		require.Equal(t, holes, unknown, decl.GetName())
		require.Equal(t, checked.Rows, realized.Rows)
		require.Equal(t, checked.TargetFingerprint(), realized.TargetFingerprint())
		require.Empty(t, checked.FieldValues(realized.States[0]), "a check reads no state fields")
	}
	require.Positive(t, holed, "the fixture declares hole rows")

	realizer, err = NewRealizer(load(t), DefaultScope)
	require.NoError(t, err)
	nexus := realizer.b
	table, err := nexus.claimed(nexus.machines["nexusSystem"])
	require.NoError(t, err)
	const field = "temporal.features.nexuscaller.system.state-field.nexusSystem."
	require.Equal(t, []interp.Atom{{ID: field + "phase", Value: "succeeded"}, {ID: field + "attempts", Value: "1"},
		{ID: field + "scheduleToClose", Value: "unset"}, {ID: field + "scheduleToStart", Value: "unset"},
		{ID: field + "startToClose", Value: "unset"}, {ID: field + "nexusProduct", Value: "succeeded"}},
		table.FieldValues("succeeded-1-unset-unset-unset"))
	require.Contains(t, table.Claims(), interp.Claim{Member: "temporal.features.nexuscaller.system.action.nexusSystem.reply-handlerError-false",
		Action: "temporal.features.nexuscaller.system.action.reply", Field: "reply", ClassName: "handlerError (retryable := false)", Example: "BadRequest"})
}

// A Realizer reads the Behavior Fingerprint of each table it binds once, and it is the table's own.
// A changed copy of a bound table is another table to it, fingerprinted afresh, and the bound
// table keeps its own.
func TestARealizerFingerprintsEachBoundTableOnce(t *testing.T) {
	realizer, err := NewRealizer(load(t), DefaultScope)
	require.NoError(t, err)
	tables := 0
	for name := range realizer.b.machines {
		table := realizer.b.subject(name).table
		if table == nil {
			continue
		}
		tables++
		want := table.TargetFingerprint()
		require.Equal(t, want, realizer.TargetFingerprint(table), name)
		require.Equal(t, want, realizer.TargetFingerprint(table), name)
		changed := *table
		changed.Facts = append(slices.Clone(table.Facts), "unbound")
		require.NotEqual(t, want, changed.TargetFingerprint(), name)
		require.Equal(t, changed.TargetFingerprint(), realizer.TargetFingerprint(&changed), name)
		require.Equal(t, want, realizer.TargetFingerprint(table), name)
	}
	require.Positive(t, tables)
	require.Len(t, realizer.prints, 2*tables, "one fingerprint for each bound table and each changed copy")
}

// A Realizer binds only a Model admission lets through, and gives only the Queries that Model
// declares, by the key Check gives each one's receipt: a Query it binds always has a receipt.
func TestARealizerGivesOnlyTheQueriesOfAnAdmittedModel(t *testing.T) {
	m := load(t)
	unadmitted := proto.Clone(m).(*umpirespb.Model)
	unadmitted.GetRealizations()[0].Name = ""
	realizer, err := NewRealizer(unadmitted, DefaultScope)
	require.Nil(t, realizer)
	require.ErrorContains(t, err, "a realization has no name")

	realizer, err = NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	answered := 0
	for _, receipt := range Check(m, DefaultScope).Receipts {
		if receipt.Subject != QuerySubject {
			continue
		}
		answered++
		declared, err := realizer.Declared(receipt.Key)
		require.NoError(t, err)
		require.Equal(t, receipt.Key.Name, declared.Query.GetName())
		require.Equal(t, receipt.Position, interp.Where(declared.Query.GetPosition()))
		bound, err := realizer.Find(receipt.Key)
		require.NoError(t, err)
		require.Equal(t, receipt.Key.Name, bound.Name)
		require.Equal(t, receipt.Limits, bound.Limits)
	}
	require.Equal(t, len(m.GetQueries()), answered)

	// A declared Query copied under another name is no Query of the Model: no key names it.
	key := ClaimKey{Family: "temporal.features.nexuscaller.system", Owner: "nexusSystem", Name: "syncCompletion"}
	_, err = realizer.Find(key)
	require.NoError(t, err)
	for _, c := range []struct {
		name string
		key  ClaimKey
		want string
		at   string
	}{
		{"another name", ClaimKey{Family: key.Family, Owner: key.Owner, Name: "syncCompletionAgain"}, "no Query syncCompletionAgain", m.GetSource()},
		{"another machine", ClaimKey{Family: key.Family, Owner: "nexusProduct", Name: key.Name},
			"query syncCompletion runs on nexusSystem of temporal.features.nexuscaller.system, not on nexusProduct of temporal.features.nexuscaller.system",
			"model/temporal/features/nexuscaller/system/System.scala:"},
		{"another family", ClaimKey{Family: "temporal.shared.worker", Owner: key.Owner, Name: key.Name},
			"query syncCompletion runs on nexusSystem of temporal.features.nexuscaller.system, not on nexusSystem of temporal.shared.worker",
			"model/temporal/features/nexuscaller/system/System.scala:"},
		{"no name", ClaimKey{Family: key.Family, Owner: key.Owner}, "no Query ", m.GetSource()},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, find := range []func() error{
				func() error { _, err := realizer.Find(c.key); return err },
				func() error { _, err := realizer.Declared(c.key); return err },
			} {
				err := find()
				require.ErrorContains(t, err, c.want)
				var located *interp.Error
				require.ErrorAs(t, err, &located)
				require.Contains(t, located.Position, c.at)
			}
		})
	}
}
