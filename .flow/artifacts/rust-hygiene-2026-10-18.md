# Rust hygiene and WASM gate checkpoint

Neither fn-187 nor fn-189 is complete. The R1-R4 source batch is now integrated
in primary; R4 compilation, generator validation, qualification and review remain
open. All qualification budgets and final acceptance gates remain intact.

## Latest continuation checkpoint

### Post-deletion frozen candidate — 2026-10-19

Seven remaining public test roots are removed across four files (295 lines).
`/home/agent/wt/evidence/fn187/r3-current/seven-public-removals-application-2026-10-19.json`
proves retained lines are byte-identical to exact backed-up preimages. Packet
`remaining-public-deletion-named-entry-2026-10-19/` retains the assertion map,
patch and recoverable whole-file preimages. All142 private binding keys/reasons
are unchanged; no public test was relabeled as a private invariant.

Combined-mode baseline profiling exposed a fixture race:3/8 runs sampled only
the unnamed entry rather than its named loop. The only fixture change names that
WAT entry `$hot_entry`; instructions, assertions and budgets remain identical.
`guest-symbolization-name-only-proof-2026-10-19.json` reconstructs the preimages
by removing only that name in primary and R3. Focused confirmation passes20/20;
the complete original combined-mode suite passes. Failed attempts remain retained.
The four earlier passing coverage certificates are now historical, not rebound
to the changed engine test file.

Six-mode Rustfmt/strict Clippy passes (terminal session76393), and69 coverage
tooling tests pass (terminal session34557). Post-deletion preservation proves all
526 original contracts and215 helper cases unchanged. The accepted R1/R2 staged
diff remains SHA2560a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299.

Session70048 terminated0 after `run-post-delete-conservation-2026-10-19.py`: new
before/r3-post-delete-conservation phases rebuild baseline and candidate for all
five modes. Every required full/corpus/invariant, frozen object/source/tool,
compiled-roster, cursor and strict conservation check passes. Baseline-covered
production lines preserved are15626/14971/14963/15103/14276 in default,
diagnostic-profile, diagnostic-fuel-off, guest-profile and combined modes.
All loss/union-gap lists are empty. Default also passes Go full/corpus/invariants
and statement conservation. Post-run verification proves all ten baseline/candidate
source/tool/object pins still match; log is
post-delete-final-pin-verification-2026-10-19.log. These results qualify the exact
pre-assertion-repair source, not the changed Go adapters or completion of R3.

The periodic final-seven-root correctness/standards audits return0 findings;
verbatim reports are fn187-seven-public-removals-review-2026-10-19.md. The
canonical Codex fan-out51eff737bb014145b605e88037d1e2a9 reviewed only the empty
HEAD-to-HEAD committed range. All three outputs explicitly decline task
certification; its mechanical SHIP receipt is not accepted for R3. Owner-only
project commits remain intact. Fresh native read-only correctness/standards
audits cover the actual worktree/index delta and untracked consumers as fallback.
The standards report returns0 Should Fix/1 Consider (private broker golden
bookkeeping coupling). Full correctness review returns3 Should Fix: discarded
Go idle return1000, constructor World capacity identity, and exact network
resource-exhaustion sentinel identity. All three are repaired after intended RED;
focused GREEN and four production-source overlay mutations verify the restored
assertions. Both corpus regenerations agree byte-for-byte, all eight normal
corpus/control roots pass without skips, and fresh preservation checks retain
all526 original contracts and215 original helper cases. Source rereview resolves
all three findings. Reports and disposition are
fn187-r3-working-tree-review-2026-10-19.md.

Fresh five-mode verification passed under owned session25445, terminal exit0, log
`/home/agent/wt/evidence/fn187/r3-current/typed-assertion-replay-2026-10-19.log`.
All five full/corpus/invariant/cursor/conservation gates pass, with zero production
line losses or union gaps; default Go full/corpus/invariants and statement
conservation also pass. Session36566 subsequently verified all ten baseline and
candidate source/tool/object pins against current inputs. Authenticated frozen Rust objects are reused only
after proving unchanged compiler inputs; every test arm/profile/comparison is
fresh. Reviewed input manifest SHA256 is
`a2a0fce819a1761d392aba7fe443de7e140065a240bb80199f7815b70ddbe38d`.
Bounded rereview resolves two replay provenance findings: exact reviewed Go,
fixture-proof and per-mode script/selector hashes are checked before capture.
Old passing phases remain pre-repair evidence, never relabeled. Final independent
actual-worktree correctness verdict is SHIP for R3,0 findings. `flowctl done`
then `show` verifies task .3 done. Standards remains0 Should Fix/1 Consider.

Task .4 is claimed in progress. Fresh source-only worker
`/root/fn187_r4_modules` implements the complete approved model/protocol/namespace/
handoff/dead-code ownership refactor in the existing functional checkout.
Root retains compiler/runtime/review/identity qualification ownership; owner
retains staging/commits. The worker is now source-frozen, including an explicit
cluster/handoff owner. First lint found14 unused imports/items; the second
passed the ordinary binary then found five stale test references/imports.
Worker corrections are included. Primary strict six-mode Rustfmt/Clippy now
passes, terminal session9541 exit0; make validate passes, session53809 exit0,
including template/output, compatibility pack and qualification manifest checks.
Integrated retained Rust matrix59182 terminated0:201 default,55 conformance unit,
204 diagnostic-profile,205 fuel-off,206 guest-profile and207 combined tests pass.
Only original2/3 ignored benchmark roots stay ignored. Source/testdata maps are
checked before/after each mode and match primary after terminal success.
Logs/source receipt are in r5-current/r4-integrated-rust-tests-2026-10-19/.
Go/default-host/final qualification and independent batch review remain open.

R5 physical counts are now retained in
`.flow/artifacts/fn187-r5-physical-counts-2026-10-19.md`, with the current R1-R5
mapping appended to `fn187-functional-2026-10-17.md`. Root independently verifies
all131 counted candidate inputs and inventory digest
da1132cf955b09c42aa2d64dad4f5dd79fc9fac45bdfde2daffed61ad3c59bdd.
All Rust src+tests physical lines decrease42803→42016; Go WASI tests21898→21061.
The historical src/non-inline bucket increases because separate test-only corpus
readers are classified there; no compiled-production reduction is claimed.

Combined qualification preparation is frozen under
`/home/agent/wt/evidence/fn187/final-integrated-2026-10-19/`:
8494 source inputs, manifest SHA256
9a74770b9e03de25e05910c0070decbed73c3a841c95b5d2421b4d18f77e62f8.
All manifest entries pass the independent preparation check (64710 exit0).
Approved recovery/replication fixture copies are staged byte-identically under
`.flow/tmp/fn187-final-integrated-2026-10-19/temporal-staged`.
No qualification result is yet claimed for this freeze. Restore/publish ordinary
and conformance helpers from current source after the profile matrix before Go
gates, and record actual helper/model/runtime identities.

Remaining qualification scope is not merely ten workload cells: retain the
applicable full ladder in
`/home/agent/wt/evidence/fn189/final/driver/fn189-run-qualification.sh` (89 reference
stages), plus current retained-inspection and matched-release obligations.
Its old clean-checkout/Rust1.94.1 assumptions and moved go-test producers cannot
be executed unchanged: bind current dirty-owner source freeze/Rust1.99.1 and use
the documented wasmmeasure command mapping. Keep runtime32 seeds/aggregate,
cluster provider/functional/busy boundary and grouped tests, stream/replay,
terminal/artifact/operator, actual storage/network/cluster, workload/SQLite
selection and retained-inspection contracts at their original scope. Reuse only
source-appropriate prior static evidence, explicitly marked not fresh. Native
backend execution, fn185 profiling and ignored benchmarks stay out of scope;
portable native-source/pin checks are not native qualification.

Primary integration is applied, terminal session40259 followed by successful
final hash verification. Evidence:
`/home/agent/wt/evidence/fn187/r5-current/integrated-batch-2026-10-19/`.
The guarded plan reconciles2603 owned paths (124 source/docs,2479 data), preserving
primary owner changes via recorded R3 baseline matches and three-way document
merges. Recoverable preimages are retained. An initial postcheck caught one
Unicode line-separator split in a copied comment; exact candidate bytes were
restored and every planned final hash then matched. Both indexes remain unchanged,
and primary diff check passes. All82 Rust source/config/test paths match the
worker; all3026 testdata inputs match the accepted R3 freeze. Unrelated native,
Makefile and Flow owner changes remain untouched. Old R3 phase manifests retain
their exact pre-R4 source meaning, not current-primary qualification.

R5 original-baseline timing preparation runs serially under root session56914:
`/home/agent/wt/evidence/fn187/r5-current/time-original-canonical-2026-10-19.py`.
New clean detached worktree `/home/agent/wt/wasm-fn187-original-timing` is at exact
original commit bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe, retained while timing
runs; the older baseline's unrelated cluster-test edit is untouched. The exact
original attempt terminated56914 exit2 after305.387s: the cluster fixture's
2000-iteration guest exits rather than exhausting its unchanged20000 fuel.
Failed original evidence is retained. A single-value baseline-only fixture
correction2000→4000 passes the exact original isolation test (30441 exit0), with
all assertions, production bytes and fuel unchanged. Corrected source diff is
SHA2562a7e7562f54cfb4f0e1829962749c769dea75bcca6108051fee46e4519129bff.
Corrected timing session29449 terminated exit2 after439.999s, with ordinary-helper
link-count2 failures in Go host tests. Both failed setup attempts are retained.
The supported explicit independent-helper setup is now running under10953:
`r5-current/time-published-original-canonical-2026-10-19.py`, logs/receipts under
`published-original-canonical-2026-10-19/`. The helper is an authenticated independent
copy of the original default Cargo artifact, with link count1 and matching SHA;
no source/admission changes beyond the disclosed fixture correction. One warmup
and three measured samples are intended; no timing result is claimed yet.
The candidate arm must use the same explicit-helper/serial environment.
No link-count, memory, replay, identity or other admission check is weakened.
Original Rust1.94.1 versus candidate1.99.1 is a compiler
confound to disclose, not a pure refactor speedup. Both arms must use identical
serial resource settings and the same host. Existing21 worktrees remain for
open fn185/fn187/fn189 work, uncommitted corpus/module work or retained qualification
reproduction; no preexisting worktree or branch was removed this turn.

