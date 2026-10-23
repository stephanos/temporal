# Provisional fn-232.13 source consolidation

Candidate branch/worktree: `fn232-retirement`, `/home/agent/wt/fn232-retirement`.
Base: `81660e841106a59ed26f7cb227462d1a85a93a9f`. This is prospective isolated
source work under the milestone provisional-interface policy. Root owns lifecycle,
integration, review, source freeze, resource release and acceptance. Task .13 is
not started or completed by this checkpoint. No build, guest execution, parity
campaign, qualification cycle, CI run, push or PR occurred here.

## Dependency cut and source ownership

The default build is Rust `gomad`. The Go CLI, runner, evidence producers,
installation/backend adapters, process supervisors and old qualification commands
are removed. Their fixture-only directories remain for Rust/frozen-Go references;
the three stale diagnostic-identity baselines are removed with their Go runner.
The simulation-plan vector moves into the retained model cluster testdata.

Retained Go code owns model/choice/clock/cluster/network/volume/World, strict JSON,
protocol generation, runtime source, overlay/toolchain and model conformance.
Pure WASI reference code and fixtures move to `internal/model/wasi`, including the
unchanged GMSTORE3 decoder. Capture and softassert move into `internal/model`;
filesystem and Temporal schema capture move into `internal/support`. There is no
retained source import of the deleted host/runner/evidence packages. The exact
old cluster.go input remains as `internal/model/wasi/testdata/reference/cluster.go.txt`
(SHA256 `6820058471b7d9d054a5ea80c71253f5db798823ec4103e2b454d45e3bc4c67c`),
read by Rust's original transport pin test. Frozen Go source snippets deliberately
keep their historical package/source-layout paths.

Mandatory-workload owner separately writes the narrow projection façade at
`internal/model/wasi/storage_projection.go` and the independent native observer at
`internal/model/cmd/sqlite-native`; those files are not duplicated in this branch.
Its checkpoint must integrate before native SQLite projection compilation.

`make validate` executes all retained model/oracle/toolchain/support Go tests twice,
including formerly unselected network adapter/cluster and broker lifecycle corpus
producers. The helper transcript and namespace/volume transport checks require
actual binaries through explicit environment paths. Rust suite and diagnostic
helper-corpus feature lanes remain explicit; final live-root reach still belongs
to the frozen driver's complete receiving contract. Archcheck refuses default Go
packages outside the retained set as `retired-host-source`. Removed two obsolete
lint exceptions and their deleted-source-specific checks; no waiver was added.

## Cargo workspace and identity rotations

`tools/gomad/rust/{gomad,wasmhost}` are one Cargo workspace, default member `gomad`,
with one `rust/Cargo.lock`, one `rust/rust-toolchain.toml`, and one shared target
(or the explicitly assigned external `CARGO_TARGET_DIR`). No package version,
checksum or original helper dependency-array changed: the lock is the exact
127-package helper lock plus the existing gomad workspace package. The standalone
protocol binary and its process-transport integration tests require explicit
`reference-helper`; `conformance` enables that feature. Ordinary host/release
builds do not build or install a separate helper. Relative source paths,
generator outputs, Rust include paths, build identities, Make/CI consumers and
root source-search exclusions migrate together.

The directory/manifest/lock changes deliberately rotate Rust source identities
before the final freeze. Old golden or command receipts were not rewritten.
The retained Go logical WASI identity changes from
`sha256:5fddc30fd325f3168d32a0084a3e76e3a10b341da7aee86ddcd9f3b2a8b91bc3`
to `sha256:5d3d138dadd52ebfeaee7d45cb5077d4212ebe7131685e37b384364e65186e02`
through extraction; relocation itself does not change its filename-relative
hash. `model-hash.py` reproduces the historical hash from the exact base before
reporting the current hash. Exactly 123 canonical fixture metadata strings
(62 network adapter, 51 network cluster, 10 broker lifecycle) and the Rust WASI
reference pin were rebound. A byte comparison proves every fixture change is
only that exact hash replacement. The network/broker model algorithm sources
remain byte-identical. Actual Go corpus reproduction remains unrun.

## R13 gate and code-size closeout

