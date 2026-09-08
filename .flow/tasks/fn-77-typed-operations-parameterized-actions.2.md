---
satisfies: [R2, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.2 Admit exact bounded structural operation values

## Description
Admit exact bounded structural operation values for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Operation/**; model/Umpire/Value.lean; model/Umpire/Value/**; tools/umpire/cmd/umpire-gen-lean-api/**; model/Temporal/API/**
**Touches:** [model/Umpire/Operation/**, model/Umpire/Value.lean, model/Umpire/Value/**, tools/umpire/cmd/umpire-gen-lean-api/**, model/Temporal/API/**]

### Approach
- Implement a schema-driven checked concrete value carrier alongside descriptive summaries; cover nested messages, exact bytes, typed integer/enum values, presence, oneof discriminants, ordered repeated values and canonical decoded maps.
- Use schema identity from task1 and bounded recursive traversal with depth/work/collection/byte limits. Keep schema recursion metadata complete while rejecting exhausted concrete access; never fabricate empty subtrees.
- Normalize map inputs according to decoded protobuf semantics (last value for duplicate raw keys, canonical typed-key order after decoding). Pin exact signed/unsigned ranges and open-enum unknown numbers; reject unsupported closed-enum/special forms explicitly.
- Declare floating-point field evaluation/operators unsupported initially with responsible source diagnostics; do not substitute approximate/text comparisons. Preserve complete structural discovery and every required qualifying clause.
- Prove checked canonical encode/decode preserves exact values and presence; retain old descriptive representations and record supported forms in module docs.

### Investigation targets
**Required:**
- model/Temporal/API/Proto.lean:15 — Bytes/MessageRef summaries are not concrete values.
- tools/umpire/cmd/umpire-gen-lean-api/model.go:255 — descriptor presence/map metadata.
- tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go:749 — recursive reference substitution.
- model/Testpilot/Authoring.lean:35 — existing portable concrete bytes/numeric constructors.
- common/testing/testpilot/internal/ir/runtime_value.go — actual codec semantics to match.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir`
`cd model && mise exec -- lake build Testpilot.Tests Umpire.TargetTests`

`cd model && mise exec -- lake build Umpire.Value.Tests`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Value.Tests into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Independent fixtures preserve required exact forms, including same-length different bytes, optional absent/default, unknown enums, integer boundaries, repeated order and keyed map normalization.
- [ ] Recursive depth, payload/collection bounds and unsupported forms fail explicitly without truncation or fabricated values.
- [ ] Canonical round-trip proof covers admitted exact values and schema identity; pre-existing descriptive consumers and bytes remain compatible.
- [ ] New owner tests are wired into normal Lean test roots; focused generator/codec/model checks pass.

## Done summary
# fn77.2 implementation handover

Task2 and review round1’s bounded string-default fix are implemented and frozen for conductor re-review. Native status remains `in_progress`; `commits=[]`. No staging, commits, pushes, worktrees, Flow mutations, reviews, delegation, or next-task launch were performed by this worker.

impl-review skipped(policy: host-deferred; conductor owns gate)

### Review round1 fix

Finding `finding-726c2d75ddb12ed4fbaf23fa14e4cab4` from `/tmp/fn77-task2-impl-review.json` is fixed, pending conductor re-review. `concreteDefault` now emits strings through `leanString`, following the established dynamic-config generator pattern: Lean-supported quote/backslash/whitespace escapes, Unicode escapes for other ASCII controls/DEL, and exact Unicode characters. No other generator paths or semantic owners were changed.

The existing proto2 `LegacyOptions.name` fixture now declares a default containing bell, backspace, formfeed, vertical tab, quote, backslash, newline, carriage return, tab, NUL, DEL, é and 😀. Generated Lean checks independently require character codes `[7, 8, 12, 11, 34, 92, 10, 13, 9, 0, 127, 233, 128512]`. Before the production fix, owner regeneration succeeded but actual fixture compilation exited 1 with `Fixture/API.lean:110:88: error: invalid escape sequence`. After the fix, the same generation/compilation and exact-character check exit 0. The red/green logs are `/tmp/fn77-task2-r1-red-lean.log` and `/tmp/fn77-task2-r1-green-lean.log`.

Five paths changed in this review fix: value_schema.go, the fixture proto, its descriptor input.pb, generated Fixture/API.lean, and checks.lean. The final task-only patch now contains 18 paths. The entire original descriptor closure is equal after removing only the intended name default, verified with an independent protobuf decoder and exact codepoint list. Fixture Proto/Types and the complete Temporal facade remain byte-identical to their reviewed versions. The initial handover/patch and six prospective pre-fix copies are preserved under `/tmp/fn77-task2-r1-baseline`; the ORIGINAL task2 baseline is unchanged.

Fresh serial round1 gates all completed: tagged generator/internal-IR Go tests; owner fixture and Temporal regeneration; make umpire-check-lean-api (including actual fixture compilation); all 299 aggregate Lean jobs; nonfixing make lint-code; descriptor-closure oracle; complete affected fixture trust capture/parser/mapping; and preservation checks. Lint exits 2 with exactly the inherited 1,284 occurrences / 825 distinct issue lines, including identical paths/lines/columns/messages; no findings waived and separate Make go-vet not reached. Round1 fixture trust is 390 before/390 after, every qualified declaration and raw axiom array identical. Core/Temporal source hashes are unchanged, so their complete prior raw trust capture remains valid. Exact commands, exits, elapsed times and log hashes are in the refreshed evidence.

### Delivered behavior

The generated schema graph now includes exact value shapes: all ten protobuf integer kinds, named open/closed int32 enums, fields/cardinality, explicit/implicit presence, exact defaults, oneof groups, typed map keys, message references and unsupported special forms. Complete recursive discovery remains generator-owned. All 1,864 Temporal schema nodes preserve their prior name, syntax, descriptor hex, file context and references exactly; removing the shape extension yields the original Temporal facade byte-for-byte. The nine basic-fixture nodes have only the intentional string-default/schema-identity change described above. Descriptive Proto/Types files, including Bytes and MessageRef, remain unchanged. The fixture proto/input.pb extension is explicitly accounted for by the complete descriptor-closure oracle.

`Umpire.Value.Checked` retains the generated owner and typed witness indices. Its exact structural data supports nested bytes/messages, kind/range-checked integers, unknown OPEN enum int32 numbers, explicit presence, ordered repeated values, and typed canonical maps. Checked literals reject duplicate map keys; raw decoded maps normalize last value before producing unique sorted keys. Unknown fields, duplicate fields, oneof conflicts, closed-enum unknowns, type/range errors and exhausted limits fail atomically with an owning path/reason. Depth, work, payload bytes, encoded bytes and collections are bounded; encoded overhead can exhaust byte/work limits earlier than payload alone. No payload digest, size substitute or fabricated subtree is used.

The version-1 structural binary codec is an internal exact carrier format, not protobuf/Testpilot wire. Schema/version/side metadata is outside concrete payload bytes. `Encoding.decodeNat_encodeNat` and `Encoding.decode_encode` prove the actual byte encoder/decoder suffix laws universally. `Value.decode_encode` proves canonical correspondence for every admitted bounded value under retained schema identity. The carrier stores semantic admission/resource invariants, not an encoder/decoder equality certificate. Canonical decoding rejects noncanonical encodings; raw decoding is the separate last-value normalization boundary.

Float32/64 schema/default metadata remains discoverable, while concrete evaluation returns explicit unsupported diagnostics, including NaN/infinity/signed-zero controls. Existing portable float support is unchanged. This restriction does not waive any eventual qualifying clause.

### Verification

All exact commands, environments, elapsed times, actual exits and log hashes are in `/tmp/fn77-task2-evidence.json` and the preserved command journal. Pinned mise tooling, TMPDIR=/private/tmp, xcrun CC/SDKROOT and LEAN_NUM_THREADS=1 were used. Recovery Go/Lean gates ran serially.

- Existing pre-edit Quick baselines passed: tagged generator/internal-IR Go tests and Testpilot.Tests/Umpire.TargetTests.
- Final corrected tagged Go tests passed for the generator and `common/testing/testpilot/internal/ir`.
- Final `make umpire-check-lean-api` passed, including real/basic/empty generated checks, after the round1 generator fix.
- Final explicit `lake build Testpilot.Tests Umpire.TargetTests Umpire.Value.Tests UmpireTests` passed all 299 jobs. Value.Tests is wired into the normal aggregate.
- Semantic red/green controls failed on actual admission/default/range behavior before implementation. Five isolated compiled mutants produced semantic test failures for range checks, OPEN enums, map ordering, literal duplicates and depth; `/tmp/fn77-task2-mutants.json` records exits and logs. A separate canonical-limit regression failed on the lost owning diagnostic before its fix.
- Independent tests cover same-length distinct bytes, absence/defaults, every integer-kind boundary and overflow, enums, oneof, repeated order, four map-key kinds, duplicate policy, recursive/deep/wide N/N+1 limits, independently exhausted work/bytes, Unicode, unsupported floats, malformed/schema-mismatched codecs, and universal round-trip. The Go oracle uses independently specified actual protobuf bytes, not generator-rendered expectations. Fixture tests do not rewrite their inputs; regeneration used owner routes only.
- Nonfixing `make lint-code GOLANGCI_LINT_FIX=false` exits 2 with the **exact inherited multiset**, including file/line/column/message: 1,284 occurrences / 825 distinct. Zero added or removed diagnostics. Both introduced findings were fixed. Separate Make go-vet was not reached. The stale ENOSPC 1284→0 comparison is preserved separately and is not a waiver or passing evidence. The final round1 lint/Go/generated/aggregate gates cover the current source; no source edits followed them.

### Trust and preservation

The authoritative final captures are `/tmp/fn77-task2-after-trust-recovery-final.log` (unchanged core source) and `/tmp/fn77-task2-r1-after-fixture-trust.log`, with their parsed JSON and `.mapping.json` files. Each exited 0, has complete selection/end counts, full raw transitive axiom arrays and declaration bodies, and exhaustive strict parsing. Raw captures were hash-verified and transparently HFS-compressed before parsed copies; final metadata/content verification receipts accompany them.

Core: 3,857 before / 4,735 after, all original declarations retained, 878 new. Fixture: 387 before / 390 after, all originals retained, three new. **No retained declaration has axiom growth.** An initial completed recovery capture exposed sparse-match propext growth; disabling sparse-match elaboration locally in ValueSchema removed it. That intermediate capture remains available but is superseded by the final one.

`/tmp/fn77-task2-trust-comparison.json` maps every retained/new declaration and generated auxiliary. New proof assumptions remain within the existing Operation dependent-admission kernel-logic boundary (`propext`, `Quot.sound`, `Classical.choice`); each maps to a captured existing analogue. The two binary suffix proofs use propext/Quot.sound; the checked facade proof additionally uses Classical.choice. There are no custom/compiler-trust axioms, sorry, dependency or toolchain changes. No unrelated proof owners were modified.

`/tmp/fn77-task2-baseline` retains the pre-edit HEAD/index/status, 7,674-source hash manifest, 43 original byte copies and original absences, and full pre-edit trust evidence. All copied originals verify. All original comments in touched files remain. The 715-file task1 immutable integrity manifest verifies unchanged. `/tmp/fn77-task2-preservation.json` records updated generated compatibility, unrelated source and index comparison; its preserved prior version records the original-to-initial-handover comparison.

An external commit advanced HEAD from 375abfe180dba72da6dd357e6abe33fa75a292a7 to d9f5a809a5e26333d2261d4f1aa4b5b561ca6aae and changed 237 index entries, including some work in progress. The worker made no commit. Original and final staged diffs are empty. External fn67 metadata/task additions, roadmap and AGENTS edits are preserved and excluded from the task patch. The patch is verified against original captured bytes, independently of the external HEAD.

### Recovery caveats and later owners

The interrupted original `/tmp/fn77-task2-after-trust.log` is truncated/failed, preserved with its original hash, and **not proof evidence**. Its original child exit is UNKNOWN per conductor after ENOSPC/failed receipt write; the earlier journal's exit1 is not used as reliable terminal evidence. Original worker/baseline logs remain. The first recovery Go command used an incorrect nonexistent package path and failed setup; the corrected established path passed. The first recovery compression preserved content but changed mtime, explicitly recorded; authoritative final capture compression preserves mode/mtime/content. Earlier development had overlapping Go/Lean and two accidental Lean overlaps; final recovery and round1 Go/Lean gates were serial. No unrelated editor process was killed.

Task7 must align Go admission/catalog behavior to preserve unknown OPEN enum int32 values; current runtime rejects them irrespective of openness. No runtime Go files changed here and no full cross-language completion is claimed. Task3 owns typed field references/denotation, task4 instances, tasks5/6 Properties/captures, task7 portable evaluator, task8 whole lowering. Task2 supplies the complete carrier/proof prerequisites. Combined fn77 full compatibility/live/load gates remain task11 by the explicit worker scope. No requested task2 fidelity is deferred.

Frozen artifacts: `/tmp/fn77-task2-changed-paths.txt` (18 paths), `/tmp/fn77-task2.patch` (original-to-final including binary input.pb, independently applied to isolated original source copies and all resulting hashes verified), `/tmp/fn77-task2-frozen-hashes.json`, `/tmp/fn77-task2-evidence.json`. Conductor owns review and completion. No further source edits or jobs follow this handover.

stage: impl-review - ran; SHIP after one bounded fix round (model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir, mise exec -- lake build Testpilot.Tests Umpire.TargetTests Umpire.Value.Tests UmpireTests, mise exec -- lake env sh /tmp/fn77-task2-fixture-trust.sh, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" make umpire-gen-lean-api-fixture, mise exec -- lake env sh ../tools/umpire/cmd/umpire-gen-lean-api/check-fixtures.sh, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" make umpire-gen-lean-api, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" make umpire-check-lean-api, mise exec -- lake build Testpilot.Tests Umpire.TargetTests Umpire.Value.Tests UmpireTests, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- go run /tmp/fn77-task2-r1-closure.go, mise exec -- lake env sh /tmp/fn77-task2-fixture-trust.sh, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" make lint-code GOLANGCI_LINT_FIX=false
- PRs: