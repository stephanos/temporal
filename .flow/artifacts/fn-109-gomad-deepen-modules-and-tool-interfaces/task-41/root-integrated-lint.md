# Canonical JSON integrated lint checkpoint

The original integrated gate measured 324 remaining findings on the frozen amended candidate, compared with 325 before this correction. Root compared the complete raw diagnostic blocks and found exactly the canonical.go:120 exhaustive diagnostic removed. All 324 other blocks are byte-identical and no new finding appeared.

Root ran this exact command from the repository root on 2026-10-05 at 13:37:09 UTC. It finished at 13:37:41 UTC with Make exit 2 after 32.356986182 seconds.

```bash
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype
```

The command used the recorded offline stock Go 1.27.1 environment, empty GOFLAGS, GOMAXPROCS=2 and unset Gomad seed controls in `root-integrated-command.sh`. HEAD was reviewed admission `deb4330663e83c669f365d2428fdbb65e1ff2a68`; the uncommitted canonical production SHA-256 was `738375708f711bb44094f57a346ec9131159f5ef61da402b3a62ec82b15f9a41`, and the additive test SHA-256 was `274b979156bd6f4ff47c6d5daa454dee51a670d65c3ba694ebed87e647bff747`.

The trace selected the same 55 ordinary host packages and original `disable_grpc_modules,,test_dep,` tags, configuration, base revision and new-from-rev policy. Root did not narrow the gate or add exclusions. The selected source/toolchain/config/module protection checks matched before and after execution. Nine integrated input pins matched twice; 994 unchanged selected BASE inputs matched twice; both candidate files matched afterward. This establishes stability for the retained selector, not the whole repository or every inherited environment variable.

| Linter | Before | Final |
| --- | ---: | ---: |
| errcheck | 252 | 252 |
| exhaustive | 5 | 4 |
| forbidigo | 11 | 11 |
| gci | 1 | 1 |
| staticcheck | 56 | 56 |
| Total | 325 | 324 |

Golangci-lint exited 1, its recursive Make exited 2 and the top-level Make exited 2. Makefile:505 failed before the following errortype vet recipe, so full errortype was unreached. The separately executed canonical-package errortype pass does not establish full-gate acceptance.

The remaining four exhaustive diagnostics belong to compatibility-pack refresh pinimpact.Status, internal compatibility-pack FactKind policy, Runner choice-divergence and Runner completion Strategy switches. They retain their existing control flow and owners until separately admitted corrections. The errcheck, forbidigo, gci and staticcheck blocks remain under their existing correction or source owners; task21 consumes the exact residual log and returns gaps to those owners. Task41 authorizes no changes to those files or gate settings.

`root-integrated-final.log` has SHA-256 `1bb1271a13e1ab87cf3ec0fadedd8ffb22e8064046c5ec5cf590678cdfac7544`. `root-integrated-baseline.log` retains the original 325-finding comparison. The root command, manifests and before/after checks accompany the logs. The scoped lint result is green; original integrated, full/default/functional/affected-consumer/formal/native-Darwin and first-baseline acceptance remains incomplete. Missing transferred Linux execution stays with fn-128 and does not block source progress.
