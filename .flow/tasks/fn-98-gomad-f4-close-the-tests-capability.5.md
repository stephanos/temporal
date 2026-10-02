---
satisfies: [R1, R2]
---
# fn-98-gomad-f4-close-the-tests-capability.5 Remove os/exec and os/signal admissions from the modernc libc packs

## Description
Completion review finding (P1): modernc-libc-xsys-v041, -v047 (darwin) and -v047-linux-amd64 admit `import:os/exec` and `import:os/signal` for modernc.org/libc, violating F4's 'no pack admits os/exec' constraint (and the spec's os/exec/os/signal/os/user ban). Rewrite the libc adapter's prepared sources so those imports are gone (the system/popen/signal paths refuse deterministically, like the fx/SDK/otel adapters), regenerate the affected requests/reviews/packs through discover/review/generate (linux pack: regenerate only what can be done from darwin; if the linux pack cannot be regenerated here, record it as needing a linux run and keep its request consistent), add a validation test that rejects any pack admitting os/exec, os/signal, or os/user, and re-run darwin closure analysis of ./tests, compatibility-pack-qualification, core set, and the Temporal set.

## Acceptance
- no pack admits os/exec, os/signal, or os/user; a validation test enforces it
- darwin ./tests closure still 0 blockers; pack qualification, core set, Temporal set pass

## Done summary
The modernc libc adapter now replaces `system`/`pause` (darwin), `signal` (libc_unix.go), and `signal`/`system` (musl) with unconditional refusals and drops the `os/exec`/`os/signal` imports. Preparation fails closed if any rewritten source still imports `os/exec`, `os/signal`, or `os/user`. `ValidatePack` now rejects any pack that admits those three imports, and request validation rejects allowing them while denied records stay valid. Tests: `TestDecodePackV2RejectsWeakOrNonCanonicalPolicy` (admits cases), `TestValidateRequestRejectsUnadmittableCapabilities`, and `TestRejectLibcHostImportsRejectsForbiddenImports`.

The darwin v041, v047, and v047-isatty-v021 packs were rediscovered, reviewed, and regenerated with the observed darwin pin. The linux pin and the v047-linux-amd64 request evidence were derived from the linux/amd64 `go list` file set; the same method reproduces the observed darwin pin exactly. They still need a linux/amd64 `compatibility-pack-qualification` run, and a wrong value fails closed. After the change, the `./tests` closure still reports supported with 0 blockers, the core set passes 5/5, and the Temporal set meets its expectations (16/0/2/0, 18/18). README and the F4 status in MILESTONES.md record the change; the F8 section is untouched.

Memory auto-capture was skipped because `.flow/memory` is not initialized.

stage: impl-review - ran [2026-09-27] codex fan-out NEEDS_WORK (1 P2: denied-only forbidden facts invalidated requests) -> fixed -> SHIP

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ec3053f1223aa664b88ff09b43136818992f583a, 2449051e200aaf629df033c0bd17a8d395cc7d09
- Tests: baseline: green (make -C tools/gomad3 validate, pre-edit; spec defines no Quick commands), make -C tools/gomad3 validate (pass, after fix commit), make -C tools/gomad3 compatibility-pack-qualification (pass, 8/8 darwin requests, rerun after fix commit), make -C tools/gomad3 test-host (pass, 44 packages, rerun after fix commit), go test -tags test_dep ./internal/compatibilitypack/... ./deterministicio/ ./cmd/gomadtool/ (pass; new tests confirmed red before fix), make -C tools/gomad3 core-qualification-set (expectations-met=true, supported=5 unsupported=0 failed=0, 5/5), gomad analyze --capability-mode=closure --format=json --build-tag=disable_grpc_modules --build-tag=gomad --build-tag=test_dep go-test ./tests (classification=supported, 0 blockers, 0 guarded, 0 eliminated), make qualification-set with tools/gomad3integration/qualification/temporal.json (gomad3-qualification set half; expectations-met=true supported=16 unsupported=0 failed=2 intermittent tier3 infrastructure-errors=0 completed=18/18), linux/amd64: modernc-libc-xsys-v047-linux-amd64 evidence and linux prepared source-set pin derived on darwin, not observed; needs a linux/amd64 compatibility-pack-qualification run
- PRs: