# Task15 proof retention tooling: prepared, collection unexecuted

`collect.py` and `verify.py` retain selected evidence, with no cleanup or restore operation. No private proof archive, binary copy or recovery CAS write has run during this preparation. Four safe controls exercise corrupt-member SHA/size/mode, missing-member, traversal-path and modified-input refusal using tiny synthetic files only. Temporary fixture cleanup is confined to those controls.

## Exact immutable selection and supplemental parent components

The original `proof-retention-preparation.json` remains byte-for-byte unchanged at SHA256 `cf8af5a758144fd6ece56e0849404d01387b38f5cbf9834b44693022b0ea3228`. `derived-preparation.json` preserves its exact first4799 selected rows and adds precisely two original parent bootstrap ELFs, for4801 members. Both current parent binaries matched the proof's recorded `native-binaries.json` hashes during preparation. The shared target as a whole remains protected; these selected components do not authorize changing it.

- Parent xtask: `9134d098c6dc5316cb446d70c07956b047b7985ca825413713b78b5e572ba8d7`.
- Parent guest-driver: `2bca7d08c3734750b41126e33a60c2e76fd66917001a2c82e9b5f72dc546c368`.

Already-selected pinned libstd is referenced as parent support without duplicating its archive member. This is evidence retention, not a hermetic OS image.

The private proof's current mutable driver alias (`b2cae3f9...`) and both exact recorded gate-finish driver copies (`cf6181b1...`) are all retained. The recovery manifest maps the original alias path to both retained gate-finish members and separately identifies the later alias member. Original result/receipt bytes and current binaries are never rewritten. The map describes a possible separately authorized recovery; this tooling implements no restoration.

## Root-only future execution

Root must review `tools-manifest.json`, its SHA256 and the derived input SHA256 in `handover.json` before execution. Collection has an explicit execution flag and rejects changed code/control/input seals, changed source bytes/modes/sizes, unsafe/duplicate names and changed original proof authority.

```
PYTHONDONTWRITEBYTECODE=1 python3 .flow/tmp/fn25415-validation/retention-tools/collect.py \
  --input .flow/tmp/fn25415-validation/retention-tools/derived-preparation.json \
  --expected-input-sha256 <reviewed-derived-input-SHA256> \
  --expected-tools-sha256 <reviewed-tools-manifest-SHA256> \
  --execute
```

Without `--execute`, no source collection or CAS mutation occurs. The collector streams exact regular files to a deterministic gzip/tar with explicit regular-member metadata, rechecks each original while reading and after collection, verifies archive contents, and publishes by new hard links. No existing object or manifest is overwritten. Unique staging and any unique published objects remain intact on failure; there is no production cleanup function.

The CAS is `/Users/stephan/Workspace/skunkworks/umpire/temporal/.recovery/fn25415-native-proof`, outside the private proof and scratch. It holds content-addressed original-preparation, derived-preparation and payload objects, plus a separately content-addressed recovery manifest. The tar contains exactly4801 selected file members. Unknown/unclassified files remain in place, and the parent shared target remains protected.

After collection, root must independently run the verifier using the returned manifest address and SHA:

```
PYTHONDONTWRITEBYTECODE=1 python3 .flow/tmp/fn25415-validation/retention-tools/verify.py \
  --manifest <CAS-manifests-SHA256.json> \
  --expected-manifest-sha256 <returned-manifest-SHA256> \
  --receipt .flow/tmp/fn25415-validation/retention-tools/independent-verification.json
```

Verification reads only sealed CAS bytes, never the originals. It checks object/manifest content addresses, every ordered member's regular-file type, safe name, mode, size and SHA, exact original-selection preservation, the two parent components against the archived original native-binaries record, pinned support and the gate-finish restore map. It never extracts an archive. A fresh receipt uses exclusive creation.

Neither accepted collection nor verification grants cleanup authority. Root retains source/review/closure, ownership-refresh, unknown-file classification and cache-cleanup decisions. No Flow, Git, runtime build/proof or cleanup action is part of this tooling preparation.
