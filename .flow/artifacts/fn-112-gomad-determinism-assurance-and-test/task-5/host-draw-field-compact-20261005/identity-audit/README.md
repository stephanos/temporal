# Independent diagnostic identity calculation

This task-local Node helper recalculates the choices fixture from the retained fixture at `1147416b2e` and the source/schema contracts. It never writes product files, invokes Go, alters the Darwin guard, or claims native execution. The companion capture helper writes receipts only beside itself and refuses to overwrite an existing receipt label.

From the repository root, run `node <this-directory>/capture.mjs baseline --source-ref=1147416b2e`. After the implementation owner freezes generation, run `node <this-directory>/capture.mjs final --source-ref=worktree`. `node <this-directory>/derive.mjs --source-ref=worktree --emit` emits candidate canonical JSON to stdout without a final newline. `--expected-source=<hex>` optionally requires a particular source fingerprint. Product publication belongs to the authorized implementation owner.

The calculation first reproduces six retained properties: canonical fixture bytes, trace SHA-256, tape SHA-256, failure signature, record hash, and portable-plan SHA-256. It verifies the current fake preparer's BuildKey and target bytes against the fixture, keeps its one-decision fake executor unchanged, checks that the plain fixture matches baseline, and requires the Darwin guard and complete byte assertion to remain unchanged. It refuses changes to the owner contracts used by the independent calculation.

The source fingerprint follows `internal/gomadtool/generation/protocol/protocol.go:594`. The execution implementation follows `choice/trace.go:100`. The one fake choice record follows `runner/runner_test.go:2122,2542`, and its tape follows `choice/tape.go:362,484` plus `choice/schema/choicewire.json` and the generated wire encoder. Failure and record projections follow `record/identity.go`; their domain hashes follow `record/record.go:95,100`. The portable-plan digest hashes its canonical bytes, as `runner/portable_plan.go:201` does.

Only seven JSON pointers may change: three choice implementation identities, the tape digest, failure signature, record hash, and portable-plan digest. Supplied BuildKey, platform metadata, trace bytes, schemas, counts, and every other fixture field remain fixed. A source-derived identity update does not authorize changing independently supplied compatibility pins or retained historic journals.

The report compares the baseline generated/source fingerprint and its expected implementation identity with the retained golden. This distinguishes pre-existing fixture staleness from the compact-field rename's additional identity change. Final candidate evidence must be captured after generation is stable. Native Darwin focused, host, and full qualification remain open regardless of a successful independent calculation.

## Baseline result

At `1147416b2e6465de7b631e1e4b695eba33ebf000`, the source fingerprint and generated choice constant both equal `d3a4a66b4be3547d165886de0e526b657450f0a3861bf5584a725eb77e93a068`. Combined with the retained fake BuildKey, that produces `sha256:567d9463e68a353a28ab9f137f15490af5b7a9312433c75f5dc28705e71e92a0`. The choices fixture instead retains `sha256:c5bd3f6abcf0ed320ef699d4b2120e217ddd49e96f3010eaed090bafb9b79ea4`. The golden was already stale before the rename. A final refresh will correct this baseline residual and bind the compact-field candidate; the rename did not cause the original mismatch.

`baseline-fixed-parser.json` records exit 0 from 2026-10-05T22:38:30.481Z to 22:38:31.715Z, with unchanged hashes for all 994 tracked Gomad product files. Its raw stdout records all six retained self-checks, the seven changed JSON pointers, and per-input bindings. `baseline.json` and its streams preserve the first exit-1 calculator run: its generated-rune parser mistook the byte literal `'}'` for the end of the Go array. The parser was corrected to delimit at the source line; no product file changed in either run. Later receipts bind their exact helper hash.

The final parser recognizes Go's named escapes, hexadecimal `\x`, Unicode `\u` and `\U`, and three-digit octal rune escapes. It still requires exactly 32 values in the byte range. The `baseline-complete-rune-parser` receipt supersedes earlier helper versions for subsequent candidate calculations; earlier successful and failed captures remain intact.

## Frozen candidate after publication

The implementation owner reported generation complete, the golden published, and final sources frozen before these captures. `post-publication-frozen.json` records exit 0 from 2026-10-05T22:56:13.008Z to 22:56:13.536Z. Its raw stdout proves all six retained self-checks, exactly the seven admitted JSON-pointer differences against the fixture at `1147416b2e`, and `current_golden_matches_candidate: true`. The plain fixture, fake BuildKey, eleven owner-contract files, native Darwin guard, and whole-byte assertion remain unchanged.

The recomputed source fingerprint equals the generated constant at `2acd63cceb126f37c179bf0dd47c188c60894bc63ddbd0deb06a7a056a1f025a`. With the retained fake BuildKey, the candidate choice implementation is `sha256:e8d4ef82d115bedf966b47bcdf8d7f601be4af7fc6ca5af4c4cfb04e48eb2054`.

`post-publication-emitted.json` records a fresh `--emit` run with that exact expected source fingerprint, exit 0 from 2026-10-05T22:56:27.019Z to 22:56:28.293Z. Its raw stdout is the 13,271-byte canonical fixture, with SHA-256 `1356bc8358d2d71a4586a1d4b2cd8107ee572960c859651da40c261b19335011`. A subsequent read-only `cmp` of this stdout against `tools/gomad3/runner/testdata/diagnostic-identity-choices.json` returned exit 0 with no output.

Both capture receipts bind helper SHA-256 `4c24da7606148ffbb033ea4914d35ce0c6c24bdff4838867797a4516da70bd12`, empty stderr, and the same before/after hash over 994 tracked Gomad product files, `d9c4f5fd806c07ac8bd135cda78e3b2406a90791920f1196b71522e77fc70870`. Neither capture changed a product source. These are independent source/schema and byte-comparison receipts. `TestDiagnosticsOffPreservesExistingCanonicalIdentities` was not executed on its native Darwin host, and native qualification remains open.
