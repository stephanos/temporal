# fn-232.13 installed CLI lifecycle preservation

Source batch based on `cfbe9e39ed`; owner branch `fn232-installed-lifecycle`, worktree
`/home/agent/wt/fn232-installed-lifecycle`. This is implementation and focused
verification, **not final qualification or actual signal/guest execution**.

Original source is `1d695061b6382009d0659238c652a5a765b1d5f5`.
The independent `/home/agent/evidence/fn232-test-wasm-reach/host-contract-map.json`
is SHA-256 `6560d83e77f635757ca2d8abb0cb4d341da0382ee461e22623d232e2afad62e8`.
The declared corpus records each of the seven original callable spans, helper
span hashes and unchanged guest/config fixture byte hashes. Original roots,
helper bodies, tables, loops and dispatch phase ordering were read, including
`replayRetainedCluster`: retained-schema and forbidden-option refusal precedes
installation resolution. No original Go or guest source was edited.

## Native changes

* Seed campaign JSON results add `artifacts`, an array of `path`, `record_hash`,
  `replay_argv` and `replay_command`. Each unique reference is opened through the
  pinned campaign store, all payloads are authenticated, and its record must
  equal the committed outcome before an invocation is advertised. The argv is
  `gomad replay ABSOLUTE_ARTIFACT`; shell text quotes its operand, including
  apostrophes. Native retained replay with no installation selector remains
  available. Existing result outcome/identity fields are unchanged.
* A narrow readonly helper facade reuses the existing bounded `RecoveryImage::decode`
  codec. Native inspection authenticates typed retained cluster context and each
  actual terminal storage image, selects the latest incarnation for each node,
  and reports actual decoded volume IDs/counts. Independent node ownership is
  preserved. Configuration declarations alone cannot provide this evidence.
  No qualification parser or model/guest wire format was changed.
* Text artifact inspection displays authenticated per-instance stdout tails,
  byte counts, virtual elapsed time and the actual capacity bound. Raw retained
  records and payloads remain unchanged. Text escapes control characters.
* Explicit `--installation` is forwarded to replay and resume. Grammar and
  retained replay admission retain precedence; forbidden retained-cluster flags
  remain status 2. Missing/incompatible selected manifests return status 3.
  Profile, helper, host and runtime pins are bound to retained input identity;
  retained source preparation provenance also binds its compiler identity.
  Resume checks the selected context before opening its writer. This does not
  reopen source, compiler or runtime directories. No-selector replay/resume and
  installation-independent inspect/report retain their existing behavior.
* `campaign-check.py` authenticates every advertised path, retained record hash,
  argv and parsed shell command against the independently authenticated native
  outcomes, then excludes this additive presentation field from the existing
  semantic golden projection. Missing, extra, duplicate or changed advertisements
  fail. Raw stdout/stderr and their receipts remain exact; no behavioral fields,
  identity digests or existing golden bytes are normalized or regenerated.

## Original predicates and runtime assertions

The entry point is `tools/gomad/testdata/parity/installed-lifecycle-check.py`;
its portable predicate tests are `installed-lifecycle-test.py`. Actual commands
use existing `rust/driver/driver.py::command`, its admission launcher, command
and process journals, PIDs, raw numeric exits and separate raw stdout/stderr.
The result inventory binds every retained command file. Existing campaign and
artifact authenticators check native canonical bytes, payload hashes, ordered
journal chains, outcomes, receipts, prepared identity and execution bounds.

