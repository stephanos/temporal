- [x] Analysis and compatibility review use the preparation owner's inspection operation; neither creates a preparation root nor calls `PrepareTargetBuildAdapters` directly.
- [x] Workspace cleanup is explicit and its failure is surfaced; a caller-supplied preparation root for review is still rejected.
- [x] Closure inspection is shown not to compile or execute; linked inspection compiles without launching the target.
- [x] Unsupported closure, malformed linked records and invalid sums keep their classifications (`UnsupportedCapabilityError`, invalid capability review) and CLI exit statuses.
- [x] `gomad analyze` text/JSON output and `gomadtool compatibility-pack` behaviour are byte-identical for fixed fixtures; `make validate-compatibility` passes.