The approved recurring readiness selector is also fixed in both copies. Outer
`TestTemporalBrokerReadinessSource` passes (session68015); direct staged execution
passes both predicate roots and47 subtests without skips (session93077). Evidence
is `/home/agent/wt/evidence/fn189/readiness-source-selector-2026-10-19.log` and
`readiness-source-selector-controls-2026-10-19.jsonl`. Earlier5394-entry host
freeze predates the engine naming and selector edits and is not current-source
qualification. Final combined gates/review and admitted matched comparison remain.

The following sections are historical checkpoints. Their live-process and
unapplied-patch statements are superseded by this section.

### Current primary host gates — 2026-10-19

Owned session44332 terminated exit0. On primary HEADd200fb9be2, stock Go1.27.1
default build, vet with test_dep and integrated `make -C tools/gomad validate`
all passed, each with retained command/exit/log and a passing5394-entry source
check. This current roster includes the earlier5393 source paths plus the owner's
new native stdlib manifest; primary unrelated dirty files were preserved.
Evidence is `/home/agent/wt/evidence/fn189/host-gates-current-2026-10-19/`.
Validation includes generated/toolchain/pack/inventory source checks, not deferred
native backend qualification. Harness/architecture and World race gates remain
separate until actual terminal evidence is recorded.

Harness session33534 and World session56071 have now both terminated exit0 with
5394/5394 source hashes passing after each. Archcheck actually executes in31.533s;
ordinary package results include valid Go cache hits, and all three World race
package results are cached. These canonical command passes are not fresh World
process/race executions or native qualification. Their complete logs/exits remain
beside build/vet/validate evidence. No source change or commit accompanied them.

Fresh World race execution is now proved separately: session13147 terminated0,
running `go test -race -tags=test_dep -p=1 -parallel=1 -count=1
./internal/model/world/...` with stock Go1.27.1 and sanitized overrides.
World, mailbox and process pass uncached in2.816s/1.031s/1.021s respectively.
`world-fresh.exit`, `world-fresh.log` and `world-fresh.source-check.log` retain
the actual result and5394/5394 passing source hashes in the same evidence root.
This supersedes the cached-only World limitation, not deferred native qualification.

### R3 broker fixture provenance correction and fresh default freeze

Session67815 is terminal1, not live: guest-profile baseline tests all pass, but
strict full export admission rejects two entries for one previously unseen
debugid wrapper. The full307-entry diagnostic inventory contains no nonzero-hash
or project mismatch. The new wrapper maps uniquely to registry
debugid0.8.0/src/lib.rs, SHA256d39adde2…, exactly
`_RNvMs0_CshOPwBy4Ngnb_7debugidNtB5_7DebugId8breakpad`. It also exposes the
guest-feature dependency variant of the already reviewed aarch64 unwinder,
`_RNvNtNtCsl1f379ryPF2_26wasmtime_internal_unwinder4arch7aarch6417get_stack_pointer`,
mapping to the unchanged47.0.3 source SHA2564daf28dc…. All unknown/project/nonzero
mismatches remain errors. No broad debugid/unwinder-crate allowance was added.

New symbol acceptance regressions first fail for the intended refusal; then69
tooling tests pass, including other-symbol/version/source/hash, ambiguous,
missing and modified-source refusals. Logs are
`debugid-exact-variant-red-2026-10-19.log`,
`guest-unwinder-exact-variant-red-2026-10-19.log` and
`guest-exact-variants-green-2026-10-19.log`.
`debugid-exact-source-disposition-2026-10-19.json` inventories all307 retained
diagnostics only; it does not relabel the failed historical phase.

The repair lives only in NEW `guest-debugid-tools-2026-10-19/{guest-profile,
guest-profile-fuel-off}` tool roots; canonical and prior executed tools remain
unchanged. All three passing default/diagnostic source/tool/object pins verify
again, with15626/14971/14963 baseline covered lines and zero losses/union gaps.
Fresh guest retry session52099 uses `run-guest-exact-conservation-2026-10-19.py`
and new before/r3-guest-exact-conservation phases. Both original guest baselines
and candidates must execute full/corpus/invariant/cursor/comparison gates; no
objects, source pins or receipts are rebound. Original tests and staged index
remain untouched. The source-only deletion-packet agent has returned; it has no
live compiler/runtime work and its patch remains unapplied.

Three clean modes now pass full conservation: default, diagnostic-profile and
diagnostic-fuel-off. For both diagnostic modes, baseline full/corpus, candidate
full/corpus/138-invariant arms, fresh bounded cursor proof and the separate full
line comparison all exit0. Diagnostic-profile retains14971 original covered
lines with zero loss/union gaps. Guest modes remain pending under live
session67815. The candidate source and all original tests remain unchanged.
The prepared deletion-only packet is
`remaining-public-deletion-packet-2026-10-19/`: four files, seven public roots,
295 deletions/zero additions; SHA256
48620d31b84095d226d24badd50f323a3293fca1611a4fd27dd7e87b728d0ddd.
Root independently rechecks all35 source/fixture bindings and patch dry-run.
These are static preparation checks, not deletion permission or a review verdict.
Original request-limit differences for the diagnostic cluster refusal remain
explicit in its assertion map; no exact request-byte equality is claimed there.

Clean-retry default evidence now passes every required arm and comparison.
Session67815 remains live, proceeding to diagnostic-profile. Both baseline and
candidate initial builds rebuild all eight project executables, fresh=false,
without compiler warnings; early production source-root admission passes.
Default baseline Rust/Go full+corpus and candidate Rust/Go full+corpus+invariant
arms and post-Go exports all exit0. The fresh bounded cursor proof exits0;
the separate full Rust conservation certificate records15626 original covered
production lines, zero losses and zero corpus+invariant union gaps. Go's exact
statement conservation likewise records empty loss/union-gap lists.
Source pins are356730b76c2d15229a9dc25290f78e66564df47edc73cc20fbe9465c0b444b41
andc65699316ed27a08781895ce0662d34583a594ae06f63257b81ced184e03dde1.
Evidence is under `before-feature-clean-conservation-2026-10-19/default`,
`r3-feature-clean-conservation-2026-10-19/default` and the latter phase's
`go/statement-conservation.json`. The cursor receipt is only its bounded probe,
not a stand-alone full certificate. All four nondefault comparisons, remaining
public assertion-level deletion proof, post-deletion verification and review
remain required. A separate source-only agent may prepare future deletion
patches outside frozen inputs; it cannot apply them or run tests.

Superseding terminal checkpoint: session42656 ended1 at cursor proof after all
default candidate Rust/Go full, corpus and invariant arms and post-Go exports
returned0. Baseline had full/corpus arms only. No comparison certificate passed.
The strict foreign-source guard found candidate paths in baseline production
mappings. Independent per-object empty-profile exports in
`foreign-root-diagnosis-2026-10-19/` isolate the fault: baseline ordinary helper
has0 primary/37 candidate project source paths; baseline conformance helper has
24 primary/0 candidate paths. Cargo's baseline JSON reports ordinary helper and
two test executables fresh=true despite its primary manifest path. The shared
target reused prior candidate artifacts. This invalidates the phase as an
original-baseline conservation reference; do not normalize paths or relabel it.

Two new early-admission regression tests fail for the intended missing refusal,
then all66 tooling tests pass after the minimal correction. `coverage.inventory`
rejects every reused project executable before freezing, and production mapping
admission invokes the unchanged strict source-root parser before pinning.
RED/GREEN logs are `foreign-root-guards-{red,green}-2026-10-19.log`.
Coverage/new-test hashes are6cd7e12d…/7c9cadea… and are mirrored into all four
unexecuted mode-specific tool roots and their manifests. No project/unknown
diagnostic exception, source-root normalization or production behavior changed.

Fresh retry session67815 runs `run-feature-clean-conservation-2026-10-19.py`
under new phases `before-feature-clean-conservation-2026-10-19` and
`r3-feature-clean-conservation-2026-10-19`. Before each initial no-run build it
cleans only package gomad-wasmhost in the exact owned R3 coverage target; registry
dependencies, source, index and independently frozen historical objects remain.
All original full/corpus/invariant/Go/cursor/comparison obligations still apply.
Old attempts, commands, raw profiles and actual failed guards remain retained.

Current continuation: six-mode lint session93845 and five-mode compiled-roster
session64340 both terminate0. Source-bound inventory refresh admits only four
unconditional test-machinery roots in the default selector: functional204,
machinery40, corpus28, private133 and combined invariants134. Existing private
reasons and142 bindings are unchanged. Coverage-tooling tests pass64/64 in
session12058. Per-feature tool copies retain exact compiled partitions: diagnostic
functional210/corpus26/invariants138/public5/machinery41; fuel-off212/26/138/6/42;
guest213/26/140/6/41; combined215/26/140/7/42. One resource root is still separate
from functional selection but included in actual full execution; default ignores
two benchmarks and each nondefault mode ignores three. No ignored benchmark is
executed and no public root is admitted as private. Hash/count/subset/disjoint
partition checks pass for all five modes in
`feature-inventory-validation-2026-10-19.log`.

Fresh phases `before-feature-conservation-2026-10-19` and
`r3-feature-conservation-2026-10-19` are executing under root session42656 via
`run-feature-conservation-2026-10-19.py`. Default baseline warning-free build,
Rust full/corpus, Go full/corpus and post-Go Rust exports pass. Candidate default
warning-free build and Rust full/corpus pass; remaining candidate invariant/Go
arms, cursor proof, comparisons and all four nondefault modes remain unproved.
Do not edit pinned source or tools during execution. Failed historical phases
remain unchanged. Fresh preservation receipts confirm526 original JSON contracts
unchanged and215 original helper files byte-identical under
`feature-current-preserve526-2026-10-19.json` and
`feature-current-preserve215-2026-10-19.json`. These checks do not replace final
coverage, assertion-level deletion proof, review or qualification.