| Original root / helper predicates | Native receiving assertion |
| --- | --- |
| `TestInstalledOperatorKillResume`: exactly 12 successes, seeds 1–12, 30 s execution, 2 m overall, parallel 1, 100 ms grace, choices and diagnostics | `kill_resume(12)` uses the original empty Go test bytes from the declared corpus, checks all 12 ordered outcomes and the complete baseline native journal. |
| `wasmRunKillResume`: running/resumable committed prefix; real SIGSTOP of CLI group and descendants, all stopped; lock-free inspect while writer remains owned; incomplete durable prefix | Process registration supplies the actual CLI PID; `processes`/`owned_processes` follow descendants across groups; `lifecycle` cross-checks public inspection against independently authenticated journal outcomes/admissions/receipts. A real stopped CLI and incomplete prefix are mandatory; no fixture exit can replace either signal. |
| Actual SIGKILL, raw signaled exit, descendants absent, frozen log unchanged and exact post-kill committed count | Signal observations retain PID, owned process set, stopped states and signal times; raw process exit must equal `-SIGKILL`; post-kill journal bytes must equal the frozen bytes and inspection must expose the same outcomes. Polling remains 120 s and command admission remains bounded by 180 s. |
| `TestInstalledOperatorTypedScenarioKillResume`: 24 mixed success/target outcomes, first interruptible prefix containing both domains, in-flight unreceipted admission, retained successes/failures | `kill_resume(24)` uses unchanged durable guest plus original `mixed` boot argument; success limit 32 / 536870912 bytes, seeds 1–24, 30 s / 3 m. `completed_prefix` requires both domains and authenticates every completed artifact byte. |
| Typed detach; incompatible selected installation resume status 3; missing prepared module / corrupt journal status 2; no completed evidence changed; recover does not publish incomplete work | Private input paths and scenario are renamed unavailable; damaged copies are independently authenticated before mutation. Refused resume/recover compares all completed artifact inventories before/after. Recovery checks the unchanged committed count and no complete event. |
| Resume is published; complete semantics equal baseline; frozen artifacts and lifecycle/network/storage logical records unchanged | Completion requires the authenticated native complete event plus public stop reason. `semantics` retains seed, ordinal, exact native record, executed steps, divergence, soft assertions and feature facts; only artifact paths/storage byte bookkeeping are excluded. `cluster_identities` compares literal native per-seed record hashes, including their authenticated World/network/storage/tape stream identities. |
| `TestInstalledOperatorAsymmetricGuestFailureEndsPromptly`: two and three nodes, a fails, b/c wait; authored start/wait ordering; 3 m watchdog does not mask <=10 s failure | `asymmetric` copies the original deadlock scenario and applies exactly the original table mutations. `prompt_cases` measures command elapsed time, requires status 1 and target_failure, exactly one retained outcome, every node incarnation and per-node failing/waiting marker. All three original stderr markers are checked in authenticated native per-instance stderr, with the original 64 KiB aggregate read cap. Fresh replay must reproduce with status 1. |
| `TestInstalledOperatorClusterFailureEndsPromptly`: deadlock and capacity statuses 3 within 10 s, 3 m execution/4 m overall; exact infrastructure/capacity classification and bounds | `prompt` requires the original output fragments, including global deadlock and payload-bytes limit=4096 used=4096 delta=64. No timeout extension or success substitution is accepted. |
| Capacity has one retained cluster, two interrupted incarnations, nonempty per-instance choice tapes, observation limit 4096 and nonzero virtual elapsed; four partial output markers | `cluster` and `prompt_cases` authenticate native cluster-replay/cluster_summary and all per-instance descriptors. They check interruption, tapes, original_limits, elapsed time and all client/server stdout/stderr markers. |
| Capacity inspect aggregate byte counts, per-instance interruption/counts/tail, elapsed/exhausted fields; text includes ping round 2, exact exhausted line and a/1 instance; replay status 1 reproduced | The native inspection file catalogue and authenticated record preserve per-instance byte totals; the checker sums these descriptors and independently read streams (combined output file counts also checked if present). Interruption and replay prefixes come from the typed retained inputs. The text assertions remain literal. Native JSON names are not required to reproduce the old Go field names. |
| `TestInstalledOperatorTypedScenarioDetachedReplay`: recover/canary/network, seed 7, exactly one artifact each; source/input/compiler/runtime/config/preparation/engine cache/scenario unavailable; discover/inspect still work | `detached` seals every private input and exact installation namespace control bytes before launch, then renames the complete original unavailable set. Discovery requires typed scenarios, no simulation, cooperative profile and explore/inspect/replay/recover/resume. |
| Exactly 6/5/8 executed lifecycle transitions; no global exact tape; durable 3 incarnations with a/1 and failed canary b interrupted, choice tapes and storage identities; durable/network markers and canary peer suppression | The independently authenticated outcome `scenario_steps` and replay configuration preserve the executed prefix; canary uses its original five-action prefix. Per-instance interruption/tape/storage descriptor facts and all original markers are checked; global public `choices` must be null. Existing scenario authenticator checks declared boot arguments, storage model identities, modules and evidence bounds. |
| Actual retained durable volume snapshot contains exactly one volume | `durable_volumes` requires one actual decoded volume, owned by a/2 with ID a-store. Native inspection authenticates the storage-image descriptor, configuration binding and bytes, then uses the existing RecoveryImage decoder; latest-incarnation selection prevents duplicate counting while independent nodes retain separate ownership. Actual 0/1/2-volume codec and native inspection tests cover the source seam; guest execution remains pending. Declared volume topology is insufficient. |
| Each retained target/backend file missing/tampered is refused with status 2; incompatible installation status 3; detached replay status 0/1, reproduced=true, choice-replay=none | `damage` separately copies an already authenticated native artifact and removes/flips each retained module/backend payload; each command retains its exact exit. Fresh replay also compares the entire retained tree before/after. |
| `TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs`: stock and cooperative, seed 7, both unchanged failure/captured-input fixtures, profile discovery, advertised installed replay invocation, bounded stdout and exact replay capabilities | `operator` is run separately for each profile, seals the exact original fixture bytes/config/captured control, checks both stdout markers within 8 MiB, native profile/choice metadata, discovery, inspect, advertised argv and actual advertised source-less replay. New native advertisement is installation-free rather than adding a needless selector; explicitly selected replay is tested separately. |
| Optional original frozen qualification branch: measured deterministic failing target, two executions/two replays, cost/accounting, bounded response and independent publication receipt validation; report without installation | The frozen-source lane makes this original conditional branch mandatory. `operator_qualification` invokes public qualify using the original spec bounds and reuses the existing qualification receiver's raw canonical Result/publication receipt/campaign/artifact authenticator. It independently checks deterministic=true, target_success=false, two executions/two replays, nonfailed cost, nonzero preparation/execution RSS and captured command PID. `detached_qualification_report` invokes public report after detachment with an explicitly missing installation, requires the identical Result and independently validated publication accounting, and verifies no evidence changed. Source and portable predicates are ready; actual qualification/receipt proof remains pending. |
| Original source/compiler/inputs/config/preparation cache (+cooperative runtime) unavailable; discover/verify-only 0, replay 1 reproduced; exactly one private .gomad-cache-v3; poison/symlink fail 3; restored replay 1 | `operator` retains engine-cache, removes exactly the original build inputs, checks one regular mode-0600 nonempty cache entry, flips its last byte and separately substitutes a symlink, restores bytes/path in finally blocks, then requires reproduced=true. Typed detach separately removes engine-cache and therefore tests cold retained replay. Typed linked cache forwarding is owned by the separate linked-cache batch; do not infer this branch passed from storing the installation field. |
| `TestInstalledOperatorReplaysRetainedClusterWithInstalledHelper`: authenticated external typed cluster, outcome-dependent replay 0/1 reproduced=true/choice-replay=none, missing installation 3, forbidden flags 2 before resolution, corrupt payload 3 | `external_cluster` requires a sealed complete external artifact inventory, authenticates every payload, preserves the exact negative invocation matrix and independently copies/changes its nonempty typed replay payload. |

