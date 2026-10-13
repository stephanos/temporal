---
satisfies: [R1]
---
# fn-153-gomad-retire-canonical-json-and-private.2 Centralize staged and streamed whole-file publication

## Description
Implements R1 for this owner; see the parent Architecture and Delivery sections.

**Size:** L
**Files:** internal/hostfs replacement/staging and host-helper owner, all surviving private helper consumers, WASM qualification publication, and the Rust helper cache.
**Touches:** [tools/gomad3/internal/hostfs/**, tools/gomad3/artifact/**, tools/gomad3/captured/**, tools/gomad3/runner/portable_plan*.go, tools/gomad3/runner/internal/minimizer/**, tools/gomad3/runner/internal/campaign/**, tools/gomad3/backends/native/toolchain/**, tools/gomad3/backends/native/upgrade/adapterregen/**, tools/gomad3/backends/wasm/qualification/**, tools/gomad3/backends/wasm/wasmhost/Cargo.toml, tools/gomad3/backends/wasm/wasmhost/Cargo.lock, tools/gomad3/backends/wasm/wasmhost/src/cache.rs, tools/gomad3/go.mod, tools/gomad3/go.sum]

### Approach

- Consume fn152's landed hostfs seam without changing protected log/open/lock ownership. Implement one synced-and-closed staged-file handle with path accessor, PublishContext returning publication status plus error, and error-reporting cleanup.
- StageContext accepts a same-directory temp pattern, requested mode and bounded fill callback. The caller validates its staged path before publication. Existing Replace/ReplaceContext wrap this primitive.
- Keep primary and cleanup errors discoverable via errors.Is/As. Track rename success separately from directory open/sync/close errors and cancellation before publish.
- Behavior pin: replay existing replacement controls plus injected create/chmod/fill/write/sync/close/rename/directory/cleanup failures. The helper does not create new containment or locking promises.
- Owner amendment reconciliation: the post-fn152 tree has six surviving Go directory-sync implementations (artifact, portable plan, minimizer, campaign, native toolchain and adapter regeneration), not the amendment's stale count of five. Consolidate all six, the artifact/hostfs/captured link-count implementations and artifact's no-replace rename variants into internal/hostfs so only one implementation of each operation remains.
- Use `golang.org/x/sys/unix` for Linux `Renameat2` and Darwin `RenamexNp`; retain the existing unsupported-platform and Windows behavior. Route the owner-amended WASM qualification Result and receipt publication through the shared staging/publication owner without weakening immutable no-clobber paths or durability. General qualification report/soak migration remains owned by dependent task fn153.7.
- Replace the Rust cache's raw libc declarations and unsafe calls with a direct `rustix` dependency already present transitively in Cargo.lock. Preserve owner-only, no-follow, no-clobber and cleanup behavior.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/hostfs/replace.go:15`
- `tools/gomad3/internal/hostfs/replace_test.go:11`
- `tools/gomad3/qualification/qualification.go:224`
- `tools/gomad3/backends/wasm/qualification/result.go:192`
- `tools/gomad3/artifact/store.go:570`
- `tools/gomad3/runner/portable_plan.go:422`
- `tools/gomad3/runner/internal/minimizer/workspace.go:523`
- `tools/gomad3/runner/internal/campaign/files.go:40`
- `tools/gomad3/backends/native/toolchain/build.go:739`
- `tools/gomad3/backends/native/upgrade/adapterregen/transaction.go:356`
- `tools/gomad3/artifact/rename_noreplace.go:3`
- `tools/gomad3/artifact/link_count_unix.go:11`
- `tools/gomad3/captured/hardlink_unix.go:1`
- `tools/gomad3/backends/wasm/wasmhost/src/cache.rs:160`
- `tools/gomad3/internal/hostfs/open.go:23`

### Verification

Focused commands:

```sh
go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs ./artifact ./captured ./qualification ./backends/wasm/qualification ./runner ./runner/internal/minimizer ./runner/internal/campaign ./backends/native/toolchain ./backends/native/upgrade/adapterregen
go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck
GOOS=darwin GOARCH=arm64 go -C tools/gomad3 test -tags test_dep -run '^$' ./internal/hostfs ./artifact ./captured
GOOS=linux GOARCH=arm64 go -C tools/gomad3 test -tags test_dep -run '^$' ./internal/hostfs ./artifact ./captured
cargo test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml cache
```

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Byte replacement and streamed staging share one file-publication implementation; modes, same-directory atomic replacement and cancellation behavior pass.
- [ ] Failure before rename preserves the old destination and cleans staged state; after-rename failures report published=true without claiming durability.
- [ ] Fault controls cover primary/close/cleanup joins, cancellation, cleanup idempotence and caller-visible published status; existing guarded opens/locks retain their tests.
- [ ] Exactly one Go implementation remains for directory sync, link-count extraction and no-replace rename; every surviving consumer uses internal/hostfs and Darwin/Linux cross-builds pass.
- [ ] WASM qualification Result and receipt publication use the shared staged owner without weakening no-clobber publication, receipt durability, error identity or cleanup semantics; general qualification remains scoped to fn153.7.
- [ ] The Rust cache uses rustix rather than raw libc declarations while preserving owner-only, no-follow, no-clobber and cleanup behavior.

## Done summary
Centralized same-directory staged whole-file publication and consolidated directory sync, link-count, and no-replace rename ownership in hostfs. Migrated artifact, captured, Runner, native toolchain builder/patch/source, adapter regeneration, and WASM Result/receipt consumers while preserving publication disposition, error identity, cleanup, cancellation, validation, and durability ordering. WASM collision losers now durably remove their stage and flush the winner directory. The Rust cache uses rustix instead of raw libc calls. Supported changed-package, race, architecture, cross-platform compile, Rust cache, and repository validation gates pass; the literal aggregate command remains host-inconclusive where runner needs the absent patched toolchain and deterministic-I/O platform, and adapterregen fixtures lack staged inputs.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7a15ae608b7947b0321fa31cba1188393e32602e, b9060bb1a966a618f50f7041abb69af31557de1b, cf5b26ca1f7a4e177186750d3f485dfb52f8694b
- Tests: changed Go packages, including complete native toolchain and WASM qualification suites: PASS, focused publication cleanup/error-identity/failpoint controls: PASS, race: internal/hostfs, public hostfs, and WASM qualification: PASS, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck: PASS, Linux arm64, Darwin arm64, and Windows amd64 hostfs/artifact/captured compile checks: PASS, cargo test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml cache: PASS (9 tests), make -C tools/gomad3 validate: PASS, flowctl codex impl-review re-review: SHIP, both prior P1 findings fixed and no new findings, literal aggregate focused Go command: INCONCLUSIVE on linux/arm64 because runner requires the absent patched toolchain and deterministic-I/O host, while adapterregen fixtures lack staged version/Makefile inputs; constituent changed packages compile/test green
- PRs:
## Host helper consolidation amendment (2026-10-12)

Owner decision: consolidate the five `syncDirectory` copies, three link-count checks and `artifact/rename_noreplace_*` into `internal/hostfs` using `golang.org/x/sys/unix`; replace the Rust helper cache's raw `extern "C"` calls with `rustix`; route WASM qualification publication through the shared replace helper. Acceptance adds: one implementation of each helper remains; Darwin and Linux variants build; publication error identity and durability claims are unchanged.

## Revival trigger (2026-10-13)

Revived after fn153.1 completed with a SHIP implementation review and its integrated strict-decoder, architecture and validation gates green. The forced local claim is only needed because Flow chain admission requires the completed fn152 parent on an unpushed remote branch; pushing remains outside the user's requested scope.
