# Linux/macOS guest byte-identity certification

> HTML render lens (local file): `.flow/artifacts/fn-232-linux-macos-guest-byte-identity/spec.html`. Regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context

Model authors need a Linux/macOS byte-identity check for both stock and Temporal guests. The owner moved this verification out of fn-230 so its Linux build, embedding and freshness work can proceed. This follow-up retains the original cross-OS requirement and remains open until actual macOS execution proves it.

## Edge Cases & Constraints

Use fn-230's completed build step, the same source revision and pinned toolchain versions on Linux/arm64 and macOS, and clean checkouts with independent initially absent target directories. Record the macOS architecture. No reachable macOS executor leaves verification incomplete. Never compare artifacts from different revisions or substitute two directories on one OS for cross-OS evidence.

## Acceptance Criteria

- **R1:** The stock and Temporal guest artifacts are byte-identical across clean Linux/arm64 and macOS builds of the same source with the pinned toolchain. Each OS also demonstrates path independence using two clean absolute source paths and independent targets. Errors: a build failure, byte mismatch, source/toolchain mismatch, dirty source or unavailable executor leaves certification incomplete; retain the evidence and investigate rather than rewriting artifacts or weakening the check.
- **R2:** The design summary and focused build guide record the source pin, exact commands, OS/architecture, toolchain versions, clean-source checks, target independence, artifact sizes, hashes and actual byte-comparison results. Distinguish historical Linux proofs from the new certification. Errors: missing evidence or unexecuted macOS commands cannot support a completion claim.

## Early proof point

Task fn-232-linux-macos-guest-byte-identity.1 validates actual same-source cross-OS identity. A failed comparison requires a separately scoped reproducibility fix before certification.

## Boundaries

- No prerequisite edge from fn-230 or its consumers back to this certification.
- No changes to panic semantics, target, ABI, protocol, Model, Realization, dependency versions or build architecture. A mismatch needing implementation changes requires a separately scoped fix.
- No remote provisioning, paid infrastructure or credential changes without authorization.

## Decision Context

The owner requested a separate spec/task for macOS proof. A successor spec preserves the check without keeping fn-230's closure and downstream work blocked by an unavailable executor. Keeping it as another fn-230 task would retain that whole-spec gate.

## Quick commands

```sh
bash -n model/scripts/guest.sh
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Same-source/toolchain stock and Temporal identity across both operating systems and independent paths, with explicit failure cases. | fn-232-linux-macos-guest-byte-identity.1 | None |
| R2 | Commands, platform, cleanliness, target, hashes, sizes and actual comparison evidence in the existing proof documentation. | fn-232-linux-macos-guest-byte-identity.1 | None |
