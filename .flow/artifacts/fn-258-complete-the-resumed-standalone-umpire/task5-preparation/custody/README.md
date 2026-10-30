# Verified source and dirty-index custody

Result: **PASS**. This preserves candidate039, candidate075 and the six source-only freezes before receiving snapshot construction. No original source ref, index, working file, tracker or standalone receiver was changed. Same-named refs and temporary HEAD selections exist only inside the isolated recovery bare repository.

## Recovery objects

- Exact eight-ref source bundle: `e5ba826718df8c270dc269e42c49dde0349d107396f47dc15537c534acb65170` (167,528,083 bytes).
- Captured shallow-boundary bytes: `ef7d07756cb29242403b96f2ed65951e9edd7f8e20293ed90cefb4add39740b7`.
- Candidate039 raw-file/index/patch archive: `2801b0647868bddf3aeb54cf1de03bce510e6423ede9ad7a737c7ad92b46662d` (1,372,414 bytes).
- Candidate075 raw-file/index/patch archive: `63792d98bc1ce4d63dc3f7b699cc61c0763e60dd2ffc1776145cda7c8bf9ab3d` (1,328,628 bytes).

All objects are under umbrella `.recovery/fn258-task5-custody/objects/<sha256>`. The [manifest](custody-manifest.json) records sizes, full CAS paths, raw file modes/hashes, index blob objects, archive links, source identities and before/after fingerprints. [Verification](custody-verification.json) binds the final manifest and scripts. `.recovery/fn258-task5-custody/manifest.json` points back to this canonical evidence.

## Exact preserved source refs

| Ref under `refs/heads/` | Commit |
| --- | --- |
| `agent/attempt-fencing-prototype` | `bac2d44038362121939b0904e1993e2e8e25f787` |
| `checklocks-matcher-data` | `a325f98d350dff75ff16c82a9b1ab29dd9bda83b` |
| `w-1d-core` | `243f9dd525b16b7ab15b7f1fa5d486e0f7ce822a` |
| `w-1d-shared` | `86e4ee2505d9c3f379d820fb1d5bb1a658629287` |
| `w-perf` | `f4fba65eb634f11f547679279597bf48a69fa797` |
| `w-perf-dev` | `f4fba65eb634f11f547679279597bf48a69fa797` |
| `work/fn254-source-assembly` | `7cf05d1c5336b89c17c4fb13f6fced57db1ced14` |
| `work/fn2553-prep-fuzz` | `1df6f6b75d47a06642bd30b9bbe0d47b6d040ef8` |

The independently recovered source bundle has the exact same available 7,210-commit graph and passes strict Git integrity checks, without alternates to the original repository. The source is shallow: unavailable earlier parents are not recovered or claimed. Captured shallow bytes must be restored into a fresh isolated bare repository before unbundling.

## Dirty and index restoration

| Candidate | Raw dirty/untracked paths | Exact index records | Unmerged index stages |
| --- | ---: | ---: | ---: |
| 039 | 10 | 16798 | 3 |
| 075 | 45 | 16772 | 0 |

Candidate039 preserves the unresolved support.rs working bytes and all stage1/2/3 blobs; no conflict was resolved. Candidate075 preserves five modified files and forty untracked files. Restored byte/mode/symlink manifests, scoped Git status, complete index stages, and staged/unstaged binary patch bytes match exactly. The archived raw indexes remain byte-identical. Git verification uses a separate index copy because Git may refresh stat-cache bytes even during read-oriented commands; its semantic stage records are also checked afterward.

Only dirty/untracked paths are materialized during verification, reusing one independently recovered object store. Other committed working files remain reconstructible from the source bundle; two full source checkouts were not duplicated. Current ignored inventories are empty in both dirty checkouts. This is not retrospective proof about historical ignored or untracked bytes.

Full original tracked/untracked file byte/mode/symlink fingerprints, raw indexes, status inventories and HEADs match before/after; all eight original refs and source shallow metadata remain unchanged. The six source-only freezes retain exact refs/commits, zero active registered checkouts and explicit fn258.5 keeper actions. No feature implementation or accepted transfer is inferred. Historical ignored-state uncertainty remains explicit, without worktree retirement.

## Reverification and independent recovery

Run from the Temporal repository root:

```sh
python3 -B .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/custody/verify-custody.py
```

This hashes CAS objects, verifies source refs/available graph, restores each archive into a new isolated recovery directory, checks raw files/index stages/patches, and compares originals again. It creates no receiving refs. `capture-custody.py` refuses to overwrite an existing capture; a changed source state requires a separately admitted new custody version.

For recovery without the original object database: initialize a fresh bare repository, copy the captured shallow object to its `shallow` file, run `git -C <isolated-bare> bundle unbundle <source-bundle-object>`, and create only the eight same-named source refs at the exact manifest OIDs inside that new repository. Restore archive members without following symlinks; import the supplemental index blobs listed in each archive, preserve `index/raw-index`, and use a disposable index copy plus the captured source HEAD to inspect stages and patches. The verifier implements this recipe and records the actual isolated restoration paths. Do not blindly apply combined unmerged patches or stage the original checkout.

Initial refused verification attempts and the first receipt/capture script are retained in CAS. The final verifier corrects only a NUL-delimited index-record summary count; content/index/patch comparisons were not weakened. CAS content and original source inputs remain preserved.
