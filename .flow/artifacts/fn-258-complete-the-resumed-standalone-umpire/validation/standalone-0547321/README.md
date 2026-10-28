# Standalone required-TLC validation at 0547321

`make validate` completed at standalone `05473210b48c3db862f22051eb7154ce98258f98`, tree `46829511b6d582b40f3fa5095f42bdf980c7d466`, on Linux aarch64. Independent retained-receipt verification passes: all 18 ordered parent stages and all 17 ordered source-only child stages succeeded, with exclusive stage authority and no count errors. The standalone result is complete, child exit is zero, and all 1,195 source inputs match the pinned Git tree and both actual source copies byte-for-byte with executable modes unchanged. The parent embeds the exact child stage authority once, retaining its digest and original path.

The `validate` tier required TLC, used the dev profile, one Cargo build job and the required nextest profile with four test slots. Apalache was skipped. Each independent tests stage reports:

| Producer | Discovered | Executed | Ignored | Skipped |
| --- | ---: | ---: | ---: | ---: |
| nextest | 910 | 908 | 2 | 0 |
| Cargo doctests | 5 | 4 | 1 | 0 |
| stage authority | 915 | 912 | 3 | 0 |

Parent and child are separate executions; their counts are not combined with the inclusive standalone stage. Raw observations include 836 TLC attempts and completions per tests stage. Published stage TLC count fields remain null; the retained diagnostic event totals do not replace that authority or close work-count obligations. Go-stage test totals likewise remain unobserved in the stage counts. All six native standalone containment controls and both source-only setup commands passed.

The gate took 1,679.762 seconds against its 60-second soft target. The edited-checker loop took 24.162 seconds against its 10-second soft target. Both are explicit misses; no performance obligation is closed. Release workspace testing, paired consumer/local-edit evidence, optional Apalache execution and publication are separate claims. The original command's completed stage/timing log is retained; the conductor observed success, while no separate launcher exit-status file was supplied to this retention lane.

Recovery retains every result-named helper binary, generator, TLC jar, wasm guest and native parity payload, corresponding parent outputs, guest lock/compile identities, compiler wrappers and the six versioned tool executable identities. Every previously sealed artifact hash and directory inventory matches. The exact source snapshot and complete parent/child receipt directories—including claims, raw producer/TLC events and logs—are independently verified compressed recovery objects. Shared Rust/Java/Go tool installations are not claimed to be self-contained or eligible for cleanup merely because their executable identities were retained.

The [recovery manifest](recovery-manifest.json) maps 320 named records to 158 unique SHA-256 objects (294,884,237 bytes) under the umbrella `.recovery/fn258-standalone-proof/objects/` store, outside both build targets. [Archive manifests](archive-manifests.json) record every source and authority member. [Verification](verification.json) passed all 420 checks. Independent recovery-only verification passed all 158 objects and 7,979 archive members, including file modes, without accessing original caches. The [recovery-only verifier](verify-recovery.py) reads no original build cache:

```sh
python3 .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/validation/standalone-0547321/verify-recovery.py
```

Restore a named proof binary by copying its `recovery_object` to its recorded destination and applying its recorded mode. Restore the source or complete receipt hierarchy by extracting the corresponding content-addressed tar.gz object from `archive-manifests.json` into a fresh directory. Readable copies of inputs, tools, result, setup, ordered/individual stage receipts and logs are alongside this report.

Both proof roots remain protected. The ownership snapshot found no matching cwd, path argument, environment, descriptor or descriptor-lock owner; system PIDs 10, 542 and 557 were unreadable and remain unknown. The parent target occupies about 6.43 GB and the child target 6.66 GB. After conductor review/closure and explicit owner release, these reproducible target outputs can be considered for cleanup only after repeating the live ownership checks and verifying recovery again. Both target roots currently lack `CACHEDIR.TAG`; a later cargo-clean safety refusal must not be treated as deletion authorization. No cleanup was performed, and no source, worktree, shared cache, Flow status or Git ref changed.
