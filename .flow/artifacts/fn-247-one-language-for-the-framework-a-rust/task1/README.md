# Shared process extraction evidence

Admission was recorded before source edits at `cf339e835809cca4ce3cc06a9dda5d66948e02eb`.
The fn-237.3 implementation, primary-review and closure commits are ancestors (see
`admission.json`). Its guest SHA-256 is
`eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`.
`baseline-artifacts.json` records actual bytes/hashes for 752 guest, golden and
compatibility files, including all 15 immutable originals and independent authority.
The original verifier passed before the extraction. No artifact was refreshed.

`corpus.rs` is an observed invocation corpus, compiled before the move against
`umpire_tla::process`, then after it against both that compatibility path and the new
direct owner. The three stdout and stderr pairs compare byte-for-byte. The bounded
read refusal checks its exact dynamic scratch path before printing the fixed record;
no guest, golden or hash bytes are normalized. `run-corpus.py` records the invocation
recipe; its build metadata and executables remain in task-owned `.flow/tmp/fn2471/`.

`control-inventory.json` captures the pre-edit Python tests and Rust Python-child
sites. `control-owner-mapping.json` maps each of the 48 retained controls one-to-one:
eight Rust controls moved unchanged; 40 Python controls remain pending their assigned
Rust-port task. This checkpoint claims no completed Rust supervisor or Python removal.
`verify-extraction.py` verifies both moved sources byte-for-byte against the baseline,
all recorded artifact hashes, and the constrained local-only Cargo graph delta.
`package-features.json` and before/after package inventories show no existing package,
version, checksum or feature changed, including guest dependencies.

Baseline and post-extraction focused backend commands passed (159 tests each), as did
the retained Python discovery suite (6 tests each). `suite-comparison.json` checks the
same passing test identities after the process module moves to its own test binary.
An initial baseline launch lost its host result handle; it was inconclusive, and the
focused suite was repeated once to capture its exit status. Only the observed exit-0
run is used as baseline. Format, scoped Clippy with warnings denied and scoped docs
with warnings denied passed. Raw logs are retained here. No full Model gate or live
server run belongs to this mechanical extraction checkpoint; integration remains with
the spec's final task. Linux descendant controls ran on Linux; native macOS execution
is not claimed. The Python suite also exercised its simulated Darwin policy controls.