Continuation: session57726 is terminal exit1, not live. Its fresh Rust full,
corpus and134-invariant arms pass, but candidate Go full exposed the same stale
109cabe… digest in62 added network-adapter and51 added network-cluster cases.
The existing Go oracles regenerated those113 files. Every exact preimage SHA256
is reconstructed by reversing only the digest; a second regeneration is byte
identical. Together with the10 broker cases, all123 added identity-bearing cases
now bind9b087b96…. The separate183 original network cases remain unchanged.
The complete63-root Go candidate preflight passes with zero failures, using the
frozen ordinary/conformance helpers and a separate profile directory:
`go-metadata-preflight-2026-10-19/test.jsonl`. Failed coverage phases remain intact;
neither has been relabeled as a successful conservation certificate.

Before another full freeze, source research found nondefault feature gaps.
Actual fresh Rust1.99.1 diagnostic-profile helper-corpus execution confirms RED,
session10159 terminal101, under `diagnostic-corpus-preflight-2026-10-19/test.log`:
missing explicit feature profile expectations and a cluster guard treating raw
profile stderr as empty. No production guard, profile projection or original
assertion has been relaxed. A bounded implementation subagent is preparing
additive feature expectations and exact public scenario mappings, preserving
original fixture bytes and all original tests. Root remains sole runtime owner.
Explicit per-feature tool-root copies are prepared by an unexecuted launcher,
not changes to frozen canonical tools; public roots are not being blanket-added
to private invariants. Final corpus/conservation execution remains outstanding.

The feature repair adds11 public scenario cases and source-SHA-pinned sidecars
outside the original helper-case directory. Core frames/evidence and preexisting
exact profile counters are checked against the original semantic baseline before
sidecar comparison or regeneration; only declared feature configuration/fuel
differences and the diagnostic profile addition are permitted. New negative
controls pass in the actual diagnostic preflight. Session69769 ended101 with
22 passing guard roots and the expected missing-sidecar corpus failure.
First regeneration session49462 ended1 after26.38s:367 partial sidecars were
retained, but the handoff-boundary fixture still applied its default completion
guard to a diagnostic refusal. Source/root index and215 original helper hashes
remain unchanged except the explicitly owned test-side correction.

Context-only reproduction session17547 ended101 and retains the actual refusal
in `diagnostic-handoff-context-2026-10-19/test.log`: exit0, exactly one
`cluster_result`, termination unsupported, no instances, zero virtual elapsed,
empty evidence/residual stderr, with the diagnostic message that clusters are
not executed. The narrow feature guard now asserts that exact response; all
default completion/World/network/inspection assertions remain intact. A mutation
control rejects process/frame/evidence drift. Original JSON bytes and every
original test remain; no production behavior or admission policy changed.
A fresh sidecar-regeneration retry runs serially under root ownership, retaining
both failed attempts. It requires two byte-identical regenerations and a normal
check for each mode, preserving original source/index bytes. Full coverage and
independent review are still unproved.

Retry session91832 is now terminal1. Diagnostic-profile passes all24 helper roots
for both regenerations and the normal check, with byte-identical sidecars and
source/index preservation checks. The next fuel-off regeneration exposes a
test-side semantic-adapter error: it rewrites offered fuel into early refusal
results, whereas production `Outcome::new` correctly leaves fuel0 before store
execution; `terminal_outcome` assigns offered fuel only at the actual terminal
store boundary. The preserved default execution-stage metrics distinguish those
paths. Its23 guard roots pass, but the corpus records102 literal fuel mismatches.
The agent is narrowing that test-side eligibility rule with explicit negative
controls; production and original expected bytes are not changing. Retain all
partial fuel-off sidecars and the passing diagnostic-mode evidence. No full
feature or conservation acceptance is claimed.

The corrected classifier uses the preserved instantiation/store stage, not the
observed output or execution timing alone: failed instantiation also reaches
terminal_outcome. Early results keep literal fuel0; store-terminal results must
retain offered fuel. Unknown/inconsistent baseline shapes refuse. Mutation guards
cover both directions. Session95347 now terminates0: fuel-off26 roots,
guest-profile25 roots and combined26 roots each pass first regeneration, a
byte-identical second regeneration and normal checking. Sidecar counts are
368/373/190 respectively. Each step checks original source/index preservation.
The earlier diagnostic pass has368 sidecars; its24-root result predates the added
fuel-classifier unit controls, so final current-source all-feature execution is
still required. Original215 helper SHA checks pass again.

`assertion-conservation.json` under `.flow/tmp/fn187-feature-corpus/` maps five
original public roots to11 added cases plus the existing cluster-refusal case.
Its22 source hashes verify; no originals have been deleted. Canonical lint-rust
then finds four local pedantic/style issues (single-match, boolean simplification,
101-line function and redundant clone). The scoped agent is correcting those
without allowances, fixture changes or weakened checks before the final freeze.

The four style fixes are now verified: session93845 terminated0 for canonical
`make -C tools/gomad/internal/backends/wasm lint-rust`. Rustfmt and all six
strict Clippy configurations pass with no added exemptions. Evidence is
`feature-corpus-lint-rust-retry-2026-10-19.log`. The feature helper module is
SHA25600630eeb555870bdaa035ab487fb6a4388755acf9d8c52d32d56aed547214670;
the refreshed assertion mapping binds22 files and99 line references, SHA256
8c59ccc3fe4ecb750fa4cfa807970dd590613484b85a0022dc532450485faf29.
Staged diff remains0a343ce3… unchanged. Current-source compiled test rosters and
coverage inventories are being refreshed before new immutable phase captures;
no final coverage certificate, deletion permission or integration is claimed.

The previous default phases completed baseline Rust corpus9 roots, candidate
Rust corpus28 roots and candidate invariants134 roots. Baseline Go full and
corpus arms also passed. Candidate Go full failed its two broker-lifecycle
roots: all10 added generated fixtures still carried implementation109cabe…,
predating the reviewed process/measurement ownership change to9b087b96….
The failed command, raw profiles and source manifest remain in the old phase;
it is not a final current-candidate conservation certificate.

The existing Go `TestBrokerLifecycleCorpusIsCurrent` oracle regenerated those
10 fixtures in the R3 checkout. Replacing only the new digest with the old digest
reconstructs every exact old SHA256 from the frozen source manifest: no request,
observation, ordering, capacity or assertion byte changed. Both broker roots and
the compiled-model identity control pass without regeneration enabled; a second
oracle regeneration is byte-identical across all10 files. Evidence is
`/home/agent/wt/evidence/fn187/r3-current/broker-lifecycle-rust199-*2026-10-19*`.
No generator, identity check or production source was modified. Staged diff
remains0a343ce3… unchanged. The correction uses the existing Go oracle, not edited
expected outcomes or a weakened identity gate.

Fresh candidate phase `r3-rust199-broker-identity-2026-10-19/default` is now running
under owned session57726 via `run-rust199-identity-freeze.py`. It captures fresh
source/tool pins, builds matching instrumented Rust1.99.1 objects, and serially
runs all Rust/Go full, corpus and invariant arms with strict exports. The earlier
baseline phase is unchanged and verifies. Comparisons and nondefault feature
configurations remain required; no task completion or integration is claimed.

### Matched-release candidate stopped: required memory admission failed

This terminal checkpoint supersedes the live session9428 descriptions below.
The historical baseline passed all12 scenarios and its retained report remains
valid. The candidate command terminated exit1 after repeated unchanged8GiB
MemAvailable admission failures; the root safely interrupted its owned CLI once
failed rows were confirmed. No successful candidate comparison or speedup exists.
Raw errors and partial rows remain under
`/home/agent/wt/evidence/fn189/matched-release-current-2026-10-18/`.
No comparison helper remains running.

Read-only source investigation confirms that all four phase-boundary admission
checks are intentional: `TestDiagnosticClusterPairAdmission` requires refusal
with8GiB minus1KiB and no dispatch/measurement for each phase. They must not be
moved or relaxed to pass this run. The separate helper RSS limit is not an
exception to the minimum available-memory policy. Generic collection continuing
after an ordinary failed sample is also explicitly tested; no runner policy was
changed. Source inspection cannot attribute the shortage to a specific process
because refusal receipts do not retain the sampled available-byte value.

Post-cancel host memory is approximately9.11GiB available out of19.78GiB total.
Process cwd/ancestry inspection identifies most other large CLI processes with
the separate `umpire` project. The remaining Gomad Claude process is attached to
an interactive terminal, not a positively owned completed headless review. None
was terminated. The memory guard stays intact; fn187 coverage work can proceed
serially without restarting the failed cluster producer into the same window.

### Approved namespace-readiness qualification — current execution

The owner approved register-namespace, verify real standby readiness, then start
workflow. Flow records the narrow exception; upstream validation, all history,
durability, canary and capacity/resource assertions remain unchanged. Both
fixture copies now implement that ordering. The new29-case predicate root and
existing replication-readiness root pass after intended RED, as recorded below.

