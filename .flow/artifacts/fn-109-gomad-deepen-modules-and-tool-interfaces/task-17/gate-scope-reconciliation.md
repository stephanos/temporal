# Process tests must reach the canonical simulation gate

Fresh frozen-source reviewer `network_handle_frozen_review` identified that
`Makefile:147` excludes every new `TestProcessNetworkHandle*` case, despite
their presence in Runner's integration list. The conductor verified the
filter and both CI invocations of `make test-simulation`. Direct root-module
process tests skip without Runner transport, so those invocations cannot
replace the missing integration selection.

Task 17's Description/Touches now include Makefile and a focused nested-module
gate-selection regression. The sole implementation worker owns the correction:
select all thirteen actual-process cases in the canonical gate, retain the
separate forward-mode delay regression and strict watchdog exclusion, and
prove the old filter fails the new selection assertion. Acceptance, native
requirements and D12/resolved-D14 dispositions are unchanged.

The fifteen-file frozen manifest remains the pre-correction source identity.
Freeze and hash the scoped correction after focused checks, then re-review it;
do not overwrite the old-source failure or treat this discovery as native
execution evidence. Writing-for-agents influenced the clarification by placing
the reachable-gate obligation beside the existing process-test requirement.
