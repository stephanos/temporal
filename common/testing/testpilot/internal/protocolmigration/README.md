# Protocol migration oracle

fn-87 changes the Testpilot protocol without changing any Verdict. This package proves it: every
fixture written before the migration, decoded through a frozen snapshot of the protocol it was
written against and mapped under a declared list of steps, equals the fixture its generator writes
today.

## Baseline

`testdata/baseline/` is frozen. It holds:

- `descriptors.binpb`: the descriptor set of the pre-migration protocol, taken once at commit
  `d9c75773098e5c5fbd3ea01a26fdd30a79a74f6b` from the repository root with protoc 29.5:

  ```bash
  mise exec -- protoc --include_imports --include_source_info \
    --descriptor_set_out=common/testing/testpilot/internal/protocolmigration/testdata/baseline/descriptors.binpb \
    -I proto/internal \
    temporal/server/api/testpilot/v1/case.proto temporal/server/api/testpilot/v1/run.proto
  ```

- `fixtures/`: a byte-for-byte copy of every fixture at the same commit, under its
  repository-relative path: the functional `tests/testcore/testpilot/testdata/*-case.json` and
  everything under `common/testing/testpilot/testdata/case-runtime-conformance/`.

Never re-snapshot or re-copy. A second snapshot taken mid-migration would hide every change made
before it. The repository `.gitignore` ignores `testdata/` directories outside its allowlist, so the
tree was committed with `git add -f`; it is tracked, and being frozen it is never added again.

## What the test checks

`TestBaselineFixturesMapToRegeneratedFixtures` pairs the baseline and regenerated fixture sets one
to one (an added or deleted fixture fails, unless `Added` declares the addition) and, for each pair:

- a Case (`case.json`, `*-case.json`) decodes strictly through the snapshot, is mapped, and
  compared with the regenerated Case after both decode strictly into the generated types;
- `correlated.json` is compared entry by entry the same way for `case`, `runnableCase` and
  `events`, while `expected` and `incomplete` must equal the baseline;
- `expected.json` is the Verdict pin: byte-identical, unless a declared step changes its JSON.

A failure names the fixture and the first differing field, or the step that failed.

## Adding a step

A task that changes the protocol or a fixture appends Steps to `Declared` in `mapping.go`, in the
order the changes land. Each Step names itself and the requirement it implements. Steps run on
the fixture's JSON tree, where every object carries the snapshot message it encoded; steps address
messages by their baseline full names even after an earlier step renamed them.

Reuse the helpers where they fit: `RenameField`, `RenameEnumLiteral`, `RenameMessage`, `DropField` and
`RewriteMessages`. A step that is not a rename validates what it assumes (a derived field equals
its recomputed value, a dropped bound matches a declared loosened bound) instead of discarding
data; `DropField` requires that check.

A task that adds a Case rather than migrating one, such as a new conformance rejection, lists each
new file in `Added` with its requirement. No baseline can pin it, so its generator's validation is its
check; a declared addition that is not regenerated, or that has a baseline, fails.

A step spells the baseline names its change retires. Split those literals (`"Run" + "Status"`) as
the other files the retired-vocabulary scan reads do, so the scan keeps holding the names everywhere
else.