Primary HEAD is now `60e4dc6ca1` (an externally created checkpoint, not a commit
by this continuation). Rehashing the previous5393-file source roster finds
exactly two source changes, the approved recovery/replication fixture files.
New manifest is
`/home/agent/wt/evidence/fn189/namespace-readiness-2026-10-18/qualification/source.sha256`,
SHA256 `4fb14d40ced978f5dbd046f17a18a6372bce83041990afeeee9b5e2795c2b661`.
Fresh actual independent-cluster success then canary qualification is live under
owned session5455, with `run-cluster.sh`, `cluster-success.log` and
`cluster-canary.log` in that directory. The success cell has now passed in783.99s,
with its child process terminal exit0 and `cluster-success.exit`0. Its retained
result is `cluster/wasm-qualifications/v1/result-f4065475a18ae1118fad022c3c0802492160365d36cdbca05531ec2e3c22ed98.json`.
Qualified/deterministic/target-success are true; both recorded executions have
attempted exact replays with match=true. All20 samples are present. Budget passes
with zero violations, helper peak3379494912 bytes and operator peak1143832576,
below its recommendation. Four pre/post-crash observations and history/convergence
oracles pass. Actual module is `sha256:b1ac4d230191fa3a52623f05aa165a2625816303d7bc46f76c8402790c5283ba`.
Canary completed successfully as a negative-control gate in578.20s; the child
and owned session5455 terminated exit0. Its retained result is
`cluster/wasm-qualifications/v1/result-205794ab1a14ccddd807dde377d93910003cff48fbfeedb4b0ef3e77bdf240d1.json`.
As expected, qualified=false and target_success=false: the intended target
oracle failure is deterministic, with both exact replays attempted and matching.
All20 samples are present, WithinBudget=true and violations are empty. Helper
peak is3375185920 bytes and operator peak1064353792 bytes; neither exceeds its
budget/recommendation. The post-run5393-file source check also passes.
The same module and budgets were used, with warm
preparation2.682811418s. The launcher uses new staged fixture copies,
unchanged detached request/broker inputs and budgets, the stock standalone Go,
unchanged3700def4… helper and an8GiB pre-helper safety observation. It is the
sole compiler/runtime owner. The earlier failures and diagnostic captures
remain intact. `complete-workload-corpus.sh` completed under owned session9477,
terminal exit0: the other eight exact Makefile cells passed in separate fresh
processes, followed by the unchanged mandatory ten-cell aggregate. Every cell
is deterministic with two matching exact replays, the expected success/canary
outcome and WithinBudget=true. The final5393-file source check passes. No
old-source partial results were imported. `matrix/aggregate.log` records10
cells, producer/publication CPU2434670252000ns, checker CPU181603000ns and
checker lifetime RSS41910272 bytes. Both onebox cells use module
`sha256:962406ec671028e2a5e1fc11d82fa6950b1cdfc21cafc5137f765a30fe2a4963`;
SQLite uses `sha256:53cf282fa4b64776501b064fb7c0f8dd649b292590c64b82d0823221151ac3d4`.

The first supplementary SQLite attempt, owned session45395, terminated exit1
before executing selection0: the separate preparation root changed the
path-aware runtime inventory fingerprint. Retained source/provenance comparison
shows identical helper/compiler/model/engine and module; after normalizing only
the preparation-root prefix, the only inventory content difference is the
path-bearing generated `overlay.json`. The primary inventory fingerprint is
`88fce42bd5856fd2a0688df8ba9de19f3a8fa20bd610b0b31a631d39143ecbdb`;
the distinct-root inventory reproduces as
`e4e1840dd4698c5de5f6b808a4686820623245648b82102338eb62be2dc7e09e`.
No source, report bytes, comparison or guard was changed to absorb this refusal.

