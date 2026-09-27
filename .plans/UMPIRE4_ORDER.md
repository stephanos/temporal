# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

fn-88 to fn-92 are queued. fn-83, fn-84, fn-87, fn-85, fn-86, fn-46, fn-33, fn-22, fn-26 and
fn-29 are delivered, each with SHIP implementation and completion reviews; their task receipts in
`.flow/` and the git history carry the details.

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

### Delivery queue

1. **fn-88 — Veil concrete checker as the Umpire search engine**
   ([spec](../.flow/specs/fn-88-veil-concrete-checker-as-the-umpire.md)). Makes Veil's concrete
   model-checker library the engine behind `Umpire.Search`, consuming the `FiniteTable` the
   `machine` command already enumerates, with Properties and Scenarios lowered to monitor automata
   through the lowering the Case Producer uses. Its first task is a developer-machine compatibility
   probe of the checker library under the repository toolchain; `adopt` continues, and
   `defer-incompatible` closes the rest as not applicable and hands the engine question to a
   separate `FiniteTable` to TLA+ exporter spec. Today's `Search.lean` stays as the frozen reference
   engine and differential oracle. The reasoning is sections 1 and 6 of
   [UMPIRE4_DIRECTION](UMPIRE4_DIRECTION.md). It adopts no Veil DSL and no SMT path, and it neither
   runs fn-23's sandboxed gate nor resumes fn-24 or fn-25.
2. **fn-90 — Resolve the intermittent live Testpilot failures**
   ([spec](../.flow/specs/fn-90-resolve-the-intermittent-live-testpilot.md)). Re-measures the
   umpire-run namespace-delete timeout, the Nexus evidence-ordering mismatch and the async-Nexus
   INCONCLUSIVE Run on today's test identities with a repetition harness, then fixes each at its
   cause; a retry is a signature-exact, issue-linked last resort inside the live test and never
   applies to a VIOLATED Run. Done is zero failures over the per-test loops and five consecutive
   green `make umpire-check-live-tests` runs.
3. **fn-89 — One Contract Rule per entity**
   ([spec](../.flow/specs/fn-89-one-contract-rule-per-entity.md)), after fn-90, since both touch the
   Nexus pair Case. A Rule over N instances crosses the wire once with its per-instance values
   instead of N lowered copies; Verdicts stay identical, pinned by a differential test, and only the
   multi-instance pair fixture changes.
4. **fn-91 — Say opaque handle in the Driver contract**
   ([spec](../.flow/specs/fn-91-say-opaque-handle-in-the-driver-contract.md)). A mechanical,
   behavior-preserving rename of the Driver seam's hand-written Go from capability to the opaque
   handle family, with the retired-vocabulary gate holding the old names. Independent of the others;
   land it before fn-79 resumes on the same seam.

4. **fn-92 — Compose entity machines into one Model**
   ([spec](../.flow/specs/fn-92-compose-entity-machines-into-one-system.md)). Adds a `compose`
   command that builds one Model from entity machines with declared action synchronization over a
   reachable-state enumeration, `restrict:` and `extend:` keys that derive machines from a source
   table, and `Workflow` and `Worker` entity modules the Start and Outage use cases share; hosts the
   first cross-entity claims as `verify` Queries. Version one realizes no Case over a composition and
   leaves the caller module, its fixtures, and the canary's pinned Case identity untouched; moving
   the operation entity is the named follow-up. Depends on fn-88, fn-89 and fn-90.

### Carried forward, not specs

- **A field path through a repeated field** (deferred from fn-86 R2): selecting the element the
  observation correlates to the row's entity instance. `Temporal.Case.FieldPath` rejects every
  repeated-field path today.
- **The canary CI job's cold Lean build** (from fn-29): the `canary` job in
  `.github/workflows/umpire.yml` is the first to build Lean targets (`canary-check-case`,
  `umpire-check-evaluation-profiles`) with `cache: false` inside a 30-minute timeout; confirm the
  first CI run fits.

## Gate baselines

Re-measured 2026-09-20 on a four-core, 16 GB cloud session at the fn-86 closeout:

| Gate | This session |
| ---- | ------------ |
| `make umpire-check-regression` | exit 0 after `go clean -cache` -- 590 Lean jobs, the offline checks, **29 passing live identities** |
| `make lint-model` | exit 0, measured 2026-09-26 on a macOS host with `LEAN_NUM_THREADS=1`: the import graph passes with the authoring-path rule and both controlled violations asserted, the declaration linters report no findings, and the final `lake --wfail lint --builtin-only` step reports no warnings. The 163 generated findings were fixed in `umpire-gen-lean-api`, which writes `Method`'s `DecidableEq` and `Repr` instances without instances of its phantom payload types, and declares fieldless messages with no placeholder field and fieldless oneof arms as nullary constructors. The 24 builtin warnings in handwritten modules were fixed at their source: `enum` registers an unused-variables ignore function so its constructor fields are spared as `inductive`'s are, a `machine` attaches its doc comment to the declared Model, `Umpire/Command/Syntax.lean` uses `Level.zero` and `String.trimAscii`, and `Umpire/Value/Encoding.lean` unfolds `encodeNat` with `rw` rather than `simp` |
| `make lint-code` | 0 issues over the changed packages (`GOLANGCI_LINT_BASE_REV=9484405 make lint-code-fast`); the full `make lint-code` is not measurable in a shallow clone with no `main` merge base |