`make parity` calls `scripts/all-verb-parity` with authenticated bindings and a
fresh work root. The fixed existing executable lanes cover ordinary public verbs,
raw authentication and own-lineage goldens; unavailable Go verbs have actual
availability/refusal observations. Source/compiler/custody inventories and explicit
source-bound role-build receipts are mandatory. The current conformance reference
helper's explicit feature is `conformance`; the compiled matrix stress role is
separate from the fixture-enabled replay stress role. The mandatory CI source job
requires authenticated external provisioning and fails without it; no job ran.

Pending endpoints fail preflight as UNEXECUTED: .11 installation/plan/recover
files must integrate; `stress-parity-lane.py` must adapt the existing same-frozen-
candidate/cycle actual authenticated fn188.5 matrix receipt through the fixed
Rust driver reader, with actual Go availability and role exclusion. It must not
invent a second matrix, weaken fn188 obligations, or fabricate Go support.
`all-verb-README.md` documents the exact binding schema. Eleven negative source
checks validate omission/tamper/role/receipt/refusal handling; they do not prove
runtime parity. R13 and R12 remain unaccepted.

`scripts/source-size.py` and its README provide a reproducible Git-tree counter.
It records exact revisions, complete path lists/hashes, counter version/SHA/Git
revision, per-file bytes/total/nonblank/physical lexical code lines, whole-repo net
bytes/files and subsystem/language/production-test-generated groups. Inline tests
remain classified with their source file; this is explicitly not cloc. The
preexisting Rust helper, new Rust host, retained WASI reference, other retained Go
and support languages are separate. Run final counts only on the final integrated
frozen source against the authenticated Go source revision (frozen source archive
revision `1d7b37f00a74a85399993f1edcad6fe742e778e3` if root confirms that custody as
the comparison baseline). No final comparison was claimed or run here.

## Verification receipts and remaining gates

Source-only commands, each exit 0:

- `cargo +1.99.1 metadata --manifest-path tools/gomad/rust/Cargo.toml --offline --locked --no-deps --format-version 1` (two workspace packages; sole default member gomad).
- `cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check`.
- `make -C tools/gomad --no-print-directory -n build validate test-harness parity lint-rust`.
- `python3 tools/gomad/testdata/parity/all-verb-selftest.py` (11 passed).
- `python3 -m unittest discover -s tools/gomad/scripts/tests -p test_source_size.py -v` (3 passed).
- `python3 .flow/artifacts/fn232-retirement/model-hash.py` (historical reproduction and current measured identity above).
- `gofmt` parsing of edited Go sources; static current local-Go-import and Rust literal-include path checks; exact lock-entry/checksum comparison; exact 123 fixture metadata-only comparison; `git diff --check`.

No compilation or runtime result is implied. Required next gates include integrating
all upstream .11/live/mandatory/profiling/stress/driver checkpoints, reviewing this
cut and all source-identity migrations, then resource-granted retained-Go build/
vet/lint/tests and full Cargo role build/test/Clippy/corpus reproduction. Resolve
any resulting discrepancy without weakening original fixture assertions. Then
freeze all features/identities, run actual every-verb differential parity, all
transferred live/runtime/workload/replay/inspection/stress obligations and the
unchanged 3,600-second final Rust qualification cycle. The frozen Go baseline,
archives and binaries stay external fn232 reference custody until root accepts
those gates; no custody was removed here.

Only this owned worktree remains required for integration/review. Other registered
worktrees and evidence roots belong to root/other workers and were untouched.
The owned temporary hash-calculator file was removed after retaining its reporting
copy. No shared cache/output was removed. Disk last checked: 21 GiB free on the
shared overlay; do not start builds without root's resource grant and a fresh check.

## Independent-review repairs after b64f3dcb89

Both MED source findings were repaired without running Go/Cargo builds or guests.
The all-verb gate now requires the fresh work root outside authenticated trees,
independently copies the original pinned source and cluster-source trees into
owned disposable derived source directories, verifies byte-for-byte input
snapshots before/after copy, and retains their original-to-derived inventories.
Only those derived trees reach reference lanes that create Go wrappers. Original
source bindings still undergo the unchanged final strict authentication. Added
source fixtures prove wrappers alter only the derived copies and a corrupted
copy refuses; all 13 Python gate checks exit 0. This is not actual corpus parity.