A fresh supplementary four-state SQLite attempt completed under owned
session87330, terminal exit0, via `run-sqlite-selections-shared-preparation.sh`. It binds the current durable
success result `result-855d64bf8f2e1030fefc0fa97c29cfdfdc740bbf592506021e85c31221e2c278.json`,
runs each selection in its own process using the primary preparation/cache
location, then relocates only the newly published content-addressed Result and
receipt pair into `sqlite-selections-shared-preparation/wasm-qualifications/v1`.
Artifact paths and immutable report bytes remain unchanged, with a relocation
hash journal. The exact catalogue gate validates those four supplementary pairs.
The launcher also requires the original ten publication pairs to be byte-exact
and reruns the mandatory corpus gate on the original ten-cell tree. This is no
claim of native backend qualification. All four selections pass; the exact
catalogue gate passes with four reviewed states/four measured cells, producer
and terminal-publication CPU114323487000ns, reader CPU552630000ns and reader
lifetime RSS38256640 bytes. The original publication hashes and all5393 source
hashes verify. The original ten-cell mandatory corpus recheck also passes.
`run-temporal-inspection.sh` has now started the retained
shard/workflow/onebox inspection gate under owned session58874, now terminal
exit0. All three subtests passed: shard120.14s, workflow122.25s and
onebox1138.35s (whole gate1380.74s). Exact retained replay, inspected-prefix
verification, witness matching, stacks, heap, probes and Temporal snapshot
assertions pass, with evidence bytes unchanged. All5393 post-run source hashes
verify, and the temporary fixture copies were cleaned up. Shard/workflow
retained record hashes are respectively
`sha256:961e3605020fb1b4df6b4ebc288cc6cd1ceca1bc4aadc51e2cf489dcc621c050`
and `sha256:9deee67c963bc3aa5f1e9c82a4083ba071f1caa24c387f78da5278fec210c795`.
The inspection gate passes; neither spec is done.
The onebox recording has reached the intended inspection canary after workflow
timers and activity retries complete, with retained stdout
`workflow completed [step 0 attempt 2 step 1 attempt 2]` and the explicit canary
message. Its retained artifact is
`temporal-inspection/onebox/artifacts/v1/campaign-20261019T000453.949249338Z-8b08d10bdcef10d98eeafee9d5e71dc7/failures/sha256-5d52b590e4781e6931a82752600f4df3`,
record `sha256:ca5704fa71a5607ec16993c878d350d40d2642d25661e85f58384cdeebb8e003`.
The completed onebox inspection resolved history incarnation1 at choice30197,
trace45706, order489457, virtual time180009000000ns. Its ordinary recording,
exact replay and inspection assertions all passed; no source or budget changes
were made during the gate.
Current HEAD is `d200fb9be26b916fab433db1ebe882eb54e06e31`, another externally
created checkpoint; the actual source manifest remains unchanged. Both current
fn-187 default source/tool snapshots verify, and its staged patch remains
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`.

The required matched release comparison has now started via
`/home/agent/wt/evidence/fn189/matched-release-current-2026-10-18/run.sh`.
Owned session9428 is live. Both baseline source controls pass (compiled model
identity and complete/nonpositive authorization-envelope controls). The
authentic historical baseline release-helper build passed in193.59s with child
exit0; admitted helper SHA256 is
`8f576c508cc5fe81680d6eae79e839f562e5777194953dcda7c33b6e575771a2`.
Its build/admission receipts remain under
`baseline-build/release-helper-3849881496/` in the measurement evidence directory.
`TestMatchedReleaseClusterBaseline` passed in2099.80s, child exit0: all12
scenarios, one warmup and three measured repetitions per seed7/19/42, passed
modeled whole-cluster crash, durable reopen, interrupted-prefix replay, full
replay and the recovery oracle. The final report confirms scenario_passed=true,
all12 rows passed and3 distinct choice traces. Its retained path is
`baseline-measure/matched-release-baseline-2534281347/matched-release-report.json`,
SHA256 `d1d7ab0879f9d862a5d9f85c6244969c408bc91518ef9fd967ec89a6f28a3b60`.
Authorized-envelope medians for seeds7/19/42 are125.304/116.882/114.205 seconds;
separate parent-CPU medians are93.290/87.075/84.296 seconds. Raw samples, ranges,
compiler and helper observations remain in the report and original producer
artifacts; these numbers are not a speedup claim.
The candidate measurement command build and authentic Rust1.99.1 release-helper
build both passed, child exits0. The independently built/admitted measurement
helper SHA256 is `53d1167e004c921beec8db79e9c304183790e45a5ab8670d5e91502942cd9743`,
with original receipts under `candidate-build/release-helper-1261089091/`.
This is a distinct measurement helper identity, not a relabeling of the earlier
qualified3700def4… helper; both identities and their source/build evidence remain
retained. The actual `matched-release-candidate` command is now live under the
same owned session9428. Guest preparation completed and the actual
`seed7-rep0` warmup directory and live cluster helper are observed; no candidate
scenario completion or complete comparison is yet claimed. Its authentic module
admission binds source HEADd200fb9be2, module
`2c6433d52554097ebdd597a272a426ff89403ba18adf3e52f107c4eaea47e7bb`,
runtime key7c220e2c…, runtime implementation6940d421… and model9b087b96…;
the raw receipt is
`candidate-measure/matched-release-candidate-186343486/module-build-3821070380/module-admission.json`.
These are the measurement producer's actual identities, not renamed qualification
module identities. Primary and
clean-candidate5393-file source checks passed before dispatch. The producer's
`native-tmp` scratch root under that evidence directory is retained while its
builds/executions are live; this does not execute the deferred native backend.
It serializes authentic helper builds and actual12-scenario baseline/candidate
producers (seeds7/19/42, one warmup plus three measured repetitions each), with
separate parent/compiler/helper costs and real authorization/classified-reaped
boundaries. This is retained fn-178 R4/R8 acceptance, not fn-185 profiling.
The historical baseline keeps its pinned Rust1.94.1; the current candidate uses
Rust1.99.1, both on the same authenticated stock Go1.27.1 and host. Record this
toolchain difference with the results, not as an isolated attribution of speedup
to only the broker implementation. No candidate measurement pass or speedup is claimed.

Git worktree creation session32193 terminated exit0. New clean detached checkout
`/home/agent/wt/fn189-release-candidate` is held at `d200fb9be2` because measurement
admission requires a clean tree; it has no source changes. All5393 source entries
match the qualified primary candidate after remapping only the root path, recorded
in `qualification/release-candidate-source-check.log`. It is retained until its
pending measurement and evidence retention finish; the existing20 worktrees and
their documented retention reasons remain unchanged (21 total now). Primary
evidence notes remain dirty here; no commit, push or PR was made by this run.

The paired default Rust full-coverage owner61194 is terminal exit0, baseline352
passed/2ignored and candidate201passed/2ignored. Both strict exports pass with
inventoried external diagnostics. Remaining arms/comparisons and other feature
modes still gate R3; no conservation certificate or original deletion is claimed.
The staged R3 patch is still0a343ce3… and its sources remain unintegrated.

The initial two source-audit dispatches failed before reading a diff because
their shell ignored `workdir`; neither issued a clean verdict. Fresh-context
retries used explicit repository paths. Their scoped same-family GPT reports
follow verbatim; neither supplies final task acceptance or runtime qualification.

### Correctness axis — namespace readiness

## Quality Audit — Correctness axis: namespace readiness delta

- Files changed: 2 · Critical 0 · Should Fix 0 · Consider 1.
- No blocking correctness finding. Live qualification remains with the conductor.

### Consider

- **tools/gomad/internal/backends/wasm/wasi/integration_test.go:3517** (Conf 100): The recurring source gate selects only the older readiness test, omitting the new 29 namespace controls — extend its `-run` selector to include `TestReplicatedNamespaceReadiness`.

### Evidence

- **Walked:** registration → validated standby frontend `:7233` → same ID/REGISTERED/global/exact replicas → workflow creation. `DescribeNamespace` reads standby persistence without cross-cluster forwarding; RPC/polling contexts share the 90-second bound.
- **Executed:** scoped diff, copy comparisons and retained-log reads. Both copies match; retained logs show the undefined-predicate failure followed by both readiness tests passing. No builds, tests, runtime operations or writes performed.

### Preservation

- **Walked:** local recovery registration/start ordering and existing five-event, durable-result and observation assertions remain intact. Reopen, oracle, canary and resource-budget callers are unchanged.

### Test Budget

- Approximately 80 control-test lines excluding initial fixture setup versus 43 helper/caller/import lines; below 2:1. Existing assertions were preserved.

Correctness axis:1 Consider; no blocking finding. The recurring selector will be
expanded after the current source-frozen runtime owner terminates.

### Standards axis — namespace readiness

## Quality Audit — Standards axis: namespace readiness

### Summary

- Files changed: 2 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No actionable standards findings in the diff against `60e4dc6ca1539f78b822ea31ba0dd7c36bef1243`.

### What's Good

- **tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go:117** — Walked: private predicate and bounded `await.Require` keep readiness checks local; named table cases reuse a cloned response.
- **tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go:142** — Walked: extracting registration creates the required ordering seam without widening public APIs or retaining a second workflow-start path.

### Scope

- Executed: scoped diff and line counts; neither file crosses 1,000 lines.
- No builds, tests, runtime qualification, or writes performed.

Standards axis:0 findings; no worst tier.

### Helper publication continuation — current source freeze

This is the current checkpoint; live-handle and paired-phase descriptions below
are historical where superseded here. Neither spec is complete. The ordinary
canonical full gate passed on helper `098add07…`, then runtime session73782
terminated exit2 before the first guest: Cargo's fresh default build published
a helper with link count2, correctly refused by the unchanged execution-input
ownership check. This is not a determinism failure or a runtime seed pass.
The failure remains in
`/home/agent/wt/evidence/fn189/rust199-full-gate-identity-fixed-2026-10-18/runtime.log`
and `runtime.exit`. The rebuilt helper identity differs from that earlier full
gate, so a fresh full gate is required, not a silent reuse of that pass.

The actual production Make recipe now always copies through a fresh temporary
inode, verifies bytes, sets0755 and atomically renames the result. Default target
and already-matching custom outputs no longer bypass normalization. No ownership
check, qualification threshold or budget changed. New
`tools/gomad/internal/backends/wasm/helper_build_test.go` executes the actual Make
recipe against a controlled external Cargo-output fixture. Intended RED catches
default count2 and matching-custom count3; GREEN passes default/custom/matching
cases through the real hostfs bounded read and verifies permissions. Logs are
`helper-publication-red.log` and `helper-publication-green.log` in the failed
qualification directory above. Primary/R3 Make and test copies match exactly.

Actual canonical helper build now passes; published helper is regular,0755,
link count1, SHA256
`3700def42eec69623251d87e97ba0ddde41906863569edc17bab93bb95344525`.
Fresh primary full gate PASSED, observed owned session60237 exit0, with log
`/home/agent/wt/evidence/fn189/rust199-helper-publication-fixed-2026-10-18/test-wasm.log`.
Log SHA256 is `3dd27f156d3cf14f7f546e29acc82783b9d2cf89387c432553bb69bae97a9fda`;
the WASI package passes in390.460s, including100 fresh-guest output/transcript
equalities and the unchanged ownership/identity/capacity controls. The actual
helper hash remains3700def4…; guest-test logs record that same identity.
The guarded fresh32-seed runtime run PASSED, observed owned session25163 exit0,
using `run-runtime.sh`/`runtime.log` in that same directory. All32 reports retain
25 recordings and one exact replay:800 recordings and32 replays. The aggregate
passes, with30 choice digests,32 outputs and both canary outcomes. The source
checks before and after pass for all5393 entries. Runtime log SHA256 is
`6cd1bde84dabd614e1e86a33b1dd5368b2105c15be958ff41545fe97ac25d7a2`.
The initial workload run, session60324, terminated exit2 after both controlled
cells passed. Cooperative admission correctly refused the module-cache compiler
distribution. No code or threshold was changed: the existing standalone stock
Go at `tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go` is a regular,
single-link0755 executable outside the module cache, version1.27.1, with the
same authenticated executable SHA256 `1675694e…`. The failed partial attempt
and `workloads.exit` remain intact.

A fresh canonical ten-cell attempt terminated under owned session69652, exit2,
using `run-workloads-stock-distribution.sh`, `prepare-inputs-stock-distribution.log`
and `workloads-stock-distribution.log` in that same evidence directory. Results
go to a separate `workloads-stock-distribution/` tree, with new inputs and
byte-identical staged fixture copies. All four fixture cells, both durable-SQLite
cells and both onebox cells have passed:8of10. Onebox success/canary take274.81s/
126.15s. The first multi-instance cluster cell failed in369.80s with
`finish simulation: WASI capacity: network-v2-evidence-bytes`. Cluster canary and
the mandatory aggregate did not execute. No ten-cell/aggregate pass is claimed.
The actual failure is the128MiB reconstructed network-document limit; missing
accepted samples cause secondary resource-accounting errors, not proof of an RSS
budget overrun. Empty executions do not establish whether guests committed.
Read-only diagnosis confirmed each document combines per-instance operations and
expanded World with the cumulative shared network Record. Which component
dominates remains unmeasured. No source, fixture, oracle or budget was changed.
All previously owned runtime/compiler handles are terminal. A separately labeled
nonacceptance diagnostic reproduction is the next investigation step.
The first actual diagnostic terminated under owned session16579, exit1, with files in
`/home/agent/wt/evidence/fn189/cluster-network-capacity-diagnostic-2026-10-18/`.
The explicit Go build overlay changes only host-side capture before classification
and error-path component publication; production source and helper stay frozen.
`run.log` and `run2.log` retain two pre-execution launcher refusals (mandatory
intent without a manifest, then a missing relative-source `./` prefix).
`run3.log` records the same actual guest module SHA256 `5b26fdef…`, preparation
1m26.443787083s. The capture proves active reached bidirectional replication
readiness and emitted the correct five-event observation. Its terminal is
`capacity: network-v2-transcript-bytes`; the reconstructed-document capacity
masked that earlier failure. Operations occupy134217519 bytes, native Record
2212091 and expanded World4718357. Of471834 operations-log entries,460633 are
polls and456907 polls remain pending, through logical241.830977071s. Shared
Record duplication is not the dominant size cause. Exact original component
captures remain in `parts-run3/`, with raw network and stdout in
`terminal-3900236330/`. Pre/post source-manifest checks pass. A second provisional
run added only host-side stdout mirrors for both nodes to distinguish standby
progress from active waiting. That run, owned session93293, terminated exit1 in
187.51s, again with exactly the same component sizes and active observation.
Both nodes' stdout proves local and peer History readiness at logical32s;
standby emits no workflow observation before the capacity failure. Pre/post
source checks pass for all5393 entries, and the helper SHA remains3700def4….
No diagnostic timing or provisional result is qualification acceptance.

Offline HTTP/2/HPACK and generated-protobuf decoding of the original retained
operations provides a more specific upstream compatibility finding. All three
workflow tasks1048581/1048586/1048594 leave active at logical32.000000001s.
Standby acknowledges low watermark1048595 at32.100000001s. The namespace-create
message for the same namespace is not sent until32.863623988s. Exact filtered
evidence is `protocol-order.json`, SHA256
`fc0b589dc5d9f90196124c67e28ff972e61f7d7cbd05e3fef000a61173a55144`;
the full decoded packets/messages and diagnostic Go readers are retained beside
it. No malformed HTTP/2 or HPACK decode was observed; ordinary GOAWAY frames
remain represented rather than treated as parse errors.

Upstream ancestor `f2b571e2eee8a9143fbc4711f3db8b5a37146b3b` added default-true
`ValidateReplicationTaskSourceCluster`. At
`service/history/replication/executable_task.go:890`, a missing namespace now
returns apply=false without eager namespace synchronization. The versioned
transition executor treats apply=false as successful skip. This matches the
observed task acknowledgment before namespace propagation and subsequent absent
standby history; exact branch execution was not logged because the unchanged
fixture uses a no-op logger. Namespace/workflow ordering is the leading concrete
compatibility cause, not proven defective WASM routing or shared-record growth.
Independent read-only research verified the exact packet order, configuration
and executor branches, and the upstream
`TestGetNamespaceInfo_NamespaceNotFoundSkipsRefreshAndProcessing` explicitly
locks the new behavior. It found no namespace-validation override or eager
`GetNamespace` request in the retained traffic.
Disabling this upstream validation, increasing limits, weakening history oracles
or relabeling provisional output is not acceptable. A proper fixture change
would require namespace readiness on standby before creating workflow tasks, but
fn189's explicit no-test-rewriting/unchanged-fixture contract requires owner
direction before applying that change. No source or acceptance criterion has
been changed. All compiler/runtime handles are terminal.
Dedicated extra gates have not started. The selected-SQLite launcher now requires
the new attempt's zero-exit record and reads its result tree, not the failed
partial attempt. The launchers
(`run-runtime.sh`, `run-workloads.sh`, `run-extra.sh`) pass syntax validation and
require preceding zero-exit records; this is not qualification acceptance.
Its source manifest
has5393 absolute regular-file entries, including the untracked regression test
and module mise file, SHA256
`c3bcfd4b825e397e3c35dea1c112e933485f43dcd00d472c302ca08ae23812dd`.

All64 coverage-tooling tests still pass in `helper-publication-tooling-green.log`.
All17 tools,153 source bindings and15 selector records were checked. Fresh
paired source-only phases under `/home/agent/wt/evidence/fn187/r3-current/` are
`before-rust199-helper-publication-2026-10-18/default`, source pin
`b023222af555bce5c0643e4141ebd44c0efcdf089ec1a29c3be77209fd48b5ae`,
and `r3-rust199-helper-publication-2026-10-18/default`, source pin
`7c4b30874446661c0af86a871cc311a9ed3ff73dec6afd62523edd4a3b99fd50`.
Previous paired snapshots preserve their pre-fix Make bytes and are historical.
No new coverage object/profile/certificate exists. Staged R3 patch remains
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`.

