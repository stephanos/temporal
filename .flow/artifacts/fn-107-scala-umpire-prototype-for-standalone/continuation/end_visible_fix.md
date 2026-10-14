# Decision: non-Boolean `ends` / `visible` / `visible_outcomes` results in goir

Status: proposed and unrun. No source was edited.

## Finding (verified against reviewed task-15 source, hash `bd9bc488…` of `model/scalav2/goir/machine.go`)

- **`ends`** (`goir/machine.go:561-573`): the Model's `ends` lambda is evaluated per state with `in.Apply(end, []Value{s})` (`:566`), and then `if v.Bool` (`:570`) reads the result without checking `v.Kind`. An `Int`, `Text`, enum or record result silently reads as "not an end".
- **`visible` / `visible_outcomes`** (`goir/machine.go:671-696`): the `sees` closure returns `b.Bool` from `in.Call(function, …)` (`:677-678`) without a Kind check. A non-Boolean result reads as "not seen". That silently turns a visible stutter into an accepted invisible one, which is exactly the R3 failure "visible-output stutter never becomes successful refinement".
- **Admission does not catch either case.** `Validate` walks the `ends` expression and checks only the arity of the `visible`/`visible_outcomes` functions (`goir/load.go:637-647`). It does not check result types.
- **Same pattern elsewhere, deliberately out of scope.** `Call` reads a precondition with `ok.Bool` unchecked (`goir/eval.go:238-240`). Recorded as an observation only (see Scope).

## Decision: task 3 owns a narrow, test-first guard

Reasons:

1. **Task 3 depends on these values directly.** Its refinement receipts compare `RefineTables` results with `mm.Refinement`/`mm.Rejected` (`goir/machine.go:605-642`). Its monitor `AtEnds` reading and progress receipts consume `Table.Ends`. A silent `false` would make both disagree without any error, and no key-level engine change can detect that.
2. **The path is inside `model/scalav2/goir/**`,** which is task 3's Touches. It is also inside task 16's Touches (`model/scalav2/**`), so it cannot run in the new `model/go/umpire/**` prerequisite or concurrently with task 16. Task 3 already runs after task 16.
3. **Task 15 is done and reviewed (R2).** Reopening it for a guard of about 10 lines is heavier than a bounded red/green step inside task 3, and task 15's acceptance stays true: tables and fingerprints for well-typed Models are unchanged.
4. **It is a malformed-IR case.** Under the parent spec ("Unsupported source constructs and malformed IR are definition/admission errors, never semantic holes"), the correct outcome is a located error, not a new semantic rule. No GOV-02 exception is involved.

## Red (write first; must fail on current source)

Add to `model/scalav2/goir/interpret_test.go`, following the existing `proto.Clone(lifted(t, "declarations"))` mutation pattern (`interpret_test.go:232-299`). Use the fixture via `lifted` and never pin `Position.file`.

```go
func TestAClaimFunctionMustReturnABoolean(t *testing.T) {
	intLit := func(at *modelirspb.Position) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{Kind: &modelirspb.Value_Int{Int: 3}}}}
	}
	for name, mutate := range map[string]func(m *modelirspb.Model){
		"ends": func(m *modelirspb.Model) { /* disk's ends lambda body → intLit */ },
		"visible": func(m *modelirspb.Model) { f := function(m, "disk.visible"); f.Body = intLit(f.GetBody().GetPosition()) },
		"visibleOutcomes": func(m *modelirspb.Model) { f := function(m, "disk.visibleOutcomes"); f.Body = intLit(f.GetBody().GetPosition()) },
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "declarations")).(*modelirspb.Model)
			mutate(m)
			_, err := Build(m)
			var located *Error
			require.ErrorAs(t, err, &located)
			require.NotEmpty(t, located.Position)
			require.ErrorContains(t, err, "not a Boolean")
		})
	}
}
```

Expected red on current source:
- `ends`: `Build` succeeds and disk's `Table.Ends` is empty.
- `visible`: `Build` succeeds with `disk.Rejected == nil` and the refinement accepted.
- `visibleOutcomes`: `Build` succeeds silently.

`Value_Int{Int: …}` matches `ir.proto:291` (`int64 int = 2`).

Control (already green, keep it): `TestVisibleProjectionOfARefinement` (`interpret_test.go:279-299`) and the admitted fixture tables stay byte-identical.

## Green (minimal guard in `goir/**` only)

- In `startsAndEnds` (`machine.go:566-572`): after `Apply`, return `errorAt(decl.GetEnds().GetPosition(), "%s: ends is %s at %s, not a Boolean", decl.GetName(), v.Key(), s.Key())` when `v.Kind != BoolValue`.
- In `seen`'s `sees` closure (`machine.go:673-679`): return `errorAt(mm.Decl.GetPosition(), "%s: %s is %s for %s, not a Boolean", mm.Decl.GetName(), function, b.Key(), v.Key())` when there is no error and `b.Kind != BoolValue`.
- Keep evaluation errors, including `*Hole`, propagating exactly as today. Whether a hole inside `ends`/`visible` means incomplete evidence is the S6 decision task 3 records. The guard neither changes it nor wraps it.
- Preserve existing comments. No change to `Validate`, `eval.go`, tables for well-typed Models, fixture expectations or fingerprints.

## Scope limits

- Not included:
  - a general result-type checker in `Validate`;
  - the `requires` precondition reading (`eval.go:238-240`);
  - `Apply` argument-count checking;
  - claim functions (`holds`, monitor `next`/`violated`/`after`, progress `from`/`to`), which task 3's adapter already Kind-checks at its own boundary (control-matrix rows 8b and 8c).
- If the reviewer wants the `requires` sibling covered, that is a separate bounded follow-up with its own red test, not part of this guard.
