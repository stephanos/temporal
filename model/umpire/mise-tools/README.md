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

This is an UNVALIDATED source-preparation candidate. Its changed config/data paths
and admission inputs have not been exercised. Existing research proves actual
TLC and Apalache mise feasibility with different resolver paths, not this provider.
The source controls cover command selection, failed installs, scoped environment,
data symlink refusal and runtime corrupt-JAR refusal. Before acceptance, run those
controls and renew real clean installs/admission, bad checksum, partial trees,
concurrent installs, interruption/cleanup and archive path-safety controls against
these inputs. Do not infer hostile-archive safety from the pinned happy path.

No lockfile is adopted here. The retired installer was the last tar/flate2 consumer; their direct dependencies
are removed. Both workspace lockfiles prune their now-unreachable package closure; the admitted
validation lane must confirm locked Cargo admission.
