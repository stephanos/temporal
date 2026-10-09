# Verified architecture digest progress

The two architecture source-digest writes are lint-clean while preserving their literal identities. Root accepts a separate source-progress checkpoint after the fresh [SOURCE-PROGRESS PASS](source-progress-review.md). Task19 remains in_progress and its original acceptance remains open.

Root read both complete production diffs and the additive fixture. Each production file changes one statement. The format, operands, enumeration, filtering, error paths, comments, pins and APIs stay unchanged. Root independently derived the two file hashes and full sorted framing, producing fixture digest `67ad170a9788e8b8d82ab27c22a54c6f98af43713537fa5775a11f1424d4bcd5`.

## Candidate and packet identities

Base commit is `21b30b4604788c9e2e54b6c33f3e47045725daf9` on `gomad`. Root verified all 1044 current source inputs, 13 actual tool inputs and nine final control inputs against their manifests.

| Artifact | SHA-256 |
| --- | --- |
| Final source manifest | `eb4136469addda8629a61bab47a4a95c015ca60f1547dc17b10edb048b93ecfd` |
| Final tool manifest | `73c86870edf2cdafb4bebf6fc17c6ca3c0ba258bfc8c4316eeafc63f18ad4c55` |
| Final control manifest | `2a808448e42918b74e0576a8020e1f45bf83109670ec13b6c19a74a2d36182a7` |
| Worker handover | `4be6d3c519a44ff58ffa8680946dc5ad14b5f683b03b4db54ffd324bcc1c5dd4` |
| Worker evidence | `4475988482c07b60ade98deee6b13d41418ac9802977820b0cfbf5d8e24ffb4b` |
| Hardened preservation script | `94bfff84047b9ea1a1062feafc967e218167d392e53e8273c6874d8043a75ad2` |
| Fresh independent review | `82f16c354f086092f29b636ed4427a54addc13b07065cda68d34f7e5ceb043ab` |

Root audited all 21 worker receipt hashes, 42 raw stream hashes and 63 referenced manifest hashes. Root reparsed each JSON test stream and matched its actual pass/fail/skip counts to the receipt. Before/after controls each retain 22 named passes; the complete architecture package retains 183; root architecture checks retain 14. Each suite has zero failures/skips. HostPackageVet retains 55 packages for each supported source set and the actual host. Source listings, check-only validation, formatting and standalone architecture errortype retain their successful receipts.

## Independent root reruns

After the worker confirmed every handle terminal and released the execution lane, root ran the following commands serially through the frozen `run.mjs` recorder. Every command exited0 with unchanged source/tool/control bindings and complete retained raw streams.

| Receipt | Actual result | Receipt SHA-256 |
| --- | --- | --- |
| [Root identity controls](root-digest-controls-receipt.json) | 22 named passes, seven top-level tests, zero failures/skips, 7.484 seconds | `5d5c2726f0ce43af5fc483d9f6c97eab72ae6b335ad8995cb2dcbbaf184e4b1a` |
| [Root preservation](root-digest-preservation-receipt.json) | Literal framing, exact production statements, 401 original inputs and both full lint comparisons, 1.409 seconds | `facd5d73183e4d004877033aa79661d6459dd591d1faf915959aa8f0b1f1f4e5` |
| [Root fast Make lint](root-digest-fast-lint-receipt.json) | 55 packages, zero reported issues with explicit base21b filter, errortype reached, 3.118 seconds | `0431c8b3a3218daf0d2812b14295356b97dab87d9a309b43e7791bbe325c9ebd` |

The scoped analyzer supplies actual RED with two errcheck findings and GREEN with zero. Behavioral preservation controls pass before and after because the repair preserves runtime behavior. The original-base Make lint remains exit2, falling from215 to213 findings; integrated errortype is unreached. The later-base comparison falls from24 to22 and retains only that filtered meaning. Root reran the exact residual-byte comparison and retained [baseline accounting](lint-baseline-accounting.md), then appended clarification through Flow to tasks9/40 without rewriting historical raw packets. Changed-code fast lint does not discharge original-base acceptance.

After staging, root required `git ls-files --error-unmatch` to find the additive fixture and replayed the complete hardened preservation proof. [Staged preservation receipt](root-digest-staged-preservation-receipt.json), SHA-256 `562595e5b7e62af2c3deaceba720c519a15bffa5a028055df862b7c95c33f47c`, is exit0 with unchanged source/tool/control inputs. This exercises the formerly failing index state. Root's four supplemental receipts remain separate from the unchanged 21-receipt worker packet. The final audit verifies all 25 receipts, 50 raw streams, 75 referenced manifests and 1044 current source inputs; every root artifact link resolves. Flow validation passes all 22 specs and 198 tasks, retaining the two historical uncovered-requirement warnings for fn104/fn107.

## Evidence corrections and retained limits

The early package/validation overlap remains disclosed in the worker packet. Root accepts only the serialized replacement as authoritative validation. The synchronous gate driver later failed an evidence assertion expecting empty clean-lint output. Its failed receipt and exact script preimage remain archived; the analyzer's actual `0 issues.\n` output is now checked.

Root then found that current-index enumeration made preservation proof fail once the additive fixture became tracked and could omit deleted original paths. The original worker now selects all 401 protected paths from the fixed baseline tree, verifies each current file and separately binds the additive fixture to the frozen source manifest. The index-dependent script/control version and prior receipts remain archived. The hardened proof passed under a new write-once receipt and root independently replayed it. No source changed during either evidence correction.

The writer and fresh reviewer requested `gpt-6.1-sol/high`, from the same GPT family. Task-aware selectors returned `Tier: session` with `jev-unavailable(no_key)`; actual executed-model telemetry is unavailable. The first scout dispatch preceded its required selector, as already disclosed in admission. Renewed original-inventory research and implementation/review selection followed the recorded selector procedure.

Task18 stays Todo with its dependency retained. Root and reviewer verified 13 of14 historical manifest entries; later Makefile changes prevent a claim of complete current predecessor qualification. Task19's full R8/R18/R19, first-baseline, affected-consumer, preservation and formal requirements stay open wherever unproved. The genuine task40 pipe-Close fault and prior upgrade publication ENOENT proofs remain unexecuted/unexplained. Native fn128/fn149 remain deferred/unverified. Actual execution uses stock Go1.27.1 linux/arm64 on aarch64; supported-source static checks supply no native test-host pass.

Root verified the two user-owned Turbo files retain their original hashes and excludes them from staging. No push, PR, CI, native revival or history rewrite is authorized.

stage: impl-review - skipped(policy: original-base integrated lint retains213 findings; fresh source-progress review is separate from formal SHIP)
stage: completion-review - skipped(policy: task19 and the milestone remain incomplete)
stage: plan-sync - skipped(config: planSync.enabled false; no task completed)
stage: tracker-sync - skipped(config: sync active false)
stage: qa - skipped(policy: developer architecture checker has no live-app flow; focused source controls retained)
