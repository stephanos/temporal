# Mutable call-binding soundness checks

Source-only hypotheses sent to the task19 writer before source freeze.
The conductor ran no Go tests and makes no demonstrated-failure or formal
review claim. `effects.go` after inspection had SHA-256
`61042a6274d4490e526595fb3c0345d1f026d9a762695f7ab3a9b7c8ae791010`;
the writer remains active, so this is not a frozen source identity.

## Cross-slot aliases

`callValueKey` normalizes and serializes each argument separately, creating
a fresh `bindingKey` seen map for every slot. The call key also serializes
receiver and captures separately. This preserves each slot's structure but
loses sharing between slots.

Suggested negative fixture: an ordinary helper package outside pure roots
defines `Box{F func()}`, `Clean`, `Dirty` (which reads `time.Now`), and
`Probe(x,y *Box) { x.F = Dirty; y.F() }`. A pure root first invokes `Probe`
with two separate clean boxes, then with one fresh clean box in both slots.
The latter call reaches `Dirty` at runtime. Structurally identical per-slot
keys can reuse the first call's result and suppress that effect.

Use one canonical ordered binding graph if memo identity must distinguish
cross-argument, receiver and capture aliases. Retain cycle handling and
finite allocation-site summaries; do not replace a termination defect with
unbounded raw keys.

## Replaying mutation on memo reuse

`call` returns a cached result immediately, while assignment to a selector
mutates the argument's abstract fields. Its memo entry stores only the result.

Suggested independent negative fixture: helper `Set(x *Box) { x.F = Dirty }`;
a pure root creates separate clean boxes `a` and `b`, invokes `Set(a)`, then
`Set(b)`, then `b.F()`. If the second `Set` reuses the same clean-input memo
entry without replaying its argument update, the analyzer can miss `Dirty`.

Validate both fixtures and receiver/capture variants before freeze. A repair
must preserve mutation semantics across memo reuse, or conservatively avoid
unsafe result-only reuse. A source-only warning is not RED/GREEN evidence;
the implementation writer owns reproduction, repair and regression tests.
