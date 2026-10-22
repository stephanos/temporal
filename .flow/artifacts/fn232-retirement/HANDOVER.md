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