Restored the exact original pure invariant tests
`TestHelperModelRejectsRetiredNetworkEvidence`,
`TestNamespaceHelperOutputRefusesInconsistentOwnership` and
`TestNamespacePartialEvidenceResultDoesNotAliasTransport` under model/wasi.
The ownership test sets namespaceOwned=false and requires refusal; the alias
test mutates the returned partial transcript then checks the transport buffer.
These tests have no host Run or guest fixture dependency. They remain unrun
under the root resource hold. Gofmt parsing, diff checks and exact original-test
source comparisons pass; the retained reference logical hash remains 5d3d…6e02.

Upstream source integration remains pending root's stable-head relay. Preserve
live_controller_tests.rs private35 under the moved Rust helper and migrate its
explicit final-driver command to rust/wasmhost/Cargo.toml. The live-broker-test.py
source assertions require authenticated original Go integration/provider test
bytes from configured frozen custody or retained fixtures, never a skipped guard.
Mandatory projection façade/observer checkpoint 537be69421 uses model/wasi;
its helper known-image source seam must survive the Rust helper move.

The e253 repair accidentally overwrote the existing helper_model_test.go while
restoring its missing network refusal test. Re-review caught the deletion of
nine retained pure tests and their helper functions. The correction restores the
complete b64 file byte-for-byte and appends only the exact original missing test.
A read-only function inventory confirms all twelve preexisting test/helper
functions are unchanged and the appended test is the sole addition (ten tests
total). No original test/helper content is removed. Actual Go tests remain unrun
under the root resource hold; gofmt parsing and diff checks exit 0.

## Prospective source integration of 82ba157e16

Merged the root's reviewed main source through 82ba157e16 (source e5ea438c90)
into this owned prospective branch. Main retirement remains unaccepted; no
main checkout was changed. File-location conflicts retain every incoming helper
file in rust/wasmhost, including typed module metadata, complete World/deflate
codecs, private35 live-controller test and their fixtures/licenses. The README
keeps the current .11 installation documentation and single-workspace ownership.
All 2,372 incoming helper source/fixture files map to the new location; the only
137 byte differences are the existing 123 identity metadata rotations plus 14
manifest/build/input-path/pin/docs files. The helper's original 127 lock entries,
dependency lists, exact versions and checksums remain unchanged.

The source-only broker observer tests now read byte-pinned complete original
integration/provider Go test snapshots under testdata/parity/reference, retained
from exact incoming revision 82ba157e16. Their assertion/subtest/call-preservation
checks remain unchanged and both selected Python-only checks pass; no Go probe
or live observer was invoked. Snapshot provenance and hashes are retained in
live-broker-source.json, with independent expected hashes in the source checker.
External actual Go observation still uses its explicit authenticated --source;
it never executes these inert snapshots. New gzip Go oracle and live_backend
fixture modules have exact archcheck fixture classifications; no production
package or dependency boundary was widened.

Integration source checks, exit 0: workspace fmt-check and offline locked
no-deps metadata;13 all-verb negative checks;3 counter checks;2 broker source
assertion-preservation checks; current local-Go-import and Rust include/path-
attribute existence scans; explicit private35 test retention; exact 127 lock-
entry comparison; diff checks. No Cargo/Go compilation, Go unit test, guest,
parity lane, profile, qualification or CI execution occurred.

Follow-up after root grants resources: Go build/vet and focused pure-reference
invariant tests (including all nine preserved helper tests plus restored network
refusal and both namespace tests); complete make validate and retained model
corpus reproduction; Rust ordinary/fixture/helper role build/test/Clippy through
the shared workspace and current source; actual private35 gate with its original
input/bound/output root; complete live-broker observer checks against original
custody; final all-verb local parity and final frozen Rust qualification only
after mandatory precrash image/Go import, fn231 and fn233 source integration.
Mandatory new projection/precrash source remains a later root-owned merge.
No final source freeze or existing evidence reuse for changed identities is implied.

## Prospective source integration of 1b61ab0a52 and real moved-Go validation

Merged root main 1b61ab0a52, including reviewed mandatory precrash-image/native
import/observer/driver source and completed .11 installation corpus, into this
prospective branch. Root now owns .13 in_progress; this branch still supplies no
main retirement, final freeze, qualification or acceptance. fn231/fn233 source
integration remains pending. The new recovery-image test reads the existing
corpus at `../wasi/testdata/volume-conformance/vectors/image-admission.json` from
the real relocated volume package. No overlay or copied package validates it.

