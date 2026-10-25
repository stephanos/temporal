# Joined native Apalache installation proof

At frozen joined source `523d131d2cf992b04a43b4d595ea0f1a434f44b6`, the
actual shared `model/scripts/xtask.sh` bootstrap built/resolved the native tool
with exit 0. The tool fetched the official v0.62.4 release from GitHub over
HTTPS into a new disposable installation root, admitted its pinned archive hash,
read the exact regular jar member using tar/flate2, validated the jar hash and
published it with exit 0. The network progress log records the 183.1 MiB release
transfer. This was not a fixture or cached-archive substitution. Successful
production admission is the archive/member validation observation; staging,
including the downloaded archive, was removed by the installer as designed.

The admitted jar remains available at:
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-real-install-523d131/model/build/apalache/apalache-0.62.4/lib/apalache.jar`

It is a regular 197019382-byte file with SHA-256
`e5223915507b65cc5f1638bd037ede60f8a4fc1644aff46355779f9b05df9262`.
The archive pin remains
`b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4`.
The single pin source and successful production implementation are bound in
`source-binding.json`; no pin/archive/download overrides were supplied.

Running the same native command again with `PATH=/nonexistent-fn2476` returned
the same absolute jar path, exit 0 and empty stderr. Since curl cannot be resolved
in that PATH, this proves successful cached reuse without a second download.
Only the persistent `.install.lock` and final installation remain in the cache;
there is no leftover private staging directory.

`real-install-proof.json` verifies regular installation components and jar bytes,
including byte-for-byte equality and SHA-256 equality against the existing main
`model/build/apalache/apalache-0.62.4/lib/apalache.jar`. That existing main cache
was neither removed nor replaced. Backend `Apalache::discover(repo_root)` admits
the jar under that main path, with no external-jar environment override. The
final required-backend pipeline will therefore consume the exact validated
content from the existing main cache, not the disposable installation pathname.
No backend run or final Model gate is claimed by this installation proof.

Before metadata reuse, all three Cargo files (workspace manifest, lockfile and
xtask manifest) were proven byte-identical to lane `17bce75e1c`. The joined native
`guest-lock` command then processed the exact pre-lane metadata/lock, post-lane
metadata/lock and freshly collected joined metadata/lock, each with exit 0.
Complete post-lane and joined lock maps are equal. All six guest identities are
identical across all three maps. The task 6 dependency admission, feature
inventory and no-new-package guest-closure proof therefore apply unchanged to
the joined tool. Native maps, joined metadata, binding hashes, exact commands,
exit codes and artifact hashes are retained alongside this record.

A first host-shell attempt failed to locate the bootstrap after a startup cwd
reset (exit 127 before bootstrap execution). It was inconclusive and was
corrected with an explicit absolute script path in a BASH_ENV-disabled
orchestration shell. The actual bootstrap and all subsequent commands passed.
This proof changed only task 6 scratch/evidence files; it made no source, index,
Flow or canonical guest changes. No command remains running. The conductor owns
review, the final required-backend pipeline and completion.
