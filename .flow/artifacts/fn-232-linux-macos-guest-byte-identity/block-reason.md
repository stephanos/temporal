Certification failed at source `8d429e97409b1537ecb347c359129bffafa63f78` on
2026-10-21. All four clean builds completed, and both guests were path-independent
within each OS. All eight Linux/macOS comparisons failed. Stock first differs at
zero-based offset 443 and Temporal at 635. See the retained
proof at `.flow/artifacts/fn-232-linux-macos-guest-byte-identity/proof.md`,
platform transcripts, manifests and `comparisons.json`.

R1 remains unmet. The spec requires a separately scoped reproducibility fix
before another certification attempt. No build recipe, dependency version or
Wasm artifact was changed to hide the mismatch. Keep fn-232 open.
