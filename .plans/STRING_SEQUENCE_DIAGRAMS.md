# Visual sequence diagrams in Scala Models

Exploration note, 2026-10-08. These are design ideas, not an approved implementation plan.
All proposed APIs, diagram syntax and generation markers below are illustrative.

## What we want

A representation embedded in Scala source that makes interactions and ordering easier to understand
than ordinary code. It should be unambiguous to parse, visually readable in the source file itself,
and straightforward for tooling to format and check. Ease of writing is secondary to ease of reading
and confidence that the presentation is correct.

The motivating opportunity is spatial expression. A diagram can show participants, communication,
overlapping lifetimes and a failure between two events together. A table that merely compresses
repeated Scala calls does not fully meet that ambition.

Two possible forms are in scope:

- An executable diagram literal, whose structure declares a Scenario or, eventually, a protocol.
- An automatically generated diagram comment above ordinary Scala declarations. The code remains
  authoritative, and a gate checks that the comment is current.

Generated comments are the current recommendation for an initial experiment. This recommendation
does not settle whether diagrams should eventually become an authoring language.

## Where Umpire could benefit

### Activity and the task queue

`model/temporal/features/activity/standalone/system/DispatchWithTaskQueue.scala` already declares scenarios
whose important feature is the ordering between participants:

- `staleDeliveryAfterPause`: dispatch and persist, pause the activity, then deliver.
- `deliveredAgainAfterLostAck`: deliver, lose the acknowledgment, then deliver again.
- `crashAfterAdmissionCommit`: crash between admission and acknowledgment, then retry delivery.

The composition supplies two members and explicit synchronization points. These make a useful first
subject because a drawing can distinguish a member's own step from one that advances both members.

A possible generated comment for the existing lost-acknowledgment schedule is:

```scala
// @diagram deliveredAgainAfterLostAck
//
//  Activity                  Queue
//     |                        |
//     +------- dispatch -------+
//     |                        o addActivityTask
//     |                        o persistTask
//     +-------- admit ---------+
//     |                        o fault.ackLoss
//     +-------- admit ---------+
//     |                        |
//
//  Pinned schedule. Each crossbar is one atomic synchronization.
// @end-diagram
val deliveredAgainAfterLostAck = c.scenario.actions(
  c.synced(_.activity -> history.dispatch),
  c.own(_.queue, queue.addActivityTask),
  c.own(_.queue, queue.persistTask),
  c.synced(_.activity -> worker.poll),
  c.own(_.queue, fault.ackLoss),
  c.synced(_.activity -> worker.poll)
)
```

The names `dispatch` and `admit` are declared synchronization names. In particular, `admit` does not
assert successful admission. Crossbars avoid inventing a message direction that the synchronization
does not declare.

### Nexus close and reset

`model/temporal/features/nexus/workflow/system/ClosePolicy.scala` separates finishing work, delivering
its completion, resetting the caller and retaining an outcome. Scenarios such as
`resetBetweenDeliveries`, `canceledAcrossReset` and `closedThenFinished` could benefit from diagrams
that expose which event crosses a reset or closure.

These are authored close/reset designs, including deliberate negative controls. A diagram must not
present them as claims about currently implemented server behavior. The model also distinguishes
operation, run and delivery identities; a rendering must preserve those distinctions.

### Channels and lifetimes

`model/framework/Channel.scala` declares queued messages, delivery, loss and redelivery separately.
Where participant information exists, a diagram could show a message remaining in flight across a
fault or being delivered again.

Timelines are another candidate for pending pause/cancellation, held attempts and timeout windows.
They are especially useful when the question concerns overlapping lifetimes rather than who sends
a message. Any displayed duration must come from an explicit bound or observation.

## Prior art to borrow from

