---
satisfies: [R1, R4, R5]
---
# fn-56-split-lean-api-generator-planning-from.1 Separate normalized Lean planning and deterministic naming

## Description
Keep plan construction as the sole descriptor-normalization interface while moving its deterministic name allocator into a focused private module. Make plan validation structural and record every decision that the renderer needs.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go`, `tools/umpire/cmd/umpire-gen-lean-api/lean_names.go`, `tools/umpire/cmd/umpire-gen-lean-api/lean_plan_test.go`
**Touches:** [tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go, tools/umpire/cmd/umpire-gen-lean-api/lean_names.go, tools/umpire/cmd/umpire-gen-lean-api/lean_plan_test.go]

### Approach
- Retain plan records, `buildLeanPlan`, message/enum/service planning, recursive ordering, reference normalization, and plan validation in `lean_plan.go`.
- Move `nameRequest`, scoped allocation, identifier normalization, reserved-name handling, numeric suffixes, and digest fallback intact to `lean_names.go`.
- Record the support namespace and complete module/import decisions in the validated plan so rendering needs no configuration lookup.
- Replace validation through formatted Lean text with structural type validation while preserving every current diagnostic and validation order.
- Reframe focused tests to assert normalized plan data, deterministic naming, and every R1 planner failure without calling rendering. Use complete expected errors plus multi-invalid inputs that pin first-failure precedence, and prove each failure returns no usable plan.
- Preserve all existing comments and keep every new declaration package-private.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go:14-229` — plan representation and construction interface
- `tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go:283-470` — package/declaration planning and name requests
- `tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go:499-898` — field/type planning and validation coupling
- `tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go:1035-1219` — allocator, identifier normalization, and collision fallback
- `tools/umpire/cmd/umpire-gen-lean-api/lean_plan_test.go:10-352` — collision, reference, recursion, module, and diagnostic coverage
- `tools/umpire/cmd/umpire-gen-lean-api/message_graph.go:10-185` — deterministic dependency and recursion input

**Optional** (reference as needed):
- `tools/umpire/cmd/umpire-gen-lean-api/model.go:84-100` — neutral projection interface

### Key context
- The plan is the deep private module; naming is internal implementation, not a caller seam.
- Do not change diagnostic strings/order, introduce templates, export internals, add dependencies, or rewrite generated fixtures.

## Acceptance
- [ ] R1 is satisfied by one validated private plan containing every naming, type, dependency, module, import, and support-namespace decision.
- [ ] Table-driven planner tests cover empty packages, duplicate identities/fields, unresolved parents, unknown named types, missing/mismatched oneofs, unsupported scalar kinds, support-name collisions, invalid type shapes, incomplete imports, namespace mismatch, dependency-order violations, and service-order mismatch with complete expected errors.
- [ ] Multi-invalid planner inputs pin the existing validation precedence, and every planning failure returns no usable plan or partial declaration set.
- [ ] Focused tests assert normalized plan data and malformed structural types without invoking source rendering.
- [ ] Reordered inputs, reserved identifiers, recursive components, and collision fallback retain deterministic planned names.
- [ ] Existing comments are preserved and no exported declaration, interface, package, dependency, generated-file edit, or drift-check machinery is added.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api` passes.

## Done summary
Separated deterministic Lean name allocation into a private sibling module, made plan validation structural, and expanded planner-only coverage for normalized data, diagnostics, and failure precedence without changing generated output.
## Evidence
- Commits: 762224bdb
- Tests: go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api, cd model && mise exec -- lake build Temporal.API, env CC=/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/clang CXX=/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/clang++ SDKROOT=/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep,integration --timeout 10m --fix=false --new-from-rev=b75931433 --config=.github/.golangci.yml ./tools/umpire/cmd/umpire-gen-lean-api/...
- PRs: