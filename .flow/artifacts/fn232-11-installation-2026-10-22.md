# fn-232.11 installation receiver checkpoint

Task `fn-232-gomad-rewrite-the-host-side-in-rust.11` remains **in_progress**.
Implementation and the completed actual public corpus are ready for conductor
acceptance. This handover retains the earlier failed attempts and repairs below;
its earlier queued/pending notes describe those checkpoints, superseded by the
final observations at the end. Root owns integrated review and Flow. No final
R12 candidate qualification is claimed.

The Rust CLI now owns install, discover, documented doctor refusal/reporting,
public seed/scoped-choice plan and journal-only recovery. Installation separates
manifest identity admission from build-input admission, publishes exclusively
with mode 0600, authenticates the pinned offline overlay generator inputs and
complete guest configuration, and rejects Go lineage. Discovery survives
removed build inputs and resolves explicit, environment and binary-sibling
selection without fallback from an invalid chosen manifest. The linked helper
is the default; explicit helper admission accepts only the current executable
or byte-identical copies, records the supplied digest and never executes them.
This is the parent-authorized exact-binary exception, not admission of an
independently built standalone helper.

Public seed plan retains a portable authenticated prepared module receipt with
zero guest executions (fn-222 preparation receiver). Scoped choice plan reuses
the existing scope/base-resolution owners and records its required base
execution (fn-230.1 receiver). Recovery repairs existing seed/round journals
without guest starts; repeated recovery reports no change. Native analyze and
Go's actual public plan/selected-doctor refusals remain explicit corpus cases.

## Verification observed

The focused Rust source checks below used the assigned worktree and dedicated
`CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-install`, jobs 4 and Rust
`+1.99.1`; exact commands are in the evidence JSON. Numeric exit observations,
not log scraping, determined status.

- Baseline host tests: exit 0, 66 tests; `.flow/tmp/install-baseline-rust.log`.
- Focused host fixture tests: exit 0, 66 tests;
  `.flow/tmp/install-bounded-host-tests.log`.
- Fixture all-target Clippy with `-D warnings`: exit 0;
  `.flow/tmp/install-bounded-clippy.log`.
- Imported helper aggregate execution regression: exit 0, 3 tests;
  `.flow/tmp/install-merged-helper-bounds.log`. Sink capacity/overflow: exit 0,
  2 tests; metrics retention: exit 0, 1 test, adjacent sink/metrics logs.
- Actual public numeric/config duplicate boundary: intended RED exit 1 against
  the earlier immutable fixture, then GREEN exit 0 for twelve base-0 numeric
  cases and nested duplicate-config refusal. Raw evidence:
  `/home/agent/wt/evidence/fn232-install-corpus/{tiny-red,bounded-green}`.
- Initial public grammar corpus: exit 0, fresh Go/Rust commands, repeated
  executions and byte-identical second regeneration;
  `/home/agent/wt/evidence/fn232-install-grammar-first`.
- Final fixture build including imported metrics, aggregate and typed-role
  corrections: exit 0, 26.03 seconds;
  `.flow/tmp/install-final-immutable-build.log`.
- Formatting, Python syntax and `git diff --check`: exit 0 at checkpoint.

The unchanged Go model/generator validate baseline is reused from root's
`49ea645940`, `.flow/artifacts/fn-232.12-retained-validation.md`: exact
`CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-retained-validation make -C tools/gomad validate`, session 3110 exit 0.
No shared Go generator/protocol inputs were edited here.

Pre-review immutable fixture (retained for actual RED proofs):
`/home/agent/wt/evidence/fn232-install-final-fixture/gomad-567df2b234e8d79931399740cb7111cd0bf16e5e87934ddc13d1aeeaaf0b47f9`.
Its SHA-256 is the filename suffix. Exact 4,453-file source inventories before
and after build plus binary metadata are retained beside it. The sole source
inventory delta was the controlled Python detached-compiler distribution
layout correction during compilation; no Rust/compiler identity input changed.

## Independent review repair batch

Source review found that matching scoped public plans could discard the
successful base under `keep_successes=none`. The repair passes the existing
plan-only execution mode into retention, forcing scoped ordinal-zero retention
with the existing capacity exemption. Ordinary matching scoped exploration
retention is unchanged. The existing public corpus authenticates both matching
and empty-scope retained base artifacts; this actual public proof remains queued.