| Prior art | Useful contribution | Boundary for Umpire |
| --- | --- | --- |
| [RxJS marble tests](https://github.com/ReactiveX/rxjs/blob/master/apps/rxjs.dev/content/guide/testing/marble-testing.md) | Executable visual strings with symbols bound to actual values; aligned streams expose event relationships. | Their dashes encode virtual time. Umpire should not assign elapsed time to padding or arrow length. |
| [Message Sequence Charts, ITU Z.120](https://www.itu.int/ITU-T/recommendations/rec.aspx?rec=z.120) | A trace language with graphical and textual syntax for interactions between components. | Borrow precise interaction semantics without adopting the entire language. |
| [Live Sequence Charts and PlayGo](https://wiki.weizmann.ac.il/playgo/index.php?title=Language_%26_Concepts) | Executable charts distinguish may/must and execute/observe modalities and can express forbidden behavior. | Umpire already separates allowed transitions from Properties and progress obligations. Chart syntax must preserve that separation. |
| [ditaa](https://github.com/stathissideris/ditaa) | ASCII geometry is the source, parsed into a drawing. This closely matches the visual-source ambition. | Graphical connectivity alone supplies no distributed-system semantics. |
| [PlantUML ASCII output](https://plantuml.com/ascii-art) | Sequence diagrams can be rendered as ASCII or Unicode with width controls. A possible renderer or source of layout conventions. | Its textual input would need to be generated from Umpire. Rendering an arrow does not establish its validity. |
| [WaveDrom](https://wavedrom.com/tutorial.html) | Aligned signal lanes and compact encodings of state over time. Useful inspiration for lifetime views. | Its time-period encoding and hardware vocabulary are not Umpire semantics. |
| [Scalameta quasiquotes](https://scalameta.org/docs/trees/quasiquotes) | Structured syntax with embedded Scala values provides a precedent for typed holes. | A design reference, not a proposed new dependency. |

The promising combination is MSC-inspired semantics, Scala references for identity and a compact
ASCII presentation. Library adoption is a separate decision.

## Decide what a diagram means

Three artifacts need different labels and potentially different rendering rules:

1. **Schedule.** The author requests particular action classes in an exact order. It does not show
   which outcomes occur or establish that a complete execution is possible.
2. **Witness or counterexample.** The checker supplies a particular execution, including its choices,
   outcomes, facts and state changes. The drawing identifies the Query and the result it depicts.
3. **Protocol.** The author specifies relationships that admit multiple schedules, such as delivery
   following sending while another event is unordered with them.

`model/framework/Claims.scala` currently represents a Scenario as an exact action schedule or free
exploration. Even a pinned schedule may have several executions because effects are nondeterministic.
A partial-order protocol would require additional semantics and support in the checking pipeline.
Rendering one chosen linearization must not suggest that every depicted order is required.

The following distinctions must remain explicit:

- A send and its delivery can be separate steps with channel state between them. A single horizontal
  arrow must not silently merge them into an atomic operation.
- A composition synchronization is one atomic step of two members. It does not inherently declare
  a sender and receiver.
- A scheduled action is not an observed fact or a successful result.
- An expected state is an assertion to check, not a filter that removes executions which violate it.
- The placement of an event on one lane must not introduce undeclared ordering on another lane in
  a future partial-order form.
- Drawing dimensions convey layout. Time and progress bounds need explicit model meaning.
- Crash, message loss, timeout, rejection and permanent termination are different events. A generic
  cross or broken line should not stand for all of them.

For example, `[paused]` or `[no attempt admitted]` underneath an interaction must be identified as an
assertion, an evaluated witness state or explanatory prose. A generator cannot choose a favorable
outcome from a nondeterministic step and present it as the schedule's guaranteed result.

## What can be derived today

The current declarations supply action order, action classes and inputs, composition members,
member-local steps and synchronization membership. `ActionDecl` also carries an actor and optional
target entity, plus timer/internal/channel information.

They do not provide a universal sender/receiver relation for every action. Actors, entities,
composition members and physical services are different concepts. A diagram must choose and label
its level of abstraction. An internal action with no endpoints should remain a local event unless
the model explicitly supplies the missing information.

Generated comments therefore do not remove all design work. They remove the need to parse drawings
as executable declarations, while participant selection and truthful rendering still need rules.
Additional presentation metadata could name lanes or select fields to display, but should refer to
existing declarations rather than repeat their action sequence or invent behavior.

Detailed state timelines require evaluation. Under the current architecture, the Go reader computes
Model behavior. A diagram generator should consume its results instead of introducing another
evaluator in Scala.

## Executable literals in Scala

Scala custom interpolators receive literal fragments separately from expression arguments and can
return structured values. They can retain action references without converting them to text.
See [Scala string interpolation](https://docs.scala-lang.org/overviews/core/string-interpolation.html).

An illustrative form, with `dispatch`, `pause` and `admit` bound to existing composed steps, is:

```scala
sequence"""
  Activity              Queue
     |                    |
     +--------------------+ $dispatch
     o                    | $pause
     +--------------------+ $admit
"""
```

The diagram's geometry locates each step. Typed references identify it, and validation checks that
the drawn participants match its declaration. Labels such as `effects.pause` written as ordinary
text are not automatically Scala references.

There are two reference strategies:

- **Typed holes.** `$pause` or `${c.own(_.activity, client.pause)}` preserves Scala name resolution.
  Long expressions clutter the picture; short aliases make the reader consult their definitions.
- **Explicitly bound textual names.** A supplied vocabulary gives a cleaner drawing, at the cost of
  additional name validation and weaker ordinary IDE rename support inside the literal.

A runtime parser alone is insufficient for Umpire's pipeline. `model/irgen/Lift.scala` reads written
TASTy trees, and the lifter recognizes particular authoring forms. It would need explicit support
for a diagram literal. A shared parser could serve runtime construction, lifting and formatting;
the adapters would retain typed references and source positions.

A macro could give earlier literal diagnostics, but is optional and would need consideration against
the existing author-surface restrictions in `DSL_OPERATORS.md`. Macro expansion should not be assumed
to produce trees the current lifter accepts.

For a parseable drawing, constrain the grammar to a small set of lifelines, endpoints, connectors,
event labels and explicit grouping markers. Reject ambiguous connections instead of guessing.
Normalize common indentation, prohibit tabs in the drawing, and keep spacing changes from changing
time or order. A formatter must preserve connectivity and event order, with a parse/format/parse
round trip yielding the same structure. Embedded Scala expressions need to remain opaque tokens
rather than be reformatted by the diagram parser.

## Generated comments above Scala

Generated comments retain existing Scala declarations as the source of truth. The generator owns
only a marked region associated with a declaration, preferably by a stable declaration identity
rather than its current line number. Human explanations remain outside that region.

A possible workflow is:

1. Read the compiled/lifted declarations and any explicit presentation settings.
2. Produce a deterministic diagram for each opted-in declaration.
3. In update mode, replace only managed comment regions.
4. In check mode, compare expected output with the source and report a diff without writing.
5. Perform the final lift after source comments have settled.

`model/check/CommentRule.scala` permits only `//` comments, so generated regions use line comments.
Diagram formatting would need its own handling integrated with `make fmt-model` and `make lint-model`;
ordinary Scala formatting is not a diagram layout engine. The existing 100-column convention is a
useful presentation constraint.

Inserting comments changes line numbers recorded in the IR. Diagram content must exclude source
positions and timestamps so generation settles. After an update, the final lift must record the
new source positions. This ordering needs care to avoid repeated stale-artifact reports.

Presentation rules should keep participant order stable, preserve full action inputs, wrap labels
predictably and retain all relevant events. If a diagram exceeds its width or complexity limit,
report that limitation or move the detailed view elsewhere; do not silently hide steps.

A generated comment needs no parser for its internal drawing. Its text is checked against canonical
output. That is a substantial reduction in implementation scope compared with executable literals.

## Tradeoffs

| Form | Benefit | Cost |
| --- | --- | --- |
| Executable diagram literal | The diagram directly declares the scenario, with no repeated action list. Could eventually express partial orders naturally. | Parser, reference binding, lifter support, source diagnostics, formatting and possibly new Scenario semantics. |
| Generated comment | Visual reading in ordinary editors and code review; Scala types and refactoring remain intact. | More source volume, regeneration, layout churn and changes to recorded source positions. |
| Generated editor view or report | Richer diagrams without enlarging source files. Suitable for witnesses and counterexamples. | Readers must open another view; ordinary source diffs do not contain the diagram. |

All forms risk giving a drawing more authority than its evidence supports. Schedule, witness and
protocol labels are part of correctness. Freshness checking prevents stale comments, but does not
prove that the renderer has interpreted the model correctly.

Generated comments should be selective. A picture above every short Scenario could make a file
harder to navigate. The useful subjects are interactions where spatial arrangement exposes a race,
failure boundary, repeated delivery or lifetime relationship.

ASCII is a reasonable initial format for editor and diff portability. Unicode box drawing could
improve appearance later, but font widths and connector conventions need consistent handling.
Neither form should require color to convey semantics.

## Suggested first experiment

Compare generated comments for three existing scenarios:

- Activity's `deliveredAgainAfterLostAck`.
- Activity's `crashAfterAdmissionCommit`.
- Nexus's `resetBetweenDeliveries`.

Judge them by whether a reader can explain the critical ordering and participants without inspecting
the action list, whether every visual relationship is justified by declarations, and whether the
diagram fits comfortably beside the code. Change one action or input and inspect the resulting diff.
Repeat generation to check that the layout settles.

This experiment would establish the rendering vocabulary and its limits. An executable literal
could later use that vocabulary if authoring directly through diagrams adds enough value.

Open decisions include which declarations opt in, whether lanes denote actors or model members,
which presentation metadata is needed, whether any state assertions belong in source diagrams,
and whether future protocol diagrams should admit multiple interleavings. None is settled here.
