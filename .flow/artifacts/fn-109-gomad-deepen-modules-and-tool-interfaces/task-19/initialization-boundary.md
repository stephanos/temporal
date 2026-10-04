# Initialization boundary and corrected source evidence

The named pure roots and their imports must not introduce module/dependency
initialization effects. The checker follows the platform-selected import graph,
including blank and transitive imports, and checks every nonstandard package
variable initializer and every `init` declaration. Multiple `init` declarations
are inspected individually; the function-name index is not their inventory.
The same typed callback and recursive fixed-point analysis applies to these
initializers. Third-party packages are indexed for concrete callback provenance,
not treated as pure by package identity. Callable unrelated siblings remain
outside the named roots, but package initialization really does run their
initializers when their package is imported.

Process startup is a different boundary, already described by ARCHITECTURE.md's
runtime-startup audit marker and preservation of runtime.main/package startup.
Stock Go 1.27.1 runtime/proc.go calls nanotime and initializes runtime tasks
before executing module initialization tasks in dependency order. Its SHA256 is
`40d15ce858cf058cff92606001a29ee1689db88c121a2f0c283163a0d64a7375`.
The standard crypto/internal/fips140 init reads the startup GODEBUG configuration
(`fips140.go` SHA256
`3428ba28d5aa395cdd3a9b01897b38cc017392e02b4bdfacc13e3e38b466ad11`);
internal/godebug init registers runtime configuration callbacks (`godebug.go`
SHA256 `cd492e76ae63872ac4310fc09b2e097229588dbf114d14d713be3f027b0233c8`).
These observations do not establish standard-library startup as pure.

The initialization-only boundary requires actual go-list `Standard` identity
and a matching SHA256 inventory of every immediate `.go` file in each imported
standard package directory. Missing/changed identities reject. The recorded
Go 1.27.1 directory pins live in startup_sources.go. They were derived from the
union of Linux/arm64, Linux/amd64 and Darwin/arm64 dependency metadata of record,
World, capability policy, exploration, execution, campaign and target. Those
mixed-package dependency inventories include source siblings, but do not add
callable pure roots. Digest format is sorted `sha256  basename\n`, then SHA256
over that inventory, matching the existing memory-summary source contract.

The existing mixed-callback fixtures import net and crypto/rand even in their
pure controls. Their qualified/actual dependency closures are included in the
startup identity inventory too; their callable network/entropy operations remain
host-effect seeds. The first root-fixture run retained missing-startup identities
as RED, then the exact imported source identities were added. `C` is a compiler
pseudo-import, not a separately initialized Go package: it is skipped only in
an actual standard package with nonempty go-list CgoFiles, whose own startup
source identity is checked. A nonstandard unresolved `C` import still fails.

This boundary applies only while inventorying standard package startup. It does
not classify their functions as pure, does not suppress module/dependency
initializer calls, and does not set a startup exemption flag in the call
analyzer. For example, an imported dependency `init(){time.Now()}` remains a
host-clock finding even though time's startup identity is pinned. Lazy Local
timezone initialization is an operational call, not package startup:
time/zoneinfo.go's Location.get invokes localOnce.Do(initLocal). Consequently
time.Parse and unproven/Local ParseInLocation still reject; explicit UTC remains
the admitted repair.

The first actual-tree initializer run found bodyless sync/atomic.LoadUint32 and
StoreUint32 through x/mod's lazy-regexp -> sync.Once call chain. Their summaries
are exact memory-only symbols with the complete sync/atomic Go-source identity
`c99607f2227aa6e7770eb017ae4faecf4f3f30b09fa7229273e012e6b58e361d`.
No package-wide atomic exemption or pointer/callback storage exemption was added.

Retained RED: round2-red-dependency-initialization-2.log. Corrected controls:
called/blank/variable/repeated/transitive dependency initialization, third-party
replace-module initialization, effectful Once callbacks, pure local/Once and
standard startup, and deliberately changed standard startup source. These run
through the production checker on both qualified source sets; the altered-source
guard uses Linux/amd64 metadata. round2-green-checker-units-1.log exits 0 in
37.936s. The actual module passes both qualified source sets in
round2-actual-host-effects-3.log (exit 0, 23.542s).

round2-actual-host-effects-1.log collected zero tests because its selection used
an obsolete test name: this is inconclusive, not a passing gate. The corrected
selection first reported the atomic summary gap in
round2-actual-host-effects-2.log, then passed after that precise correction.
