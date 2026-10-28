Coordinator format repair and duplicate collapse only. All three raw reviewer verdicts are SHIP; raw reports remain authoritative. Reviewers read source and call sites; sandbox access to main-checkout runtime artifacts was denied, so runtime evidence was independently verified by the conductor.

## Issue 1
- **Severity**: P2
- **Title**: Release identities are mirrored without a drift control
- **Confidence**: 75
- **Classification**: introduced
- **Problem**: JarTool selectors/install paths, backend version pins, installer SHA literals and mise declarations repeat the same current release identities. They agree now, but a future partial version/hash bump can pass existing controls and fail discovery or admission.
- **Suggestion**: Derive runtime selectors from shared pins and/or add a focused Rust-to-mise drift control.
- **File:Line**: model/umpire/backends/process/src/tool_provider.rs:21
- **R-IDs**: []

## Issue 2
- **Severity**: P3
- **Title**: Missing configuration receives a missing-JAR diagnostic
- **Confidence**: 100
- **Classification**: introduced
- **Problem**: The contained-path helper is reused for mise-tools configuration; missing config files receive a missing pinned jar message, and TLC may append a fetch hint that cannot restore absent source configuration.
- **Suggestion**: Use a provider-configuration-specific diagnostic.
- **File:Line**: model/umpire/backends/process/src/tool_provider.rs:49
- **R-IDs**: []

## Issue 3
- **Severity**: P2
- **Title**: Policy dispatch adds a behaviorally redundant branch
- **Confidence**: 100
- **Classification**: introduced
- **Problem**: Checker routes to Running::spawn while Orchestration routes to spawn_with_policy, but Running::spawn delegates to spawn_with_policy with Checker; both arms reach the same effective operation.
- **Suggestion**: Use one spawn_with_policy call.
- **File:Line**: model/umpire/backends/process/src/support.rs:182
- **R-IDs**: []

## Issue 4
- **Severity**: P3
- **Title**: Two contained-path walkers remain after provider replacement
- **Confidence**: 100
- **Classification**: introduced
- **Problem**: The legacy PinnedJar::admit path and new provider path keep similar containment walkers, while the former currently has only in-tree test callers.
- **Suggestion**: Consider consolidating admission without breaking any retained public contract.
- **File:Line**: model/umpire/backends/process/src/jar.rs:69
- **R-IDs**: []

## Issue 5
- **Severity**: P2
- **Title**: Standalone TLC setup invokes a second native-tool bootstrap
- **Confidence**: 75
- **Classification**: introduced
- **Problem**: Standalone setup already builds native tools, then fetch-tlc.sh delegates through xtask.sh to the xtask-bootstrap target and may compile that closure again. Reviewers flag the additional fixed setup cost as nonblocking while final timings remain assigned to task8.
- **Suggestion**: Call the already-built native fetch command in standalone setup instead of the shell bridge; preserve the clean environment and final provider.
- **File:Line**: model/umpire/dev/xtask/src/gate/standalone.rs:418
- **R-IDs**: R7

## Issue 6
- **Severity**: P3
- **Title**: Admission-order documentation is imprecise
- **Confidence**: 100
- **Classification**: introduced
- **Problem**: The admit_mise doc describes resolving then retaining admission, while actual operation admits the contained regular file and SHA before exact mise where resolution, pinned Java and the version probe; the model README omits the resolution step.
- **Suggestion**: Describe the actual admission order.
- **File:Line**: model/umpire/backends/process/src/jar.rs:21
- **R-IDs**: []

```json
{"classification_counts": {"introduced": 6, "pre_existing": 0}, "unaddressed": [], "suppressed_count": {"25": 6}}
```

<verdict>SHIP</verdict>
