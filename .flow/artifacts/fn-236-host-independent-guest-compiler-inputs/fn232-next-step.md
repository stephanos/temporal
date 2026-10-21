# Interim source-pin gate, resolved

The committed source pin became available during validation. fn-232 subsequently
passed its fresh four-clean-checkout certification at
`3c314fb5b85427471a1525d6f190fe5d1fb9f3b2`; see
`../fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/proof.md`.
The earlier handoff below records the gate before that commit existed.

The compiler-input mismatch is resolved by the separately tracked fn-236 fix.
All twelve actual stock/Temporal comparisons passed across four native
Linux/macOS source snapshots with independent initially absent targets.
Both OSes passed the actual stock/control and embedded Temporal host suites;
the full required-backend model gate passed on macOS. See
`.flow/artifacts/fn-236-host-independent-guest-compiler-inputs/proof.md`.

fn-232 remains open for its final committed-checkout certification. Its clean
source-pin requirement needs a commit containing the reviewed fix and refreshed
asset. The owner's AGENTS.md instruction reserves commits for the owner unless
explicitly requested. No commit was created. After that commit exists, pin it in
a fresh four-clean-checkout run, retain the comparisons and hosts, and close
fn-232 only when its original acceptance criteria pass. The earlier failed
certification receipts remain unchanged as historical evidence.
