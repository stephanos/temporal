# Local scratch retirement

After committed closure eaa618da5e, the conductor retired exactly 230 regular,
untracked files from the audited LIVE/DUP/HOLD ledger. All task workers and root
gate/review processes were terminal; no fn231 worktree or branch remained.
Normal worktree prune completed. Unrelated and uncertain worktrees remain.

Recovery is lossless under `.git/retired-scratch/fn231-final/`: `audit.md` retains
the ownership/recovery ledger, `files.txt` the exact relative paths,
`files.sha256` every original content hash, and `evidence.tar.gz` their bytes.
Archive SHA256:
554a975b603c593cba7fe2f5653568ec9ab4d6ab9881313894d7c470bb9e2d09.
Before removal, both SHA256 checks and `tar --compare --gzip` succeeded.
Every selected file was verified absent afterward. Recover into an empty
scratch directory with `tar -xzf <absolute-archive-path> -C <recovery-directory>`;
do not extract over a live checkout.

Selected files occupied 67,866,624 allocated bytes; the compressed archive is
3,547,811 logical bytes (about 3.4 MiB). Thus this reclaimed roughly 61 MiB net,
not GiB; filesystem free space still rounds to 44 GiB. Root conductor logs and
green receipts, created after the audit snapshot, remain protected in scratch
and their actual receipts are also tracked beside this record.

Explicit exclusions retained in place: all three foreign fn155-task4-review
files beneath fn2311-macro-lane, both uncertain multi-spec/config snapshots,
the three external /tmp review receipt copies, the shared fn2174 live report,
future fn227/fn211/fn232/fn233 reports/plans and active fn2301-unwind caches.
The audit itself remains available in scratch and the recovery directory.
No whole-directory recursive removal, forced worktree removal or ref deletion
was used. Empty scratch directories may remain intentionally.

Post-removal hashes reconfirm the embedded guest
10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06
and shared report a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0.
Tracked code/goldens/history and foreign fn233 plans are untouched. End-of-run
tracker check had no output with exit 0 (bridge inactive); no external write.
