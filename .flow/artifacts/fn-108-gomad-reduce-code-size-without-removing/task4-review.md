# fn-108.4 impl-review record

Raw codex bridge on the working-tree diff (commits forbidden). Model gpt-5.6-sol, reasoning effort high, read-only sandbox. One round, verdict SHIP, no findings.

## Round 1

## Findings

No BLOCKER, SHOULD-FIX, or NIT findings.

## Behavioral comparison

Success behavior is unchanged in [upgrade.go](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/upgrade/upgrade.go:512):

- Identical `json.MarshalIndent(..., "", "  ")` bytes and trailing newline.
- Same destination and `MkdirAll(parent, 0o755)`.
- Same-directory temporary file, now prefixed `.safefile-`.
- Same explicit `0o644` mode and rename-over-existing semantics.
- New behavior is intentional: [hostfs.Replace](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/internal/hostfs/replace.go:15) syncs the temporary file and destination directory and reports cleanup failures.

Error-path differences:

| Stage | Old error | New error/effect |
|---|---|---|
| Encode | `encode upgrade dossier` | Unchanged |
| Mkdir | `create upgrade dossier directory` | `publish upgrade dossier: create parent directory`; prior untouched |
| Create | `create upgrade dossier` | `publish upgrade dossier: create replacement`; prior untouched |
| Chmod | `chmod upgrade dossier`; close/cleanup ignored | `publish upgrade dossier: set replacement mode`; close/cleanup errors joined |
| Write | `write upgrade dossier`; close/cleanup ignored | `publish upgrade dossier: write replacement`; close/cleanup errors joined |
| File sync | Not performed | `publish upgrade dossier: sync replacement`; prior untouched |
| Close | `close upgrade dossier`; cleanup ignored | `publish upgrade dossier: close replacement`; cleanup reported |
| Rename | `publish upgrade dossier` | Adds inner `publish replacement`; prior untouched on failure |
| Cleanup | Ignored | `remove replacement` joined into the returned error; the temporary file may remain when removal itself fails |
| Directory open/sync/close | Not performed | Error returned after rename; the complete new dossier is already installed, but never partial bytes |

All message changes and new failure modes are within the stated intent. `Run` still publishes before returning the gate error at lines 227–231, so publication failures retain precedence.

## Architecture, documentation, and tests

- [architecture_test.go](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/architecture_test.go:457) changes exactly one edge: `upgrade → hostfs`.
- [ARCHITECTURE.md](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/ARCHITECTURE.md:692) is accurate. A directory-sync failure occurs after rename, but the text promises only that partial output cannot replace the prior dossier—not that every error preserves the prior dossier.
- The golden test asserts exact bytes, newline, mode, nested-directory creation, replacement of a longer `0o600` file, and absence of temporary files.
- Failed-gate and publication-failure tests correctly establish publication ordering and prior-dossier safety.
- The [RLIMIT_FSIZE test](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/upgrade/upgrade_unix_test.go:21) is sound on Darwin and Linux. Go’s internal write loop retries the initial limited short write and surfaces `EFBIG`.
- Not injecting close/sync/cleanup failures is acceptable: no seam exists in the permitted files, `hostfs` is explicitly out of scope, and the single returned error is transparently wrapped.
- Static verification found clean formatting, no old-prefix matches in CI/ignore/tests, the exact CI upload path, unchanged HEAD, and no staged files. The reported focused tests and validation all exited zero; I did not rerun write-requiring tests in the read-only sandbox.

Acceptance items 1–5 are satisfied.

Review used the [code-review skill](/Users/stephan/.agents/skills/code-review/SKILL.md) and [verification checklist](/Users/stephan/.agents/skills/verification-before-completion/SKILL.md). The [Flow-Next dispatcher](/Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/codex/skills/flow-next-impl-review/SKILL.md) could not execute because its launcher requires temporary-file creation, prohibited by the sandbox.

VERDICT: SHIP
