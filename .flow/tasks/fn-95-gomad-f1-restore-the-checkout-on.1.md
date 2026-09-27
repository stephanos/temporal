---
satisfies: [R1]
---
# fn-95-gomad-f1-restore-the-checkout-on.1 Build the go1.27.1 toolchain on darwin/arm64 and pass gomad doctor

## Description
Run `make gomad3` with GOROOT on a stock go1.27.1; confirm `tools/gomad3/.bin/gomad doctor` reports the runner, all adapters, and the artifact store available. Repair any darwin build failure.

## Acceptance
- `make gomad3` exits 0 on darwin/arm64
- `gomad doctor` reports runner available

## Done summary
`make gomad3` exits 0 on darwin/arm64 with GOROOT on a stock go1.27.1: the go1.27.1 toolchain built from source (key 85c444f9...) and `.bin/gomad` built. `gomad doctor` reports available=true, with the runner, all 7 adapters and the artifact store ok. No darwin build failure surfaced, so this task needed no code change. The trailing `repair:` line in the doctor text output always prints and is informational.

stage: impl-review - ran (triage_skip SHIP: empty task diff)

Note: commit 1bc1d4affa (Merge CLAUDE.md into AGENTS.md) landed concurrently from outside this task and is not part of it.
## Evidence
- Commits:
- Tests: baseline: none (spec defines no Quick commands), make gomad3 (GOROOT=stock go1.27.1, darwin/arm64) rc=0; toolchain key 85c444f98ab905a782983191f7d47dd4ce9687a606051d1c87f9ffdf71a39f16, tools/gomad3/.bin/gomad doctor rc=0: available=true; host, toolchain, runner, 7 adapters, artifacts all ok
- PRs: