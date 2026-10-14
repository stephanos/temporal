I’ll read the task spec first, then trace the changed code and focused tests against its acceptance criteria.
The fixture covers classed synchronized actions and dotted member actions. I’m checking bare classed `when:` references and key collisions next; both affect the new composition paths.
I found that bare classed `when:` references still require an exact catalog key, so `when: reply` cannot cover `reply-ok` and `reply-error` as specified. The lint command is blocked by the read-only sandbox; I’m using focused in-memory Lean checks to verify the edge cases.
The focused Lean check reproduced both bare-reference failures. I also confirmed that a sync name can collide with a generated member-action key. I’m checking domain compatibility and Scenario start resolution to pin down their observable effects.
1. **Bare classed `when:` references are rejected**
   - **Severity:** P2
   - **Confidence:** 100
   - **Classification:** introduced
   - **File:Line:** `model/Umpire/Command/Syntax.lean:475`
   - **R-IDs:** [R3]
   - **Problem:** Resolution still requires an exact action catalog key. On the supplied fixture, both `when: reply` and `when: job.poke` fail with “unknown Model action,” because only their classed keys exist. The task explicitly requires bare references covering every class.
   - **Suggestion:** Expand bare references into their matching classed keys and enumerate the Property across those actions. Pin both synchronized and member-qualified forms.

2. **Sync names can collide with generated member-action keys**
   - **Severity:** P2
   - **Confidence:** 100
   - **Classification:** introduced
   - **File:Line:** `model/Umpire/Command/Syntax.lean:2784`
   - **R-IDs:** [R3, R4]
   - **Problem:** `taken` checks member field names and previous sync names, but excludes generated action keys. Renaming the fixture’s synchronized `halt` to `agent_resume` produces two `agent_resume` catalog entries. The walk indexes results by that key, so one action overwrites the other’s results. A focused probe reproduced overwritten rows; the resulting catalog also fails uniqueness.
   - **Suggestion:** Reject collisions across the complete candidate-action key set before walking, with a located diagnostic naming both actions.

3. **Input compatibility compares spellings instead of domains**
   - **Severity:** P2
   - **Confidence:** 75
   - **Classification:** introduced
   - **File:Line:** `model/Umpire/Command/Compose.lean:172`
   - **R-IDs:** [R3]
   - **Problem:** The mismatch check compares action-key suffix lists. Two distinct input domains with identically named constructors therefore pass. A focused elaboration successfully synchronized `Job.Reply` with a separately declared `Other.Reply`, both containing `ok` and `error`, despite the required input-domain mismatch refusal. Their values are silently paired by spelling.
   - **Suggestion:** Compare participants’ ordered input-domain identities from `Registry.ActionEntry.inputFields`, retaining the intended classless-participant exception.

4. **Scenario starts only resolve a member’s first state field**
   - **Severity:** P2
   - **Confidence:** 100
   - **Classification:** introduced
   - **File:Line:** `model/Umpire/Command/Syntax.lean:602`
   - **R-IDs:** [R3]
   - **Problem:** Composition declarations resolve `job.pending` by inspecting every member state field, but Scenario resolution compares only the first `-`-separated key component. Moving `poked` before `phase` in the fixture produces the valid start key `false-pending`; composing succeeds, but `scenario … starts: job.pending` fails. Field order should not invalidate this member-qualified reference.
   - **Suggestion:** Resolve Scenario starts using the same structural value matching as composition declarations, rather than parsing the key’s first component.

Focused in-memory Lean probes reproduced these cases. The prescribed build and lint commands were blocked by filesystem writes denied in the read-only sandbox.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Worker module belongs to task .1. |
| R2 | deferred | Entity uniqueness lint belongs to task .1. |
| R3 | partial | Command, unions, and fixtures exist; findings above identify resolution and validation gaps. |
| R4 | partial | Reachable walk and fingerprint pins exist; action-key collisions corrupt its results. |
| R5 | deferred | Composition proofs and checks belong to task .3. |
| R6 | deferred | Derived machines belong to task .4. |
| R7 | deferred | Shipped claims belong to subsequent tasks. |
| R8 | deferred | Shipped composition counts belong to subsequent tasks. |
| R9 | deferred | Epic-wide compatibility gate; no listed fixture artifacts changed here. |
| R10 | deferred | Existing production Query compatibility belongs to subsequent tasks. |
| R11 | met | Both set entry paths reject compositions; fixture pins the Query path. |
| R12 | deferred | Authoring documentation belongs to task .6. |
| R13 | deferred | Governance drafts belong to task .6. |
| R14 | partial | All four fixture Queries are added to the differential; shipped and derived Queries remain deferred. |
| R15 | partial | Member state-field lowering is implemented; field-addressed requirements belong to task .7. |

Unaddressed R-IDs: []

Classification counts: 4 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>