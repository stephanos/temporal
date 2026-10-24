---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.6 Port Apalache installation with exact archive, publication and concurrency controls

## Description
Port Apalache installation and its security/concurrency controls (R2, R3), after guest integration establishes shared xtask helpers. This isolates archive admission from the other helper ports.

**Size:** M
**Files:** `model/dev/xtask/{src/fetch_apalache*,tests/apalache*}`, `model/scripts/{fetch-apalache.py,tests/test_apalache.py}`, `model/umpire/backends/apalache/tests/controls.rs`, `Makefile`.
**Touches:** [model/dev/xtask/**, model/scripts/fetch-apalache.py, model/scripts/tests/test_apalache.py, model/umpire/backends/apalache/tests/controls.rs, Makefile]

### Approach

- Read `.flow/tmp/fn247-archive-feasibility/report.md` and its actual28-case evidence first. Python getmember chooses the LAST duplicate; do not invent duplicate refusal. Plain `tar -xOf` concatenates duplicate data, contiguous type7 is a Python file, BSD link metadata cannot be decided from the first mode byte, and gzip trailer differences occur only after deliberately replacing the archive pin in research.
- Retain the actual fixed SHA-pinned release through the old path at implementation time and prove exact-member metadata and old/new bytes plus appropriately tested GNU/BSD command behavior. Filter the known ASCII member before numeric listing, validate complete selected records and keep extraction stdout-only. No arbitrary caller-supplied archive grammar is introduced. Preserve every existing test counterpart. Distinguish BSD executable tests/source evidence from native macOS execution; if tooling is absent, report the remaining contract evidence and consider a disposable executable without a project prerequisite or new unavailable-hardware gate. Ambiguous listing or a failed actual-pin proof returns a concrete counterexample before any dependency amendment. Earlier process/guest tasks remain valid.
- Keep one Rust pin authority for version/archive hash/jar hash; update the backend's source-include pin consistency control to reference it without copying a second authority.
- Preserve download curl arguments, existing path/status interface, regular no-follow lock and exclusive concurrency, containment/component/symlink checks, partial/wrong installation refusal and private staging cleanup. Verify archive hash before inspection, require the exact named regular archive member, verify jar hash, and atomically publish only validated bytes. Do not replace member-type checks with unchecked tar extraction.
- Port every mocked-download/malicious-lock/archive/staging/concurrency test to Rust. Concurrent installers must still perform one publication and return the same admitted path. Switch only the installer Makefile caller, then remove its Python source/control.

### Investigation targets

**Required**:
- `model/scripts/fetch-apalache.py:18` — containment, lock and publication.
- `model/scripts/fetch-apalache.py:52` — download/archive/member/hash contract.
- `model/scripts/tests/test_apalache.py:41` — complete old failure controls.
- `model/umpire/backends/apalache/tests/controls.rs:248` — pin-consistency caller.
- `Makefile:1277` — required backend preflight.

### Quick commands

```bash
cd model && mise exec -- cargo test -p xtask --test apalache_install --locked
cd model && mise exec -- cargo test -p umpire-apalache --test controls --locked
```

Use controlled download/archive fixtures for installer tests; real required Apalache execution belongs to the final gate.

## Acceptance
- [ ] Actual pinned-release old/new comparison proves exact-name/regular-member admission and stdout-only extraction, with complete selected-record checks; archive/jar hashes and version remain under one Rust pin authority. Duplicate refusal is not invented from the research fixtures.
- [ ] Every old installer case has a Rust counterpart; wrong/partial/symlink/nonregular-lock/hash/member/cleanup controls fail as before and concurrent installers publish once.
- [ ] Recorded deterministic installer diagnostics/statuses compare equal; dynamic curl/path output is validated by the parent contract.
- [ ] Makefile and Apalache pin-consistency controls use Rust; installer Python/control removed only after parity proof.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