## Source/portable verification and remaining execution

All 26 portable tests pass. They exercise malformed/incomplete/public-prefix rejection, exact wall
and capacity limits, discovery command omissions, asymmetric node ordering,
source-byte changes, evidence mutation/symlinks, transitive process ownership,
record/step semantics, advertisement/path/hash/argv/command mismatches and guest
configuration limits. They do not prove Gomad signal behavior or guest execution.

Focused native tests publish and reopen actual native artifacts and campaign
stores without running guests. The causal RED compiled with forwarding wrappers
and failed all four observable missing behaviors; GREEN covers the repairs and
compatible source-less selected replay, incompatible profile/helper rejection,
corrupt retained output refusal and shell-sensitive artifact paths. The current
native suite has seven passing tests. The helper's actual 0/1/2-volume recovery
image test also passes and rejects malformed/truncated/corrupt/bounded input;
the final test tightening recomputes digest bindings for truncated and
unsupported-version inputs so those cases reach structural decoding directly.
The final helper test passes with that tightening. The readonly accessor and its
linked-facade test are explicitly excluded from conformance, where the linked
modules are unavailable; no feature was enabled implicitly.

Evidence is retained under `/home/agent/evidence/fn232-installed-lifecycle-*`:
`native-red2.log` (causal RED), `native-final.log` (seven native tests),
`volume-final.log` (helper volume test), `portable-final.log` (26 tests),
`qualification-red.log`, `volume-red.log`, `volume-owner-red.log`,
`volume-inspect-red.log`, `clippy3.log` and `main-check.log` (before the final
storage additions), the portable RED logs and help output. The initial
`native-red.log` is setup-only (a test fixture type-inference error), not causal
RED proof. Later final verification receipts supersede intermediate GREEN logs.
No Go build, guest build, installed CLI launch, actual signal campaign, fresh
Go/Rust differential, full golden run or qualification cycle has been run here.

The root granted only focused pinned Rust 1.99.1 debug tests/checks on the
exclusive `/home/agent/.cache/codex-build/fn232-install` target, offline/locked,
jobs 1, `CARGO_INCREMENTAL=0`, preserving existing outputs. Initial target was
1.7 GiB; after the focused checks it was approximately 2.8 GiB. No cleaning or
release build was performed. Root reclaimed released intermediate space and
explicitly granted the final exclusive window. With `CARGO_INCREMENTAL=0` on
every command, both crates' library/test strict Clippy (`-D warnings`) and the
Gomad binary check passed. The final affected native host suite passed seven
tests and the helper decoder passed its actual 0/1/2-volume table. Logs are
`clippy-final.log`, `helper-clippy-final.log`, `main-check-final.log`,
`native-frozen.log`, and `volume-frozen.log` under the evidence prefix above.
All sessions are terminal and the compile window was explicitly released. The
install target is approximately 3.5 GiB; final observed global free space was
1.3 GiB, so no further Cargo commands were started. Root owns space reclamation.
These GREEN receipts precede the sparse-stream review follow-up below. Current worktree, target and raw logs stay under
open fn-232.13 ownership for integration and the pending frozen execution.

### Actual command plan after integration and explicit runtime grant

Freeze the combined candidate and its ordinary release build receipt using the
existing configuration/role helpers. Prepare separate private roots for each
family/profile. Copy the exact original guest fixture bytes listed by the corpus;
prepare captured-input configuration with the original default capture limits.
The typed config is the unchanged scenario/config plus
`GOMAD_REQUIRE_CAPTURED_INPUT=1` and the captured `/inputs/control`; operator
config uses the original profile defaults and `immutable operator input`.
Use a private compiler clone, runtime copy, configuration, preparation and engine
cache. No shared checkout/compiler/cache path may be detached or poisoned.
Installations must bind the frozen helper/host, default fuel 4000000000 and
memory 268435456, and their actual admission configuration. Actual install setup
belongs to the installation owner and must have its own sealed command receipts.

Each lifecycle input JSON has schema `gomad.installed-lifecycle-inputs/v1` and:
`qualification_config`, `repo`, `binary`, `binary_role: release`, `installation`,
`files` (absolute-path SHA-256 inventory), `private_root`, `source`, `inputs`,
`compiler_root`, `runtime_root` (cooperative), `guest_config`,
`preparation_cache`, `compile_cache`, and exact `detach` paths. External retained
cluster runs instead require `external_artifact` and its complete `trees`
inventory. Operator runs also supply `profile`; their frozen qualification configuration
must include the authenticated `source_manifest` and `source_sha256`, and the
original conditional qualification branch is mandatory. Empty or foreign candidate,
role, source, namespace, metadata or payload inputs fail admission.

Run each family with a fresh work directory inside its private root:

```sh
python3 tools/gomad/testdata/parity/installed-lifecycle-check.py \
  --config /ABS/sealed-family-input.json --work /ABS/private/fresh-work \
  --family prompt
```

Repeat separately for `kill-12`, `kill-24`, `typed-detached`, `operator` stock,
`operator` cooperative and `external-cluster`. The declarations preserve each
original seed, timeout, signal, retention cap and read bound. The two signal
families require Linux and actual SIGSTOP/SIGKILL, not fixture interruption.
Keep command/process/signal journals and every raw stream/artifact; authenticate
result inventories from the final receiving reader. Actual decoded-image inspection, conditional qualification publication,
linked-cache poison/reuse and signal proof remain explicit pending actual-run
predicates, never synthesized PASS results.

The candidate receiving binding is only
`operator/detached-installed-discover-inspect-replay`, after all its exact
predicate dependencies and actual captured commands are reconciled. No binding
is made now. This batch does **not** bind
`public-verbs/fresh-differential-and-raw-golden-integrity`: the projector repair
is source work, not a fresh differential or raw-golden execution result.


## Independent native executable publication

The separately checkpointed follow-up preserves
`tools/gomad/internal/backends/wasm/helper_build_test.go#TestHelperBuildPublishesIndependentExecutable`
(original blob `acfb086581971204f2018e444192dcf7916384d2`, function span SHA-256
`6922e308a5d37f2e17b42e3b8f01e48fb646350b2fd4f1497531b0d8b6abaaef`).
The complete original root and all three table cases were read from the pinned
host-contract map. The native public `make build` replaces the retired standalone
helper as the ordinary executable; the explicit `wasm-test-helper` target retains
its reference transport feature and is tested separately.

`rust/publish-executable.sh` copies Cargo's regular output to a fresh temporary
inode, sets mode 0755 and atomically renames it to the stable default publication
path. This works when source and publication path match, when a custom target
cache is selected, and when an existing publication hard-links the original
fixture. It rejects symbolic/nonregular sources or destinations, preserving a
prior publication if admission fails. No host runtime or standalone feature is
silently changed.

The concrete receiving pointer is:

```sh
python3 tools/gomad/testdata/parity/installed-publication-test.py
```

This invokes the **actual copied public Makefile and actual publication helper**
with the original deliberately controlled Cargo publisher. Both native targets
run each original default/custom/custom-matching-output case. Each checks exact
original executable bytes, mode 0755, distinct inode, one link, unchanged fixture
mode, independence after mutating the source inode, and exact pinned Cargo
package/feature arguments. The default build must not produce the standalone
helper; its Cargo binary remains feature-gated. Negative cases reject symlink,
missing and directory inputs without altering the previous output. The original
mechanism uses a controlled publisher and does not build or execute the fixture;
it proves the real shell/filesystem publication behavior and grants no actual
compiler, guest, ordinary release-build or qualification credit.

Causal RED against the unmodified native Makefile failed four mode checks and
two missing custom-cache publication checks. GREEN has three tests, including
six complete public-recipe table executions. Raw logs are
`/home/agent/evidence/fn232-installed-publication-red.log` and `-green.log`.
The architecture receiver owner may bind this exact test pointer; frozen actual
Cargo build-role receipts remain the static build owner's separate obligation.


## Sparse cluster streams review follow-up

Native-only checkpoint `2048a06928` repairs text inspection of native clusters
whose helper legitimately omitted empty stdout/stderr streams. Every declared
stream still requires a matching manifest payload and exact byte count/hash;
unknown names or either one-sided presence are refused. Only jointly absent
descriptor and payload produce empty output. Added tests publish real native
artifacts for silent, stdout-only and stderr-only incarnations, plus missing
payload, undeclared payload, unsupported name and mismatched digest negatives.
These new native tests are covered by the separate frozen-source development
unit gate below; they are not inferred from the earlier seven-test receipt. The checker independently applies the same presence
invariant; its portable causal RED and 25-test GREEN are retained. Cache owner
received the native-only cherry pointer; no cache code was changed here.

