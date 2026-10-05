# Selected v041 preservation research

Read-only research found a source restoration path within fn-113.3's existing write surface. The deletion commit is `7fd67d5aaeca5d658ca45d571b7671a67ddbfc31`; its parent `56148912df17e105dab3ec4b9e250ff5ef813318` retains the exact selected fixture, pack, request, report and original assertions. This report records research, not an admitted restoration or completed R18 proof.

The six deleted files are `internal/compatibilitypack/packs/modernc-libc-xsys-v041.json`, `requests/modernc-libc-xsys-v041.json`, `reports/modernc-libc-xsys-v041.md` and `testdata/v041/{go.mod,go.sum,libc_test.go}`, under `tools/gomad3`. The fixture selects x/sys v0.41.0, libc v1.72.3, memory v1.11.0 and isatty v0.0.20. Its target is `go-test .` with `test_dep` and `^TestLibcCompatibilityClosure$`.

A complete source candidate also needs the sorted `working-directories.json` mapping, regenerated compatibility-pack outputs and inventory, and additive v041 evidence/policy/mutation controls retaining current v047 coverage. Preserve the current shared-table Makefile qualification flow. Recovering files alone does not prove current selection, approval or workload availability.

The historical v041 profile `sha256:9cd0cff9595bb7f79ec3de247031165cc96d7b40c0e8133051ec17afbf6ac7c0` and libc/memory pins match the current Darwin v047 request. Research found no demonstrated stale-pin mismatch. Current discovery must compare the complete fresh review digest with historical approval `sha256:0ea7c3348fb859e204f7cbfd36db3a735b99261597a3e33260e725b375fe8820`. Preserve existing approval only when that comparison proves equality; changed evidence requires clearing approval and the existing explicit approval flow. No new approval or repinning is authorized by this report.

`compatibility-pack qualify` checks the freshly prepared closure and discovered review digest. It does not execute the fixture. Current native Darwin closure qualification, actual Runner workload execution/replay, original dependencies and formal review remain required. Synthetic generated-pack tests and historical Darwin receipts cannot establish those claims. Linux remains under fn-128.

The research agent made no source, state, cache, generator, test or commit changes. The current source-progress correction still consists solely of the refresh status case.