The all-verb installation caller forwards the authenticated same-candidate
`stress-release` binding to the required `--role-binary`; its source-bound build
receipt requires helper-stress-jitter and its SHA must differ from release.
This satisfies the behavioral-parity `--installation-role-binary` receiving
contract without executing that supplied role as a helper. The literal role
hashes in the .11 golden must be deliberately reviewed/regenerated for the
final frozen role binaries; no golden was rewritten here.

Root granted only focused pure Go validation. Package inspection found no
TestMain/init in the three selected packages; the five selected root tests use
in-memory recovery/import/enumeration and native request admission, without
helper build/guest execution. Pinned compiler reports go1.27.1 linux/arm64. From
`tools/gomad`, the final exact command was:

```sh
env -u GOROOT -u GOMADSEED -u GOMAD_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAXPROCS=1 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -v -tags test_dep -p 1 -parallel 1 -count=1 -timeout=2m -run '^(TestCrashRecoveryImage(AdmissionCorpus|PreservesOriginalEnumeration|RefusesMalformedPendingState)|TestSQLiteCrashCatalogueUsesOriginalEnumeration|TestActualStorageInputRejectsWrongBoundsAndChangedPayload)$' ./internal/model/volume ./internal/model/wasi ./internal/model/cmd/sqlite-native
```

Authoritative final exit 0: five root PASS, zero FAIL/SKIP across all three real
moved packages, with their original positive/negative subtests preserved. Raw
outputs are `validation/pure-go-check{1,2,3,4}.log`. First compilation exposed an
unused sync import in extracted transport.go; the first shell wrapper returned
cat's status and did not capture Go's status, but its raw output explicitly
records build failure. Second attempt exit 1 exposed the missing math/rand/v2
test import and duplicate := after schema extraction. Those source errors were
fixed; third and final fourth attempts exit 0. No failure log was relabeled.
This validates actual relocated import/facade source, replacing the earlier
temporary-overlay check; it is not actual guest/native projection equivalence
or qualification.

Removing the unused transport import legitimately rotates the current logical
Go WASI reference identity from 5d3d…6e02 to
`sha256:bd45d4831ca0f417b6411a29b70cf1070a779e5f04591a679c180d925b586d15`.
The retained calculator again reproduces the historical 5fdd…91bc3 before deriving
this current identity. Exactly 123 current reference-fixture origin strings plus
the current Rust pin and live Go identity test expectation follow that measured
rotation; every fixture change is an exact identity-only replacement. Original
Go control source, model algorithms and all historical receipts remain unchanged.
Thirteen all-verb negative checks and static diff checks also exit 0. No Rust
build/Cargo command, whole-Go suite, guest, profile, parity lane, qualification,
CI, Flow operation, PR or push ran in this wave. Final integrated code-size
counts remain deferred until the final frozen source.

## Prospective broker-correction integration through 374c62ecce

Root independently reviewed 913bc6d48f as SHIP with no blocking source findings,
including the current measured bd45…6d15 identity, exact 123 metadata-only
rotations and five-root actual relocated-Go validation receipt. That review does
not accept final retirement or qualification.

Merged reviewed root main 374c62ecce into this prospective branch. The moved
broker causal revocation source, linked-cluster coordinator and complete expanded
live-controller tests match the corresponding reviewed main source byte-for-byte;
the already integrated mandatory precrash image/native projection source remains
present. No main checkout or frozen external Go control source was changed.

Static-only checks exit 0: direct pinned1.99.1 rustfmt --edition2024 --check on the
three changed Rust files; exact incoming broker/coordinator/controller source
comparisons; all Rust include/path-attribute resolution; unchanged 127 original
helper lock entries; source-tool Python AST parsing; original/current logical
model-hash reporting (current still bd45…6d15); diff checks. No Cargo, Go, guest,
new Python test suite, profile, actual parity or qualification ran in this wave.
Root's integrated helper test session51796 was still pending at dispatch; its
outcome belongs to root and is not claimed here. fn231/fn233 source integration
and the final endpoint/identity freeze remain outstanding before any final cycle.

