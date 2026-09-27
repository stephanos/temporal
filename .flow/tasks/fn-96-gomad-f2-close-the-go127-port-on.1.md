---
satisfies: [R1, R2, R3]
---
# fn-96-gomad-f2-close-the-go127-port-on.1 Run the darwin upgrade dossier and review the go1.26.4 to go1.27.1 boundary diff

## Description
Run `make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=<last go1.26.4 commit>`, review each boundary diff entry, rerun with GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256 set to the reviewed digest, and record the digest and reviewed entries.

## Acceptance
- dossier reports every non-root gate passed
- boundary diff empty or approved by recorded digest

## Done summary
Ran the darwin/arm64 upgrade dossier against the go1.26.4 baseline 75f4d101c5, reviewed every boundary-diff entry, and reran with the approved digest. Every non-root gate passes and the boundary diff is approved. The dossier stays `qualified=false` only because the DTrace clock audit needs root. The first run's clock-audit gate had failed for a different reason. The `testdata/clock_audit` fixture was lost in the gomadv3 rename, and the script looked for `clock_audit.d` under `scripts/` although it lives in the module root. Commit 68d36aadfe restores the fixture from history and fixes the probe path. `make clock-audit` now builds the probe, finds the `main.auditStart` marker, and stops with `gomad3 clock audit requires root DTrace privileges`.

- R1: `tools/gomad3/.toolchain/bin/go version` reports go1.27.1 darwin/arm64, and `make -C tools/gomad3 validate-toolchain` passes.
- R2: the approved dossier shows these gates passed: manifest-validation, toolchain-and-compiler, host-world-and-probes, builder, runtime, and disabled-upstream. host-clock-escape failed with exit 2 and the output `gomad3 clock audit requires root DTrace privileges`. `boundary_changes_approved=true`. The retained gomad3-core corpus shows 5/5 supported with exact replay, expectations met, and report sha256:0b4f0de9954463d28c7698404a6d5cd8468d872f75ea9f9df61d49cf910298f5.
- Approved boundary diff digest (value for the `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` repository variable): `sha256:86f18fc8cda31fe234d345f70384e8d5ae94e9cbb883beb5f8399e73f6d4798f`. The diff goes from go1.26.4-v2 to go1.27.1-v1, with 0 added, 0 removed and 131 changed.
- Reviewed entries:
  - `manifest`: `go_version` go1.26.4 to go1.27.1. `manifest_version` is excluded from the metadata comparison but moves go1.26.4-v2 to go1.27.1-v1. `hook_policies`, `reviewed_candidates` and `platforms` are unchanged. The `os.Pipe` linux/amd64 `platform_overrides` block only moved within its entry, which canonical JSON ignores.
  - `os.(*File).Chdir` (os/file_posix.go): `declaration_sha256` changed from 867cdc5d… to eac53648…. Upstream added a `testlog.Logger()` block that logs `Getwd()` after a successful `Fchdir`. The entry is modeled, so the hook replaces the body and the testlog record never runs under the model.
  - `net.(*Resolver).LookupSRV` (net/lookup.go): `declaration_sha256` changed from 815db3a4… to 041fa9c5…. The only change is a doc comment on the returned cname. The body is identical.
  - 128 other entries changed only `package_sha256`:
    - `os`: 59 entries, f3919bd2… to 20a14239…
    - `net`: 68 entries, 7895b7d6… to e6786904…
    - `os/signal.Stop`: d2e88503… to d2c250b5…
  - `os/signal.Stop`'s own body hashes identically in both versions.
  - The upstream package changes that account for the new `package_sha256` values are all outside the intercepted declarations:
    - `dirFS.ReadLink` error path
    - `ReadFile` doc comment
    - darwin `readdir` EBADF skip
    - root_*.go
    - `signal.Notify` lazy handler
    - `signalError.Is`
  - `os/user.Current` is unchanged, since the package is byte-identical upstream.
- Dossier JSON copies are in the scratchpad: `upgrade-dossier-darwin-arm64-go1.27.1.json` (approved) and `upgrade-dossier-run1-unapproved.json`. The core-qualification artifacts (121 MB) were deleted.

stage: impl-review - ran [codex fan-out rid 0863b36c157b43709729b6c70178da1a, 3/3 draws SHIP] SHIP
## Evidence
- Commits: 68d36aadfe6481eec0532d722991c52e13228c3d
- Tests: baseline: none (spec defines no Quick commands), make -C tools/gomad3 validate-toolchain (rc=0), tools/gomad3/.toolchain/bin/go version -> go1.27.1 darwin/arm64, make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=75f4d101c5 (dossier published; boundary diff sha256:86f18fc8cda31fe234d345f70384e8d5ae94e9cbb883beb5f8399e73f6d4798f unapproved; host-clock-escape failed: testdata/clock_audit missing), make -C tools/gomad3 clock-audit after fix (rc=2: 'gomad3 clock audit requires root DTrace privileges'), make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=75f4d101c5 GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256=sha256:86f18fc8cda31fe234d345f70384e8d5ae94e9cbb883beb5f8399e73f6d4798f (boundary_changes_approved=true; 6/6 non-root gates passed; host-clock-escape failed on root only; gomad3-core 5/5 supported, replay exact; qualified=false)
- PRs: