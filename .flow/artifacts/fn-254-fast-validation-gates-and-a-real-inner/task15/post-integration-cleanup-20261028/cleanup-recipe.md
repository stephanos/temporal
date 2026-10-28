# Exact private cache cleanup: prepared, unexecuted

Root must read and authorize this recipe before running it. This preparation executed only four tiny fixture admission controls; no real Cargo clean, cache deletion or CAS verification was run by the worker. Existing sealed tooling inputs are unchanged.

The script has no target option. Its sole mutation target is the fixed literal `/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n/target`, guarded by exact parent/realpath/inode and no-symlink checks. It independently verifies both reviewed custody manifests, checks sealed full classification, and refreshes exact file membership, SHA/size/mode/inodes and private handles before mutation. It then runs only the four pinned Cargo clean argument arrays for current, hash-verified Cargo cache tags. There is no forced tag.

Before removing any untagged remainder, it repeats the exact subset inventory and hash/inode/mode checks. New directories are tolerated; new files or symlinks/specials refuse. Every remaining file must be unchanged and already retained or covered by admitted compiler provenance. Removal uses symlink-resistant `shutil.rmtree` on the guarded literal only, with no wildcard.

CAS metadata, copied framework/proof files outside the target and protected directory identities are checked before and after. The active parent target is checked by exact directory identity, not a false assertion that task12-owned contents stay idle or byte-stable. Both custody archives are reverified after mutation or partial failure. Atomic receipts preserve commands, process terminal statuses, output hashes, allocations, free space, protected checks and partial failures.

The explicit daemon acknowledgement admits only the recorded UID0 socat10/dockerd542/containerd557 visibility limitations. New unknown unreadable processes or readable private handles refuse. The final receipt keeps that limitation visible.

## Root-only execution recipe

```sh
PYTHONDONTWRITEBYTECODE=1 python3 .flow/tmp/fn25415-validation/cache-cleanup-preparation/cleanup.py \
  --expected-primary-sha256 68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610 \
  --expected-supplemental-sha256 e22efa29f106d7a2f8bf96374b080ee8d5c67b73b1dc4adc864daf7b9554a06a \
  --expected-classification-sha256 78fa352ec05730c873597c1b928d80734e779884861dbc1a2915181997d6e9a5 \
  --expected-tools-sha256 8b11813eda5a3a5482ca065f8603e57fdaf43b946b3db40acc67345128550de2 \
  --expected-script-sha256 8ccd5dfc80ace0263379ba09da2fa9584d10e61fdf0aedff0c9fd08040a5860f \
  --accept-recorded-daemon-visibility-limits \
  --receipt .flow/tmp/fn25415-validation/cache-cleanup-preparation/cleanup-execution.json \
  --execute
```

Use a new receipt filename if that exact receipt already exists; exclusive creation preserves previous evidence. Without `--execute`, the script reports prepared-only. No additional permission or task/source/Flow mutation is embedded in this helper.