## Prospective fn231 source integration through 9a96fcc914

Merged root's reviewed minimizer source main9a96fcc914c17917b23ad0878d8aec698aba5efa
without conflicts. All twenty incoming Rust files match reviewed main bytes
exactly, preserving the producing-round scope/search context, context-qualified
store publication/opening, ledger/digest binding, checkpoint admission and
reporting. Existing moved helper broker and mandatory precrash sources remain
unchanged. Direct pinned rustfmt checks for all twenty files, all Rust include/
path-attribute resolution, source-tool Python AST parsing, exact original127
helper lock entries and diff checks exit0. No Cargo, Go, guest or test ran here.

Parent reports the ordinary private35 gate passed all sixteen original cases
with every retained World/network assertion. Parent's subsequent full77 attempt
failed its thirteenth root after twelve passes (original34 storage model identity
mismatch);64 roots remain unrun and diagnosis belongs to root. These reported
outcomes do not qualify this prospective identity. Integrated minimizer runtime,
strict lint, launcher proof and remaining source integration/final freeze remain
required; worker unit results and source review are not actual runtime acceptance.

## Prospective launcher/minimizer integration through 00a97c42

Merged root main00a97c42 (reviewed source checkpoint432362fa5b) into the isolated
prospective branch. There was one file-location conflict: incoming helper test
`tests/linked_streaming.rs` was accepted at `rust/wasmhost/tests/linked_streaming.rs`.
One additional new nested helper source path was not automatically relocated by
Git: `src/linked/streaming_tests.rs`; it was mechanically moved from the obsolete
helper directory into `rust/wasmhost/src/linked/streaming_tests.rs`. Every incoming
changed Rust file is byte-identical to its exact00a97c42 counterpart after those
path moves. There were no content conflicts or behavioral edits. Existing model/
helper/workspace ownership, mandatory precrash source, broker corrections and
minimizer context/publication semantics remain intact.

Static checks exit0: direct pinned rustfmt --edition2024 --check on all incoming
changed Rust files; exact incoming-byte comparisons; complete incoming helper
path coverage in the relocated crate; all Rust include/path attributes; incoming
Python AST parsing; exact127 original helper lock entries; diff checks. No build,
Cargo, Go, guest, qualification, fixture pin rotation or identity freeze occurred.
No newer interception/observer checkpoint was cherry-picked. Main's source freeze
for fn231 actualscope97675 and unrelated user files were untouched. All prior
pending runtime/lint/final-freeze/qualification gates remain required.

## Prospective source synchronization through cfd72770db

Merged root maincfd72770db into the isolated retirement branch without conflicts
or extra relocation fixes. All twenty-two incoming Rust files match their exact
reviewed-main bytes after mapping helper paths into rust/wasmhost. This preserves
the current module fix, interception/cache/uint/network aggregate controls,
linked-helper cleanup, private package catalogue/launcher collection, pool locks,
store guards and existing runtime_workload/search-context/minimizer bindings.
No production behavior, field context, dependency, fixture pin or source identity
was changed by assumption. The V6 broker checker auto-merged with retained source
snapshot admission; the complete pinned integration/provider snapshots still
match current main's original Go test files byte-for-byte.

Static-only checks exit0: direct pinned1.99.1 rustfmt --edition2024 --check for all
22 incoming Rust files, exact incoming-byte comparisons, full incoming helper
path coverage, all Rust include/path attributes, incoming Python AST parsing,
unchanged127 original helper lock entries/dependencies/checksums and diff checks.
No Cargo build, Go, guest, golden, actual parity or qualification ran; root's
active build cache and main checkout were untouched.

Synchronization still requires root's pending separate Python origin/stored_bytes
checker fix and any subsequent reviewed source checkpoint, followed by the actual
integrated source/runtime/lint/launcher/parity gates and final endpoint/identity
freeze. Main Go retirement and qualification remain root-owned and unaccepted by
this prospective source synchronization. No history rewrite, push or Flow
lifecycle operation occurred.

## Python origin/stored_bytes synchronization through 999ff5aba1