The root subsequently granted only the exact frozen source `874f6fc62d` on a
separate host-filesystem development target, with native unit fixtures admitted
at >=256 MiB free. Both new tests passed individually, and strict affected host
library/test Clippy passed. No pre-fix native RED was executed under this exact
source freeze; portable causal RED is retained separately. No Go, guest,
installed CLI, signal or qualification launch is implied.


### Final sparse-stream development unit gate

Exact code source was `874f6fc62df0af89516e1c39e3f9ef0566c8a4ca`, clean. The root
approved the development-only target
`/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks`,
with compiler/linker `TMPDIR` in that target's `tmp` child. Every Cargo command
used `CARGO_INCREMENTAL=0`, pinned `+1.99.1`, `--offline --locked -j1`, default
DEBUG and no added features. The old native install target was untouched.

```sh
CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks \
TMPDIR=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks/tmp \
CARGO_INCREMENTAL=0 cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml \
  --offline --locked -j1 -p gomad --lib --no-run --message-format=json

CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks \
TMPDIR=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks/tmp \
CARGO_INCREMENTAL=0 cargo +1.99.1 clippy --manifest-path tools/gomad/rust/Cargo.toml \
  --offline --locked -j1 -p gomad --lib --tests -- -D warnings
```

The first command compiled without running tests (98 s). Its Cargo JSON selected
exactly one Gomad library test executable; the independent development input
receipt binds that executable's SHA-256, source commit, test names and both
filesystems. Only these two native functions were then executed, each using the
existing `driver.command` process/PID/argv/raw exit/stdout/stderr capture:

* `installed_lifecycle_tests::installed_lifecycle_inspect_accepts_silent_and_single_stream_cluster_instances`
* `installed_lifecycle_tests::installed_lifecycle_inspect_refuses_inconsistent_or_unrecognized_cluster_streams`

Both passed individually. Their actual native Store-published fixtures used
`TMPDIR=/home/agent/evidence/fn232-installed-lifecycle-tests-tmp`; available native
space was checked >=256 MiB before each test, and no fixture directories remained
afterward. Raw command/process captures and the executable/source receipt are in
`/home/agent/evidence/fn232-installed-lifecycle-hostfs-unitgate`. Compiler JSON,
compiler stderr and strict Clippy logs are respectively `hostfs-no-run.jsonl`,
`hostfs-no-run.log`, and `hostfs-clippy.log` under the installed-lifecycle evidence
prefix. Strict Clippy exited 0 (47 s).

The initial conservative target estimate was 4–8 GiB; actual final target usage
was 1.6 GiB, with 90 GiB available on the host filesystem and 723 MiB available
on the native fixture filesystem. Every process/session is terminal and the
exclusive development compile window was explicitly released to root. These are
development unit checks only: the host filesystem is not being admitted for
qualification evidence, final binary gates, actual guest execution or their
native-storage requirements.


## Actual producer stream catalogue correction

A further cache re-review found the earlier closed stdout/stderr name predicate
incorrect. The actual `replay::cluster::project_encoded` producer iterates every
entry of each linked execution's evidence map into the incarnation `streams`
object and corresponding retained files. Choice, transcript, network, storage
and storage-image evidence are legitimate entries. The prior unknown-name
negative and restrictive predicate are removed, superseding that part of the
previous source/verification checkpoints.

Native-only correction `bd81e2869d` plus fixture follow-up `f3af92e1c0`:

* Inspector checks every descriptor against the actual manifest via the same
  typed `Payload::bind` logic extracted from existing `Payload::read`. The public
  read path retains its previous admission and error behavior. Safe evidence
  paths use the existing `store::open::components` contract; there is no closed
  kind-name vocabulary or new fixed evidence-size cap.
* Every actual payload is authenticated by the existing `authenticate_all`
  streaming verifier before inspection. That verifier is narrowly factored and
  reused for `payload_tail`: 16 KiB read buffers, before/after pinned metadata
  checks, exact byte/hash verification, and at most the requested 4096-byte
  stdout suffix retained. Nonoutput evidence is never materialized by the new
  display path. Hash/size checks are not skipped for nonoutput entries.
* Per incarnation, checked sums include **all** streams against the retained
  native evidence budget and combined stdout+stderr against its output budget.
  Each descriptor also binds its appropriate native bound. The actual catalogue
  must match declared paths in both directions. Only absent stdout/stderr
  descriptors with absent payloads map to empty displayed output.
* The new native fixture constructs a linked cluster execution and calls actual
  `project_encoded`, then publishes its actual projected files via `Store`.
  The only original fixture payload reused is the target module; unrelated
  global output files are excluded. The positive case contains choice,
  transcript, network, storage, storage-image and stdout entries, with transcript
  larger than the output budget but within the evidence budget. Stdout spans a
  streaming-buffer boundary and its authenticated tail is exactly 4096 bytes.
  Stream bytes are unit-test data, not claimed executed Go/model/tape evidence.
