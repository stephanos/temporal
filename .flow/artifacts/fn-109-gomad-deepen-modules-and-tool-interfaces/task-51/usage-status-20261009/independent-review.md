# Task51 independent source-progress review

Verdict: SOURCE_PROGRESS_PASS

Ready to merge: No

The admitted test correction and additive public controls support a separate reviewed progress commit. Ordinary package coverage and original-base lint remain red, so this review supplies no formal SHIP or Done verdict.

## Scope and bindings

This fresh reviewer inspected the integrated uncommitted candidate against branch gomad HEAD/base `e09187751326abf393011052dd08fdfc9af61900`. Requested reviewer and writer were both `gpt-6.1-sol` at high effort, from the same GPT family. Actual execution-model telemetry is unobserved. Root reported the session tier fallback `jev-unavailable(no_key)`.

The review binds these inputs by independently recomputing their hashes:

| Input | SHA256 |
| --- | --- |
| handover.md | `f3c6232bafb9e95e798679fca68d39eb2ece3942baea03b9482b79164364f8ca` |
| evidence.json | `43d9773438e54356e3f24f12c3858afce26d7b9a9d5661e7bbf5111616f83586` |
| Final source inventory, 1,109 paths | `c059aae0da0c99ec252b72d128c1a8f1233a517a0749767dcea30e2259bb49a8` |
| Predecessor source manifest, 1,108 paths | `d11d5b9700d75b6cbd0d02d1d44bbaedd832fd1e34f2f02bc534af0abf31204c` |
| Predecessor tool manifest, 26 entries | `8f2aef1b9b27c90165396c6a2f15e92947df84bf05f853aec877a2ffce3a266a` |

I read AGENTS.md, the Gomad README, MILESTONES.md, the complete task51 description and admission, the relevant parent preservation requirements/amendment, task8's original usage decision, changed and unchanged fixture helpers, production dispatch, and the retained command/proof scripts. Read-only Node audits verified 34 current receipts, 68 raw streams, 285 retained control hash references, the 29 receipt references in worker evidence, Git traces, all tool files, generator inventory and source hashes. The 34 receipts comprise 29 worker gates, packet seal, handover correction proof and three later root checks. I ran no Go, build, lint, generator, native, Git mutation or lifecycle command. My only file write is this report.

## Strengths

- `tools/gomad3/cmd/gomadtool/maintainer_output_test.go:145` implements exactly the admitted literal default 2 and compatibility-pack-only 1. Independently reversing those insertions and the invalid-row datum reproduces the complete base file. Other assertions, arguments, operation cases and subtest order remain byte-identical.
- `tools/gomad3/cmd/gomadtool/usage_status_preservation_test.go:12` exercises public `run` with the exact missing/unknown arguments and independent literal usage bytes. Lines 33-68 observe real writer calls, actual read-only-file EBADF, healthy delivery, zero stdout attempts and unchanged publication. Existing helpers retain their established cleanup behavior. Writer scratch directories remain outside the publication snapshot.
- `tools/gomad3/cmd/gomadtool/compatibility_pack.go:23` and `:42` preserve the original a3b9 usage contract. The immutable first-baseline source independently confirms failed delivery 1 and successful delivery 2. Comparing every original Gomad file with e091 confirms all 1,037 unaffected files, including production and task8's permanent closed-writer fixture, remain unchanged.
- `fixture-red.stdout` retains the actual existing `primary status = 1, want 2` failure. The identical focused command in `fixture-green` exits 0 with 26 named passes. The new and permanent controls already pass before the expected-datum edit and still pass afterward, with 12 named passes each. Task46/task50 controls pass with 76 named records, also confirmed by root-focused.
- `mutant-inputs.json` and the retained scratch bytes bind exactly the failed-usage returns at production lines 25 and 44. Both mutant suites fail for literal expected 1 versus actual 2 in the two intended terminal leaves, with healthy leaves passing. No production mutation entered the candidate.
- Scoped lint remains 63 diagnostics and configured original-base lint remains 208. Independent comparison confirms unchanged complete diagnostic/message/source-line records. The actual fixes-disabled fast gate covers 55 host packages with zero findings under the admission-base filter; its scope cannot establish full-lint acceptance. Affected vet, standalone errortype, architecture/purity/public-boundary controls, both static source selections, formatting and check-only validation retain successful receipts.
- The final candidate preserves all 160 generator inputs and 53 generated outputs against the predecessor manifest. No generator/template/pin or policy change is introduced. The handover correction changes exactly one unaffected-file count and two relative links; evidence changes only its summary hash. Initial handover/evidence and the failed original mutation auditor remain retained.

## Issues

Critical: None in the admitted changes or their source-progress evidence.

Important acceptance blockers remain open:

- `ordinary-package-final.stdout:247` retains the failure at `compatibility_pack_refresh_test.go:187` in `TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes/unselected_output`. The complete package run exits 1 with 193 named passes, two failed records and zero skips. Trace2 independently shows init 0 followed by add 128 at the same absolute directory, with no repository recognition recorded for add. The baseline counterpart succeeds. Its cause remains unknown, and task51 supplies no fix or later full-package pass.
- `scoped-final.stdout:190` retains 63 issues with exit 1. `integrated-final.stdout:640` retains 208 issues with exit 2 against original base `951c5516e9e7b3066e7e069adda9565cfd68844c`. Integrated errortype remains unreached. Standalone errortype and filtered fast success cannot close these gates.

Minor: The original mutation auditor counted intermediate public subtest ancestors as failed leaves and returned 1. `proof.mjs:51` corrects the audit by selecting terminal failed leaves from the existing raw runs. The raw failure and original auditor remain immutable. This recovery establishes the intended mutation rejection without a repeated suite and leaves no actionable source fix.

## Plan alignment and recommendation

The candidate satisfies the bounded source amendment and additive preservation scope in task51. The dated parent decision explicitly permits the sole post-first-baseline expected datum. Historical task46 status-2 passes retain their original meaning and packets; task8 keeps the original usage authority, task46 keeps its stdout ownership, task50 keeps its generator ownership and task21 consumes task51 through its direct dependency. Root clarified that authoritative lifecycle state is merged from `.git/flow-state`; bare tracked JSON defaults alone do not contradict the in-progress handover.

Recommend committing this reviewed source progress with task51 acceptance open, then honoring the user's requested stop. There are no critical or important task51 correction requests for the worker. Original first-baseline/fixed-identity, predecessor, preservation, full/default/affected-consumer/functional/formal source obligations stay open wherever unproved. Native fn128/fn149 remain deferred and unverified. Both-source-set metadata and stock linux/arm64 source tests establish no supported-native execution qualification. This report authorizes no task completion, native revival, PR, push or CI action.
