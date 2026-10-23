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