Review also found missing Go clock zero defaults. The typed adapter handles Go zero defaults for omitted/null typed members,
including clocks, entropy arrays, empty capture metadata, primitive values and
collections, without changing semantic admission bounds. The pinned
Go decoder explicitly uses case-insensitive matching with sensitive delimiters,
and rejects differently spelled aliases of the same typed field. A private
Serde adapter obtains actual schema member names from derives, matches EqualFold
(including Kelvin-K and long-S for ASCII schema names), retains `_`/`-`, rejects
duplicate aliases and leaves arbitrary map keys/raw Values unchanged. The
original configuration and its digest preimage remain literal.

The actual pinned Go decoder/model oracle passed 21 inputs (exit 0); source,
command, inputs and raw results are retained at
`/home/agent/wt/evidence/fn232-install-corpus/config-alias-oracle`. Rust agrees
on all 21 accepted/refused results and admitted clocks; `A`/`a` payload map keys
remain distinct. Actual pre-review public Rust calls failed the intended mixed
case/null accepted-Go inputs before a deliberately missing compiler: aggregate
RED exit 1, three child exits 2, no cache/compiler/guest, retained under
`/home/agent/wt/evidence/fn232-install-corpus/old-config-red`.

Development Cargo clock probe RED exited 101 for omitted read-step admission;
focused GREEN exited 0 for two probes covering seven clock/default/bounds cases
and the 21 exact Go inputs (`.flow/tmp/install-review-clock-red.log` and
`.flow/tmp/install-review-config-green.log`). Temporary observable probes were
removed after the run; retained source is `.flow/tmp/install-review-probes.rs`.
The lasting public matrix now has 27 explicit config/alias/null/range/refusal
rows plus three actual Temporal descriptor cases.
Initial focused Clippy exited 101 for one elidable lifetime; its mechanical fix
passes Clippy `--lib --features determinism-fixture -- -D warnings` (exit 0,
`.flow/tmp/install-review-repair-clippy-green.log`). Formatting, five Python
syntax checks and diff checks pass. No fixture rebuild or guest execution has
occurred after this repair; the successor fixture and complete corpus remain
pending the shared host window.

The follow-up source review identified the captured descriptor's typed decoder
bypassing the field adapter. Capture now authenticates the original raw byte
length/SHA-256 first, parses strict raw JSON, then applies the same typed decoder
to the descriptor and its nested fields. Arbitrary raw Values and payload map
keys retain their literal bytes. The adapter also verifies complete sequence/map
consumption: a 33-byte entropy array had previously escaped the array visitor's
length check, while Go rejects both 31- and 33-byte arrays.

A second pinned Go oracle passed 34 exact decoder/model inputs (exit 0), including
actual Temporal descriptor alias/duplicate/delimiter mutations with recomputed
raw descriptor length/hash; null entropy/capture/manifest/limits and primitive
items; invalid zero limits; and exact entropy array arity. Its source, inputs,
command and observations are at
`/home/agent/wt/evidence/fn232-install-corpus/config-alias-oracle2`.
The original Rust source failed ten concrete comparisons (RED exit 101,
`.flow/tmp/install-review-null-red.log`); the initial repair failed only the
33-byte tail (exit 101, `.flow/tmp/install-review-null-green.log`). After exact
sequence/map end checks, all 34 decoder/model cases agree (exit 0, one focused
probe, `.flow/tmp/install-review-null-green-final.log`). The probe additionally
checks admitted clock, entropy, arguments and unchanged original config. Its
source was removed after validation and retained in
`.flow/tmp/install-null-probe-passed.rs`; raw Rust comparisons remain beside the
Go oracle. Final production library Clippy passes (exit 0,
`.flow/tmp/install-review-null-clippy.log`), with formatting/Python syntax/diff
checks passing. These bounded probes start no guest or overlay/compiler
preparation. No successor fixture build or full public corpus has run yet.

## Merged receipt metadata seam

Root integrated the reviewed installation source while preserving task 13's new
optional typed module metadata. That merge exposed a receiver gap: receipt
validation restored module/provenance and checked typed canonical provenance,
but bypassed the shared actual-module metadata authenticator. The source-only
integration patch is
`.flow/artifacts/fn232-11-receipt-metadata-integration.patch`. Its private `authenticate_provenance_module` wrapper parses the
actual retained optional metadata after canonical typed admission and calls
`prepare::authenticate_module_metadata(top, inputs, module)`. That existing
owner binds the typed top-level value to its input preimage and actual WASM
imports/memory pages; no metadata algorithm is duplicated. Its honest legacy
`None`/absent-input admission remains unchanged. Fresh preparation emits actual
metadata. The wrapper keeps the existing 100-line receipt validator unchanged in size.
The isolated branch deliberately does not reference the not-yet-
received live API in compiled source; root applies the narrow patch to the
merged production owner.

Two public receipt cases now mutate the actual retained top-level imports or
initial memory pages, reseal the provenance file, artifact manifest and receipt,
and independently authenticate all outer artifact hashes before requiring the
named metadata refusal. Inputs, build key, actual module, configuration and
record remain unchanged, so the unpatched merged reader would admit these
lying top-level values. Original bytes are restored and reopened after each
case. These source cases require the fresh merged producer's actual metadata;
none is fabricated for a positive admission. Patch application check, Python
syntax and diff checks pass (exit 0). The merged-source inspection also found the receipt validator compared the
engine to the bare release constant. It now compares the actual compiled
`Conditions::configuration()`, matching the already-correct producer and
preserving diagnostic/stress role suffixes. Runtime RED/GREEN remains deferred during
the profile gate; no Cargo, Go, fixture build or guest was started for this
integration patch.

## Pending acceptance

R10 implementation and focused boundaries are present; its complete actual
installation/plan/recovery corpus is pending. R13/R14 acceptance requires the
full fresh public Go/Rust lane, two independent executions/regenerations with
byte agreement, and strict fresh-root golden checking. Only grammar has that
complete observed evidence so far. Full attempts `full-first.log` and
`full-second.log` honestly exited 1: literal choice capability versions differ
(Go v3 versus authorized new Rust v4), then real missing Temporal build tags.
The corpus now retains both complete literal capability objects; production
Temporal tags were corrected to the exact original four. Complete Temporal
configuration equality, descriptor bytes and payload hashes already passed in
those failed attempts. Neither failed attempt is acceptance.

The full corpus is frozen and waiting for root's explicit guest window after
the profiling quiet pair. No further guest or parity process is running here.
Ordinary release fixture exclusion and final integrated identity/golden refresh
remain conductor integration checks. Task 13 owns final frozen qualification,
its typed module metadata/provenance addition and retirement; this checkpoint
makes no R12 candidate qualification claim. No Go retirement, CI, push or PR.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

Integrated public lane 01 used immutable fixture `63c43b4525d16dae1aa1432ddbff8424516c8595c31b3f67c37eaadcda133500`
from main `82ba157e16`. Session 19299 / shell PID 2804814 terminated exit 1:
110 direct commands completed, then seed preparation refused the nested-module
fixture addressed from its parent module. Raw outputs remain under
`/home/agent/wt/evidence/fn232-install-integrated-01` and adjacent `.log`;
no installation golden was published. The fixture has its own unchanged go.mod.
The harness now uses that module directory with target `.` for seed plan,
Rust recovery, Go reference recovery and build-input refusals. Its support
fixture root remains separate for portable-copy and scoped-plan checks.
Pinned Go `list -json` reproduced parent-target refusal exit 1 and admitted the
nested-directory target exit 0, binding exact Module.GoMod, Dir and main.go.
Raw two-command receipt and source hashes:
`/home/agent/wt/evidence/fn232-install-nested-module-regression`.
Python AST checks (2 files) and git diff --check both exit 0. No production
source or guest bounds changed. Fresh full lanes remain pending integration.

Integrated lane 03 (session 90033) terminated exit 1 after seed preparation and
first authentic portable opener both succeeded. Its later collision probe kept
`--working-dir` pointing to the deliberately deleted source directory, so CLI
validation correctly refused that directory before output collision admission.
The harness now uses a distinct directory containing only the original go.mod
for collision probes; the original compiler and source remain deleted. This
preserves both exact collision messages, unchanged output/receipt assertions,
detached opener, metadata attacks and all missing-input proofs. An actual tiny
probe first confirmed an empty directory without go.mod is also invalid, then
both corrected collision commands returned status 2 with `output already
exists`, without changing original plan/receipt or recreating source/compiler.
Raw evidence: `/home/agent/wt/evidence/fn232-install-collision-cwd-regression`.
Lane 01 and 03 failures remain intact; 02 and 04 were not started. No production
source, compiler build or guest execution was changed by this repair.

