# Original runtime-cost request binding

The thin immutable-Go invocation controller previously compared every native
request with the native preparation configuration. That configuration includes
campaign argv and environment normalization; the original direct `wasi.Run`
request has separate, ordered argv/environment. The generic binding is retained.
The runtime-cost pair now selects a separate closed binding for the original
runtime request and the actual native preparation normalization.

This path pins the unchanged 368-byte runtime source, empty compile arguments and
tags, seed 7, original argv and ordered environment, entropy, clock, model limits,
namespace and original execution bounds. It authenticates the same current module
and provenance through the existing run custody checks. It adds no Go host logic.

The positive corrected-request fixture fails against the historical controller
from `cda65407c6` with `native request differs from exact prepared configuration`
(exit 1): `/home/agent/measurement-original-runtime-binding-historical-red.log`.
The new fixture and eleven mutations pass with the explicit runtime binding;
existing generic-binding tests remain unchanged. Focused 11 tests and all 318
Python driver tests pass, exit 0, in
`measurement-original-runtime-binding-green.log` and
`measurement-original-runtime-binding-driver.log` under `/home/agent`.

Only `driver/reference_profile.py` and its test change executable/controller
source. The separately admitted ordinary Rust Release build remains source
`cda65407c6ab0d72d8c3061bca8b87b32def7b40`; this controller follow-up does not relabel
that binary. These fixtures and its actual build do not establish original
three-leg runtime acceptance or final frozen qualification.

The reference-helper build custody now also seals tracked Rust modules at the
package root, including `build_identity.rs`, alongside the existing `src` tree.
Actual untracked root modules are refused rather than added to the receipt.
The existing exact Git-byte comparison authenticates each included module.
A causal fixture first failed because the identity module was omitted; the fixed
fixture also refuses a changed or missing tracked root module and an extra
untracked root module, and accepts a historical Git tree without that module.
Focused helper-custody tests pass in
`measurement-reference-helper-root-source-custody.log`; all eleven reference
controller tests pass in `measurement-reference-helper-root-source-green.log`.
No previous helper build receipt is modified.

Independent actual-read review found that historical `derived-v2` changed one
original symlink literal from `./versioned/v15/index_template_v7.json` to the same
path without `./`. The previous verifier accepted its matching manifest and tree
without comparing them to the original archive. Its actual unsafe acceptance is
retained in `measurement-reference-v2-source-custody-red.log`; the old source and
runtime records remain unchanged. The three-leg v3 records establish raw behavior
matching, with an original-source custody gap, rather than accepted exact-original
source execution.

The verifier now reuses the original-ringpop closed archive/source inventories,
permits only the pinned invocation-glue addition, checks the immutable archive
revision, file/directory modes, and closed directory membership. Coherent changes
to both manifest and symlink/file, a mode change, and an extra empty directory fail.
The causal fixture failed before the repair, then passed; all 318 driver tests
passed in `measurement-reference-archive-binding-driver.log`. An actual read now
refuses old v2 and accepts fresh literal-preserving v3, recorded in
`measurement-reference-archive-binding-actual-read.log`.

The fresh derivation also independently compares all 9,690 archive payloads with
Git blobs and actual source bytes/modes, including both literal symlinks. The one
`160000` gitlink is explicitly unexpanded. The full proof is
`derived-v3-git-archive-source-custody.json` under the owned checkout runtime-v2
preparation root. A fresh Go harness build and ownership-pair rerun are still
required before exact-original runtime acceptance.

Fresh review identified a valid restrictive-umask case: the tar data filter leaves
directory modes to mkdir, so umask `077` produced `0700` directories while the
new verifier expected the safe archive mode. Extraction now explicitly applies
`member.mode & 0755` only to validated real directories contained in the private
source root, after the data filter. The source root remains private. File modes,
executable bits and symlink literals retain their existing checks. This is an
explicit safe tar-directory policy; Git does not store directory permission bits.
The actual restrictive-umask fixture failed before the fix. All 12 reference tests,
the full original-archive custody fixture under umask `077`, and all 319 driver
tests passed, exit 0. Logs under `/home/agent` are
`measurement-reference-archive-{umask-green,restrictive-actual,umask-driver}.log`.

Fresh v4 is independently clear for the scoped development runtime comparison.
The new Go build uses the exact original `1d695061b6` archive and only the pinned
invocation glue. The separate recipe-catalogue revision is `81660e8411`; the old
read report that used that catalogue revision under `original_revision` remains
unchanged. New reports name both lineages explicitly, along with Rust source
`cda65407c6` and controller `187792871a`.

Actual Go, helper and uncached native PIDs 846815, 846899 and 846490 exited zero
and were reaped. Each made 200,037 calls and 100,000 decoded select polls. All six
raw streams match: 20-byte stdout, empty stderr/network, 778-byte WASI transcript,
19,200,000-byte choices and 16-byte terminal. The WASI digest is
`f6b374b7322317fbb53747ff791d37d960a94f1bf4e15838438c88e005b7a9e7`.
The independent reviewer checked all 9,690 source payloads, 1,045 archive directory
modes, both literal symlinks, the unexpanded gitlink, 414 ordinary build inputs,
119 helper source inputs and 133 launch-input hashes. The repaired verifier
continues to refuse unchanged defective v2 evidence.

The precise acceptance, raw receipt hashes and remaining gates are recorded in
`fn232-original-runtime-development-proof.json`. This does not credit an original
test function, full Temporal execution, frozen qualification or a performance
claim. The separate full terminal module compiler endpoint passed and retained its
342,575,376-byte module; no guest ran in that preparation proof.
