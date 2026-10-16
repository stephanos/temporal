---
satisfies: [R2]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.2 Move measurement producers out of test files into wasmmeasure

## Description
Inventory the env-gated producers in wasi/integration_test.go and sibling files (runtime cost profile, matched release cluster baseline/candidate, warm preparation, namespace measurement sample, diagnostic cluster pair, guest profile helper build, terminal-exit compiler controls, cold memory guard). Create internal/backends/wasm/cmd/wasmmeasure with one subcommand per producer and explicit flags replacing every GOMAD_WASM_* variable; share the admission, hashing, host-evidence and report-writing helpers by moving them into a non-test package. Preserve evidence identity assertions and refusal rules from QUALIFICATION.md. Delete the test wrappers. Update QUALIFICATION.md, the wasm README and the rerun sections of the fn-172/174/178 artifacts with the new invocations without rewriting their results. Preserve the existing profile collector in the moved command; fn-185 task .1 follows this spec and extends it there.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/cmd/wasmmeasure/**, tools/gomad/internal/backends/wasm/internal/measure/**, tools/gomad/internal/backends/wasm/wasi/integration_test.go and sibling *_test.go, QUALIFICATION.md, README.md, .flow/artifacts/fn17{2,4,8}-*.md

## Acceptance
- [ ] No test under wasi skips on a GOMAD_WASM_* measurement variable; producers run only through wasmmeasure
- [ ] Each subcommand refuses without explicit host and identity flags and keeps the admission rules
- [ ] integration_test.go shrinks by the producer code; count retained
- [ ] Docs and artifact rerun sections point at the command

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
