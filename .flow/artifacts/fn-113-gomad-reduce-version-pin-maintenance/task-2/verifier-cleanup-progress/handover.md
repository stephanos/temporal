# Verifier cleanup source progress

Both registered-adapter verifier paths now report scratch removal failures at their existing deferred release boundary. Successful cleanup preserves the original error object and unwrap shape; a sole cleanup failure returns directly, and simultaneous failures join the primary error first.

Task fn-113-gomad-reduce-version-pin-maintenance.2 remains in_progress for the conductor. This bounded correction closes no original task1 dependency, R3 regeneration acceptance, native workload, full/default, affected-consumer or formal review requirement. Linux qualification remains deferred and unverified under fn128. The conductor owns review, lifecycle and the separate source-progress commit. The worker staged, committed and pushed nothing.

Tier: judge unavailable(no_key), project explicit model retained.
stage: impl-review - skipped(policy: conductor owns source-progress review; original red acceptance gates remain open)

The production diff changes only the two named returns and their existing scratch defers. Comments, validation and listing order, module and source pins, generated inputs, native guards and publication logic are unchanged. Similar-code investigation reused target/adapter_source_set.go and artifact/open.go's conditional cleanup composition, and the tests reuse copyFixtureTree without changing pinnedReleaseGo or newRegenerationFixture. No production seam, helper or dependency was added.

## Proof and limits

On unchanged ce126fe8d308da268fecf14e57dc7832a46c5084 production source, four success/primary controls passed. Before any production edit, all four real cleanup faults failed as intended. Each capability probe and the residual owned scratch directory returned a real permission error on uid1000 linux/arm64. The two malformed-listing cases exposed wrapped json.SyntaxError alone; the two valid-listing cases returned nil despite failed removal. Fixture cleanup restored permissions and removed the retained scratch with checked operations even after fatal assertions.

The final 192-line test repeats the same proof with exact original production files supplied through Go's file overlay. The final test hash is identical in that RED and GREEN pair. RED exits1 with four normal controls passing and four real-fault assertions failing. GREEN exits0 with eight verifier controls plus six existing portable controls passing. All four permission faults executed; no cases skipped. Primary-only failures preserve the exact message and single-error unwrap. Simultaneous errors expose json.SyntaxError and os.PathError through errors.As and fs.ErrPermission through errors.Is, with primary first. Sole cleanup failures return the raw os.PathError. Success and primary-only cases leave the private scratch parent empty.

The test resolves PATH Go with GOTOOLCHAIN=local, checks go1.27.1 and reads GOMODCACHE by lines. Complete cached Sentry/libc trees supply pinned source fixtures; no host path is hardcoded in the test. The real-Go wrapper lists both qualified source sets without executing a patched target. Permission controls probe the actual host and explicitly skip bypassing hosts. Restoration is registered before invocation and validates its removal target as a named verifier scratch child of the private TMPDIR.

Unfiltered scoped lint changes from four findings to the same two residual findings in adapter_registry.go:390 and profile.go:211. Exactly the two owned errcheck diagnostics disappear, with no additions or suppressions. Final vet, errortype, check-only validate, four architecture checks and mandatory fast lint exit0. Fast lint filters308 residual configured findings to zero changed-line findings. The inherited full lint baseline of310 therefore improves by two; full lint remains red.

One initial selector inadvertently included three existing fixture families that require the absent patched driver. Its exit1 and exact failures are retained in evidence.json as inherited tooling observations; six portable controls passed in the same invocation. These unavailable-driver families were not retried. An initial /usr/bin/time wrapper failed before validation started; Bash time supplied the successful baseline. The deprecated runtime.GOROOT test resolver produced one temporary staticcheck diagnostic and a failed fast-lint observation before correction. The final fixture uses exec.LookPath and passes the same unfiltered lint comparison.

## Evidence

evidence.json binds the final source, exact base overlay, Go/tool/config hashes, commands, exits, durations and raw log hashes. Compact RED/GREEN and lint logs sit alongside it. Large streams and the untouched raw copies remain in .flow/tmp/verifier-cleanup. The committed log rendering normalizes whitespace on blank lines only; all six retained copies currently hash identically to their raw inputs.

Defect route:

- prior fixes: local all-ref file history shows the existing regeneration implementation and merges, with both ignored defers still present in the dispatched base. Memory returned profile-pack and module-download issues, neither a cleanup fix. PR and tracker searches are unchecked because gh returned HTTP401.
- diagnosis: public success and malformed-listing controls eliminate pin/fixture/JSON-setup failure; real capability probes and residual unreadable directories confirm removal denial. Base discards that denial at the two deferred RemoveAll calls, and the scoped correction exposes it without changing the primary result.
- introduced by: skipped bisect because no known-good cleanup revision was supplied. The original regeneration implementation appears in file history at076cdcc344.
- base: real-fault RED before edits plus final-test RED against exact ce126fe8d3 source copies. Head: matched final-test GREEN against the frozen correction, with all commands terminal.
- live: no live surface; proof uses public library verification with real filesystem and Go processes.

The conductor must retain a fresh source-progress review and original open acceptance when it records the checkpoint. No formal SHIP verdict was issued by this worker.
