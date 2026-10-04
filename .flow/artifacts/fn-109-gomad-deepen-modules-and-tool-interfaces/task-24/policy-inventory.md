# Repository-relative lint policy

Root, Gomad and mixedbrain lint now use the same repository-relative exclusion
base. The sole config change adds `run.relative-path-mode: gitroot`.
[config-change.diff](config-change.diff) retains the exact one-line addition.
All exclusion expressions, settings, enabled linters, forbid/text patterns,
formatter policy, comparison revisions and fix flags remain unchanged.

The pre-edit YAML/regexp controls refute the historical doubled-escape
hypothesis. [path-expression-bytes.log](path-expression-bytes.log) retains every
path scalar's original bytes. `5c 2e` represents one backslash before a dot;
the tools rule also contains `5c 2f`, one backslash before a slash.
The `parsed-expression` entries in [policy-red-final.log](policy-red-final.log)
retain loaded values. Quoted/log-serialized values escape those backslashes
again for display. [Root's correction](root-diagnosis-correction.md) supersedes
the historical diagnosis while preserving the old task-23 report.

Rule indices below are zero-based. Every row with a rule expression receives
independently literal match and nonmatch cases in
`TestLintPolicyPathExpressions`. Repeated test/helper scopes share a table row
but each rule is compiled and exercised separately. Unanchored expressions
retain their original substring/prefix behavior.

| Location | Actual expression | Intended existing scope |
| --- | --- | --- |
| global paths 0 | `^api` | repository API prefix |
| global paths 1 | `^proto` | repository proto prefix |
| global paths 2 | `^.git` | existing wildcard-dot Git prefix, including `.github` |
| rule 0 path-except | `_test\.go\|tests/.+\.go` | enforce sleep policy only in tests |
| rule 1 path-except | `chasm/lib/.*\.go$` | enforce clock policy only in Chasm library Go source |
| rule 2 path | `chasm/lib/.*_test\.go$` | exempt Chasm library tests from clock policy |
| rule 3 path-except | `common/persistence/cassandra/.*\.go$` | timestamp policy only in Cassandra persistence |
| rule 4 path | `_test\.go$` | timestamp exemptions for tests |
| rule 5 path | `_test\.go\|tests/.+\.go\|common/testing/` | panic exemptions for tests and testing helpers |
| rule 6 path | `chasm/lib/activity/model/.*\.go$` | existing activity-model panic exemption |
| rule 7 path | `tests/testcore/.*\.go$` | existing legacy base exemption in testcore |
| rule 8 path | `common/namespace/(namespace\|replication_resolver)(_test)?\.go$` | namespace definitions and their direct tests |
| rule 9 path-except | `tests/.+_test\.go` | background-context policy only in functional tests |
| rule 10 path | `tests/(nexus_standalone\|nexus_workflow\|schedule\|schedule_migration)_test\.go$` | existing four legacy Eventually exemptions |
| rule 11 path | `tests/(nexus_standalone\|nexus_workflow)_test\.go$` | existing two CollectT exemptions |
| rule 13 path | `_test\.go\|tests/.+\.go\|common/testing/` | existing test/helper complexity exemption |
| rule 14 path | `_test\.go\|tests/.+\.go\|common/testing/` | existing test/helper dot-import/type-assertion exemption |
| rule 15 path | `^tools\/.+\.go` | existing tools-source revive exemption |

Rules 12 and 16 have no path/path-except expression. The inventory comprises
three global expressions and fifteen rule expressions, with zero regex edits.

The final clean fixture RED runs against config SHA
`86d71dda338f89c748a7ecae99e989d03b71b8693280adddae3970eba04c930a`.
Its root excluded batch reports three forbidden findings and one revive finding;
the root ordinary and nested tools findings have incorrect `../` paths, and
the nested tools batch also reports its revive-only else branch. These are
the intended regression failures. Earlier `policy-red` and
`policy-red-clean-fixture` runs contain fixture setup noise; `policy-green`
is a failed intermediate check with a staticcheck fixture finding. They remain
raw historical observations and are not final pass evidence. The final fixture
uses formatted, directory-matching packages and a revive-only branch.

[The final full helper run](helper-contracts-final.receipt.json) actually
executes the verified v2.13.0 binary. Its root excluded batch has no findings;
the ordinary root and nested Gomad application panic each remain exactly one
forbidigo finding with repository-relative filenames. Test panic findings and
tools revive findings remain excluded only by preexisting rules. The mixedbrain
fixture preserves the existing `tests/.+\.go` panic exemption, which includes
non-test files under `tests`. It makes no claim that those files are governed
by application panic policy. The default test reports a skip when its explicit
binary input is absent; this dispatch supplied and executed that input.

Git-root matching activates the existing `^.git` reporting limitation.
The real root fast log records 27 findings excluded by this unchanged global
rule, and the actual `.github/actions/policy` fixture is excluded. Matching
restoration does not establish hidden-action lint cleanliness.

[preservation.log](preservation.log) records a zero-exit diff against the
task base for all tracked Go source, module manifests, Makefiles, workflows
and old task-23 artifacts. The new regression file is the sole Go addition.
Each final gate checks its before/after source and tool hashes. Original
functional expectations, version pins, native defaults and historical receipts
remain unchanged.
