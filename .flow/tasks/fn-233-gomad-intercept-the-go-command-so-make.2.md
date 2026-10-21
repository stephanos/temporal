---
satisfies: [R2, R3, R8]
---
# fn-233-gomad-intercept-the-go-command-so-make.2 go shim: argv[0] dispatch, invocation classification, guest build environment, refusals and byte-exact host passthrough

## Description
**Touches:** tools/gomad/rust/gomad/src/main.rs (argv[0] dispatch), tools/gomad/rust/gomad/src/shim/{classify.rs,guest_env.rs,passthrough.rs}, tools/gomad/testdata/parity/**.
When started as go, read the shim contract variables and classify: guest test (go test without caller -exec: apply pinned compiler env and insert Go-driver flags before any -args/--args boundary, GOOS=wasip1, GOARCH=wasm, CGO_ENABLED=0, GOTOOLCHAIN=local, overlay flag, installation build tags, -count=1 unless caller set -count, and -exec <gomad> run-guest); guest run/build for main packages matching the guest list (build -o PATH writes PATH.wasm plus a launcher script at PATH); caller-owned execution when the caller supplied -exec (guest build env applied, caller -exec kept and invoked by Go; only a no-op launcher such as true is compile-only); host passthrough for every other subcommand and unmatched run/build, byte-exact arguments, environment and compiler. Refuse before any build with the existing unsupported-option status and a one-line message: -race, -fuzz, -cover/-coverprofile/-coverpkg, -msan, -asan, CGO_ENABLED=1, excluded -tags, GOOS/GOARCH other than unset or wasip1/wasm.

## Acceptance
Insert injected Go-driver flags before -args/--args, preserving caller argument bytes/order and effective flag policy; arguments after the boundary remain guest arguments. Include GOFLAGS admission and golden boundary/flag-lookalike cases. Create the immutable per-invocation metadata and pass its path/digest to only that compiler subprocess and its launchers.

Golden cases for every classification branch and every refusal; go env GOOS under the shim reports the host; go test -exec=true -count=0 compiles for wasip1 and launches nothing; a supplied nontrivial -exec fixture is invoked by Go and receives the module/arguments without a Gomad execution-evidence claim; caller -count preserved; multi-package go test with -run, -v and -json reports through Go with the correct exit status; passthrough argv and env are byte-exact in the fixture capture; second golden regeneration is byte-identical.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
