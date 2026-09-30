# Known-bug handling

This document records known-bug handling requirements for the Scala-authored, Go-interpreted Umpire
model.

The [shared Umpire specification](../../../.plans/UMPIRE4_SPEC.md) defines the existing architecture and
vocabulary. [SEMANTICS.md](../SEMANTICS.md) defines IR evaluation. The requirements below describe
proposed behavior, not functionality already implemented.

## Known bugs

A developer must be able to acknowledge a discovered violation without fixing it immediately.
Umpire must continue checking the correct Property, report occurrences of that known bug as warnings,
and turn recurrence into an error once the developer marks the bug fixed.

### Declaration and scope

A **Known Bug** is a versioned declaration identifying one acknowledged failure and its lifecycle
state. It records:

- A stable ID, description, and issue reference when one exists.
- The affected Property or conformance obligation and the relevant Definition IDs and Behavior
  Fingerprints.
- The discovery stage, such as Model Search or implementation execution, and any model variant,
  assumptions, bounds, or implementation versions that restrict where the acknowledgment applies.
- A witness or regression Query identifying the failure, and explicit matching criteria for its
  occurrences.
- Its state, `active` or `fixed`, and the evidence supporting a transition to `fixed`.

Keep the declaration with the owning Model and retain the regression when its state changes. A
Known Bug found in Model Search does not automatically acknowledge a runtime failure. The mapping
must establish that correspondence before the acknowledgment applies to execution results.

A Known Bug differs from a Known Gap. It acknowledges an established violation; a Known Gap records
missing support or knowledge that limits what can be established.

### Matching an occurrence

Match the specific failure, rather than suppressing every failure of a Property or every failure in
a Scenario. The match must identify the violated obligation and the distinguishing failure evidence,
including correlation relationships where relevant. Different schedules may expose the same bug,
so a match need not require identical Case bytes or an identical entire trace.

Generated IDs may differ between executions, but matching must preserve relationships between
logical operations, attempts, deliveries, and runs. A different operation's evidence cannot satisfy
the match. An ambiguous match must retain the ordinary error and report the ambiguity.

If the relevant behavior or applicable scope changes, the acknowledgment must be reviewed before it
can downgrade another occurrence. Stale metadata must not silently acknowledge a new violation.

### Reporting and failure behavior

Keep the underlying check result and its reporting severity separate. A matched violation remains a
violation, with its witness, evidence, and failed obligation available in the result. Reporting it as
a warning must not rewrite it as a satisfied Property or Verdict.

| Declaration state | Result | Reporting behavior |
| --- | --- | --- |
| No matching Known Bug | Established violation | Error; fail the check. |
| `active` | Matching violation within the declared scope | Warning naming the Known Bug; this occurrence alone does not fail the check. |
| `active` | Retained regression completes decisively without the expected violation | Warn that the bug was not reproduced and request lifecycle review. Keep it active until explicitly changed. |
| `fixed` | Matching violation recurs | Error naming the Known Bug; fail the check as a regression. |
| `fixed` | Retained regression completes decisively without the violation | Ordinary successful result within the checked scope. |
| Either state | Incomplete or inconclusive result | Preserve that status and its ordinary failure policy. It establishes neither occurrence nor absence. |

Evaluate and report every failure independently. One acknowledged occurrence must not hide an
unmatched violation, a different failure of the same Property, or an admission, infrastructure,
evidence, or cleanup error. Warning-only checks may complete successfully, but their reports must
show acknowledged violations separately from satisfied checks.

Known-bug handling belongs after semantic evaluation. It must not weaken the Property, remove a
transition from Search, change a Contract, add a Scenario guard that avoids the bug, or disable
execution's existing safety-stop and cleanup behavior.

### Lifecycle

1. Umpire discovers a violation and retains its witness and relevant scope. Discovery alone does
   not acknowledge it; a developer adds an `active` declaration after reviewing the failure.
2. Subsequent matching occurrences produce warnings. Unmatched failures continue to produce errors.
3. After correcting the affected design or implementation, the developer runs the retained
   regression and records the applicable fix evidence. Runtime reproduction and promotion continue
   to follow the shared Umpire requirements where applicable.
4. The developer explicitly marks the declaration `fixed`. Retain the failure matcher and regression
   so recurrence fails. Record revised fingerprints or version scope without losing the original
   discovery provenance.

A single clean exploratory run must not automatically mark a bug fixed. If the design model and
implementation are corrected at different times, their acknowledgments need separate stage or
version scopes. Fixing one does not establish that the other was fixed.

### Example

For a stale dispatch admitted after an activity pause commits, declare the correct admission
Property and retain the violating Query. An `active` Known Bug matching that stale-admission failure
produces a warning. A separate duplicate-admission failure still produces an error, even if it
violates the same Property.

After admission rechecks current eligibility and the retained regression confirms the correction,
mark the applicable Known Bug `fixed`. Any later stale-admission occurrence fails the check. Keep
the same promised behavior throughout the lifecycle.

### Acceptance examples

The implementation must demonstrate an active occurrence reported as a warning, the same occurrence
reported as an error after marking it fixed, and an unrelated violation that fails while another bug
is active. It must also demonstrate stale or ambiguous matching that cannot suppress an error,
inconclusive evidence that cannot establish a fix, and preservation of the original violation
evidence through reporting and replay.

Declaration syntax, serialization, and command-line presentation will be specified when this
requirement is implemented.
