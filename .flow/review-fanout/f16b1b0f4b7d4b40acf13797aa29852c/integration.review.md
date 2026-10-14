I’ll read the task spec first, then trace the changed code through its callers and run the focused checks available in this sandbox.
The classed-action fixtures are covered, but bare classed references and composed key collisions need closer tracing through the existing lookup code. I’m checking those paths next.
Bare `when:` references are still checked against complete action keys, which appears to reject a classed action unless its class is supplied. I’m reproducing that and checking whether synchronization compares input types or just class spellings. The focused Lake build has produced no output so far; I’m using Lean directly against the existing build artifacts for these targeted checks.
Both prescribed checks are blocked by the read-only sandbox: Lake cannot write `Syntax.setup.json`, and lint cannot create its inventory temporary file. A targeted Lean check did confirm that a sync named `agent_resume` produces the same catalog key as the member action `agent.resume`. I’m finishing source-only checks of the command-level failures.
The targeted reproductions confirm four command-level issues: bare classed `when:` references are rejected, distinct input types with matching class spellings are accepted for synchronization, sync names can duplicate member action keys, and Scenario starts fail when the member’s phase is not its first field. I’m checking the separate state-key collision case before writing the verdict.
1. **Severity:** P1  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Syntax.lean:475`  
   **R-IDs:** [R3]  
   **Problem:** Bare references to classed actions still require an exact catalog key. Against the supplied fixture, both `when: reply` and `when: job.poke` fail with “unknown Model action”; only class-qualified references succeed. This misses the explicitly required bare-action behavior.  
   **Suggestion:** Expand a bare action reference into its catalog classes and enumerate the Property over each corresponding trigger. Add tests for both synchronized and member actions.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Compose.lean:173`  
   **R-IDs:** [R3]  
   **Problem:** Input compatibility compares class-key suffixes, not input domains. A reproduction synchronized `Job.Reply` with a distinct `OtherReply` enum because both declare `ok` and `error`; the composition elaborated successfully. The generated action carries the first participant’s type while selecting the other participant’s rows by spelling, silently accepting the domain mismatch the command promises to reject.  
   **Suggestion:** Compare the classed participants’ ordered input-domain types using their registry declarations. Preserve the explicit exception for classless participants.

3. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Syntax.lean:2784`  
   **R-IDs:** [R3]  
   **Problem:** Sync names are checked against member field names and other sync names, but not generated member-action keys. Naming the fixture’s halt synchronization `agent_resume` collides with `agent.resume`. The command elaborates and emits `"agent_resume"` twice in its Action catalog. The walk also indexes both actions under the same key, and downstream table admission rejects the duplicate catalog.  
   **Suggestion:** Check uniqueness across all generated action keys before walking, and report collisions at the responsible sync declaration.

4. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Compose.lean:86`  
   **R-IDs:** [R3, R4]  
   **Problem:** Joining member state keys with `_` is not injective over the existing legal key grammar. `["a_b", "c"]` and `["a", "b_c"]` both encode as `"a_b_c"`. A targeted walk returned both states with that duplicate key. Because `found` also uses the joined string, rows can overwrite each other before the invalid catalog reaches admission. Rejecting underscores in member *field names* does not constrain member state keys.  
   **Suggestion:** Keep component tuples as internal row-map keys and detect ambiguous emitted state keys with a located error before emitting the table.

5. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Syntax.lean:602`  
   **R-IDs:** [R3]  
   **Problem:** Composed Scenario start resolution assumes the named value occupies the first segment of the member’s state key: `Compose.actionName key == value.toString`. Existing machines allow the phase field elsewhere. With `{ poked : Bool, phase : JobPhase }`, `compose starts: [job.pending]` succeeds, but a Scenario using the same reference fails because its start key is `false-pending`.  
   **Suggestion:** Resolve starts from structured member-field values, consistently with the composition declaration, instead of parsing the first key segment.

## Requirements coverage

Coverage is scoped to task `.2`; remaining epic work is explicitly deferred.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Worker module belongs to `.1`. |
| R2 | deferred | Entity uniqueness lint belongs to `.1`. |
| R3 | partial | Command and unions implemented; findings 1–5 remain. |
| R4 | partial | Reachable walk and ordering tests exist; state-key collisions violate injectivity. |
| R5 | deferred | Agreement proofs and checks belong to `.3`. |
| R6 | deferred | Derived machines belong to `.4`. |
| R7 | deferred | Shipped claims belong to `.3`, `.5`, and `.7`. |
| R8 | deferred | Production state-count pins belong to `.3` and `.5`. |
| R9 | deferred | Cross-spec compatibility gates belong to later tasks. |
| R10 | deferred | Existing-query compatibility belongs to `.4` and `.5`. |
| R11 | met | Composition rejection covers direct and Query-based set references; fixture pins the latter. |
| R12 | deferred | Documentation belongs to `.6`. |
| R13 | deferred | Governance drafts belong to `.6`. |
| R14 | partial | All four fixture Queries appear in the differential; production coverage is deferred. |
| R15 | partial | Member-field lowering implemented; field-addressed enumeration belongs to `.7`. |

Unaddressed R-IDs: [R3, R4]

Targeted Lean reproductions confirmed all five findings. The prescribed build and lint commands were blocked by sandbox write restrictions, so their passing status could not be verified.

Classification counts: 5 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":5,"pre_existing":0},"unaddressed":["R3","R4"]}
```

<verdict>NEEDS_WORK</verdict>