Integrated lane 05 (session 11602) terminated exit 1 after portable preparation,
opening and both corrected collision refusals passed. Its next harness assertion
counted the shared `.campaign-targets` evidence pool as a second campaign.
The container actually held exactly that pool and the one receipt-named campaign.
The harness now snapshots all container entry names, asserts exactly the one
receipt-named `campaign-*` directory and an empty journal before collision calls,
then requires the exact unchanged entries, one campaign and empty journal after.
The bounded source sweep found no sibling all-entry campaign-count assertions
in plan-check, recover-check or installation-lane; existing Go checks enumerate
actual journals. No production/evidence-store behavior changed. Full lane 05
raw failure remains under its original evidence root and adjacent log.

Focused actual plan lane 01 (session 8543) terminated exit 1 only at the final
cross-lineage module equality premise. Detached opening, both resealed metadata
attacks, matching/empty scoped retained bases and stripped-symbol refusal passed;
Go public plan refused 2 and actual Go explore succeeded 0. Frozen Go v3 runtime
and current Rust v4 runtime have authorized fn230 site-attribution input changes:
first differing generated file gomad_wasm.go is
`2716bf5007ef48bc788726b4f0dd3e036e7655fac7aade785793bc0f9af6198a`
(Go) versus `dd217006f5c56bc9ea5fe8d53911fc852ef8e9e6fcbfe0edc287c1d95af8ecc8`
(Rust), adding scheduler site/flag arrays and decision wire-v2 attribution.
Generated runtime2.go is
`2610721f6ce37cac278ba8ed4df31e9184f53b92b1ac24ce99b20434cb960614`
versus `d25bd036dee54ceb8762195695715774c2008c91d1dad933123d97a5c8a61c4d`,
adding g.gomadCreationSite/Flags. Exact generated diff and both source hashes are
retained in focused-plan-01/overlay-difference.txt. Go module is
`175e59ce0266c83d22c33ec76fe80251bbb6ddff2e8e74878d350d06e7dba2bb`
(2554819 bytes); Rust is
`a54c96c9e44342a5df0e3e9eeb11a6a191d5d99d854351d1900fb0deed934c91`
(2556163 bytes). Code, data and buildid differ because inputs differ.

Root explicitly authorized replacing this invalid equality premise with separate
lineage authentication: each actual module matches its own authenticated artifact,
provenance and receipt; exact same compiler hash/version still compare; literal
Go v3/Rust v4 profiles and both module/runtime/implementation/overlay hashes are
retained in the golden. Overlay bytes must match their own source inventory.
Selection, arguments, Go refusal/explore and Rust zero-execution behavior checks
remain. No hash is substituted or normalized; no cross-lineage module byte parity
is claimed. Independent integrated batch review must cover this changed premise.

Focused plan 02 (session 80788) completed exit 0 with full raw authentic plan,
portable receipt, metadata attacks, scoped bases, stripped-symbol refusal and
separate Go/Rust lineage checks. Full lane 07 (session 19493) then terminated
exit 1 at the first Rust recovery seed preparation: the cooperative installation
requires --choices, which seed recovery omitted (round already included it).
The same missing seed trace flag was present in Go recovery. Both harnesses now
derive seed trace flags from their actual installed profile, preserving stock
and round flags. Full 07 is retained; 08 was not started.

Independent source review also found cooperative-only lineage assertions would
regress stock plan+Go-reference calls. They now keep common own-module, compiler,
version and argv checks, while stock explicitly requires absent choice profiles
and absent overlays; cooperative still requires literal Go v3/Rust v4 and full
own-overlay authentication. Stock lineage fields retain null values; no identity
is fabricated. AST3 and diff-check exit 0; actual focused recovery and stock plan
checks remain pending the conductor's integrated resource window.

Focused real recovery 01 (session 41492) completed exit 0: actual Rust and Go
seed/round journals repair torn tails, repeat as no-op, and retain authenticated
artifacts; Rust real lock and malformed refusals remain unchanged. Focused stock
plan/reference 01 (session 14352) completed exit 0 with actual stock preparation,
zero guest receipt and Go refusal/explore reference. Full lane 09 ran
21:40:00Z–21:40:38Z (session 82894), exit 1 after complete plan/recovery families,
at public Go environment origin reporting. Full 10 was not started.

