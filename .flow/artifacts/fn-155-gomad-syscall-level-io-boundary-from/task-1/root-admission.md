# fn-155.1 source-port admission

fn-155 is next after the retained task 72 verification at `6eae858d82adafaff4bf75a255ccc1cd814b3c44`. Only task .1 is ready. The remaining path begins `.1 → .2 → .8 → .3`; fn-109.17/.18 and fn-110.3 still require .7's decision.

## Selected mechanism

Use a nosplit syscall decoder to establish tracked pointer arguments before any splittable helper. Ordinary Go helpers reuse the existing standalone network model through bounded nonblocking operations. Model state changes wake virtual poll waiters directly; both sides of the host waiter count exclude virtual waits. Caller buffers may not escape into descriptor/model state.

The [safety design](safety-design.md) is a recommendation pending compiler pointer-map, moving-stack, readiness-race and native runtime evidence. The [port survey](../../../../docs/research/gomad/2026-10-10-fn155-syscall-port-readiness.md) identifies the existing source and generation inputs. Their requested models were respectively gpt-6.1-sol/high and gpt-6-astra/high in fresh contexts; actual execution-model telemetry was not exposed. Both belong to the GPT family.

## Scope amendment

`flowctl task set-description fn-155.1 --file - --json` amended the description on 2026-10-10 without changing its acceptance section. The additional implementation surfaces are:

- `tools/gomad3/Makefile`, solely for the explicit overlay-test package list.
- `tools/gomad3/choice/internal/wire/wire_generated.go`, because its identity consumes the patch digest.
- `tools/gomad3/target/internal/livecap/protocol_generated.go`, only if the port changes its declared `runtime/gomad.go` input.

Overlay outputs were already covered by `toolchain/runtime/**`. Descriptor allowlist edits do not admit unrelated generated drift. The selected pointer-handoff approach replaces the task's generic system-stack/fixed-buffer suggestion; the bounded-copy fallback remains available if a specific typed path cannot be proved safe.

Task .1 markdown SHA-256 changed from `a494c4838ca6ac996eb80ff25ecdb40bbd912982cd1922c9f4c915440188b80f` to `de37596cb110ec1491fec0b8a25b0821ba658790be78537b9a5068bc5fa73447`. Its JSON changed from `931e5a694cc3b4dfeeba57e463e3ef6e5c770ee4ae3247021be0eb1eb30a7bca` to `1459502712b01d7072d09009a4d0ba094e2825dc14d7004155299b07c464e9e0` through the CLI's updated timestamp. The other 43 protected files and normalized historical milestone content remain byte-identical.

## Execution limits

The host provides stock Go 1.27.1 on linux/arm64, not a qualified patched-runtime execution platform. Portable tests and darwin/arm64 or linux/amd64 source compilation can support source progress but cannot establish native stack relocation, zero host sockets, virtual deadlines, determinism or disabled-runtime preservation. Task .1's first-platform proof has not been transferred to fn-128 or fn-149. Neither owner is revived here.

The internal switch remains off by default. Test fixtures may explicitly link net to initialize gomadio; a raw-syscall-only target without the backend must refuse rather than use host networking or a second kernel. Task .2 owns recorded selection and startup admission; .8 owns compile-time and closure admission. No adapter deletion, generic policy widening, DNS/filesystem port, CI, PR or push is authorized.

The shared Go/build/lint/generator lane was released after task 72's checks. Grant it to the admitted .1 worker only; independent read-only preparation may run concurrently. Do not reuse task 72's Runner results as an fn-155 baseline or claim a green tree: that capture still has 164 Runner failures and 50 full-lint findings.

## Worktree cleanup request

The read-only inventory found no registered worktree or nested `.git` entry beneath the primary `.flow` directory. Its three file symlinks belong to retained refresh fixtures. No worktree, artifact, cache, branch or evidence was deleted. Worktrees elsewhere are outside the requested cleanup scope.
