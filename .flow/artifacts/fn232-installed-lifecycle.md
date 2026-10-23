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

All 24 portable tests pass. They exercise malformed/incomplete/public-prefix rejection, exact wall
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
a subsequent test-only tightening recomputes digest bindings for truncated and
unsupported-version inputs so those cases reach structural decoding directly.
That tightening awaits the final focused native test gate.

Evidence is retained under `/home/agent/evidence/fn232-installed-lifecycle-*`:
`native-red2.log` (causal RED), `native-final.log` (seven native tests),
`volume-final.log` (helper volume test), `portable-final.log` (24 tests),
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
release build was performed. At the source freeze, free disk briefly fell below
2 GiB; new compiles were held. Root reclaimed space and assigned the next compile
window to the cache worker. Remaining strict gates on this branch are the updated
helper test, host `installed_lifecycle` tests, host library/tests clippy with
`-D warnings`, and binary check, all using the same pinned offline debug target
only after root grants the next exclusive window. Current worktree, target and raw logs stay under
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