Merged root999ff5aba16dfd9adba740e62aebabdd6220f116, including the pending
origin/stored_bytes checker correction, interception proof documentation and
conductor handover. Merge conflicts0. Both incoming Python checker files match
exact main bytes and parse as Python AST; Rust source delta0 and diff checks0.
No Cargo, Go, guest, golden, test suite or immutable build/cache work ran here.
Root reports9 offline checks passing and an independent fresh source review in
progress; no review outcome or qualification pass is inferred. The root's
immutable build14971 remains root-owned. Subsequent reviewed source, integrated
runtime/lint/parity gates and final endpoint/identity freeze remain required.

## World/storage-wait synchronization through 399b646520

Merged root399b646520f1ecc4d6daefa8a9eebf5c01116408, including accepted World
fixb05ceb6def and narrow Wait storage_crash handling399b646520. No conflicts or
additional relocation fixes. Both changed helper Rust files match exact main
bytes after relocation. Direct pinned rustfmt2files, complete incoming helper
path mapping, Rust include/path-attribute resolution, original127 exact lock
entries/dependencies/checksums and diff checks exit0. No Cargo, Go, guests,
new pins or root build-cache activity occurred here.

Root reports fresh storage-wait source reviewSHIP with genuineRED101→GREEN0,
original World/network assertions, four affected storage roots/eight cases and
strict lint0. Earlier Go proof chain passed on the old951/frozen8dad source,
including single hidden replay, two-package parallel execution with15 overlap
samples/max2, and both hidden replays; its external summary is
/home/agent/evidence/fn233/interception-early-v3-20261022/early-proof-summary.json.
Those observations do not qualify this newer World/storage candidate. Root's
new fixture build63477 owns the live-corpus target/jobs2. Subsequent provisional
fn233 corpus/shim/wrapper/Make source still requires synchronization and all
integrated/frozen runtime/parity/qualification obligations remain pending.

## Strict original-Start selector synchronization through efaf38efea

Inspected the complete incoming four-file diff and merged rootefaf38efea without
conflicts. Incoming code is limited to live-broker-cluster-check.py's strict
selection of original Starta1/b1 controls and the new authenticated offline
live-broker-stream-test.py; the other two files are source/conductor handovers.
Both Python files match exact main bytes and declared SHA256 values:
74e89ad0d5c82e629ca7bc5d060070d06b357df9af6d970a46c588116694350d and
8571f6e0ea5b33525714f32dba5b2e220e98061b2405e2d5b648aac407001a96.
Rust delta0 against both399 and this branch's previous checkpoint. Direct pinned
rustfmt checks on host/helper roots, complete helper-path mapping, include/path
attributes, Python AST parsing, exact127 original helper lock entries and diff
checks exit0. No policy edit, Cargo, Go, guest, runtime test, golden or pin change
occurred. Main checkout, caches and unrelated user paths were untouched.

The incoming handovers record the completed broker77 subset and separate
private35 proof honestly under their actual compiled source/driver identities;
those results do not qualify this prospective relocated source or complete all39
roots/finalR12. The current fn231 semantic failure remains preserved, with further
fn231 fixes and the remaining fn233 corpus/shim/wrapper/Make source pending.
The live-corpus owner is separately checking final39root/21×4 reader endpoint
wiring; any later parity-script patch is also pending a future synchronization.
Final runtime/lint/parity gates, identity freeze, source-size closeout and main
retirement remain root-owned and unaccepted by this source-only checkpoint.

### Driver host/scratch admission source checkpoint

Restored the original parallel/run-validation.sh88–94 resource admission before
any source guard/product CLI/build/guest dispatch. New frozen host_admission.py
reads kernel mountinfo/statvfs and retains actual filesystem/available-space facts
on both PASS and refusal. Original scratch fuseblk/*fuse*/virtiofs refusal and
>=12GiB on BOTH evidence root and scratch parent are fixed, nonconfigurable.
Bindings now require fresh canonical external scratch plus explicit TMPDIR and
GOTMPDIR children; all commands use those paths. Direct stages require matching
initial admission and recheck actual facts. The original sent_epoch/deadline are
unchanged. Minimal configure/driver hooks are disjoint from incoming live owner's
required-module-list additions. See driver/host-admission.md.