After the namespace-ordering diagnosis, the frozen paired default Rust coverage
run is live under owned session61194. Both source-only phase verifiers passed
against their actual roots immediately before launch. The serial launcher is
`/home/agent/wt/evidence/fn187/r3-current/run-rust199-publication-pair.py`;
it builds baseline first, runs its complete original test roster and coverage
export, then builds/runs the candidate. Build JSON/stderr, command/exit records
and per-arm logs remain inside each phase above. Rust1.99.1, matching LLVM,
separate raw profiles and the private guarded Cargo target remain mandatory.
No coverage pass, deletion, integration, commit or task completion is claimed.
That paired run is now terminal exit0. The baseline full roster passes352 tests
with2 ignored; candidate passes201 with2 ignored. Both strict coverage exports
pass with their explicitly inventoried external hash-zero diagnostics. Corpus,
invariant, Go, cursor and final comparison arms remain required; no conservation
certificate is claimed. The staged R3 patch remains unchanged.

The owner subsequently approved the narrow namespace-readiness fixture change.
The exception is recorded through Flow in the fn189 spec. Intended RED is
retained in `/home/agent/wt/evidence/fn189/namespace-readiness-2026-10-18/readiness-red.log`
(owned session5080 terminal exit1, undefined readiness predicate only). The
scoped implementer prepared the registration/workflow split, bounded real
standby frontend readiness and29 predicate cases in both fixture copies.
Fresh staged GREEN verification passed under owned session31667, terminal exit0,
using `readiness-green.log` in that directory. Both the new readiness root and
existing replication-readiness root pass. No full changed-fixture qualification
pass is claimed. All owned compiler/runtime handles are terminal; the old cluster
gate remains red until a newly frozen candidate passes.

### Correctness axis — helper publication

No confirmed correctness findings in the bounded helper-publication fix.

**Walked:** fresh staging creates an independent inode; copy, chmod, comparison, and atomic rename are failure chained. The regression covers default, custom, and matching custom outputs through the production ownership check.

**Executed:** primary and R3 Makefile/test copies are byte-identical.

Tests and runtime gates were not rerun by this reviewer. No shipping verdict.

Correctness axis:0 findings; no worst tier.

### Standards axis — helper publication

## Quality Audit — Standards axis: helper publication

- Files reviewed: 2 · Should Fix 0 · Consider 0. No material standards findings.
- Evidence: cited. The Make recipe has one publication path; the Go test exercises default, custom, and matching-output cases through actual Make and the existing host filesystem API.
- Evidence: executed. Canonical and R3 copies match byte-for-byte.
- No builds, tests, runtime, LLVM, or writes performed.

Standards axis:0 findings; no worst tier. Scoped same-family Codex reviews,
not final task acceptance or an integration/shipping verdict.

### Paired Rust 1.99 tooling continuation

This subsection supersedes the export-policy and fresh-baseline tooling work
listed as pending below. Neither spec is complete; no original test was removed,
no R3 integration occurred, and its staged patch still hashes to `0a343ce3…`.

The exact external unwinder diagnostic disposition is implemented test-first.
`export-diagnostics-red.log` retains the original unknown-crate rejection;
`export-diagnostics-green.log` and `selection-pin-green.log` pass. Admission
requires the exact symbol, hash zero, unique registry mapping, package version
47.0.3, exact source path and source SHA. The existing five-crate policy is not
broadened. Negative controls reject other functions, versions, nonzero hashes,
local/absent/ambiguous mappings, changed/missing source, unexpected diagnostics
and incomplete summaries. The historical failed phase is not reclassified.
Its `full/unwinder-profdata-show.txt` and `unwinder-observation.json` retain the
nonzero hash/count683 observation, bound to six profile/object/export hashes.
Independent review checked those hashes and reproduced the complete297-record
inventory, with exactly one special disposition. Compiler causation is unproven.

Both comparison sides now require source/object pins, declared source roots and
their own LCOV normalization; neither a `before` prefix nor old fixed line numbers
provide an exemption. Cursor schema3 binds both phases and roots. Go baseline
full selection derives all65 original roots from the primary's actual four files.
Candidate full selects the reviewed union of functional/private/corpus roots,
requires every surviving original root, and rejects arbitrary full overrides.
Current candidate union is63 roots:61 WASI and2 network. Source, production
coverage and full/corpus-plus-invariant obligations remain independently checked.

Snapshots now bind the actual executed external tools/selectors, including their
frozen copies. A supplied selector must match its exact pinned path/hash. Intended
RED evidence is `external-tools-red.log` and `selection-pin-red.log`; final expanded
tooling suite is64PASS in `fresh-paired-tooling-final.log`. The worker's additional
red-green logs are `fresh-pair-red.log`, `fresh-cursor-red.log`,
`fresh-go-selection-red.log` and `fresh-pair-cursor-green-final.log`.
All17 inventory tools,152 source bindings and15 selector records were checked.

New paired phases under `/home/agent/wt/evidence/fn187/r3-current/` are
`before-rust199-paired-2026-10-18/default` (source pin
`b37a04a46cfb9e24ad9d888f42296099beff5b8744408fe933300d003993ba6b`)
and `r3-rust199-paired-2026-10-18/default` (source pin
`c1ffbcb20eb67691d7a922b378a630b2d4dfe4e62e99d3a1dcdd31c166d7da41`).
Both bind35 actual tool/selector files; baseline/candidate source inventories
contain856/2104 files. These phases have source pins only so far: no new object
or conservation certificate is claimed. Preserve all earlier phases unchanged.

The fresh primary canonical `test-wasm` gate PASSED, observed session12238 exit0;
log `/home/agent/wt/evidence/fn189/rust199-full-gate-identity-fixed-2026-10-18/test-wasm.log`,
SHA256 `e2a3ce7ed2e309bc9d83cb48cf3c6a024fab4e809f882b0c9a07cdc86f4d09f2`.
The WASI package passes in379.272s, including the corrected implementation identity.
Conformance, architecture/projection and all canonical packages pass. Opt-in
qualification/prototype skips in this ordinary gate remain separate obligations.
Its absolute regular-source qualification manifest contains5392 entries, SHA256
`f3814c8419889a0753acafe170f50a6b9b32cfb665868c2c57952455f1998340`,
including the new module-local mise file. All entries passed a fresh hash recheck.
The canonical32-seed runtime qualification is now live under owned session73782,
using `run-runtime.sh` and `runtime.log` beside that gate log. No runtime/compiler/
LLVM work may overlap this sole resource owner. Native,
profiling, ignored benchmarks, publishing and migration execution remain excluded.

Qualification launchers are retained beside that log as `run-runtime.sh`
and `run-workloads.sh`; both pass `bash -n`. Runtime has started; workloads have
not executed. They require
the preceding conductor-observed zero-exit record, fresh output roots and source
hash rechecks. Runtime keeps all32 seeds/25 recordings/one replay and enables the
existing Linux8GiB-available-memory admission guard. Workloads retain the canonical
ten cells and aggregate, staging byte-identical committed fixture files only.
The paired candidate phase's `go-production-source-tree-equality.json` independently
records exact equal hashes for all48 recursive production Go files: WASI24,
network6 and volume18, bound to both source pins. This proves source equality,
not coverage conservation. The narrower package-root-only receipt is retained too.

### Correctness axis

## Quality Audit — Correctness axis: paired fn-187 tooling checkpoint

- Reviewed 8 tooling modules and 8 test files directly; directory is Git-ignored.
- Critical 0 · Should Fix 0 · Consider 0. No confirmed correctness findings.

Evidence: **executed** all 64 Python tests, passing in 1.408s with bytecode writes disabled.

**Walked** source/tool/selector bindings, per-phase LCOV roots, full-only certification gates, exact Go statement identities, unchanged-context Rust mapping, cursor receipt validation, and the exact unwinder diagnostic exception.

The cursor proof supplies one identity mapping while full and corpus-plus-invariant coverage remain independently required.

Scope excludes live conservation execution, integration, task completion, and shipping. No source or retained-evidence changes.

Correctness axis:0 findings; no worst tier. Scoped checkpoint, not final impl-review.

### Standards axis

## Quality Audit — Standards axis: paired fn187 tooling

- Files reviewed: 12 · Should Fix 0 · Consider 0.
- No material standards findings. Reviewed the ignored files’ complete current contents; the scoped Git diff is empty because these files are untracked/ignored.
- Evidence: cited. No tests, builds, helpers, LLVM commands, or evidence mutations executed.

### What’s Good

- Source, selector, and diagnostic checks have shared owners in `phase_snapshot.py`, `compare.py`, and `coverage.py`.
- Mutation cases use named subtests, and tool mutation probes copy sources into temporary directories.
- All scoped files remain below 1,000 lines; largest is `coverage.py` at 329 lines.

Fresh-context reviewer used the inherited Codex session model; the conductor retains writer/reviewer model-family provenance.