Source cause: `cmd/gomad/internal/cli/launcher.go` first resolves leading flag,
environment and sibling into local `path`, then its line-122 closure calls
installation.Resolve with ExplicitManifest:path for every origin. The backend
`internal/backends/wasm/installation/installation.go` otherwise distinguishes
all three origins, but public Go consequently reports literal `explicit` for
environment/sibling too. Rust truthfully retains the actual origin. Root
explicitly authorized this diagnostic difference rather than copying the Go
reporting defect. The harness now verifies the exact selected path and literal
manifest digest for BOTH lineages and retains requested/reported origins and
literal digests in the golden, with declared run-path substitution only.
Invalid explicit/environment selection refusal and no fallback checks remain
unchanged. No production behavior or identity normalization was introduced.

Focused resolution 01 completed exit 0: 10 actual Go/Rust public calls verified
all literal selected path/digest/origin values and both invalid-selection
no-fallback boundaries. Full 11 ran 21:46:31Z–21:47:19Z (session 41140), exit 1,
through plan/recovery, resolution, helper admission and changed/missing runtime
checks. Its next deliberate private compiler mutation failed with PermissionError:
copy2 preserved the frozen compiler's read-only 0555 permissions. The harness
now grants owner-write only to its owned private compiler copy immediately before
mutating its bytes. Original compiler bytes/mode remain untouched and final raw
input digest guard remains required. Other copied inputs are only read/deleted;
all earlier runtime/artifact mutations actually passed. Full 11 remains intact,
12 was not started; no production source or build changed.

Complete prior corpus acceptance for immutable `63c43b45...`:
full 13 regeneration, session 24587, 21:49:14Z–21:50:33Z, exit 0;
strict full 14, session 35153, 21:50:55Z–21:52:16Z, exit 0.
Each ran 312 direct public calls plus authenticated child lanes, two independent
executions and unchanged raw-input guards; regeneration agreed byte-for-byte on
its second write. Literal golden SHA was
`bded8fc97b41954567a0e0e9c188b2ca4d9b664f8c55323644832dfc960c8a38`.
These are genuine passes for that exact prior corpus, not final R12 qualification.

Final coverage audit found two explicit parent-required helper proof negatives
missing from that corpus: a mutated copy of the current executable and an actual
differently compiled Rust role. Both actual focused calls now pass (overall exit
0, each public install refusal 2 with exact helper-identity message and no
manifest). Raw evidence:
`/home/agent/wt/evidence/fn232-install-focused-helper-identities-01`.

The actual historical diagnostic host supplied by root is independently copied
to `/home/agent/wt/evidence/fn232-install-helper-inputs/diagnostic-gomad-59de1818de322a5194922589b18387014111084d0881f9b195a10f4d7261e844`,
22652720 bytes. Its original build log, role-check JSON, source-binding artifact
and their SHA receipts are retained alongside it. The source artifact identifies
fn188 checkpoint `79dbece295` plus temporary typed-role overlay `cac5b294eb`;
this is a historical negative input, not current qualification evidence.

The diagnostic harness adds required full-lane `--installation-role-binary`
(forwarded to child `--role-binary`), with before/after input hash admission and
literal supplied/current hashes in new golden cases. No operator CLI/API changes.
It has no permanent role SHA allowlist: a future frozen cycle supplies its own
actual different-role host with original build/role receipts and deliberately
refreshes the golden. Each attempt owns executable copies, mutates one current
byte, requires exact identity refusals/no manifest, and preserves hostile-script
no-execution proof. The retirement/all-verb caller owner was notified. AST2,
JSON and diff checks exit 0; completed-corpus regeneration/strict check pending.

## Completed public corpus handover

The completed corpus passed full regeneration 15 (session 59182,
2026-10-22T21:58:45Z–22:00:03Z, exit **0**) and independent strict run 16
(session 33744, 22:00:16Z–22:01:35Z, exit **0**). Each ran **316 direct public
commands plus authenticated child lanes** over two independent attempts.
Regeneration's second write agreed byte-for-byte. Both runs required unchanged
before/after input digests and authenticated raw artifacts/journals independently
of golden comparison. Together these are four independent actual executions of
the complete owned corpus, not scraped or synthetic results.

