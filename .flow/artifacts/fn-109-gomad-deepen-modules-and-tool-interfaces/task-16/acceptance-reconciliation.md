# Task 16 characterization preservation

Task 15's unchanged-production characterization exposed two pre-existing
violations of R11's validation-before-mutation contract. The malformed-wait pin
records a waiter waking before acknowledgement rejection. The duplicate-admission
pin records arrival credit consumed before duplicate-response rejection.

Task 16's original blanket requirement that every characterization test pass
unchanged conflicts with its requirement to reject invalid work before progress.
Clarify that all valid-behavior test bodies and assertions remain unchanged;
only migration-sensitive fixture wiring adapts to the lifecycle interface.
Strengthen the two named historical-defect pins into negative regressions and
retain their failure against the old production source before fixing the owner.
No other acceptance check changes. Neither bug is waived or frozen as intended
behavior. No native, process, replay or final R11 gate is removed.

This is a scoped test-contract reconciliation based on concrete task-15
evidence, not a new implementation owner or a reduction of the milestone scope.
The standing user instruction selects the grounded recommendation without an
interactive approval step; user-owned commits remain untouched.
