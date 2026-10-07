# Patch regeneration cleanup source handover

Task `fn-109-gomad-deepen-modules-and-tool-interfaces.48` corrects the seven admitted unchecked cleanup sites. Source is frozen on branch `gomad` against BASE `101b14f882195422c31f35afc259cb25050b31e3`; the worker's Go/cache lane is terminal, with no live worker command. This is verified source progress, not completed acceptance or SHIP. Root owns Flow state, independent review, broader checks and the separate progress commit. No staging, commit, push or history operation occurred in this lane.

Only `patch_regenerate.go` and additive `patch_cleanup_test.go` changed. Deferred work removal, VERSION Close, copy-input Close and temporary removal retain their original lifetimes; cleanup succeeds without changing the primary error, a sole cleanup failure returns directly, and actual additional failures join primary first. The three early temporary Close calls precede primary formatting. The original copy-output Close/error expression, comments, one-line context, descriptor pins, validation and publication order are unchanged. Missing temporary paths are suppressed only after successful Rename; publication is retained when later work cleanup fails.

Public `RegeneratePatch` regressions use existing synthetic archive fixtures and a context whose Err callback observes the output temporary pathname. No production hook, asynchronous descriptor race or dependency was added. Exact BASE loses real temporary ENOENT/ENOTEMPTY and work EACCES; final code exposes their original OS errors. Healthy generation and cancellation controls preserve literal patch bytes, output modes and primary unwrap identity. All six cases pass, including successful Rename followed by EACCES with published bytes/0644 unchanged; restoring owned permissions and retrying succeeds. The permission cases skip explicitly under UID 0; both executed on this Linux/aarch64 UID 1000 host.

Close failure paths (VERSION, input and three early temporary branches), combined multiple-cleanup failures, and post-publication temporary-removal/directory-durability failures were not fault-executed. Their ordering is supported by source review, not a runtime claim. Existing rejection and materialization/deterministic-generation tests were retained without edits. The public success case proves expected missing-path suppression after Rename. The initial work-failure test proves `*os.PathError` discoverability through `errors.As` and a primary plus cleanup remains a two-child primary-first join. A five-line review refinement additionally checks that a sole cleanup error has the direct `*os.PathError` type. The production source is unchanged. Root's final checks and reviewer follow-up cover this strengthened test; the worker command table below covers the initial test freeze.

Frozen SHA-256:

| File | SHA-256 |
| --- | --- |
| BASE patch_regenerate.go | fa5ca13df47c5a07f7976bb6bfd694c175274669243e16d27517e4d96749441a |
| Final patch_regenerate.go | 68135e1865312126898ec578613c1717f9524400e3d8c79d6a747b0a0d99b174 |
| Initial patch_cleanup_test.go | b907fb02f97c664c48a704866b430ff96c4221cbad0149d3f64415f2cb42f9f0 |
| Final patch_cleanup_test.go | 290bce62401e74dc42a3289bef80a1f3874ab9808d59ee3b55fb1fdfd9b8dfd4 |
| Unchanged patch_test.go | 02ae15958aa4d30513ea573cf6ad591b49d41719e73c4bad3838b6373fe7ac72 |
| Unchanged maintainer_output_patch_test.go | d2d0e49ca8eb598d2939a03008bb46187863ec6002793591463fd2afd86ff283 |

All commands ran with explicit repository `cd`, pinned stock Go 1.27.1 on PATH, `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOWORK=off GOENV=off`. Go tests always select `test_dep`. Logs are under `.flow/tmp/next-gate-101b14f882/`; elapsed time is Bash `time -p` real seconds. Command labels below resolve to these exact commands:

```bash
# focused
go -C tools/gomad3 test -tags test_dep -count=1 -json ./toolchain -run 'Test(Validate|Materialize|Regenerate|PatchCleanup|PinnedArchive|PinnedContext|Ensure|Extract|SourceArchive|SourceCleanup|CopyWithContext|CopyExactly)'
# consumers
go -C tools/gomad3 test -tags test_dep -count=1 -json ./cmd/gomadtool -run 'TestRun(Patch|MaintainerOutputPatch)'
# controls
go -C tools/gomad3 test -tags test_dep -count=1 -v ./toolchain -run '^TestPatchCleanupRegenerate$/^(healthy|cancel)$'
# regression
go -C tools/gomad3 test -tags test_dep -count=1 -v ./toolchain -run '^TestPatchCleanup'
# vet
go -C tools/gomad3 vet -tags test_dep ./toolchain
# errortype, run from tools/gomad3
env GOFLAGS=-tags=test_dep /tmp/fn109-lint-tools.ZdNe1t50/errortype -test=true ./toolchain
# scoped lint, run from tools/gomad3
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --build-tags test_dep --timeout 10m --fix=false --config=/Users/stephan/Workspace/skunkworks/gomad/temporal/.github/.golangci.yml ./toolchain
```

| Command / source | Log filename | Exit | Seconds | Log SHA-256 |
| --- | --- | --- | --- | --- |
| focused / BASE | worker-base-focused.log | 0 | 0.36 | 74d4b984e06bb045720f34c2a1916f1fe6ceb2976ffc9b658597c4d5e6cd120d |
| consumers / BASE | worker-base-consumers.log | 0 | 0.35 | 9b9310397cc759045d11e187757de7e1486852098aa5fe1f1735017245ba9c80 |
| controls / BASE plus final additive test | worker-base-controls-corrected.log | 0 | 0.24 | 7b1bf78ec669aa61a72bf0b046523aba39aac3d04bc660717ce3e0d617b3b997 |
| regression / BASE plus final additive test (RED) | worker-base-red-final.log | 1 | 0.26 | da0ea540eb2ad8e46849ad6239182a9717aef1946a69674dc22debc14228df3b |
| vet / BASE | worker-base-vet.log | 0 | 0.06 | 9b74bfad5449b06a75dcb7c79a9c6383cb265d0ce573650376f75106831cea59 |
| errortype / BASE | worker-base-errortype.log | 0 | 0.43 | 7b4c89a3c3f3ae38d380faf1b61109fc578933ea7f774c8dcfcc64c8f265abf6 |
| scoped lint / BASE | worker-base-scoped-lint-corrected.log | 1 | 0.17 | b4b6f98792ada561f6cd4fdb54ea6a964126089dabfc9b541a00cbc8cb3daecc |
| regression / final (GREEN) | worker-green.log | 0 | 0.52 | 4bb7cb075a8ef7ccb7eb478d27200c7005acb1e9c71fd88b3aac2362967686c9 |
| focused / final | worker-final-focused.log | 0 | 0.44 | bbad1515ee856b7e5d4a6986dfd76ee8dccfab8c57647f2a6179a8c872a783a4 |
| consumers / final | worker-final-consumers.log | 0 | 0.51 | ab6a46ca3ed1c846f681f8a3fd69a7021c830afed3e6be1a8f31f4c5bc43bb8c |
| vet / final | worker-final-vet.log | 0 | 0.09 | d31b99da58e03a765a94a503ce0b07988a1844037370acb5112886a78d166818 |
| errortype / final | worker-final-errortype.log | 0 | 0.44 | fd4b9b3a8adfe3bb53ca2f6f81c5a6e253a410eb348a0b7ce3c7459786cb4d68 |
| scoped lint / final | worker-final-scoped-lint.log | 1 | 0.89 | cb4ca7b019f4fadf8100caab7b79b6fa4799a79d9ba67f6ec5acc9a49390058c |

Unfiltered configured scoped lint drops exactly seven findings: 23 (22 errcheck, 1 forbidigo) becomes 16 (15 errcheck, 1 forbidigo). Residual finding blocks, including source lines and carets, are byte-identical after omitting only the seven owned BASE blocks and timing/summary trailers. No header-position normalization was needed; both residual files SHA-256 `74eff6f7da7f0572f15bb973c8caf743f83604e62495656c6ee31b895186b5f5`, `diff -u` exit 0. Root owns the final full-lint comparison against its 273-finding original baseline and the architecture, validation and fast-lint commands; see its conductor checks. Direct errortype above reached the analyzer independently of the still-red scoped/full golangci stage.

Both BASE and final focused selections skip exactly `TestRegenerateMatchesCheckedPatchForPinnedArchive` and `TestPinnedContextRepresentationsMaterializeIdenticalSource`: `pinned Go source archive go1.27.1.src.tar.gz is not cached`. Their unchanged source SHA is listed above. CLI patch consumers have no skips. This stock Linux/aarch64 execution does not qualify Darwin or linux/amd64. Native Darwin full-host/builder/full/default/functional/affected-consumer, matched-first-baseline, bounded 10/100, formal and predecessor gates remain open wherever unproved. Fn-128 remains deferred and native Linux unverified.

Early retained attempts are diagnostic only: scoped lint from the root incorrectly named a nested-module package (exit 7, 0.07s, worker-base-scoped-lint.log); corrected nested cwd produced the valid baseline. An initial controls regexp admitted `work-after-cancel` (exit 1, 0.44s, worker-base-controls.log); anchored per-subtest selection produced the valid controls. Before production edits, the test's remove-only PathError operation assumption was corrected to require concrete PathError/path plus the literal OS cause, because actual RemoveAll EACCES is `openfdat`; final exact-BASE RED then reran. No test expectation was weakened to permit the lost cleanup error.

Requested implementer tier: `gpt-6.1-sol` at high; retained dispatch tier `session (jev-unavailable(no_key))`. Actual executing model metadata is not exposed, so no actual-model claim is made. Review is conductor-deferred source/evidence review; no formal implementation review or Flow done ran here. Existing unrelated `.turbo` files and conductor Flow writes were preserved. `git diff --check` passed at freeze.
