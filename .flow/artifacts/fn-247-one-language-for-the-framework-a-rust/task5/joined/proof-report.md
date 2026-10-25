# Joined guest reproduction proof

Stock and Temporal guests built from joined source `523d131d2cf992b04a43b4d595ea0f1a434f44b6` agree byte for byte across two clean detached source clones with different absolute paths and initially absent independent Cargo targets. Every command completed. The runner exited 0 and both actual `cmp` commands exited 0.

The proven Temporal output is `temporal-a.wasm` in this directory, SHA-256 `bc27af7d5b6b1ed8fa5c524015046f3a8b7c12d2d488e9ce7cb3df3ea9217872`. `new-guests.sha256` and `guest-sizes.txt` record both pairs' hashes and sizes. The old canonical asset remains untouched at `eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`; `main-asset-preserved.log` verifies it after all builds and checks. The migration to native schema 2 compiler identities accounts for the changed guest identity. No scientific expectation or source was edited for this proof.

Clone b's first guest operation was the actual shell `guest-freshness.sh` with no cached native tool or target. It built the native dispatcher, admitted installed pinned tools and Rust source/wasm readiness, built the Temporal guest, and compared it successfully against clone a's proven output. Check mode preserved that output's hash. The later cached clone a freshness check passed too. The old main asset produced the expected stale status 2 and remained unchanged. `status.log` records all observed statuses; each child log and `.rc` file remains beside it.

`source-a-tree.txt` and `different-source-b-tree.txt` record the complete committed Model trees. `clone-source-shas` records both exact commits; `clean-source-status.txt` is empty because both clones remained clean. `target-identities.txt` records separate targets. `closure-maps.sha256` records native lock maps. The proof shared downloaded registry source only, with independent compiled targets and no cache cleanup.

The platform was Linux aarch64. Tool logs record Rust 1.99.1 (`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1), Cargo, Go, protoc and mise identities. Native Darwin execution and cross-OS equality are unclaimed.

The conductor owns asset publication after its immutable review window, using `publication-commands.sh`. That file also gives precise shell freshness and independent `--check` commands against the published main asset. It has not run. By the conductor's updated staging decision, final task 3's single required complete Model pipeline owns the semantic Go host suites; this proof launches no duplicate host suite or full gate. Task 5 remains in_progress pending conductor publication/review/closure.

All new proof material remains in this unique ignored directory during the main review freeze. The conductor can copy this report, manifest, input hashes, command/status/tool logs and transcript into task 5's durable artifact directory after releasing the freeze. Build outputs and both targets stay here for publication and independent checking.
