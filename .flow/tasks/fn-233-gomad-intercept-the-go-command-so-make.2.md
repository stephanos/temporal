---
satisfies: [R2, R3, R8]
---
# fn-233-gomad-intercept-the-go-command-so-make.2 go shim: argv[0] dispatch, invocation classification, guest build environment, refusals and byte-exact host passthrough

## Description
Touches: gomad/src/main.rs (argv[0] dispatch), gomad/src/shim/{classify.rs,guest_env.rs,passthrough.rs}, golden corpus.
When started as go, read the shim contract variables and classify: guest test (go test without caller -exec: append pinned compiler env, GOOS=wasip1, GOARCH=wasm, CGO_ENABLED=0, GOTOOLCHAIN=local, overlay flag, installation build tags, -count=1 unless caller set -count, and -exec <gomad> run-guest); guest run/build for main packages matching the guest list (build -o PATH writes PATH.wasm plus a launcher script at PATH); compile-only passthrough when the caller supplied -exec (guest build env applied, caller -exec kept); host passthrough for every other subcommand and unmatched run/build, byte-exact arguments, environment and compiler. Refuse before any build with the existing unsupported-option status and a one-line message: -race, -fuzz, -cover/-coverprofile/-coverpkg, -msan, -asan, CGO_ENABLED=1, excluded -tags, GOOS/GOARCH other than unset or wasip1/wasm.

## Acceptance
Golden cases for every classification branch and every refusal; go env GOOS under the shim reports the host; go test -exec=true -count=0 compiles for wasip1 and launches nothing; caller -count preserved; multi-package go test with -run, -v and -json reports through Go with the correct exit status; passthrough argv and env are byte-exact in the fixture capture; second golden regeneration is byte-identical.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
