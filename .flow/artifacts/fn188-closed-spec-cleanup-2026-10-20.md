# Closed-spec cleanup — 2026-10-20

Owner instruction: delete a closed Flow spec's worktrees, branches and evidence.
AGENTS.md records the policy. Flow spec/task records and Git history remain;
this cleanup neither rewrites history nor changes remotes.

## Executed cleanup

- Deleted 162 exact local closed-owner branch refs after checking current Flow
  states and unchanged ref OIDs. The audit found 49 done and 10 open specs.
- Removed the clean, idle fn220 and harness-retention (fn225) worktrees and both
  fn189 comparison worktrees. Current integration moved off the closed fn189
  branch to `gomad-milestones`.
- Removed 95 closed-owned tracked evidence roots, containing 34,823 tracked
  files, plus their untracked remnants. Removed ten closed external evidence
  roots and 50 exact closed-owned children in mixed directories. The mixed
  salvage/pruning indexes now contain only open fn185 entries.
- Free disk increased from approximately 21 GiB to 87 GiB. External evidence was
  deleted without a backup; tracked content remains retrievable from Git history.

Exact ownership plans and branch deletion results are under the still-open
fn188 evidence root: `closed-branch-cleanup-plan.json`,
`closed-branch-cleanup-result.json` and `closed-evidence-cleanup-plan.json` in
`/home/agent/wt/evidence/fn188/`. Ambiguous/mixed branches were not treated as
closed solely from generic names; the plans distinguish them from closed refs.

## Live inputs transferred, not archived

- The two qualification lint executables and a rebound checksum manifest now
  belong to open fn219 at `evidence/fn219/qualification-inputs`. Bytes, executable
  modes, versions and hashes were verified before deleting the closed fn187 copies.
- Full/slim measurement source and tests now belong to
  `.flow/artifacts/fn185-performance-inputs/`; isolation measurement source and
  its four consumed binaries belong to `evidence/fn185/r6/isolation`. All binary
  hashes verify; the transferred Python suite passes 22 tests after deletion of
  the old directories. Current fn185 Flow references point to these inputs.
- `fn185-input-release-baseline` holds source commit
  `51604889c9a9062f981cd02aa3daf04ca55fd940`. Its old absolute-path admission
  was not copied or rewritten. Future measurement needs a genuine baseline
  build/admission under fn185-owned paths.
- `.flow/artifacts/wasm-qualification-driver` is live operational source owned
  by open fn219/fn185, not closed-spec evidence. Its stage contracts are retained.

The current docs and open specs no longer require deleted closed-owner reports
as acceptance inputs. fn223's older retain-every-artifact rule is superseded;
its native deletion prerequisite and identity/requalification requirements remain.

## Verification and next execution

`flowctl validate --all` reports valid structure. Working/index diff checks pass;
only documentation under `tools/gomad` changed. No production Go/Rust, model,
protocol, seed, repetition, budget or qualification oracle changed.

The fn188 runtime reference at source `03542bfd6f` completed before cleanup:
exit 0, outer wall 11.144 s, helper peak RSS 259,375,104 bytes, complete
676-event order log and authenticated success artifact. The other three
references were held; no guest was killed or restarted. Its open-spec evidence
remains in `evidence/fn188/reference-preflight-03542bfd6f`.

The previously prepared qualification freeze is obsolete after tracked cleanup.
Prepare a new clean source manifest and binding before resuming references and
the unchanged 96-stage, at-most-3600-second qualification. No qualification pass,
full stress pass or full sweep completion is claimed by this cleanup.
