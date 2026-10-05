# Existing external-pack import-policy mismatch

Discovered in task42's unchanged-production literal BASE controls at admission commit `533aa074337bc1e21e52c306ddb5a19e93fdfdbe`. This is evidence and a scope handback, not a completed repair or executable host-access claim. The task42 correction preserves it.

The loader rejects `import:os/exec`, `import:os/signal` and `import:os/user`, but accepts otherwise valid exact rules containing `import:plugin` and `import:runtime/cgo`. A validated/selected matching rule can then grant the latter capabilities. The new characterization retains complete Decision fields and literal decision bytes for this pre-existing behavior. The original matching fixture, which contains neither grant, denies all five imports with `remain_unsupported` remediation. Those are different controls and must not be conflated.

Read-only Astra source research traced the real policy path:

- `internal/compatibilitypack/schema.go:40`: the shared `unadmittableCapabilities` list has only the first three imports. Loader validation and authoring fact validation reuse it.
- `target/capability_collection.go:108`: import collection does not unconditionally refuse the latter two imports.
- `target/internal/capabilitypolicy/policy.go:132`: both are classified as forbidden imports, but `packageFindings` at line97 consumes `selection.Evaluate`; an allowed decision suppresses the import finding.
- `internal/compatibilitypack/policy.go:198`: exact capability membership grants before the final denial/remediation path. The five-import list in remediation is not an admission ban.
- `README.md:1167` says external packs cannot admit any of these five imports. `SPEC.md:273` puts plugin/native code outside support absent a qualified adapter. No checked-in pack/request names either of the latter capabilities, and no documented decision explains this discrepancy.

Independent execution restrictions remain: `target/internal/build/context.go:100` forces `CGO_ENABLED=0`; `target/target.go:838` refuses cgo-enabled provenance, with subsequent plugin-build-mode/external-linking refusals; `toolchain/runtime/overlay/src/runtime/gomad.go:145` refuses activation with `iscgo` or `gomadExternal`. Other findings and preparation may reject a target. Neither this source inspection nor the pure matching-pack controls built/executed plugin or cgo, proved host escape, or established native qualification.

A separately admitted boundary-policy correction is warranted, with an explicit decision to reject these previously accepted pack/authoring inputs, exact loader/external-pack/authoring/evaluator regression expectations, validation precedence/wrapping preservation, check-only generator evidence and original qualification handback. The shared list is the grounded production owner; inspect all consumers before changing it. R17's separation/refactor scope and task42's exhaustive correction do not themselves authorize that tightening. Do not silently repair it in the two-line lint change or describe current policy as enforcing the README's five-import prohibition.
