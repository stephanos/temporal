---
satisfies: [R9]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.4 Pure Scala gap migration transformation and copied-mini source proof

## Description
Build the deterministic Scala migration transformation and copied-mini source proof (groundwork for R9). The executable adapter and full IR-backed proof remain task 7.

**Size:** M
**Files:** new migration core/tests/fixtures; mini-only source proof; docs.
**Touches:** [model/migrate/**, model/temporal/features/activity_mini/**, model/README.md, model/SEMANTICS.md]

### Approach
- Admission prerequisite: conductor verifies fn-166.3 is done with its closure committed in the current branch. The CLI does not support cross-spec task edges; this prerequisite is not replaced with a whole-spec dependency that would create a sweep cycle.
- Consume existing IR/acceptance shapes plus resolved placement/provenance from tasks 2–3. Keep input parsing separate from the pure transformation; do not create a public sidecar format.
- Group by concrete declaration and exact reason, preserving reason text and existing source formatting outside insertion spans. Resolve old owner names explicitly; do not guess nearest declarations.
- Remove redundancy only when the supplied provenance proves the corresponding inherited finding equivalent. Return every unresolved acceptance or unsafe reduction as a diagnostic, with no partial source publication.
- Prove the annotated mini hole and preserve the existing behavior-bearing waiver with its exact reason using copied inputs and source fixtures, then compile/re-lift the result. Reconcile capability-waiver acceptance entries with `except`/`overriding`, report mismatched reasons, and never insert a second waiver annotation. Keep real sources/artifacts and all existing acceptance payloads untouched. Coordinate the future adapter location with fn-166's migration-as-code convention without depending on its whole held closure here.

### Investigation targets
Required:
- `.flow/tasks/fn-166-mini-activity-a-small-complete-model-to.3.md` — committed mini protocol producer.
- `.flow/tasks/fn-166-mini-activity-a-small-complete-model-to.5.md` — migration convention owner.
- `model/irgen/Context.scala` — resolved declarations and positions.
- `model/check/Gate.scala:1402` — current acceptance representation.
- `tools/umpire/lint/accept.go:88` — exact grouping identity.

### Verification
Add proposed `umpire.migrate.GapsSuite` with reason preservation, multiple reasons, inherited duplicates, unmappable owners/subjects, conflicting insertions and idempotent second application. Pin original inputs and non-insertion source bytes. Compile/lift copied mini output and pin its step tables. Do not claim full R9 from fixtures alone; task 7 must run the actual IR/acceptance-backed adapter.
## Acceptance
- [ ] Before admission, conductor verifies fn-166.3 done and its closure committed in the current branch.
- [ ] Transformation fixtures preserve reason text, source bytes outside insertions and original inputs; repeat application is idempotent.
- [ ] Unplaceable or unsafe reduction inputs fail without partial publication.
- [ ] Copied mini one-hole/one-waiver output compiles/lifts with equivalent tables; actual adapter/IR-backed R9 remains task 7.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