* Negative native cases preserve actual projection and authenticated native
  publication while independently changing hash, byte count, payload presence,
  descriptor presence and unsafe catalogue paths. Separate cases exceed the
  combined evidence and combined output budgets while each relevant individual
  stream remains within its own bound. The largest native fixture is about
  17 MiB; no guest or engine execution is required.

Exact native functions are:

* `installed_lifecycle_tests::installed_lifecycle_inspect_accepts_actual_project_encoded_nonoutput_streams`
* `installed_lifecycle_tests::installed_lifecycle_inspect_authenticates_actual_project_encoded_catalogue_and_budgets`

The existing sparse negative is now
`installed_lifecycle_tests::installed_lifecycle_inspect_refuses_inconsistent_cluster_streams`.
These changes are **source pending** on this branch. Cache owner received both
exact native-only checkpoints and owns the causal native RED/GREEN and affected
strict gate after Det's exclusive window. Minimal causal reversal is reinserting
the prior `streams.keys().any(name != stdout && name != stderr)` refusal at the
start of `cluster_output`, running the new actual-producer positive, then
restoring the correction before GREEN. No Cargo was run here during Det's lease.

The Python checker independently removed the same invented allowlist and now
stream-hashes each actual descriptor/catalogue entry, validates complete path
presence, and checks native evidence/combined-output budgets. Only stdout/stderr
bytes are returned to output-marker predicates, so markers in nonoutput evidence
cannot satisfy output assertions. The retained-incarnation evidence loop also
hashes files incrementally. A portable causal RED rejected a legitimate broader
catalogue through the old existing predicate; all 26 portable tests now pass,
including missing payload/descriptor/hash/size and aggregate-bound negatives.
Logs are `producer-catalogue-red.log` and updated `portable-final.log` under the
same native evidence prefix. This does not replace actual lifecycle campaigns,
SIGSTOP/SIGKILL, guest replay or qualification receipt proof.


## Current-root cache/lifecycle composition and explicit native status mapping

The cache worktree merged current root `652103a3e4`, then the independently
reviewed source batches from the installed-lifecycle branch, and later root
`292b8f8848`. Root `0fd7ae7b6e` adds Python/runtime receiver source only and is the
next integration base. No production guest configuration, original deadline,
fuel, memory, output, evidence or qualification bound changed.

Independent same-family review found two concrete receiving issues: public
cluster corruption expected status 3 while the new native CLI returned 2, and
actual projected cluster JSON lacked aggregate retained output facts. Native
causal RED is `review-repairs-red.log` under
`/home/agent/evidence/fn232-cache-lifecycle-integration`: 12 pass, exactly those
two tests fail (status 2 versus 3; null total versus 16402). The earlier
`corrupt-red.log` was a fixture setup error and supplies no causal proof.

The original status discrepancy is explicit. At original revision
`1d695061b6382009d0659238c652a5a765b1d5f5`, command/command_test.go lines
265–277 flips retained cluster payload bytes and invokes the public binary with
`--installation installed replay corrupt`, requiring 3. The same public
`invokeOperator` helper is used by command/scenario_test.go lines 364–390,
which flips the same class of retained payload, invokes
`--installation f.manifest replay corrupt`, and requires 2. Both clone a valid
artifact and alter bytes without resealing; the typed test is not a library
entrypoint. No context-based distinction that survives this native lineage was
established. Exact original source files, hashes, entrypoints and line ranges
are retained in `original-status-discrepancy.json` beside those logs.

Root explicitly approved this native mapping: unsupported lineage, command
grammar and authenticated incompatible implementation/identity return 2;
artifact integrity and execution infrastructure failures return 3, before guest
creation. The typed detached corruption root therefore maps its old numeric 2
to native 3. This is **not** unchanged numeric-exit or byte equivalence. Its
strong integrity refusal and zero-execution requirement are retained; the
external corruption root's original 3 is retained. The existing native
observed-replay raw-corruption predicates also adopt 3; their resealed identity
mutations remain 2. No new public flag or weakened acceptance alternative was
introduced. Native tests independently cover raw corruption 3, authenticated
foreign implementation 2 and unsupported storage lineage 2.

The additive Rust v9 `artifact.cluster` projection now carries checked aggregate
retained stdout/stderr counts and separate offered/discarded/truncated output
facts, elapsed time, exact exhausted bound, per-instance interruption/counts and
a bounded authenticated stdout tail. It reuses the full stream catalogue and
budget validator for JSON and text, rejects duplicates/missing observations and
orphan payload prefixes, and binds summary handles/interruption to configured
and retained typed inputs when those inputs are present. The actual producer
now retains its original `IncarnationResult.interrupted` flag in the summary.
Global `artifact.stdout`/`stderr` still describe global output files; no combined
cross-incarnation hash/order is fabricated. CLI.md declares this schema mapping.
The portable receiver asserts native cluster aggregate and instance facts
unconditionally, including offered versus retained truncation. Final fresh
two-run raw/golden lanes remain pending after candidate identities settle; no
old golden was regenerated or relabeled here.

Native lifecycle GREEN after the two causal repairs is
`review-repairs-green2.log` (14 pass, exit 0). Additional ownership/lineage
negatives and final strict/unit checks are recorded in the next final append.
One expanded negative initially reached a test fixture's unreachable arm; that
setup branch is fixed. One broad run overlapped a replacement Cargo build,
invalidating `current_exe` reads; that run is retained as invalid development
evidence. A subsequent broad run passed 290 with one newly added fixture setup
failure; the expanded storage fixture now supplies the same authenticated
cluster observation required from actual producers. None is a guest, cache,
qualification or original-bound exception. Worktree `/home/agent/wt/fn232-linked-cache`
and source donor `/home/agent/wt/fn232-installed-lifecycle` remain open-owned for
root integration. The dedicated assigned target is
`/home/agent/.cache/codex-build/fn232-cache-lifecycle-new`; no other target or
source worktree was cleaned or modified.


### Final source GREEN and independent review

Final implementation checkpoint is
`187c1774fe4d54c0ed804c5615407d09a4a14f98` on `fn232-linked-cache`, composed
through root `0fd7ae7b6e`. Independent same-family reviewer `cache_lifecycle_review`
(gpt-6.1-sol, high, fresh read-only context) returned **SHIP** for that exact
checkpoint, with no remaining concrete blocking source finding. This is a source
verdict; it supplies no actual Go lifecycle or frozen qualification pass.

The final extra-owner regression publishes an actual native Store artifact with
configured a/2 and b/2 but retained/summary only a/2, and no b payload prefix.
Before the count check it fails with `configured but unretained owner was
accepted: ()` (`owner-red.log`, exit 101). The repair requires configured and
retained counts to agree, in addition to existing membership, uniqueness and
summary-map equality. The shared actual-volume fixture proves the same input's
valid ownership/image path; no orphan-prefix refusal masks the causal negative.

All raw logs and numeric exits below are under
`/home/agent/evidence/fn232-cache-lifecycle-integration`. Commands use explicit
Rust 1.99.1, the locked offline workspace, default debug features, assigned target
`fn232-cache-lifecycle-new`, Cargo jobs 2 and native test threads 1.

| Gate | Actual retained result |
| --- | --- |
| Full `gomad --lib` suite immediately before the final owner guard | `lib-final-confirmed.log`, exit 0: 292 passed, 1 ignored at `9f017ab700` |
| Final `gomad --lib installed_lifecycle` | `lifecycle-owner-final.log`, exit 0: all 17 pass, including raw integrity 3, authenticated foreign implementation 2, lineage 2, catalogue/budgets in JSON and text, exact owner completeness and actual volume image ownership |
| Both crates `clippy --lib --tests -- -D warnings` | `clippy-owner-final.log`, exit 0 on final source |
| Helper `--test linked_compile_cache` | `helper-cache.log`, exit 0: 3 actual small-WAT tests, cold/warm zero-compile cache reuse, poison/link refusal, restored reuse and cluster host failure |
| Actual retained small-WAT selected-cache resume | `resume.log`, exit 0; also included in the full library GREEN, with absent compiler/preparation/source paths and a published private engine-cache entry |
| Portable installed lifecycle predicates | `portable-lifecycle-final.log`, exit 0: 27 pass |
| Independent executable publication predicates | `portable-publication.log`, exit 0: 3 pass |
| Workspace formatting and Git whitespace | Both exit 0 |

`final-summary.json` binds the source checkpoint, exact test executables and
SHA-256 digests, all retained exits (including setup failures and causal REDs),
source-review verdict, runtime gaps and worktree custody. Final host unit test
executable SHA-256 is
`9d36ca098cecff89303a683270398b7f39dc6161908125ca521e011e56746b62`;
helper cache integration test executable is
`7f5e1fab2622594cf1d4114d64f3a00bc23232ae6d6323aba5444735ea15a54a`.
Every assigned-target executable/process is terminal. Target usage is 2.0 GiB;
last free shared space is 67 GiB. No cache cleanup, guest-bound change, Go run,
full qualification, Flow lifecycle change, push or history rewrite occurred.

The root's subsequent determinism/stress source at `d71c795a8a` was not imported
or tested by this worker; root owns that combined source integration. Preserve
cache refusal before **every** canonical publication when adapting the separate
Completion interface. The actual two-profile installed Go cache/operator matrix,
SIGSTOP/SIGKILL recovery, changed-candidate detached replay/resume/inspection,
fresh raw/golden lanes and final frozen cycle remain pending. The concurrently
running immutable fn233 Make/Go acceptance branch and target were untouched,
and this cache implementation supplies no cache-effect claim for that binary.

Kept worktrees: `/home/agent/wt/fn232-linked-cache` for root integration and
`/home/agent/wt/fn232-installed-lifecycle` as the unchanged clean source donor;
both belong to open fn232. `worktrees-final.txt` captures the complete registry.
No additional scratch checkout is retained. Assigned target and all evidence
remain available; the exclusive cache/lifecycle build lease is released.

## Original terminal-exit controls: native applicability batch (2026-10-23)

This scoped batch starts at root `81a66b54b3`, then provisionally merges measurement
`0523f4a475` as `a9452dc32a`. The dependency merge is source composition, not an
acceptance verdict for measurement's separate batch. Its APIs and mechanical
Execution/Prepared initializer additions are required by these controls.

Immutable original `1d695061b6:tools/gomad/internal/backends/wasm/wasi/process_test.go`
lines 757–799 has seven `TestTerminalExitMeasurementControls` cases. The original
`process.go` lines 1207–1312 authenticates an external terminal frame, then
requires successful exit, no cancellation/watchdog, process group disappearance,
and terminal-to-reap at most one second. That helper transport was retired.
The native mapping therefore makes these applicability distinctions explicit:

* Prompt exit, 1.1-second slow preparation and 1.1-second delayed return run an
  actual linked binary-WASM guest through production prepare/execute timing,
  `warm::Run::observed` and its existing one-second candidate admission bound.
  Request-local preparation/cleanup delays exist only under the compiled
  determinism-fixture feature. Real `Instant` values establish the measured
  interval; delayed preparation is excluded, delayed cleanup is refused.
* Missing/invalid terminal metadata are causal typed-consumer refusal controls:
  execute a real guest first, then remove/invert its native timing metadata.
  These mutations are not claimed as actual missing external frames. An actual
  trapping guest also fails the production warm observer.
* Helper death, the 200-ms post-receipt watchdog and GroupGone use production
  measurement `build::command_bounded` with real `/bin/sh` fixture children.
  The retained receipt records signal 9 or timeout, retains the fixture stdout
  marker, and has no cleanup errors; a separate real process-group ESRCH check
  establishes disappearance. The watchdog fixture retains the original 30-second
  sleep. Its marker is a fixture receipt, not a restored helper result grammar.

These are native API/actual child preservation controls, not Go guest execution,
installed operator acceptance, or final frozen qualification. Original bounds
(one second, 1.1 seconds, 200 ms, thirty-second watchdog fixture) are preserved.
The new production observer also binds the supplied teardown interval exactly
back to actual Execution timing and the authorization-to-terminal measurement;
it refuses absent, reversed or mismatched timestamps rather than trusting the
caller's interval alone.

Root owns the coherent final inventory append. This branch intentionally leaves
inventory and identity-history unchanged, preserving the accepted 440-record
prefix. Proposed additional controlled clock site:
`host-rust:wasmhost/src/linked.rs|fixture_timing_delay|recv_timeout|1`, proposed id
`host.rust.wasmhost.src.linked.rs.fixture_timing_delay.recv.timeout.1`.
Role: bounded request-local fixture-only delay before authorization or after
terminal classification; absent from ordinary and qualification builds, no
model/guest/canonical-stream input. Evidence is the native terminal control plus
`receiving_host_effect_sites_are_inventoried` after root's append.

Focused evidence directory: `/home/agent/evidence/fn232-terminal-controls/`.
`native-controls.log` first observed the actual native/child cases (3 tests,
exit 0). `bound-red.log` removes only the production one-second candidate bound:
real 1.1-second cleanup is admitted and the assertion fails (exit 101).
`timing-red.log` removes only actual Execution timing authentication: absent
metadata is admitted and the typed refusal assertion fails (exit 101). Each
mutation was restored before subsequent checks. `final-warm-exact.log` and
`final-warm-polished.log` pass all 14 warm-filtered tests (exit 0), including the
real lifecycle controls. Original immutable full Go sources are retained beside
the logs; these are source references, not newly executed Go evidence.

Fresh same-family independent reviewer `terminal_controls_review` (gpt-6.1-sol,
high reasoning, fresh context, read-only/no builds) initially blocked on the old
0.1% interval tolerance: a real 1,000,500,000-ns interval could be reported as
1,000,000,000 ns. The fix requires exact actual/recorded equality, stores the
actual interval and checks equality again at admission. The regression separately
covers construction and later record mutation; its crafted integer times are
explicit typed-consumer proof, not an actual lifecycle delay. Reviewer then
returned scoped SHIP, with dependency acceptance/final qualification excluded.

Formatting and whitespace checks pass. Strict fixture-role Clippy was attempted
and failed on the provisional dependency's build-script `format_collect` warning.
A diagnostic run suppressing the known build/private-interface/helper warnings
then exposed 149 dependency lint errors, including stress files owned by root;
this is not a strict lint pass. The measurement owner requested neutral warm
lint repairs (docs, semicolons, direct method/default use), included here; no
stress or independently owned measurement source was changed. Root owns strict
lint on the composed current source, final inventory/ledger append, source
identity rotation, Go/installed acceptance, raw/golden regeneration and frozen
qualification. No such combined acceptance is claimed by this focused batch.

Ordinary-role focused warm tests also pass: `default-warm.log`, 11 tests,
exit 0. Fixture-only controls are excluded from that build. This source checkpoint
will next compose root `e944295e58` as an explicit dependency to receive the
already reviewed global strict-lint/source fixes; the earlier failed lint logs
remain historical evidence for the provisional source and are not erased.