Standards axis:0 findings; no worst tier. Writer/reviewer are same-family Codex;
these are source/tooling checkpoints, not a full task shipping verdict.

### Earlier sequence-adapter and full-gate checkpoint

This section supersedes the earlier live-handle and pending-adapter descriptions
below. The primary full WASM gate is terminal, exit2 after10m39s. Conformance,
projection and architecture checks passed; its sole reported failing root was
`TestImplementationIdentityBindsCompiledModelDigest`. The reviewed measurement
ownership move changed the primary's only two modified production WASI files,
`measurement.go` and `process.go`. Their contents and identity owner match R3.
The R3 baseline is older and also carries the already accepted addition of
measurement.go to identity inputs. Expected identity now records the actual
`sha256:9b087b968ee0054b2886b670934e37f0e4546939442d61fde69606ac056c4c99`;
the mutation/missing-source test now explicitly covers process.go. All four
identity roots pass in both checkouts. Logs are primary
`.flow/tmp/rust-hygiene-2026-10-18/identity-owner-after.log` and R3
`.flow/tmp/fn187-r3/identity-owner-after.log`. No fresh full-gate exit0,
runtime qualification or workload qualification is claimed.

The two read-sequence adapters are now implemented. With no-op adapters, both
negative-control roots failed for the intended reasons: changed classification
and changed storage maximum were accepted. See R3
`.flow/tmp/fn187-r3/sequence-adapters-red.log` (exit101). Actual adapters compare
full real observations, fresh cluster readers over a shared cursor/Broken, and
three storage reads on one descriptor including actual stored operation counts.
Existing EOF and storage-vector corpus roots replay the new cases as well.
Both original inline tests remain unchanged.

The first strict lint pass caught two test-code style issues, retained in
`sequence-adapters-clippy.log`. After correction, all six configurations and
formatting pass in `sequence-adapters-clippy-2.log`. Complete ordinary Rust
tests pass:201passed, zero failures, two ignored benchmarks, zero filtered,
with all407 helper cases and both originals retained. The seven target summaries
are140/4/3/7/21/18/8 passed. See `sequence-adapters-full-rust.log` in R3.
No warnings appear in that successful test log. All48 Python tooling checks pass
in `sequence-coverage-tooling.log`.

The two new negative controls are explicitly test machinery, not private
invariants. Rust selectors are now functional200/corpus28/diagnostic26/
invariants134/public2/machinery36, with held1/ignored2 unchanged. No new private
waiver was admitted. The staged patch remains unchanged.

A fresh Rust1.99.1 default coverage build completed successfully in4m59s with
zero warnings/errors in Cargo JSON and stderr. Its immutable phase is
`/home/agent/wt/evidence/fn187/r3-current/r3-rust199-sequence-conservation-2026-10-18/default/`.
The first package-clean attempt stopped because Cargo1.99 requires a root
CACHEDIR.TAG absent from this older private target. The directory was inspected:
it contains only Cargo release/platform output and rustc metadata, with an
existing platform-child cache tag. The owner added the standard tag to the exact
guarded target; no broader cleanup or protection bypass was used. The failed log
and command record are preserved as `clean-conformance-cachedir-missing.log` and
`commands-before-cache-tag.jsonl` before retry. The retry clean and fresh exact
conformance build pass, and ordinary helper restoration uses its frozen copy.

All seven frozen full-suite targets pass,201tests/2ignored, and profile merge
passes. Strict export then rejects one newly encountered external zero-hash
variant, among297 diagnostics: serde_json241/base6415/rustix10/wasmtime30 plus
one wasmtime_internal_unwinder. Functional/corpus/invariant arms did not start.
No conservation certificate is issued. Raw profiles, ordinary/conformance
objects, source/object pins and failed export remain intact. Owned sessions57822
and43422 are terminal; no compiler/test/LLVM handle remains live at this point.

The rejected symbol is
`_RNvNtNtCsbt0ASiPIg7r_26wasmtime_internal_unwinder4arch7aarch6417get_stack_pointer`.
It maps uniquely to registry package47.0.3 `src/arch/aarch64.rs:49`, source SHA256
`4daf28dcd10780235ff1216f82c6f21c1c8afd803c594a851277eca8f8e3eecd`.
The combined export reports FNDA683. llvm-profdata independently reports a real
record hash0x44b0d86e39910fc9/count683; the reported missing variant has hash0.
The existing five-crate admission remains unchanged and correctly rejects this
unrecognized crate. A filename-filter probe retained the same diagnostics and
was not substituted for the raw export. Any compatibility disposition must be
independently reviewed, exact to the function/version/source, tested against
local/ambiguous/nonzero/unknown variants, and recorded in a fresh tooling phase.

Read-only baseline research identified current primary after hygiene as the
faithful fresh pre-R3 baseline. It retains all256 original src test names and
all65 original Go roots; the sole absent integration/common name is an added
coverage-harness self-test. Cargo manifests, lock and toolchain are equal to R3.
Production Go bytes match across wasi24/network6/volume18 files, preserving exact
statement comparison. The known Rust conformance cursor seam remains the narrow
existing equivalence obligation. Do not reconstruct from the dirty old
before-guard checkout or reuse its1.94 profiles. Fresh paired phase roots,
executed-tool/selector binding, source normalization and cursor proof are needed;
the final before/after verifier must remain fail-closed. Old records stay
historical. Final full/functional/corpus/invariant/union and original-baseline
conservation remain required across supported configurations.

### Independent correctness followup

No confirmed correctness defects at confidence 75+ in this follow-up.

- Walked: `wasmhost/src/cluster/transport_input_corpus.rs:49` preserves fresh-reader lifecycle over one shared stream and `Broken`; compares actual classification, message, bytes and broken state.
- Walked: `wasmhost/src/storage_transport/vector_corpus.rs:63` reuses the descriptor across all three reads; compares actual failures and actual stored operation counts. Classification/maximum/count guards exercise execution, so they are not tautological.
- Walked: `wasi/identity_test.go:24` retains digest equality, adds mutation/missing-source controls, and updates the expected digest to `9b087b…`. Executed: `cmp` confirms matching primary/R3 measurement, process and identity sources/tests; fixture byte counts and wire hashes match their JSON declarations.

Baseline nuance: R3 also changes production `identity.go` versus its older HEAD, adding `measurement.go` to identity inputs; its current contents match primary.

Source-only checkpoint, same-family GPT review. No builds/tests/writes; no final coverage, conservation, R3 acceptance or shipping verdict.

## Tooling

Both checkouts now pin Rust 1.99.1 through `tools/gomad/mise.toml` and the crate's
`rust-toolchain.toml`, including rustfmt and Clippy. Live Makefile, CI and
measurement-admission pins agree. The old-toolchain admission regression still
uses Rust 1.94.1 as deliberately rejected evidence. Historical receipts retain
their actual compiler versions.

The Cargo policy enables Clippy all and pedantic, plus fallible_impl_from,
needless_collect, redundant_clone, dbg_macro, todo, unimplemented and
undocumented_unsafe_blocks. `lint-rust` checks formatting and all targets in six
feature configurations, with `-D warnings`; `lint-code` and WASM CI invoke it.
Exceptions are scoped and reasoned, not group-wide pedantic/restriction allowances.

Primary `make -C tools/gomad/internal/backends/wasm lint-rust` exits 0 across
default, conformance, diagnostic-profile, diagnostic-fuel-off, guest-profile and
guest-profile plus diagnostic-fuel-off. Evidence:
`.flow/tmp/rust-hygiene-2026-10-18/pedantic-primary-matrix-5.log`.
Earlier failed diagnostic passes are retained beside it. The separate R3 checkout
also passes all six configurations, including formatting and all targets:
`/home/agent/wt/wasm-fn187-functional/.flow/tmp/fn187-r3/pedantic-worker-matrix-4.log`.
That lane's failed diagnostic and compiler passes are retained alongside its log.
No post-hygiene Rust test, coverage,
resource or independent-review result is implied by Clippy.

Matching LLVM tools are installed: rustc 1.99.1 reports LLVM 23.1.1 and its
bundled llvm-cov reports `23.1.1-rust-1.99.1-stable`. The ordinary R3 full Rust
suite passes with one build job and one test thread, no coverage flags and
no ignored benchmarks executed: 199 passed, zero failures, two ignored manual
benchmarks. All three exact resource originals pass. Its first log is
`/home/agent/wt/wasm-fn187-functional/.flow/tmp/fn187-r3/rust-hygiene-default-tests-1.log`;
source/testdata hashes are retained in the adjacent `*-source.sha256` file.
Both corpus-regeneration variables, the residual-case filter and the RSS-child
marker were confirmed unset before tests began. Source hashes matched afterward.

The standards checkpoint found the R3 loader above 1,000 lines. Four unchanged
telemetry/profile projection functions and their prefix constant moved into
`tests/common/helper_corpus_projection.rs`; only visibility/import wiring changed.
The loader is now 998 lines. A byte comparison of the moved block, normalized
only for the two `pub(super)` visibility qualifiers, matched exactly. Test roots
and assertions were not changed. The complete ordinary Rust suite passed again
with the same 199/0/2 results after extraction:
`.flow/tmp/fn187-r3/rust-hygiene-default-tests-2.log` in R3. Its source hashes are
retained alongside the log. The six-configuration formatting/Clippy rerun also
exits 0 after extraction: `.flow/tmp/fn187-r3/pedantic-worker-matrix-5.log` in R3.
The standards followup below confirms the finding is resolved. Source/testdata
hash verification and both checkouts' `git diff --check` pass; the staged patch
hash remains unchanged. No final coverage or qualification is implied.

The World writer's signed time conversions were changed in its schema template,
then regenerated in both checkouts; generated Rust was not hand-edited. Existing
wire bits, admission order and capacity/refusal behavior are preservation targets.

## Architecture failure

The earlier primary full `test-wasm` run exited 2 because `wasi/measurement.go`
contained host supervision and reflection despite being a pure-model file.
Its retained gate log is
`/home/agent/wt/evidence/fn189/resource-resume-2026-10-18/test-wasm.log`.
Runtime and workload qualification did not follow that failed gate.

