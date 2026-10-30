Coordinator merge of review round69374ba736a74a3cbf466335e820aa75.
Original draw reports remain unchanged. Contracts used a multi-file anchor and
integration used an unparseable finding layout, so this file repairs their format.
The NEEDS_WORK verdict below is the draws' worst verdict, not a new verdict.

## Issue 1
- **Severity**: P1
- **Title**: Combined required-Apalache gate bypasses explicit provisioning
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: model/dev/xtask/src/gate.rs:139
- **R-IDs**: R4, R7
- **Problem**: correctness:1. The combined model:all graph dispatches framework leaves directly, bypassing Gate::full_inner and its UMPIRE_APALACHE=require fetch. On a clean checkout the required Apalache tests refuse the missing pinned JAR; the previous Make caller provisioned it first.
- **Suggestion**: Restore explicit optional provisioning before the combined graph and cover the required/default/failure paths without routine auto-install.

## Issue 2
- **Severity**: P1
- **Title**: CI mise pin is older than both newly required config floors
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .github/workflows/umpire.yml:25
- **Problem**: contracts:1. All four Umpire jobs install mise2026.8.16 while both task configurations now require2026.10.11. The pinned binary refuses these configs before their public gate or Go commands run.
- **Suggestion**: Pin the verified2026.10.11 release and its correct extracted Linux x64 binary SHA256 in each job, and verify the workflow/config compatibility.

## Issue 3
- **Severity**: P2
- **Title**: Formatting unnecessarily waits on the host-wide JVM lease
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: model/umpire/dev/xtask/src/main.rs:80
- **Problem**: correctness:2, contracts:2 and integration's second finding describe the same defect. Both dispatchers route fmt through leased supervise, so formatting in one checkout waits for unrelated gate or quick work despite launching no JVM.
- **Suggestion**: Remove fmt from both leased command lists, preserving owned leaf execution and exact formatting arguments/status.

## Issue 4
- **Severity**: P2
- **Title**: Shared supervisor scratch parent prevents a second user from running commands
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: model/umpire/dev/xtask/src/gate.rs:401
- **Problem**: integration's first finding. A first user creates the fixed TMPDIR/umpire-command parent with ordinary permissions. A second user cannot create a command directory there; an attacker-owned parent can also expose or replace another user's redirected logs. This parent was introduced for every supervised public command.
- **Suggestion**: Use the existing tempfile private temporary-directory mechanism or a validated private per-user parent.

```json
{"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[],"suppressed_count":{"25":2,"50":2}}
```

<verdict>NEEDS_WORK</verdict>