Portable driver tests: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover
-s tools/gomad/rust/driver/tests -q` exit0,35tests (including8new admissiontests);
`git diff --check` exit0. Initial run exposed existing pending-adapter unit test's
new missing preflight fixture; its isolated adapter test now explicitly mocks
admission alongside source guard, while new tests prove real dispatch ordering.
No Cargo/Go/build/guest/golden/qualification or main retirement occurred. Current
low host free space remains a real future qualification refusal. Untracked
static-stage assembler/ignored inventory are separate unfinished source work;
no generated runtime result or final source freeze is claimed by this checkpoint.

Host admission review correction: the first mountinfo selector could choose xfs
lexicographically over a same-path virtiofs overmount. Read-only RED reproducer
returned xfs for reviewer-provided stacked rows. Removed all mount-topology
inference. Actual filesystem is now the original fixed `/usr/bin/stat -f -c %T
PATH` serving-path query, frozen in configuration inputs, bounded to5seconds;
errors/timeouts/empty results refuse and retain the refusal receipt. Thresholds,
fresh private scratch/temp bindings, receipt order and send epoch unchanged.
Meaningful controlled-probe regression observes virtiofs and refuses, plus real
bounded stat facts and all probe error paths. Admission9tests exit0; complete
portable driver suite41tests exit0 (includes5separate static draft tests), diff
check exit0. No main sync/build/Go/Cargo/guest/golden/qualification occurred.

### First24 concrete driver static/source/build bindings

New static_stages.py assembles only the first24 original labels into reviewed
checked-command argv+independent raw readers, keeping all remaining72 explicitly
pending/UNEXECUTED until runtime owners compose them. Fixed recipes preserve
retained Go protocol/model/conformance roots/repeats,2replay regenerations,
compile-only/default-build/vet/lint/architecture/harness/worldrace and original
6helper test modes/2helper transcript regenerations. Consolidated strict lint
covers6host+7helper feature modes. Actual source-bound conformance/reference
build copies and frozen ordinary release rebuild equality are enforced; no old
Go host producer. No generated runtime result is supplied by assembly. All tools
(including rustdoc/Clippy siblings/compile-only true), caches, lintbase and future
frozen config paths are explicit caller bindings. Unknown/changed inputs refuse.

Exact native Go roots come from the same-selector actual JSON list/run; reader
reparses actual root/subtest events and repetitions, refuses skip/failure/empty or
invented roots. Actual Cargo test executables/list/ignored-list/run+doctests retain
raw command/process/stdout hashes; reader reparses all native names/summaries,
checks each built executable and exact repetitions. Standard reader output adds
actual testcase inventory/raw_reports/static_execution path+hash for the live
owner's final reach reader. Schema documented in driver/static-stages.md and
shared directly with live owner. Each fixed action owns exact ordered raw
commands, not merely arbitrary exit0 output.

static-ignored.json declares14 exact source/name/reason dispositions.7gzip
oracle+1retained actual-WASM metadata tests require independent actual receiving
proofs under unchanged test-wasm/complete-behavioral-reach; both originalroot34
broker aggregate tests map to transferred-live/TestClusterBrokerDurableRestartParity,
private35 to transferred-live/TestClusterShutdownLiveNodesEvidenceOrder. These
require exact later actual subvariants/raw evidence, never broad rootID alone.
3original manual throughput/calibration/profile exclusions remain unexecuted
exclusions. All deferred rows statusUNEXECUTED/source+binary hashes/reason/obligation;
unknown ignores refuse. Root has been sent the exact table for assessment;
no final obligation removed or actual excluded test claimed PASS.

Portable suite42tests exit0 (6new static tests), AST/static diffchecks exit0.
Tests exercise all96/24owned mapping,13lint modes, pinned-tool mutation,
ignored inventory, exact raw Go/Rust inventory refusals and independent raw Go
reader reparse. No Go/Cargo/build/guest/recipe/golden/qualification executed.
No new main sync; branch remains based on efaf38 source with prospective moves.

Remaining source bindings: ALL72 labels after first24 (exact inventory printed
by semantic-map.json order) remain unexecuted in assembler output; live owner
owns live/admission/final composition. Whole original test-wasm callable/subtest
reach map, positive stress same-cycle actual fullmatrix adapter, exact later
oracle/metadata/private35/broker receiving proofs, final driver composition,
integrated actual static/runtime/strictlint gates, source freeze/identity custody,
code-size final comparison and main retirement acceptance remain pending. No
current source-only check discharges those actual qualification requirements.
