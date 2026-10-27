# Checker provisioning controls

`bash scripts/fetch-tlc.sh` and `cargo xtask fetch-apalache` remain explicit
provisioning entry points from the framework root. The root model bridge
`model/scripts/fetch-tlc.sh` remains supported. Successful provisioning prints
the admitted absolute JAR path; provider output goes to stderr. A provider or
admission failure returns status 1 through xtask. No execution path installs.

The default jar environment selects only TLC; Apalache requires the named
`apalache` environment. The shared provider removes inherited `MISE_*` selectors,
uses only these local configuration files and sets the framework directory as
the ceiling above this directory. Jar-specific data/cache/state/config overrides
apply only to provisioning and exact-version resolution. Java resolution keeps
its existing framework context and ordinary mise installation. No artifacts are
copied to another custom cache. Exact install directories under
`build/mise-jars/data/installs` are admitted; mise version aliases are refused.

| Existing control | Replacement owner and retained obligation |
| --- | --- |
| Official URL and download SHA | HTTP declarations; mise verifies TLC raw asset and Apalache archive. |
| Apalache selected-member extraction | mise extracts the pinned official package; the backend separately admits `lib/apalache.jar` by its own SHA. This changes extraction scope. |
| Runtime integrity, partial/corrupt install | Shared regular-file/path admission and in-process JAR SHA remain mandatory even after successful install. Partial trees are never executed. |
| Parent symlink or path escape | Provider admits config/data ancestors before invoking mise, and exact installed JAR components before resolution/use. |
| Custom installer lock and publication | Delegated to mise per-version installation coordination. No second lock/cache/publication layer. |
| Custom staging cleanup, interrupted download | Delegated to mise. Cleanup can be best effort; this does not promise the former custom staging cleanup-error behavior. |
| Java and checker version admission | Existing bounded backend runtime probes remain; no prerequisite presence/version preflight is added. |
| Default/required backend modes | Gate ownership and opt-in selection remain with the existing stage graph. |

The jar environment owns default and optional mise lockfiles and requires locked
resolution. This avoids HTTP remote-version metadata refresh while mise retains
per-version installation coordination. Without those locks, concurrent clean
installs in mise 2026.10.11 can race while clearing empty HTTP version metadata;
locked concurrent installs preserve a single installation and return the same
admitted path. The lockfiles pin the same universal JAR and archive URLs/checksums
for Linux and macOS, including Linux musl variants.

Focused controls exercise native command selection, failure statuses, scoped
configuration, regular-file and symlink admission, corrupted installed JARs,
partial trees, concurrent installs and cancellation cleanup. Real clean installs
of both official releases also pass native backend admission and Model execution.
Download checksums and archive path traversal are enforced by mise. A symlinked
archive JAR can be extracted by mise but is refused by native admission. The
provider does not promise the former selected-member-only extraction policy.

Mise owns download and extraction leftovers. The provider removes its own bounded
capture directory on ordinary success, failure, timeout and cancellation; it does
not remove shared installation data. The interrupted HTTP control verifies
cancellation, capture cleanup and a successful explicit retry. It does not turn
mise's cleanup into the former native staging cleanup-error contract. The retired
`.install.lock` is no longer a provider input; its custom FIFO/symlink lock tests
are obsolete, while data/config ancestor and installed-file admission remain.

The retired installer was the last tar/flate2 consumer. Their direct dependencies
and unreachable package closure are removed from both Cargo workspaces. Task .16
retains the exact release, fixture, failure and control-mapping evidence under
[task16](../../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task16/).
