# Architecture fitness implementation decision

Task 19's existing Flow description and acceptance are the implementation plan.
Retained source, callback, inventory and public-signature scouts ground this
decision in the current integrated source, not historical package ownership.
The user requests autonomous recommendations and owns commits; no repeated
approval, staging or design commit is required. All tracking remains in Flow.
The unavailable writing-plans skill is replaced by the existing Flow task plan.

## Selected approach

Build a small private Go-owned checker beneath internal/gomadtool, using the
standard AST/types/build facilities and existing x/mod only. Root architecture
tests exercise the same checker against actual source and controlled fixtures.
Keep package discovery/ownership, pure-root effects and public signature
traversal distinct, with typed diagnostic categories and stable evidence paths.

Discovery uses complete ./... metadata for both qualified source sets, with
independent filesystem Go-source and nested-module inventory. Exact required
overlay/fixture/corpus exclusions must match; only .toolchain and .bin are
optional generated roots. Preserve test-only root ownership, platform selection,
all existing owner/edge rules, and fail included package-listing errors.

Effects start at all functions, methods, initializers and closures in named
pure files/modules, then follow typed transitive calls through module,
dependency and standard-library source. Mixed campaign and execution packages
remain file-scoped. Resolve concrete receivers, function/method values,
callbacks stored/captured at actual call sites, generic bindings and recursion
to a fixed point. Fail closed for unresolved effect-bearing bindings. Inspect
implicit fmt/JSON callbacks and nested value graphs; pure buffer IO, error
formatting, local allocation/mutation, locking and model time remain legal.
Permit ParseInLocation only with a proven explicit UTC location at these roots.

Public traversal covers exported functions/methods/fields/interfaces, aliases,
defined underlying types, embedded/promoted methods, containers and generic
arguments/constraints. Respect every internal path segment for an external
consumer. Private storage stays encapsulated unless publicly exposed; legal
public defined builtin graphs and reachable private same-package names remain
accepted. Check complete nested graphs rather than only outer declarations.

Early review additionally establishes publisher ownership: an importable
foreign public named type/alias is a public boundary, while its generic
arguments remain inspected. Recursing into Go's public RawMessage/Options
sealing implementation rejects a usable existing seam. Direct inaccessible
foreign identities and all Gomad-owned nested identities still reject; see
foreign-public-boundary.md and its required compile/checker controls.

Repair the three confirmed public leaks with detached Runner report values,
detached target compatibility evidence graphs and explicit projections, plus
pinimpact.Spec.PacksDirectory and the production authoring-root consumer.
Retain field/tag order, all data, nil/empty distinctions and pointer presence.
Keep default/explicit-directory loading delayed until after module validation.
Repair all three record timestamps with explicit UTC parsing, preserving original
strings, accepted grammar, full parse errors and precedence. Planned intentional
Go migrations are already inventoried; finalize exact public names after source.

The reachable pack-governance ReviewedAt parser needs the same explicit-UTC
repair while keeping its UTC/Z policy and pack bytes; see
pack-governance-timezone-scope.md. The World arbitrary-error callback conflict
uses the admitted detached terminal/reporting boundary and closed model-error
projection in world-terminal-design-decision.md. These corrective scopes are
now synchronized in the task description and spec API Contracts and retained
in the pre-edit migration inventory. Neither is a host-effect exemption.

## Alternatives and verification

Text/import scans are smaller but miss transitive and implicit effects, nested
public identities and skipped source. A whole-package standard-library allowlist
would accept effectful siblings. New x/tools/SSA dependencies add a dependency
without authorization; neither alternative meets the required boundary precision.

Use TDD: first observe real old-source architecture/public/effect RED, then
implement. Preservation tests compare independent literal values/bytes and
captured old behavior rather than compute expectations with the new projection.
Exercise negative fixtures through the actual checker with exact categories;
include positive memory formatting/JSON, recursion/callbacks, private storage,
legal internal access, platform files and optional generated-root absence.
Extend external-consumer compilation without dropping existing construction.

Baseline every Quick command before edits. Broad ./... vet is known to encounter
checkout-overlay metadata errors; retain actual baseline output before proposing
any precise host-inventory scoping correction. Do not silently ignore errors or
weaken a gate. Native qualification and prior task acceptance remain separate.
