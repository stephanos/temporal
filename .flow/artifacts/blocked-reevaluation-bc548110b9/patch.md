# fn-110 blocked-task reevaluation

Frozen source is `bc548110b9321df59d757d0e0e5c0fea464c002b`. All three tasks can resume retained source acceptance on this host. Their old native-host and 642-byte size blockers are superseded. No new implementation defect was established by this inspection. Completion still needs source evidence and review reconciliation, and tasks .3/.4 retain their predecessor edges. Task .5 remains todo and is dependency context only.

Status recommendations below are conceptual dispositions for the owner's next source-work decision. This reevaluation changes no lifecycle state and recommends no destructive reset of historical task evidence.

The missing archive is an immediately remediable local prerequisite. A read-only GET of the official pinned URL succeeded and produced the exact descriptor SHA-256. No archive was cached during this research-only task. This disproves the inherited assumption that go.dev is unreachable in this session.

The host spawn explicitly requested `gpt-6-astra/high` and retained that model in its dispatch. The judge was unavailable (`no_key`); this did not select a session-model fallback. Actual executing model metadata is unavailable. This is a blocker reassessment, not a formal implementation review or completion receipt.

| Task | Retained acceptance and inspected implementation | Present cause and recommended disposition |
| --- | --- | --- |
| **fn-110.2**, blocked, depends on completed .1 | R2/R6 retain all three scheduler extractions, comments, lock/evaluation/draw order, protected upstream timer-wake and arrival machinery, timer-presence result, size assertion, allowlists, source preservation, ordinary coverage and source review. `runtime/overlay/src/runtime/gomad.go:1363,1536,1591` contains `gomadResumeSyscall`, `gomadCheckDeadTime` and `gomadSimulationTimeQuiescenceChanged`. The patch retains the hooks at lines 255/299, transport accounting, `copyenv`/write guards and `sizeof_test.go`. The registered `locked_syscall` fixture covers locked idle-P resumption, timer-first and arrival-first order and arrival during quiescence (`internal/gomadtool/conformance/runtime_campaign.go:336`, `testdata/locked_syscall/main.go`). | **Resumable source-evidence/review backlog.** The 33,294 versus 32,652 U3 measurement is the exact waived 642-byte excess; do not invent another extraction or change the comparator. Current patch and overlay still match the compact checkpoint. Reconcile retained preservation/static receipts with the frozen input set, refresh required source checks, and obtain source completion review. Recommend return to todo/resumable source verification, not done. No native host is required for that work. |
| **fn-110.3**, blocked, depends on .2 | R3/R5/R6 retain pristine upstream `crypto/rand/rand.go`, both reader assignments and their comment, exact linknames/signatures, matching source constraints, lazy environment reset and write hooks, exact descriptor source sets, archive collision audit, generated validation and separate patch/overlay measurements. `runtime/overlay/src/crypto/rand/gomad.go:12` has both `Reader = gomadio.RandomReader()` and `rand.SetTestingReader(Reader)`. `syscall/gomad_env_unix.go` preserves `unix || (js && wasm) || plan9 || wasip1`; `gomad_unix.go` preserves `unix`. Four declarations now exist because `gomadIOProfileEnabled` was admitted after planning. The checked patch has no crypto/rand section. | **Predecessor-source acceptance plus evidence reconciliation.** Historical task3 evidence records implementation and SHIP, exact source selection and developmental entropy/environment/write comparisons. It cannot alone establish current acceptance. No missing relocation was found. After .2 source acceptance, refresh descriptor/generated validation and archive collision/pristine-source evidence, then reconcile the current review scope. Recommend todo with the existing .2 edge, or preserve blocked only with that precise retained-source dependency. Neither transferred native gates nor the superseded size paragraph justify blocking work. |
| **fn-110.4**, blocked, depends on .3 | R4/R5 retain canonical U1, repeated exact regeneration, zero-fuzz U1/U3 file equivalence, descriptor-derived archive/path, checksum/version rejection, malformed/unlisted/fuzz/no-change negatives, exact allowlists, generated validation and both-source-set static coverage. `patch_regenerate.go:21` pins one context line without a CLI option. `patch_test.go:379,404,421,465` supplies real pinned regeneration, checksum rejection, context equivalence and descriptor-driven cache lookup. Existing tests run with stock host Go. | **Predecessor-source acceptance plus refreshable pinned-evidence backlog.** Two current real-pinned tests skipped solely because `.toolchain/downloads/go1.27.1.src.tar.gz` is absent. The exact official archive is reachable and verified, so this is setup work rather than an external/native blocker. The later fn-109.48 cleanup changed the regenerator after compact checkpoint evidence; its focused cleanup tests pass now, but rerun real pinned tests on the final source. Recommend todo with existing .3 edge after evidence work is assigned; do not mark complete from synthetic PASS or pinned SKIP. |