`MeasureTerminalExit` and its supervisor types now live in the existing
`wasi/process.go` owner. The pure historical converter uses the existing closed
output types and canonical JSON comparison instead of reflection, retaining
null-versus-empty distinctions and strict field admission. Collector provenance
now also binds `process.go`. No architecture exemption or import allowance was
added; the negative import fixture now explicitly includes `measurement.go`.

The focused architecture gate reproduced the failure on both required source
platforms, then passed after the fix. Focused measurement tests also pass,
including historical output shape/unknown-field controls, source existence and
current/old compiler admission. Logs:
`.flow/tmp/rust-hygiene-2026-10-18/architecture-before.log`,
`architecture-after.log` and `measure-owner-after.log` in the same directory.
This does not replace a fresh complete `test-wasm` or qualification run.

The same focused architecture/measurement selection also passes in R3:
`.flow/tmp/fn187-r3/go-hygiene-ownership-after.log` in that checkout.
Primary source-generation tests pass for Rust World schema/tag ordering and
generation/checking of every endpoint; see
`.flow/tmp/rust-hygiene-2026-10-18/generator-after-module.log`.
The first generator test invocation used a relative path across a nested Go
module and failed setup; it was corrected to the module's full import path.

## Preservation and remaining work

The R3 coverage runner now uses Rust 1.99.1 and matching LLVM 23.1.1. Its source
pin includes both toolchain files and the WASM Makefile, and those explicit inputs
are copied into the frozen snapshot. The runner clears inherited compiler flags,
wrappers, corpus filters, regeneration, seeds and the RSS-child marker. Regression
tests failed on the stale pin, omitted inputs and inherited overrides, then passed
after correction. All 48 tooling checks pass again; see R3's
`.flow/tmp/fn187-r3/coverage-toolchain-recheck.log`.

The inventory now binds 148 current source/configuration paths and 15 tools.
All 142 retained-test source hashes/lines are refreshed, with names, paths and
reasons unchanged. The separate host-concurrency classification is unchanged.
The 13 non-binding selector/reason files are unchanged; only the two source-binding
records' hashes rotate. All 178 manifest file hashes match their current files.
This is source inventory verification, not coverage conservation. Retained bodies
are not claimed byte-identical to their pre-hygiene versions. The spec's current
Quick commands now pin 1.99.1 through `flowctl spec set-plan`.

The R3 staged patch SHA256 remains
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`.
No staging, commits, integration, push or CI run was performed. The worker's
source/hash/line bindings and coverage freeze predate these edits and must be
rebound and reverified, not treated as current certificates. Original resource
test passes used the previous compiler and remain historical evidence.

The requested crate-reuse subagent's findings are recorded separately in
`rust-crate-reuse-review-2026-10-18.md`; no new dependency was adopted.
Native execution, fn-185 profiling and ignored benchmarks remain excluded.

The fresh primary helper rebuild/full WASM gate is running under owned host
session 68363. Logs and source hashes are retained under
`/home/agent/wt/evidence/fn189/rust199-full-gate-2026-10-18/`.
The first launch stopped before compilation because `/usr/bin/time` was absent;
its log remains. The corrected launch uses Bash timing, one Cargo build job and
one Rust test thread. The ordinary helper rebuilt successfully in 3m13s; the
full gate has proceeded to the separate conformance-helper build. No passing
full-gate result is claimed yet.

## Remaining inline-test replacement decision

The followup read-only research review recommends extending the existing
`cluster/transport_input_corpus.rs` and `storage_transport/vector_corpus.rs`
component corpus owners. These are component contracts, not helper end-to-end
tests. Neither remaining original is deleted or reclassified yet.

The cluster fixture must own the exact original oversized wire bytes and a
two-read sequence. Both reads construct fresh `TaggedInput` readers sharing
one cursor and one `Broken`. Compare capacity classification/error and absent
Broken, followed by exact stripped `next` frame bytes including newline. Input
must not be rebuilt from the production `FRAME_LIMIT`, which could move the
fixture boundary silently.

The storage fixture must own the original descriptor and framed context bytes,
including canonical key ordering, `/workspace`, capacity1024, limits1000/1000,
return_image=false, all400 original operations and their3000-character paths.
Validate one descriptor, then perform three reads with fresh cursors in the
original order: maximum1 refusal, missing-end refusal at18892800, and complete
success at18892800. Compare actual stored operation counts, not input counts;
retain the explicit context-size-greater-than1MiB assertion. The existing test
module can observe stored state without adding a production API.

Fixture loaders must decode, invoke existing production operations and compare
observations; case-name-specific scenario construction is not an acceptable
replacement. Preserve accepted fixture bytes and existing guards. Demonstrate
negative controls and assertion/coverage conservation with originals excluded
before removing either original. No new invariant waiver is needed.

The four proposed fixture files are now present in R3 under
`tests/testdata/cluster_transport_input/oversized-tagged-frame-followup.{json,wire}`
and `tests/testdata/storage_transport_vectors/pending-context-over-one-mib.{json,wire}`.
They are not loader-admitted or runtime-verified. The existing EOF corpus's
one-JSON roster is deliberately still unchanged; it will reject the proposed
second JSON until the read-sequence adapter and roster are implemented together.
Both inline originals remain unchanged. Do not claim the current worker suite
green based on the earlier 199-pass receipt.

Independent input diagnostics verified the cluster wire's 8388706 total bytes,
exact original padding and next-frame bytes. Its SHA256 is
`d0129c73ce964bf162e12b0e238b6c0f7425dce5d2708f87d346dee39bd0ce93`.
Storage wire SHA256 is
`615ecd1c6d532b01d7e0d6d0f7e23d612de1b33fef2c3f0228eefc2d7d6bbb76`;
decoding its20 frames yields1259101 bytes/400operations with the independently
researched original canonical payload SHA256
`75f2ecdbd473838ca2795841bf93c217a20286a0ac642f34bb57c9ca5e6fc2e8`.
The inventory binds all four additions, bringing source/configuration paths
to152. These diagnostics establish input equivalence only, not test replacement
or coverage conservation. Compiler/runtime work remains serialized behind the
live primary full gate.

The owner subsequently authorized completed-spec worktree cleanup. Flow confirms
fn-184 and all four tasks done; its two clean, merged, unused worktrees `fn184`
and `fn184-base` were removed with Git, followed by worktree pruning. Their ignored
helper binaries were copied and SHA256-verified under
`/home/agent/wt/evidence/fn184/retired-worktrees-2026-10-18/`; its README records
commits and recovery details. No branch or commit was deleted. Twenty worktrees
remain. The fn-189 baseline and all fn-185/fn-187 worktrees remain because their
specs are open; uncommitted evidence is preserved. Historical per-worktree reasons
remain in `fn187-lightweight-correctness-2026-10-18.md`, superseded only for those
two removed fn-184 checkouts.

## Periodic independent reviews

### Correctness axis

Correctness checkpoint: no findings at confidence 75+ in the scoped hygiene and measurement changes.

- Walked: `wasi/measurement.go:109` retains typed null/empty distinctions and strict unknown-key rejection; `internal/measure/regression_more_test.go:1591` covers those cases.
- Walked: `wasmhost/src/abi.rs:419` preserves signed wire bits. Checked narrowing remains behind buffer, file, port or fixed-count bounds, including corresponding R3 sites.
- Walked: request borrows and primary test predicate rewrites preserve behavior; `internal/measure/producers.go:4920` includes the relocated host owner in source provenance.

Scope limits: source-only checkpoint against `bcb72b2bc7`; R3 deletion conservation and full coverage remain unaudited here. No compiler, tests, guests, writes or repository mutations performed. Runtime results are claimed by the dispatcher. Same-family GPT review. No final acceptance or shipping verdict.

### Standards axis

## Quality Audit — Standards axis: Rust hygiene checkpoint

### Summary

- Files changed: 65 tracked primary files plus `mise.toml`; R3 counterpart inspected. Should Fix 1 · Consider 0 · Blocking: none possible (standards axis).
- Source-only, same-family GPT review. Compiler/test results are cited owner evidence; final acceptance, coverage conservation and qualification remain open.

### Should Fix

- **tools/gomad/internal/backends/wasm/wasmhost/tests/helper_corpus.rs:989** (Conf 100): R3 crosses the 1,000-line threshold, from 972 to 1,090 lines (executed: base numstat/current line-count calculation) — extract a cohesive loader responsibility into `tests/common`, preserving assertions and test registration.

### What’s Good

- Cited: `wasmhost/Cargo.toml:25` enables pedantic and selected additional lints; inspected exceptions name individual lints and reasons. No new blanket pedantic/restriction exemption or dependency change found.
- Executed: primary/R3 toolchain files and World template/output match. Cited: WASM `Makefile:11` checks six feature modes; CI invokes that gate before building.

### Limits

- No writes, staging, compiler, tests, Go, LLVM or guest execution performed. Toolchain pins agree at 1.99.1; “latest” availability was not independently checked.

Correctness axis: zero findings at confidence 75+; worst tier none.
Standards axis: one original finding; worst tier Should Fix; resolved by the
cohesive extraction and independently checked below. No final acceptance claimed.

## Quality Audit — Standards axis: extraction follow-up

### Summary

- Finding resolved · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).

Executed: R3 `tools/gomad/internal/backends/wasm/wasmhost/tests/helper_corpus.rs` is now 998 lines; the extracted module is 98 lines. The threshold crossing is resolved.

Walked: `helper_corpus.rs:54` wires the projection module and imports its two entry points. `tests/common/helper_corpus_projection.rs:3` owns the profile prefix and four related projection functions; only `project` and `split_profile` receive `pub(super)` visibility. No introduced standards issue found.

This remains a source-only, same-family GPT checkpoint. The ordinary-suite pass and unchanged registration count are cited owner evidence; the strict lint rerun and final acceptance remain separate gates.
