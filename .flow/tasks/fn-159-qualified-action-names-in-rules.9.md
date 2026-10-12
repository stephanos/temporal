---
satisfies: [R16]
---
# fn-159-qualified-action-names-in-rules.9 One spelling of a bounded integer: UpTo[N] on the field, ranges read per field

## Description

Complete the current framework/lifter/fixture migration to field- and message-owned UpTo bounds.
Selected mini/shared author sources already use UpTo for bounded state fields; the actual remaining
paths are the lifter's state-keyed intRanges and Int-channel Finite.upTo fallback, plus old positive
language fixtures. The real Nexus Protocol migration/lift remains at the owner's sweep and is not
part of this source proof. Finite.upTo for an explicitly named opaque integer type stays supported.

**Size:** M
**Files:** per-field/message range lifting, range context, existing positive/negative language
fixtures and focused controls, Domain docs, task-named migration/proof and author guide.
**Touches:** [model/framework/Domain.scala, model/framework/Domain.test.scala, model/framework/BoundedIntegers.test.scala, model/irgen/Types.scala, model/irgen/Declarations.scala, model/irgen/Context.scala, model/irgen/test/BoundedIntegers.test.scala, model/irgen/test/FixturePositions.test.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/boundedIntegers/**, model/irgen/testdata/lifts/Rejects.scala, model/irgen/testdata/lifts/Presence.scala, model/irgen/testdata/lifts/Channels.scala, model/irgen/testdata/lifts/expected/rejects.txt, model/temporal/features/activity_mini/**/*.scala, model/temporal/foundations/**/*.scala, model/temporal/actors/**/*.scala, model/temporal/capabilities/**/*.scala, model/migrate/**, model/README.md, model/SEMANTICS.md]

### Approach

- Re-anchor after the prior current source checkpoints have been committed and integrated into
  local umpire. Capture fresh selected pre-edit jar/lift and affected fixture outputs before edits.
  Do not replay the real-feature change or block on held canonical/Go prerequisites.
- Remove the state-keyed intRanges map and readFinites extraction. A plain Int field refuses at
  its actual declaration, naming the field/type and UpTo[N], even if a local Finite[Int] could
  supply a catalog. Keep scalar Int expressions/helpers where they are not bounded Model fields.
  Keep named opaque ranges and channel Inbox association; do not ban generic Finite[Int] from
  the framework just to enforce Temporal authoring policy.
- Remove messageRef's Finite.upTo Int-message branch. Refuse a plain Int channel at its declaration
  with the UpTo[N] remedy. UpTo and named opaque messages keep the range from their own type.
- Independently test two differently bounded UpTo fields in one record, zero and literal/singleton
  bounds, rejected unbounded state fields/channels and retained named opaque ranges. Existing
  negative-bound and nonconstant-bound behavior stays truthful; no initialization bypass.
- Check optional bounded fields as well: optionType currently names an IntRange argument from
  only arg.widen.typeSymbol.fullName, so differently bounded Option[UpTo[N]] wrappers may share
  one emitted enum identity. The current mini has one correct 0..2 wrapper at
  scala.Option[framework.Domain$package$.UpTo]; no current mini range failure is claimed.
  Prove the two-bound case independently before changing it. If it fails, preserve each field's
  actual range and account for any genuine synthetic-type identity delta explicitly, rather than
  silently accepting the first wrapper's range or claiming unchanged bytes after a rename.
- Root inspection found messageType currently retains only a typeSymbol fullName when associating
  Inbox fields with channels. UpTo[1] and UpTo[3] therefore share a key. Prove differently bounded
  message channels/Inboxes independently and preserve their actual type arguments in the lookup
  if needed; do not silently map both to the last channel. Do not build an unrelated general channel
  ownership redesign or change existing same-message-channel ambiguity policy without evidence.
- Migrate the old positive CounterState, Presence.polls and bounded tally channel fixtures, including
  necessary constructors/writes/Inbox types. Retain Presence's named opaque Retries and Rejects'
  empty Unreached opaque-type negative control. Negative retirement controls may generate old
  syntax in isolated scratch tests; no positive authored Model/fixture keeps the retired state Int
  convention. Preserve helper Int signatures where widening is intentional and behavior unchanged.
- Prove affected existing fixture semantic IR remains unchanged, and preserve existing expected
  JSON bytes. Keep actual source positions honest; do not rewrite goldens, invent padding or drop
  metadata to conceal changes. If an unavoidable genuine span delta conflicts with an old byte
  expectation, record and independently check that exact delta rather than claim byte identity.
  Updated rejection text must reflect the new real error contract and actual source position.
- Ship actual compiled Fn159Task9... Scalafix source under the existing convention. Supported
  bounded-field/channel declarations, necessary writes and retired local givens migrate together,
  preserving numeric ranges and semantics. Unknown/symbolically unprovable bounds, unrelated named
  opaque declarations, ambiguous scopes and unsafe uses refuse or remain untouched as appropriate.
  Prove genuine transformation/refusal/idempotence and a compiled copied-mini exercise; no new
  generic rewrite engine, dependencies or production loader. The mini's authored UpTo fields are
  already current; a scratch legacy probe must be explicitly labelled, not a fabricated permanent
  feature addition or fake old-API compilation.
- Domain.upTo and derived-catalog comments, README Types and SEMANTICS field/channel explanations
  describe one authoring spelling. Keep accurate source-cited guide excerpts and unchanged real
  features/payloads, pins and old maps.

### Verification and boundaries

Fresh selected mini/shared compile, targeted native/domain and isolated per-field/message/opaque
lifter controls, affected fixture-only packages, migration/citation tests and complete scratch
mini pre/post IR/waiver comparison are this step's gates. Do not invoke broad Fixtures.beforeAll.
Freeze source/jar inputs before final proofs and use immutable matching jar-based classpaths.
The current mini's own bounded authoring is already UpTo: its full raw payload and source positions
must stay byte-identical unless a concrete independently proven source defect requires a declared
change. No blanket metadata normalization or expected-payload rewrite.

Run direct FEATURES=activity_mini Scala `--check-equivalence ... --steps-only` against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified task map. The immutable pre-change task8-jar complete selected capture
contains sixteen tables/11,325 rows including shared Query roots. Both it and the original ten-table
baseline pass their integrity scripts; neither nor the historical batch-1d map may be changed or
recaptured. Run changed-file format/diff checks and retain all actual failed/RED observations.

Real Nexus/source consumers, managed IR/Cases and canonical Make/Go/linked-proto/schema closure
remain held. No Go/proto/schema/Make/CI/runtime/Case/live/production-loader/generated changes.
Lint/manual acceptance is NOT RUN. Broad validation and independent review occur at the ONE
Scala batch-end gate; no per-task SHIP, full-set or canonical-green credit.

## Acceptance

- [ ] State intRanges extraction and Int-channel Finite.upTo fallback are gone; actual plain Int
  field/channel declarations refuse with located UpTo remedies, while scalar helpers and named
  opaque ranges retain their intended behavior.
- [ ] Differently bounded fields and messages independently lift their own ranges; channel/Inbox
  identity preserves relevant type arguments rather than conflating UpTo bounds.
- [ ] Positive affected fixtures use UpTo, named opaque controls stay valid and existing semantic
  payload/expected JSON bytes are preserved with honest independently checked positions/errors.
- [ ] Task-named compiled source rule has genuine supported transformation, idempotence and
  ambiguity/bound/unsafe-use refusals plus labelled copied-mini/probe compilation and lift proof.
- [ ] Complete current mini IR/waiver data, declaration/Query inventories and sixteen native tables
  preserve meaning/identity; both frozen baseline integrity checks pass and protected payloads,
  pins, source and maps remain untouched.
- [ ] Domain/README/SEMANTICS and exact guide citations describe one field/message spelling;
  failed attempts and held consumer/lint/manual/batch-end broad-review gates are recorded honestly.

## Done summary
Bounded state fields and channels now derive ranges from their own UpTo[N] types; plain Int declarations refuse at their source location, while named opaque ranges and scalar helpers remain legitimate. Proven Inbox and Option parameter-identity collisions are fixed; the task-named compiled Scalafix rule migrates supported legacy declarations/writes atomically and refuses unsafe or ambiguous cases.

Evidence: `.flow/tmp/fn159/task9/proof-notes.md`, `mini-delta.json`, `probe-delta.json`, `native-final.log` (2 tests), `lifter-frozen.log` (8), `unsafe-increment-green.log` (4), `guide-arithmetic-final.log` (30 exact citations), `proof-arithmetic-final.log` (compiled scratch legacy probe and complete copied-mini equality), `equivalence-corrected.log` (16 tables / 11,325 rows), and both integrity logs. Baseline: green selected package/lift and affected fixtures captured before edits. Corrected RED and setup/inconclusive logs are retained in the notes.

Current raw mini SHA is 3579eef0fce64da34ee8b93afdd03256c5307120ecccfb037f0eefdeae77705b, not the pre-edit b153 hash: exactly five typed optional identity/reference leaves change; ranges, positions, other raw data and waivers remain unchanged. Existing expected fixture JSON is preserved with exact tested coordinate reconciliation; Touches includes the two required reconciliation tests. Frozen source/jar manifests and original baseline/payload/pin/map/external-dirt preservation checks are retained.

Tier: session (jev-unavailable(no_key)); AGENTS implementer gpt-6.1-sol medium retained.
stage: impl-review - skipped(config: REVIEW_MODE=none)

Broad validation and independent review remain held for the Scala batch end; no per-task SHIP, canonical/full-set/consumer credit, lint/manual, production/runtime/Go/proto/schema/regeneration or tracker/plan-sync action is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b7b76c849c812461dbd2f004b9a52ee9b08a8409, 68cc753a8d771260f7f03bd42d9cadb4e8bd0d72
- Tests: Baseline green: selected package/lift and affected-fixture captures in .flow/tmp/fn159/task9/before/ and fixture-before/, native-final.log: framework.BoundedIntegersSuite, 2 passing focused controls, lifter-frozen.log: umpire.irgen.BoundedIntegersTest, 8 passing source-preparing focused controls, timeout 600 mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.BoundedIntegersTest (unsafe-increment-green.log, 4 passing tests), timeout 600 mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.GuideSourcesTest (guide-arithmetic-final.log, all 30 exact citations), timeout 600 /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/migrate/prove-mini.sh umpire.migrate.RewriteBoundedIntegersMini (proof-arithmetic-final.log, .flow/tmp/mini-migration.27qAbg), FEATURES=activity_mini timeout 600 mise exec -- scala-cli run --server=false model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn159/task9/mini.map --steps-only (16 complete tables equivalent), complete-integrity.log: exact 16 tables / 11325 rows; subset-integrity.log: historical 10 tables / 9878 rows preserved, integrity only, mini-delta.json: closed five-leaf typed identity/reference delta; test-bounded-mini-delta.mjs: 8 corruption/no-op controls refused, probe-delta.json: actual legacy source compiled and frozen pre-edit lifter compared; only exact scratch-source coordinates differ, sha256sum -c .flow/tmp/fn159/task9/frozen-sources.sha; sha256sum -c .flow/tmp/fn159/task9/goldens.sha; external patch cmp and external-untracked.sha: PASS, Scoped limitations and retained RED/setup/inconclusive observations: .flow/tmp/fn159/task9/proof-notes.md; broad/canonical/lint/manual and independent review held
- PRs: