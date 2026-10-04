# Early effect-context source research

The read-only effect_context_soundness_scout found concrete false-negative paths
in the unfinished task-19 analyzer. All were forwarded to the sole source writer.
This changes the next action: require checker-driven context/error-provenance
regressions before relying on actual-tree GREEN. It is not a formal review,
executed fixture result, frozen-source acceptance or native qualification.

The scout inspected unchanged source through 2026-10-04T04:15:20Z:

```text
effects.go       cf1ee8a5aab3ddb852ad8bae6d05c019286b4778ee66b159aaf2fc3c9c6ab064
standard.go      74ccbae96596b195de7f109b8ef46eb3d030f5008e27271388edaee00227ee33
effects_test.go  cab702282bbef8470b0185e46c5ca35846109f38df2f1eb958e706f049ffbddb
```

Paths are beneath tools/gomad3/internal/gomadtool/architecture. The writer may
subsequently change them; these findings describe the inspected snapshot only.

## Context identity loses positions and captures

effects.go bindingKey (203) flattens value graphs into an unordered set of
types, functions and flags. call (175) uses that key for completed memo entries
and active recursion. Argument positions, field paths and closure captures
are not represented. joinValues (51) also deduplicates function values without
their captures.

Minimal control: an ordinary helper package outside the pure directories
defines Clean(), Dirty() calling time.Now(), and First(a,b func()) calling a().
A pure record root calls First(Clean,Dirty), then First(Dirty,Clean). The first
clean context can memoize away the second effectful one because both keys have
the same symbol set. Dirty has no separate pure-root check.

Independent controls must cover named fields swapped between Used/Unused,
active recursion Rotate(a,b,n) calling a() then Rotate(b,a,n-1), and separately
captured closures returned by Bind(f) and subsequently invoked. Factory keys
can differ while returned literal invocation keys collide. Capture-sensitive
unions also need coverage.

Use ordered argument slots, named field/element paths, receiver and captured
object identities, with cycle handling; use the same corrected context identity
for memoization and active recursion. The existing bindingDepth boolean does
separate suppressed binding-resolution calls from reporting calls, but does
not repair these collisions. Require reason-specific time.Now host-effect
findings through the actual checker, not merely fixture presence.

## Error results lose callback provenance

standard.go fmt.Errorf (87–90) marks its result builtin/known without retaining
wrapped operands. A helper Leaf has pure Error() and effectful Is(error)
calling time.Now(). A pure record root constructs fmt.Errorf("%w", Leaf{}) and
then errors.Is(err,target). Constructor formatting follows pure Error; later
implicit (124) sees an opaque builtin result with nil elements and omits Is.
An effectful As(any) has the analogous path.

errors.Join (82) retains immediate children, so direct operand callbacks have
a path. However implicit (129) discards concrete Unwrap method returns. An
Outer with pure Error() and Unwrap() error returning Leaf escapes when passed
to errors.Is directly or through Join: runtime follows Leaf and invokes its Is,
whereas the analyzer drops the returned child. Cover both single and multi-error
Unwrap trees, cycles and actual argument context.

modeledResults (58–62) marks every static error result builtin/known. Dynamic
interface dispatch in effects.go (158) invokes a concrete implementation through
void implicit, then discards its actual return. Supplier.Get() error returning
a concrete Leaf whose Error calls time.Now therefore loses that effect when
the returned error is used. errors.Unwrap similarly loses returned provenance.

Preserve wrapped children separately from cached formatted messages; propagate
concrete interface/Unwrap results, recurse through error trees for Is/As, and
retain unknown provenance when unresolved. Only established builtin error
implementations may become known-safe. Avoid indiscriminate callback summaries:
fmt.Errorf(...).Error() returns a cached message, while Join.Error() invokes its
children; errors.Unwrap supports only Unwrap() error, whereas Is/As traverse
single and multi-error forms. These distinctions prevent false positives
without admitting the actual false negatives.

## Source identity and dispatch bounds

The inspected fmt/errors.go and fmt/print.go hashes matched the summary pins;
matching pins do not repair the provenance omissions. The errors.Join and
Is/As/Unwrap summaries used standard-library metadata but did not pin
errors/join.go or errors/wrap.go. Bind those implementations if claiming pinned
error-summary behavior, within this existing analyzer repair scope.

Tier: session (jev-unavailable(no_key)). The first selector attempts failed
state-schema validation because acceptance was an array; after correcting it
to a string, each assignment was judged once successfully. Requested routing
was gpt-6.1-sol/high, same family as the writer; actual execution metadata was
unobservable. The scout performed only reads/searches/hash inspection, no
edits, artifacts, tests, builds, generation, package loading, Flow/Git mutations,
bridges or further agents. No broad scope expansion or acceptance waiver.