The controlling contract is the [native transfer manifest](../native-scope-transfer-2026-10-07.md), especially the fn-110 row and dependency section, plus the [parent spec's owner waiver](../../specs/fn-110-gomad-minimize-the-runtime-patch.md#owner-waiver-of-the-current-u3-size-deficit-2026-10-07). Task-local Description/Acceptance amendments are authoritative over older Done summaries. Native R2-R7 slices belong to fn-149.1/.2/.4 and fn-128.1/.4/.7. Source-owned byte equivalence, pinned checks, lint, preservation, review and non-native measurements remain required. Full native `test-host` is transferred; portable failures inside it remain source-owned. The exact waiver does not cover future growth or all R8 requirements, and the measured U3 result remains an increase.

## Evidence retained and refreshed

The complete source diff from the amendment's `d28d67c40ce74dd8886cf11b36fe7d2ddaf23675` to frozen HEAD under `tools/gomad3` contains only README changes. There is no patch, overlay, test, descriptor or regenerator change across that range. Current source measurements are:

| Input | Measurement |
| --- | --- |
| Checked U1 | 24,117 bytes, 692 lines, SHA-256 `4b13066eeacc5e9d33a5ada7d3924ff346786ccaf578b9856e1be5f41a94c6f9` |
| Retained current U3 | 33,294 bytes, 1,026 lines, SHA-256 `ce45f4c0bcd45cd24ee4598e3ba18fb8106f50a705a2f2652067964305572516` |
| Original U3 comparator | 32,652 bytes, 1,007 lines, SHA-256 `950063a87d63cb01dada2e6ef232fdeb4acc52f71a008e2158885666669f80bc` |
| Current patch source inventory | 20 upstream files, 275 added lines, 60 deleted lines |
| Descriptor exact sets | 20 patch paths and 79 overlay paths |
| Current overlay | 763,849 bytes, 20,763 lines |

U1 and current inventory figures were read directly from frozen checked inputs. U3 comes from the retained [compact closure](../fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/closure.json), whose U1 digest matches current source. This reassessment did not freshly regenerate U3. The retained U1 representation saving is 9,177 bytes versus U3; it is separate from the waived U3 excess.

The compact [source review](../fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/source-review.md) found no introduced issue in seven admitted product paths. It bound 30 alpha-renaming sites, all 20 patched files plus overlay, unchanged 20/79 sets and seven derived fixture pointers. Its scope was a source-progress commit, explicitly not task completion. The [conductor verification](../fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/root-verification.md) retains source validation, package architecture, failed native controls and the limits of changed-only lint. The [task3 relocation evidence](../fn-110-gomad-minimize-the-runtime-patch/task3-relocation-evidence.md) and [task4 representation evidence](../fn-110-gomad-minimize-the-runtime-patch/task4-canonical-patch-evidence.md) remain historical developmental evidence with their original toolchain identities. Their linux/arm64 shims must not be revived or described as native qualification.

Fresh diagnostic commands used the stock Go binary at `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`, its adjacent gofmt on PATH, `GOTOOLCHAIN=local GOWORK=off GOEXPERIMENT=nogreenteagc`, and explicit `go -C tools/gomad3`:

```sh
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/bin:/bin
export GOTOOLCHAIN=local GOWORK=off GOEXPERIMENT=nogreenteagc
go -C tools/gomad3 test -tags test_dep -count=1 -run 'TestRegenerate|TestMaterialize|TestValidate|TestPinned|TestDrawInventoryRejects|TestSeededDrawInventoryRejects|TestHostClockInventoryPins|TestGoroutineCreationInventoryRejects' -v ./toolchain
go -C tools/gomad3 test -tags test_dep -count=1 -run '^(TestPatchCleanupRegenerate|TestBuildRejectsOverlayCollisionBeforePatching|TestEnsureDownloadsAndReusesVerifiedArchive|TestEnsureChecksumFailureDoesNotPublishPartialArchive)$' -v ./toolchain
patch --dry-run --batch -V none -p1 -F 0 -d /tmp/fn110-source-size-CNRyRbvw/final/a -i /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3/toolchain/runtime/go1.27.1.patch
```

The first command exited 0 in 0.612 package seconds, with 17 top-level PASS and two SKIP. `TestRegenerateMatchesCheckedPatchForPinnedArchive` and `TestPinnedContextRepresentationsMaterializeIdenticalSource` both named the missing cached archive. The second exited 0 in 0.156 package seconds, with four top-level PASS including all six cleanup subcases. The dry-run exited 0 on all 20 current patched files, using retained extracted pristine members. That dry-run does not establish full U1/U3 equivalence or archive collision absence. An initial command relying on the tool's working-directory argument failed before tests because the login shell ran at the repository root; explicit `go -C` corrected it. No source defect is inferred from that setup error.

## Missing input and the minimal route

`tools/gomad3/toolchain/version/version.json:4` pins `go1.27.1.src.tar.gz`, URL `https://go.dev/dl/go1.27.1.src.tar.gz`, SHA-256 `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`. The canonical downloads directory is absent. Searches found no cached source archive in the inspected repository, `/tmp` and Go-cache locations. They did find partial source extractions in `/tmp/fn110-source-size-CNRyRbvw` and `/tmp/fn110-alignment-1TWNKL9O`, plus stock distribution sources. The first-baseline extraction is only 924 KiB and is documented as selected patched members plus VERSION, so it cannot replace the complete verified archive. Stock distribution source and repacked tarballs likewise cannot satisfy the archive-byte checksum contract.

Read-only `curl -I -L --max-time 15` returned HTTP 200 for both the descriptor URL and its official `dl.google.com` redirect, with content length 35,109,201. This command then streamed the complete archive without a file write:

```sh
set -o pipefail
curl --fail --silent --show-error --location --max-time 45 https://go.dev/dl/go1.27.1.src.tar.gz | sha256sum
```

It exited 0 and printed the exact pinned digest above. The minimal implementation-phase route is to cache those verified bytes at the descriptor-derived `.toolchain/downloads/go1.27.1.src.tar.gz`, preserve every pin, then rerun the two real-pinned tests and retain PASS rather than SKIP. This requires neither patched-runtime compilation nor a qualified host. Download/cache publication was outside this research assignment and was not performed.

After caching, run the focused commands above and reuse the existing [source-inventory fixture](../fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/compact_checkpoint_test.go) through its [Go overlay](../fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/go-overlay.json):

```sh
go -C tools/gomad3 test -tags test_dep -count=1 -run '^TestRegenerateMatchesCheckedPatchForPinnedArchive$|^TestPinnedContextRepresentationsMaterializeIdenticalSource$' -v ./toolchain
go -C tools/gomad3 test -overlay /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/go-overlay.json -tags test_dep -count=1 -run '^TestGFieldCompactSourceInventories$' -v ./toolchain
make -C tools/gomad3 validate
```

The inventory fixture materializes the pinned source, adds the checked overlay and checks draw/seeded/clock/goroutine inventories for both descriptor platforms without building the runtime. Ordinary `TestPatchedRuntime*` inventory entrypoints instead use `builtGOROOT` (`clock_inventory_test.go:220`) and skip without `.toolchain/build-key`; that convenience wrapper does not make a native host intrinsically necessary for static acceptance. Retain real archive collision evidence through the existing source helper or verified archived receipt; the fresh synthetic collision test proves rejection semantics only. Reconcile preserved comments, timer/goroutine assertions, source identity and baseline evidence before source review. The conductor owns current lint and broader shared checks; no generator, full gate or native gate was run here.

Keep the collector-file and assembly prohibitions, fixed allowlists, negative tests and existing boundary policy. No further size-driven relocation is needed for the expressly waived candidate. Close .2, then .3, then .4 only after their retained source requirements are evidenced and reviewed. Admit .5 only after that chain, for remaining source measurements/documentation and identity explanations. Native qualifications remain unverified under their deferred owners and receive no PR, push or CI authority from this recommendation.
