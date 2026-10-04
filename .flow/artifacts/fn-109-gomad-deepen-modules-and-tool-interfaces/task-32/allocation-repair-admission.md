# Task 32 allocation-provenance corrective admission

Root accepts the empty-map-literal finding after reading the final independent
report, checks, runner, probe and causal logs, then freshly running
`final-review-audit.py` to exit 0. The current source gate remains NEEDS_WORK.
Earlier make-origin fixes close their exact reproductions, but constructor-local
element initialization does not preserve every demonstrated reference alias.

## Causal bounds

BASE and HEAD remain `f931c9879e3017b346562667b9f6fbcc4db458ec` on `gomad`.
Current `effects.go` is SHA-256
`8a077fd8a0faea457d43930bc6defc98b8f0d524333ac1a8015a29a97934cdc2`;
`standard.go` is `efad9411c3876b452d4d2f4905a4ba1cb92b43a0428ea2f039c94b3e5d9fd88d`;
the 39-fixture regression file is
`4663ca0c0035d8f8a5d9097b52e47d5a5638309a03e49ebe9d1f874f77f6a0b6`.
The root audit verifies 1,042 protected tracked inputs against aggregate
`b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61`.

The read-only allocation scout used frozen overlays bound to Git BASE or the
current candidate. Its 24-case probe has 12 introduced failures across six
dirty/clean pairs. Stock Go invokes the dirty callback once and the clean
callback zero times. Both supported source sets retain empty PackageEdges.
Each introduced pair has a known Dirty-to-time.Now path or empty findings at
BASE and an unresolved callback on the candidate.

| Allocation and mutation | Cases admitted |
| --- | --- |
| `new([1]func())`, converted pointer, index assignment | Dirty and clean |
| `new(func())`, converted pointer, pointee replacement | Dirty and clean |
| `new(map[int]func())`, converted pointer, pointee replacement | Dirty and clean |
| `new([]func())`, converted pointer, pointee replacement | Dirty and clean |
| Empty map literal, converted named map, index insertion | Dirty and clean |
| Address-of empty array literal, converted pointer, index assignment | Dirty and clean |

Nine other probe cases pass at both revisions. Three zero-variable cases fail
at both revisions and remain inherited limitations. The separate five-case
empty-container formatting probe also fails at both revisions despite literal
runtime callback counts of zero. These empty map/slice/zero-array element
controls constrain the fresh-allocation abstraction; they do not establish an
introduced regression or authorize a general formatting rewrite.

Root read the final scout note and audit, then freshly ran
`python3 .flow/tmp/task32-allocation-alias-scout-audit.py` to exit 0 before
continuing the writer. The note is SHA-256
`f4669841f34dfcf540f0de873599c831e738ee711d4d1a9ec2dbfb48276c35e3`;
the audit is `d7131c6a2dcfa74ca6f38fcb0846cafa02bd29aeed0e0ce6464dbfff16036d24`.
All four probe sessions and the scout audit were terminal. Scratch overlays
remain local. The writer must retain the admitted recipes and fresh causal
RED/GREEN proof under this task's owned artifacts so their later audit does not
depend on scratch files.

## Repair boundary

Continue the same task 32 writer only after scout commands are terminal.
Preserve the existing typed destination conversion and original 39-fixture
file prefix. Add the causal alias pairs and relevant allocation controls before
production changes, retaining source-bound RED and GREEN commands.

Use the existing shared `elements` cell and `fields["$pointee"]` abstraction at
fresh allocation origins. Distinguish known-empty contents from actual typed
zero elements. Preserve unknown contents from non-allocation producers and
recursive-type termination. Direct value conversions must not gain reference
storage semantics. A new storage representation or graph-key/memo/fixedpoint
refactor is unnecessary for the proven cases and remains outside this admission.

Only the original three source paths and new uniquely named task 32 proof
artifacts are writable. Existing tests, earlier reports, receipts, snapshots,
audits, pins, classifications, public APIs, configuration and other source stay
unchanged. Replay older audits through their exact saved-source views rather
than rewriting them. The original independent reviewer must review the frozen
corrective candidate before root stages or commits it.

Original R8/R18/R19, task 19/fn-105 D4, predecessors, task 21, first-baseline
identities and full/formal/both-native qualification remain open. This admission
does not supply formal SHIP, supported-native execution or a gate waiver.