Measured 2026-09-26 on a macOS host at the fn-29 closeout, not a re-measured baseline:
`make umpire-check-regression` exit 0 with **45 passing live identities** (13 of them
`TestTestpilotCanary*`); `make lint-code-fast` red on findings outside the canary (the typecheck
error of a deliberately broken `tools/umpire` testdata fixture and staticcheck findings in untouched
`tests/*.go`); `LEAN_NUM_THREADS=1 make lint-model` also reported an unused `[BEq α]` in
`Umpire/Command/Refinement.lean` (present since 2026-09-19 and missed by the baseline above), since
removed, and the 163 generated findings and the 24 `--wfail` warnings of its builtin-lint step have since been fixed, so it exits 0; `make umpire-check-plan-index` passes
after `.plans/index.json` was resynced with Flow the same day.

**`make lint-code` under-reports when the disk is low.** golangci-lint aborts with
`no space left on device (typecheck)` and still exits with a count — `Issues before processing:
11800, after processing: 1` in the failing case against `14507 -> 161` in a healthy one. Run
`go clean -cache` before trusting this gate.

**`make umpire-check-retired-vocabulary` is the slowest offline gate** by an order of magnitude
(about twenty minutes in a cloud session).

**Environment notes.** On macOS, mise's lean4 `clang` shadows the system one and fails cgo builds
with `stddef.h not found`; `mise.toml` sources `develop/mise-env.sh`, which sets `CC=/usr/bin/clang`
unless `CC` is already set, and the Makefile exports the xcrun clang, so no manual `CC` is needed.
Live tests still need a physical `TMPDIR`
(`TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P)`; the default macOS path traverses the `/var` symlink).
`go vet -tags test_dep` is not the gate: the live suite builds with `-tags 'test_dep integration'`,
which compiles strictly more files. In a cloud session, `mise` is a passthrough shim, so
`/opt/temporal-toolchain/*/bin` must be on `PATH` before any `make` target that uses `lake` or
`protoc`; warm the Go module cache (`go mod download`) before the first gate and discard the
`go.sum` hashes that download adds; and `git fetch origin main` gives the branch a name but not a
merge base, because the clone is shallow. A fresh clone cannot change task status: runtime state
lives in the clone's `.git` common-dir, so every task reads `todo` and `start`, `done` and
`spec close` refuse until they are replayed there.

## Deferred and superseded

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope. On resume it re-plans on fn-85
(entities, actions, sets) and takes the cancel Query and the Testpilot cancel instructions that
fn-85 left out; its re-planning note in Flow lists them, and its `## Scope` records the
cancellation-race behavior fn-86 .5 deleted with the Race prototype (2026-09-20).


**fn-70 — Scheduled canary proof of concept:** deferred by user decision; it was previously
queued after fn-78 as the second model consumer. Resume on an explicit user request. Nothing in
the delivery queue depends on it. On resume it inherits fn-80's `temporal.DeriveProfile`, so a
second model consumer no longer hand-writes a `ProfileSpec`. The bind-and-run helpers landed as
test-local `bindCase`/`runCase` in `tests/testpilot_run_case_test.go` rather than exported from the
fixture package, because exporting them would compile the whole server into a Quick command; a
canary test under `tests/` reuses them where they are. fn-83 extracts the provisioning that forced that
placement into `common/testing/testpilot/temporal/provision` and adds `umpire-run` (both landed);
on resume fn-70 consumes both and re-anchors its catalog entry on an fn-85 canary set, since fn-85
removes the `case` block that replaced the fn-68 Producer.

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Retained scope on resume: fn-78 first, with fn-68, fn-71, fn-72, and fn-73 as transitive
prerequisites.
Nine tasks cover all ten requirements, with a SHIP plan review. Implementation follows fn-77
to serialize shared Producer edits; fn-77 is not a semantic prerequisite.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer proof; fn-73 already owns the live proof that one
Case byte sequence runs against two environment bindings.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.


These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec                                                             | Revisit when                                                                                                   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)      | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family.                                                |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md)         | Optional checker adoption becomes valuable.                                                                    |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md)    | A verification receipt/profile platform is justified.                                                          |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md)  | A second verification backend is justified; caller closure remains historical.                                 |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md)        | Real Claim Assessment evidence supports release governance.                                                    |

fn-88 consumes Veil's concrete checker as a Search engine and leaves the symbolic verification
path, fn-23, fn-24 and fn-25, deferred as above.

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