Raw roots and adjacent `.log` files:
`/home/agent/wt/evidence/fn232-install-integrated-15` and `-16`.
The exact main-generated golden is committed without edits at
`tools/gomad/testdata/parity/goldens-installation/installation.json`, **1637059
bytes**, SHA-256
`654693efd83b2e9e49d0b420d623a5f364ca24c1ba58527be5b59f3ffaf4ff1b`.
It contains 158 direct case observations per attempt, 27 configuration cases,
three real Temporal descriptor cases, both explicit helper negatives and the
retained plan/recovery/lineage observations. No identity, cost or behavior was
normalized. Declared run/preparation path substitutions remain independently
checked against their actual selected paths.

Actual binary: immutable
`/home/agent/wt/evidence/fn232-integrated-install-fixture/gomad-63c43b4525d16dae1aa1432ddbff8424516c8595c31b3f67c37eaadcda133500`,
built from integrated main `82ba157e16`; its source inventory receipt is adjacent
(SHA `21175ebbdbfa3f46baaac3f57616d02b0ff740fe4018049d84883572a25bf09c`).
Final harness source integration was `76f565c9eb`. Historical role input and
its source/build/role receipts are retained under the open fn232 evidence owner
as described above. The later mandatory/helper source integrations require
final-candidate regeneration in task 13; these observations bind this actual
immutable installation candidate rather than claiming that later qualification.

The full command is recorded literally in the evidence JSON. It invokes
`tools/gomad/scripts/behavioral-parity --corpus tools/gomad/testdata/parity --verb
install` with the immutable Rust binary, explicit role binary, frozen Go
launcher/helper, pinned stock Go1.27.1 compiler, frozen Go source
`/home/agent/wt/fn232-go-source/temporal-1d7b37f00a`, stable preparation root and
fresh output root. Run 15 sets `GOMAD_GOLDEN_REGENERATE=1`; run 16 does not.
No process was restarted to observe an outcome. All owned processes are terminal.

Owned requirement evidence:

- **R10:** real install/discover/default doctor and selected doctor/analyze
  refusals; compiler/runtime/config inventory admission; missing/mismatched
  tools, Go-lineage and collision refusals; explicit/environment/sibling exact
  selection and invalid-choice no fallback; discovery after compiler/runtime
  removal; linked default and exact-copy helper positive; mutated, legacy Go,
  hostile and genuinely different Rust role negatives. No external helper is
  selected for execution. Literal Go origin-reporting difference is documented
  and retained rather than copied into Rust.
- **R13:** fresh actual Go/Rust public execution, each lineage's own authenticated
  artifacts and unchanged compiler/config inputs. Go public plan refusal remains
  2; its genuine explore reference succeeds 0. Authorized Go v3/Rust v4 generated
  runtime differences retain both literal identities, never module-byte parity.
- **R14:** complete literal golden, independent raw authentication, repeated
  executions, byte-identical second regeneration and strict fresh verification.
  Root separately built ordinary release from `e8754d1cf8`, exit 0, immutable
  `81657d713cbb33ff3ca82bf8e92c224030097bb11d341fe59ab613e4556cb5a6` under
  `fn232-integrated-ordinary-release`: actual fixture-preparation-open refuses
  unsupported 2; ordinary qualify reaches missing-spec input 2 rather than a
  nonordinary-role guard. Raw outputs/build receipt are adjacent and the build
  log is `fn232-integration-rust/ordinary-release-build.log`. This proves release
  exclusion, not qualification; its earlier help-verifier premise failure is
  retained by root.
- **fn-222 preparation receiver:** actual seed preparation, authenticated portable
  retained module/provenance and receipt, zero guest executions; reopen succeeds
  after original compiler/source removal. Resealed imports/memory metadata lies
  refuse while honest legacy absent/absent admission remains intact.
- **fn-230.1 receiver:** public unscoped plan has zero executions; matching and
  empty scopes each execute and authenticate exactly one retained base, including
  default-success retention. Original scope validators and stripped-symbol
  refusal remain exercised.
- **Recovery:** actual Rust and Go seed/round journals repair torn tails and repeat
  as no-op; Rust real writer-lock and malformed-input refusals leave state
  unchanged. Rust recovery starts no guests.

Remaining conductor actions are integrated review/acceptance and Flow lifecycle.
The task stays **in_progress** here. Task 13 owns the final frozen candidate,
full unchanged-budget qualification, new-source identity/golden refresh, and Go
retirement. No CI execution, push, PR, history rewrite or retirement occurred.
