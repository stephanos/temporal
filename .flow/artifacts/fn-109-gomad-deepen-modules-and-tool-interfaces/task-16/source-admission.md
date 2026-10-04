# Task 16 source admission

Task 15's final test/design candidate is integrated in the shared checkout.
Both initial fresh source reviews and bounded-I/O delta rechecks found no
remaining source defects. The conductor reran focused, race, 100-repeat, vet,
format and diff checks against the final frozen identities; evidence remains in
task-15/conductor-verification.md, source-audit.md and evidence.json.

Task 15 is blocked on native acceptance, not done. MILESTONES explicitly permits
source implementation to advance after the predecessor candidate is integrated
and reviewed while its prescribed native gates remain open. Task 16 is claimed
with that source-only force note, not with a dependency-completion waiver.
No full green baseline handoff or native SHIP verdict is asserted.

The implementation follows the reviewed typed semantic-transition design.
Task-16 Approach and Acceptance were reconciled through flowctl to preserve all
valid-behavior test bodies/assertions, allow fixture wiring adaptation and
require old-source RED for the two strengthened historical negative tests.
Their confirmed bugs cannot be preserved as intended behavior. All process and
native requirements remain unchanged; R11 cannot close from the design alone.

Scheduling remains one writer in the current checkout because these tasks are
a sequential chain and user-owned commits prohibit isolated worktrees. Read-only
network/filesystem scouts completed in parallel and their later-task findings
are retained; neither later task has been admitted for implementation.

Worker dispatch requests the Codex implementer `gpt-6.1-sol` at high effort,
with fresh context and no commits, staging, history changes or Flow completion.
Tier: session (jev-unavailable(no_key)); explicit routing remains authoritative.
The conductor owns verification and review after the worker freezes source.
The actual host is developmental linux/arm64 stock Go 1.27.1, with no patched
toolchain, not the old task text's darwin assumption. Supported-native
darwin/arm64 and linux/amd64 gates and all unrelated milestone obligations stay
open. No runtime workaround, collector-policy widening or D12/D14 expectation
change is authorized by this admission.
