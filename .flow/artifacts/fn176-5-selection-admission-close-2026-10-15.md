# fn-176.5 reopened correction: crash-selection admission (2026-10-15)

Completion finding `finding-b09c3c9a375db075908fa7df12d439b8` (duplicate/unknown operation
selections reached guest execution and finalized as infrastructure). Branch `fn176-close`.
Historical .5 acceptance/evidence remains at
`fba4865efd:.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.5.md`.

## Commits
- e4fb5adca6 test: reproduce crash selection admission failure (RED)
- 01c419c40a fix: admit crash selections against prior pending context
- e41f6841da fix: require crash context exactly when crash limits are set (review P2)
- fa92d0fb7c refactor: share StorageCrashContext deep clone (review P3)
- 9c854e5ee2 refactor: name chunked storage inputs explicitly (review P3 x2)
- 75c13cf844 fix: keep crash-context comparison within WASM pure imports (frozen-run finding)

## RED/GREEN
- Rust RED at e4fb5adca6: `cargo +1.94.1 test --release --locked --test engine
  invalid_storage_configuration_and_images_refuse_before_guest_start` exit 101 —
  `duplicate-selection` started the guest (execute_nanos 14500) and terminated
  `infrastructure` ("crash selection contains a duplicate operation").
- Go RED at e4fb5adca6 (helper built from that tree):
  `TestStorageCrashSelectionAndUnsolicitedCrashRefusedBeforeExecution` exit 1
  (duplicate/zero had no selection admission; generic "execution configuration").
- Context-presence RED (before e41f6841da): `TestStorageCrashContextEvidenceRefusal`
  v1-with-limits and v3-without-limits accepted (exit 1).
- Architecture RED (01c419c40a..9c854e5ee2): `TestWASMArchitecture` exit 1,
  pure-import `reflect` in wasi/storage_evidence.go and wasi/storage_selection.go.
  Found by frozen attempt 1 (`/home/agent/wt/fn176-qual-20261015T1716`, ordinary
  exit 2, superseded); GREEN after 75c13cf844.

## Source gates (GREEN)
Stock Go 1.27.1 (`.toolchain/stock/878c9b2935e2c9e6`), GOFLAGS=-p=4, Rust 1.94.1, CARGO_BUILD_JOBS=4.
At 9c854e5ee2 (helper 2f9f8a02...): rust fmt 0; rust test --release --locked 0
(197 unit + 153 integration/other passed, 2 ignored); clippy --all-targets -D warnings 0;
`make test-conformance` 0; `make -C tools/gomad validate` 0; wasi
`-run 'TestStorage|TestSQLiteStorage|TestHelper|TestImplementationIdentity|TestVolume|TestNamespace'` 0 (215s);
cluster + qualification + backend packages 0 (521s); `make lint-code-gomad-wasm`
base 7e2a9aa1c3: 0 issues. At 75c13cf844: TestWASMArchitecture, wasi
`^TestStorageCrash|^TestImplementationIdentity|^TestStorageEvidence|^TestStorageOwner` 0, vet 0, lint 0 issues;
Go model identity pin sha256:ec2e2bd80e1fb680144ee47003daf78327febd145e04874507f6b478ffd148ad.

## Frozen qualification (GREEN)
Required in full: the fix changes helper storage transport and the Go model identity,
so the source digest and helper SHA bound into every result changed.
Driver `/home/agent/wt/fn176-qual-20261015T1753/run-qualification.sh`
(SHA256 c445a1410e09bb4f2bf39f2614e5cc407210804175989127669c05759f7e36d5), adapted from the
accepted Uq7Ppu 84-stage driver: repo/evidence roots, local scratch, GOTMPDIR, -p=4,
main-checkout lint binaries, shared content-addressed GOCACHE (tests -count=1). Frozen
source 75c13cf844e4fb348978f9ac22dc3f1820e1d940, lint base a1f0be7e96e2746cb9c51598766a5ae96d8552c2.
84/84 stages exit 0; PASS 2026-10-15T19:07:17Z. Source digest
sha256:a8a9bdb06b04d8e044213e817041904ac17f189f4973cac32187bee5e102b0a6; admitted helper
70e8dbe66dbc239e98ed6e7d19d20ef29a414a54a7946631695e6814f4a8ad48.
- Five primary successes qualified (fixture x2, sqlite-durable-recovery, onebox,
  multi-instance); five canaries unqualified as expected.
- TestTemporalCommittedWorkflowFreshGuestCrashReopen PASS (330s).
- Primary SQLite catalogue (incarnation 2) carries Context with 4 pending operations.
  All four WASM SQLite selected states qualified, selections [35..38], [35..37], [], [35,36];
  result SHA256 613b46ba..., 64fb24ef..., 555c554b..., 27699a5e...;
  TestActualSQLiteSelectionCatalogue PASS. Native execution remains deferred (fn-128/fn-149).

## Independent review
`flowctl claude impl-review ... --base 7e2a9aa1c3 --spec claude:claude-opus-5-5:high`, same receipt:
round 1 NEEDS_WORK (P2 context presence not tied to limits; 3 P3) → round 2 SHIP (all fixed)
→ round 3 SHIP after 75c13cf844.
