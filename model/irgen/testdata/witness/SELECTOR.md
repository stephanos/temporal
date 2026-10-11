# Exact composed Property selectors

Task fn-140.2 emits the existing `Property.when_action` string as
`@exact-composed-class:<canonical composed class key>` when source `.when(...)`
names a `Composed`, including the final class of a witness. The ordinary core twin
uses the same `.when(composedClass)` spelling and bytes. A machine's `when_class`
and every legacy `when_action` remain unchanged.

The suffix is the exact key already used by a composition's Scenario: a member's
own class is `<field>_<action>` followed by its input value keys; a synchronized
class is the sync name followed by its complete input keys. Owner validation and
the canonical selector code remain shared with Scenario schedules. No synthetic
Action, IR field, declaration kind or global ClassKey change is introduced.

Legacy admission in `tools/umpire/ir/validate_keys.go` admits `when_action` only
when it equals `interp.ActionOf` of at least one independently derived composed
class key. `ActionOf` cuts at the first ASCII hyphen. Its output therefore never
contains a hyphen, even when a source action or sync name contains one. The exact
tag contains hyphens, so its language is disjoint from all admitted legacy action
selectors. Merely reserving an unusual prefix without that proof would not suffice:
authored names otherwise admit unrestricted strings.

The deferred Go task fn-140.7 must decode this tag in one shared selector helper,
validate its complete suffix against the independently derived class catalog of
the Property's declared composition, and select only byte-equal table class keys.
Malformed, empty and missing-class suffixes must be refused at the Property's
position. Untagged selectors keep their existing action-wide interpretation.
Admission, checking, reach/lint, lowering, slicing and Quint export must share this
contract; a successful Scala fixture is not proof of those consumers.

`WitnessFixtures` independently enumerates the fixture's member and sync class
catalogs, checks tagged selectors against them, compares complete witness/core IR
with an explicit declaration-span position map, and seeds input/member substitutions
that must change the exact expected declarations. `Together`, `Reversed` and
`Repeated` share their state type, including two occurrences of one machine.
The records selector is checked against the owning composition's exact field and
machine, rather than inferred from a fact suffix or unique machine membership.
