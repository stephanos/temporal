# Foreign public API boundary for signature checking

The early review found a real false positive beyond the three Gomad leak
groups: upgrade.CorpusEvidence.Report uses json.RawMessage, whose Go 1.27.1
selected source aliases public jsontext.Value. IsValid accepts public
jsontext.Options, an intentionally public alias to an internal sealed options
interface. Callers use omitted options or public option constructors; recursively
auditing that publisher's sealing marker rejects a usable existing seam.
The conductor inspected pinned encoding/json/v2_stream.go:190,
jsontext/options.go:45, jsontext/value.go:96 and jsonopts/options.go:15–17.

Selected ownership rule: first validate name accessibility. An importable
foreign public named type or public alias is its publisher's API boundary;
do not recursively audit the publisher's underlying implementation/method graph
as a Gomad-owned export. Generic arguments remain traversed to catch private
identities introduced by Gomad at instantiation.

All Gomad-owned public graphs still undergo complete underlying/alias/field/
embedding/promoted-method/constraint traversal. Direct foreign-internal identities
still reject, as do Gomad aliases retaining inaccessible identity. This changes
neither foreign import permissions nor the required Gomad public repairs.
No general internal-package, standard-library or alias whitelist is introduced.

Require checker-positive and real external compile-positive controls for
RawMessage construction, IsValid and public options. Negative controls must
retain explicit foreign-internal exposure, own internal aliases and private
types in foreign generic arguments. The regression catches over-traversal of a
legitimate owned boundary, rather than remove/replace recorded RawMessage data.
