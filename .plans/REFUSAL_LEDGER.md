# Refusal ledger

fn-141.6 records the current refusal inventory and the intended enforcement homes for fn-174.
**Status: inventory and decisions only; fn-174 enforcement and fixture migration are planned,
not implemented or verified here.** Fn-174.8 consumes every row and verifies the final refusal
contract. There is no deletion verdict without proof that the exact formerly refused legal
spelling exports.

## Census and scope

Census at `a20e99698df4c803b030245252fd2e19ebda062f` on branch `umpire`, with the owner's
uncommitted Lift.scala header included. The inventory's source coordinates refer to that working
tree. Recount after exporter edits; do not treat line numbers as permanent identifiers.

Production means top-level `*.scala` files in `model/framework/export` and
`model/temporal/export`, excluding `*.test.scala`, fixtures and nested directories. Reproduce:

```bash
rg --files model/framework/export model/temporal/export -g '*.scala' -g '!*.test.scala' |
  rg '^model/(framework|temporal)/export/[^/]+\.scala$' |
  xargs rg -n '\bfail\(|LiftError\('
```

There are **619 lexical occurrences**, one per matched line. Exclude only the `def fail`
declaration (`Context.scala:295`) and its shared `throw LiftError` sink (`Context.scala:303`).
The remaining **617 direct sites** are 606 `fail` calls and 11 direct `LiftError` constructors,
assigned exactly once to **61 kinds** below. Temporal exporter production files have zero direct
sites, but their delegated source policy remains recorded. Test files and recursive fixture scans
are not part of this count. Site counts are not branch-coverage counts.

| Framework exporter file | Direct refusal sites |
| --- | ---: |
| `Capabilities.scala` | 81 |
| `Claims.scala` | 50 |
| `Compositions.scala` | 50 |
| `Constants.scala` | 4 |
| `Context.scala` | 9 |
| `Declarations.scala` | 113 |
| `Expansion.scala` | 19 |
| `Expressions.scala` | 33 |
| `Gaps.scala` | 27 |
| `Lift.scala` | 5 |
| `Lifting.scala` | 3 |
| `Realizations.scala` | 148 |
| `Roles.scala` | 6 |
| `Syntax.scala` | 58 |
| `Types.scala` | 11 |
| **Total** | **617** |


The implementation scope is framework/shared code, the six samples
`model/framework/samples/{Kettle,Bank,Voting,TwoPhaseCommit,TwoLayeredCache,ReliableBroadcast}.scala`,
`model/temporal/features/activity_mini/` and existing fixture contracts. Real-feature-only
ScriptRejects obligations are **sweep-held**, never deletable on mini evidence; do not alter a
real feature to prove them. This ledger follows fn-174's approved design: declarations construct
IR, capture points lift functions, declaration IR positions may be empty, names remain, and Go
answers plus two-run determinism replace the retired per-step byte-identity requirement.

## Intended outcomes and source positions

| Outcome | Intended sole front-end enforcement home | Sites | Kinds |
| --- | --- | ---: | ---: |
| M | Compile-time capture macro / definition-site reification | 149 | 19 |
| C | Construction or conversion on the owning DSL/kit type | 379 | 29 |
| L | Source lint, using the retained inspector | 89 | 13 |
| G | Existing Go admission as the sole home | 0 | 0 |
| D | Proven deleted: exact legal spelling now exports | 0 | 0 |
| **Total** | | **617** | **61** |

These are decisions for implementation, not assertions that current factories/macros already
enforce them. Kind IDs are stable inventory labels; **G01 now has outcome C**. Its four predicates
already exist in Go, but it keeps a located construction check on constructed state/channel/action/
binding metadata in fn-174.4. This needs no new tree declaration evaluator. Fn-174.8 must verify
the author's diagnostic before dropping the old Scala path. Go independently admits IR as today;
Go predicate equivalence alone does not preserve a Scala source diagnostic.

Every kept refusal must report the author's file and line with the existing message contract.
Today's `model/framework/export/Context.scala:264–303` distinguishes written positions from
generated-node placement and adds facet context. Declaration positions omitted from exported IR
do **not** exempt factory errors from this requirement.

- **M, capture / def:** capture argument trees supply positions; `@lifted` supplies a reached def's
  definition position. Shared type/constant/role readers must be checked in both body and declaration
  contexts (.1 with .3/.4); a shared sink is counted once, not permission to require declaration
  constants to be macro literals.
- **M, selector capture needed:** M17/M19 require an inline selector capture carrying the author's
  position; ordinary runtime lambdas are insufficient. .2 owns the capture and .8 its line tests.
- **C, origin metadata needed:** reflection-derived names are insufficient. Preserve capture-origin
  diagnostic metadata on the declaration/operation, or obtain an author source-map location from
  the retained inspector. Store diagnostic metadata independently of IR Position. Where no capture
  exists, .8 provides the inspector-to-declaration map; the row's owner propagates it through the
  factory/conversion. Duplicate IDs retain both declaration origins; root failures retain the root,
  declaration being constructed and its origin. This transport is a required implementation gap.
- **L, inspector:** source lint obtains written positions directly and traverses admitted definitions,
  including those not captured or executed. Run initialization lint **before IrFile.discover**,
  which initializes owners, not merely before construct or publication.

`model/framework/dsl/RuleSource.scala:22–52` captures a position but emits runtime validation;
that is not proof of static source coverage. Its `disabledImpl` line 63 separately refuses at
compile time. .4/.8 must retain that distinction.

Owners below mean `fn-174-the-model-constructs-its-own-ir-dsl.<suffix>`: .1 body macros;
.2 realizations; .3 signature; .4 machines/rules; .5 compositions; .6 claims; .7 capabilities/gaps;
.8 driver, source-map integration and contract verification. Every row has an implementation owner;
.8 verifies all of them.

## Complete site-to-kind inventory

Exact files are under `model/framework/export/`. Commas list separate source lines, not ranges.
Outcome C retains the **semantic invariant** through constructed values; it does not authorize
reproducing an obsolete parser shape as a factory restriction. The proof obligations after the
coverage table govern those arms before the old tree reader is removed.

| Kind | Refusal kind | Outcome | Position source | Owner | Sites | Exact source sites |
| --- | --- | --- | --- | --- | ---: | --- |
| M01 | Function calls, pure expression subset, recursion and opaque callbacks | M | capture / def | fn-174.1 | 11 | `Expressions.scala:383,391,395,478,499,528,532,667,679,706,715` |
| M02 | Choice token identity, named branching, single-step alternatives and level discipline | M | capture / def | fn-174.1 | 8 | `Expressions.scala:760,781,798,827,877,886,903,921` |
| M03 | Pattern and typed-remainder completeness | M | capture / def | fn-174.1 | 7 | `Expressions.scala:992,993,995,999,1004,1027,1043` |
| M04 | Composed through selectors and function references | M | capture / def | fn-174.1 | 3 | `Expressions.scala:157,163,169` |
| M05 | Block bodies, nested blocks and state accessors | M | capture / def | fn-174.1 | 4 | `Expressions.scala:278,283,309,311` |
| M06 | Sugar expansion, cycles, callback provenance, member capture and eager argument handling | M | capture / def | fn-174.1 | 14 | `Expansion.scala:70,201,224,243,256,260,309,326,331,337,349,374,412,504` |
| M07 | Finite type representation inside functions | M | capture / def | fn-174.1 | 11 | `Types.scala:49,62,71,74,96,101,102,113,138,156,181` |
| M08 | Role legality, phase enums and nonempty role cases | M | capture / def | fn-174.1 | 6 | `Roles.scala:109,126,134,142,143,151` |
| M09 | Compile-time constant readers used by function reification | M | capture / def | fn-174.1 | 4 | `Constants.scala:56,59,67,91` |
| M10 | Block placement at function boundary | M | capture / def | fn-174.1 | 1 | `Context.scala:72` |
| L01 | Source aliases, val names and declaration placement | L | inspector | fn-174.8 | 5 | `Context.scala:90,102,107,313,332` |
| C01 | Facet-to-machine ownership and distinct type names | C | origin metadata needed | fn-174.4 | 3 | `Context.scala:419,421,447` |
| C02 | Action identity, inputs, binding coherence and signature reconstruction | C | origin metadata needed | fn-174.3 | 6 | `Declarations.scala:34,36,60,101,115,152` |
| L02 | Retired action schema spelling | L | inspector | fn-174.3 | 1 | `Declarations.scala:86` |
| M11 | Constant values and step/header functions | M | capture / def | fn-174.1 | 8 | `Declarations.scala:208,225,319,324,354,367,427,450` |
| L03 | Machine header, refinement, rules and facet section structure | L | inspector | fn-174.4 | 18 | `Declarations.scala:257,388,393,398,414,419,433,435,441,453,459,467,471,514,520,528,534,545` |
| C03 | Monitor state compatibility and entity inference | C | origin metadata needed | fn-174.4 | 2 | `Declarations.scala:501,599` |
| C04 | Rule/facet ownership, duplicate targets and disabled-vs-fired conflicts | C | origin metadata needed | fn-174.4 | 6 | `Declarations.scala:658,689,695,720,739,766` |
| L04 | Rule syntax, from ownership, qualification and facet vocabulary | L | inspector | fn-174.4 | 25 | `Declarations.scala:672,701,715,724,726,727,728,823,832,840,847,890,895,925,963,970,974,983,994,1009,1022,1037,1045,1153,1178` |
| C05 | Finite action input catalogs | C | origin metadata needed | fn-174.3 | 2 | `Declarations.scala:1077,1078` |
| M12 | Rule condition/effect captures and no-empty-effect validation | M | capture / def | fn-174.4 | 5 | `Declarations.scala:1117,1123,1133,1160,1166` |
| C06 | Derivation reconstruction and replacement invariants | C | origin metadata needed | fn-174.4 | 11 | `Declarations.scala:1254,1255,1311,1343,1359,1367,1392,1405,1419,1429,1447` |
| G01 | Channel storage, delivery and loss consistency | C | origin metadata needed | fn-174.4 | 4 | `Declarations.scala:1497,1505,1511,1518` |
| C07 | Unique action/model/declaration names | C | origin metadata needed | fn-174.3 | 3 | `Declarations.scala:1529,1651,1696` |
| M13 | Fact evidence defaults and full payload coverage | M | capture / def | fn-174.4 | 3 | `Declarations.scala:1542,1555,1617` |
| L05 | Machine object requirement and derivation cycle | L | inspector | fn-174.4 | 2 | `Declarations.scala:1662,1672` |
| C08 | Channel construction, finite messages and inbox identity | C | origin metadata needed | fn-174.3 | 8 | `Declarations.scala:1712,1717,1718,1729,1758,1764,1806,1808` |
| C09 | Monitor construction, finite state and names | C | origin metadata needed | fn-174.4 | 3 | `Declarations.scala:1860,1863,1866` |
| C10 | Assumption ownership and hole construction | C | origin metadata needed | fn-174.3 | 5 | `Declarations.scala:1887,1890,1897,1911,1926` |
| C11 | Claim reconstruction, builders, roots and bundle values | C | origin metadata needed | fn-174.6 | 31 | `Claims.scala:73,76,79,223,307,326,382,385,418,423,452,459,476,477,494,515,532,537,546,553,562,571,573,594,600,675,683,688,911,924,937` |
| C12 | Required input defaults | C | origin metadata needed | fn-174.3 | 1 | `Claims.scala:116` |
| C13 | Claim and Limits identity, duplicate witness, ownership, bounds and expectations | C | origin metadata needed | fn-174.6 | 14 | `Claims.scala:137,189,212,215,246,252,281,381,475,613,632,727,763,768` |
| L06 | Claim placement and authored literal total provenance | L | inspector | fn-174.6 | 3 | `Claims.scala:699,754,881` |
| M14 | Function-valued claim helper argument | M | capture / def | fn-174.1 | 1 | `Claims.scala:894` |
| C14 | Composition construction, sync ownership, replacements and composed claim identity | C | origin metadata needed | fn-174.5 | 29 | `Compositions.scala:48,61,87,115,173,183,280,283,302,308,319,369,376,378,394,402,408,421,427,442,448,455,461,505,513,530,536,546,563` |
| M15 | Composition selector syntax captured at its call | M | capture / def | fn-174.5 | 10 | `Compositions.scala:21,38,134,159,166,194,292,396,524,543` |
| L07 | Composition section structure | L | inspector | fn-174.5 | 11 | `Compositions.scala:97,212,223,229,238,241,248,257,266,268,271` |
| C15 | Capability construction, typed facts, bindings, waivers, bounds and expansion | C | origin metadata needed | fn-174.7 | 59 | `Capabilities.scala:108,113,120,357,371,382,403,409,432,530,542,548,569,589,596,597,602,611,615,617,629,637,644,666,674,678,684,686,691,725,727,735,737,751,758,777,788,790,806,850,858,869,879,886,890,911,932,1020,1078,1086,1165,1192,1200,1234,1239,1264,1298,1303,1305` |
| M16 | Capability function captures, override signatures and quantified predicates | M | capture / def | fn-174.7 | 14 | `Capabilities.scala:58,60,419,843,986,992,997,1045,1050,1054,1070,1115,1118,1138` |
| L08 | Capability section placement and provenance | L | inspector | fn-174.7 | 8 | `Capabilities.scala:206,249,278,296,348,354,714,799` |
| C16 | Gap subject resolution, inheritance, uniqueness and bounded expansion | C | origin metadata needed | fn-174.7 | 26 | `Gaps.scala:298,347,400,419,420,428,437,450,468,480,498,506,531,542,549,554,568,582,583,601,602,608,620,628,660,662` |
| L09 | Gap expiry references to Flow specs | L | inspector | fn-174.7 | 1 | `Gaps.scala:294` |
| C17 | Realization vocabulary, typed values, descriptors and reconstruction | C | origin metadata needed | fn-174.2 | 70 | `Realizations.scala:102,109,179,182,190,193,197,200,202,205,220,224,226,237,353,356,358,360,365,367,402,407,419,424,485,524,584,585,590,591,596,600,639,646,780,871,872,910,911,985,1005,1009,1011,1022,1062,1067,1079,1098,1126,1134,1142,1146,1148,1154,1159,1194,1240,1760,1768,1774,1797,1799,1805,2296,2297,2553,2556,2573,2632,2656` |
| M17 | Protobuf selector capture and typed paths | M | selector capture needed | fn-174.2 | 13 | `Realizations.scala:245,247,255,282,291,298,309,318,321,327,332,342,1285` |
| C18 | Elapsed Duration range, exact milliseconds and authored fields | C | origin metadata needed | fn-174.2 | 5 | `Realizations.scala:458,929,931,934,935` |
| C19 | Evidence/status family completeness, keys and fact identity | C | origin metadata needed | fn-174.2 | 13 | `Realizations.scala:1211,1215,1278,1290,1300,1311,1361,1411,2410,2427,2429,2431,2435` |
| C20 | Request bases, deadline classes, presets and derived server steps | C | origin metadata needed | fn-174.2 | 17 | `Realizations.scala:1449,1459,1467,1501,1512,1519,1523,1528,1538,1550,1575,1605,1660,1686,1714,2213,2231` |
| L10 | Realization sections, instruction/name spelling and source scopes | L | inspector | fn-174.2 | 12 | `Realizations.scala:1864,1871,1877,1882,1889,1900,2323,2468,2527,2662,2688,2697` |
| C21 | Realization/query/action ownership, command identity and explained expectations | C | origin metadata needed | fn-174.2 | 18 | `Realizations.scala:810,817,1916,1920,1948,1984,2013,2028,2125,2137,2144,2146,2247,2274,2344,2374,2383,2518` |
| C22 | Witness shape, names, owner, path, fact, repeated-last-class and Limits | C | origin metadata needed | fn-174.6 | 16 | `Syntax.scala:82,120,123,134,135,145,146,148,152,162,173,176,178,188,199,205` |
| M18 | Effect blocks, recorded statuses, rejection and phase writers | M | capture / def | fn-174.4 | 17 | `Syntax.scala:313,318,350,367,388,398,444,459,469,479,490,510,531,542,567,606,624` |
| C23 | Named input tokens and assignment shape | C | origin metadata needed | fn-174.3 | 5 | `Syntax.scala:675,684,690,703,1157` |
| M19 | Request/response selectors and literal scope capture | M | selector capture needed | fn-174.2 | 9 | `Syntax.scala:780,848,854,861,963,1019,1026,1049,1055` |
| C24 | Protobuf literal values, fields, duplicate assignments and observations | C | origin metadata needed | fn-174.2 | 11 | `Syntax.scala:801,822,823,896,915,931,1010,1085,1111,1136,1141` |
| C25 | Driver root/file admission | C | origin metadata needed | fn-174.8 | 2 | `Lift.scala:117,122` |
| L11 | Gap on declaration no IR file lifts | L | inspector | fn-174.7 | 1 | `Lift.scala:205` |
| C26 | Cross-file derived ID and equivalent Property collisions | C | origin metadata needed | fn-174.8 | 2 | `Lift.scala:244,407` |
| C27 | Root existence and supported root values | C | origin metadata needed | fn-174.8 | 3 | `Lifting.scala:32,58,86` |
| C28 | Declaration sugar reconstruction and arity | C | origin metadata needed | fn-174.6 | 4 | `Expansion.scala:119,137,145,149` |
| L12 | Direct declaration builder-chain spelling | L | inspector | fn-174.6 | 1 | `Expansion.scala:177` |
| L13 | Every declared effect returns a step on every authored result path | L | inspector | fn-174.4 | 1 | `Declarations.scala:1232` |

## Existing coverage and explicit gaps

`model/framework/export/testdata/lifts/expected/rejects.txt` contains 226 historical rows.
`model/framework/lint/RefusalContract.test.scala:8–98` adapts messages/coordinates and filters
sweep-held sources; `Fixtures.test.scala:5061–5094` uses that contract and filters marker fixtures
owned by marker lint. There are 20 historical ScriptRejects and 10 MarkerRejects rows: the current
exporter-authored snapshot slice is **196 rows**, plus split suites. This is source-contract
arithmetic, not a passing-suite claim or proof that every branch is covered.

In the table, `R`, `B`, `S` mean `model/framework/export/testdata/lifts/Rejects.scala`,
`BlockRejects.scala`, and sweep-held `ScriptRejects.scala`, respectively. Other bare
`*Rejects.scala` filenames share that fixture directory; other bare test files are under
`model/framework/export/`; `lint/` means `model/framework/lint/`. Historical fixture coordinates
must be interpreted through RefusalContract's adaptations. “Pins” identifies inspected source
assertions, not executed tests. **Each row has unproved branch-level coverage unless explicitly
narrower.** A cited fixture never proves every listed source site.

| Kind | Existing evidence and missing coverage |
| --- | --- |
| M01 | `R:70` pins recursion; `Fixtures.test.scala:4489` checks unsupported trees at their line. Opaque callback variants need per-branch confirmation. |
| M02 | `R:825,833,849,857,867,877,885,913,921,928,936` pin choice and branching errors; `R:947,954,989` pin wrong-level reads. |
| M03 | `lint/Conventions.test.scala:211,245,263` checks retained typed/literal matching. No direct negative assertion for all seven exporter arms established. |
| M04 | `R` contains through-selector/def refusals; exact branch-to-fixture association remains missing. |
| M05 | `B:16,45,141` pins nested blocks/accessors; the malformed block-body arm lacks demonstrated coverage. |
| M06 | `DefinitionExpansion.test.scala:210,217,251,270,290,306,348` asserts cycles, opaque selectors, error locations and eager reads. `R:431,448,692` pins varargs/Ok/direct-chain restrictions. |
| M07 | `R:18` pins unbounded state lists; `Fixtures.test.scala:2779` checks compiler non-finite rejection. Remaining typeRef arms need direct macro refusal fixtures. |
| M08 | `RoleRejects.scala:26,36,47,71,79,82,85,126,134,142,146,154`; `DefaultEnds.test.scala:125` checks static missing Closed. |
| M09 | No isolated fixture for all four shared constant-reader failure arms established; indirect uses do not prove each arm. |
| M10 | `B:25,35,158` pins misplaced is/effect blocks. |
| L01 | `R:229,421` pins synthetic name and cyclic alias. Generic reference/val/object-placement arms need targeted fixtures. |
| C01 | `Facets.test.scala:190,211` checks facet ownership; no fixture established for the distinct IR type-name arm. |
| C02 | `R:759` pins duplicate input; `InputDefaults.test.scala:91` asserts computed/foreign/duplicate tokens. Action-binding cycle/coherence coverage not established. |
| L02 | `Fixtures.test.scala:2732` compiles retired schema refusal; direct exporter path is a separate coverage gap. |
| M11 | `DefaultEnds.test.scala:125` and compiler missing-header tests (`Fixtures.test.scala:2710`) are related coverage, not proof of all eight exporter arms. |
| L03 | `R:1139,1206` and `lint/Order.test.scala:72` pin refinement placement; `Facets.test.scala:152,172` pins machine facet structure. |
| C03 | `R:1173` pins wrong monitor state. Multi-entity inference refusal needs an isolated fixture. |
| C04 | `R:1106`, `RuleRejects.scala:100`, `Facets.test.scala:190,211` pin disabled conflicts and facet ownership. |
| L04 | `RuleRejects.scala:16,28,36,46,54,61,68,75,82`; `Facets.test.scala:483` checks vocabulary; disabled targets retain separate assertions. |
| C05 | No isolated two-arm input-catalog refusal fixture established. |
| M12 | `R:1065,1098,1161` pins empty effects and named phase-set requirements; capture shape/arity arms still need direct negatives. |
| C06 | `R:369,372,378,388,392,395,413,1109,1112,1116,1194` pins rebind/extend/refinement/duplicate assumptions. |
| G01 (C) | `R:52,64` and `tools/umpire/ir/admission_test.go:528` assert the corresponding channel rules. All four Go predicates inspected. |
| C07 | `R:260,299,342,357`; `CollidingRejects.scala:9`; `DerivedIds.test.scala:21,34,52` asserts identity collisions. |
| M13 | `R:314,323` pins payload facts requiring evidence. Partial case coverage is in the snapshot; exact complete matrix not established. |
| L05 | `R:417` pins machine derivation cycle. Missing object-of-source arm lacks a demonstrated isolated fixture. |
| C08 | `R:28,158,175,192,207` pins capacity, finite messages and computed policies. Go also tests numeric bounds, but that does not prove Scala branch coverage. |
| C09 | `R:124` pins duplicate monitor names. `tools/umpire/ir/admission_test.go:498` covers Go finite monitor state only. |
| C10 | `ComposedProgressRejects.scala:22,24,28` and `R:342,357` pin fairness ownership/name rules. |
| C11 | `R:897` pins mixed claim bundles. Generic fold/builder parser arms lack branch-level coverage; legal computed forms are deletion candidates requiring proof. |
| C12 | `R:573` and `InputDefaults.test.scala:91` assert required input omissions at calls. |
| C13 | `R:115,118,276,285,683`; `ComposedProgressRejects.scala:17,20,45`; `Fixtures.test.scala:507` pins duplicate Properties. |
| L06 | `R:812` pins literal call-site total; `lint/PropertyGroups.test.scala` checks placement. Complete direct coverage for all three sites not established. |
| M14 | `R:720` pins opaque function parameter; the shared-helper named-def arm needs independent mapping. |
| C14 | `R:148,485,488,496,502,503,519,526,554,640,651,655,659,663,675`; `ComposedProgressRejects.scala:32,36,40` covers composed ownership. |
| M15 | `R:477` pins nested selector. `DefinitionExpansion.test.scala:217,251` tests opaque selector references; remaining selector entry points need individual negatives. |
| L07 | `R:1122,1133`; `lint/Order.test.scala:72` asserts composition section ordering. Remaining parser-shape failures need targeted tests. |
| C15 | `CapabilitySectionRejects.scala:16,23,62,71,72,83,94,105,116,127`; `Fixtures.test.scala:4764,4957` explicitly checks retry/deadline capability failures. |
| M16 | `PhaseCapabilities.test.scala:150` and `Fixtures.test.scala:4764,4957` cover role/binding refusals. Not every function default/signature arm has been mapped. |
| L08 | `Facets.test.scala:137,152,172` asserts misplaced capability/waiver provenance, including subclass bypass. |
| C16 | `GapResolution.test.scala:283,398` checks unresolved subjects, placement, duplicates and inherited findings with line assertions. |
| L09 | `GapResolution.test.scala:310,311,312` explicitly checks missing, ambiguous and malformed expiry IDs. |
| C17 | `Fixtures.test.scala:3132,3150,4531` asserts typed repeated read, unknown enum helper, constructor/field refusals. Most defensive descriptor/decoder arms remain unmapped. |
| M17 | `Fixtures.test.scala:3132`; `Observations.test.scala:107` asserts constant non-field selector refusal; `TypedDeclarations.test.scala:13` pins compiler positions for crossed roots. |
| C18 | `HintRejects.scala:44` has the adapted duration message/coordinate in `lint/RefusalContract.test.scala:24`. Unknown-field/sub-ms arms lack established Scala negative coverage. |
| C19 | Historical `S:40,64,105,137,149,157` covers status/fact/history checks but is sweep-held. No active replacement for each arm was proved. |
| C20 | `DeadlinePresets.test.scala:102,148,316` asserts explicit non-Timeout presets, unset defaults and duplicate classes. Historical `S:120,185,219` is sweep-held. |
| L10 | Historical `S:28,81,93,193,198` is sweep-held. `Fixtures.test.scala:4508` independently asserts uppercase instruction and written command spelling. |
| C21 | `Fixtures.test.scala:390,498` asserts foreign Query owner, explained expectations and double realization ownership; historical `S:31,34,170,203,226` is sweep-held. |
| C22 | No complete active witness refusal matrix established in this investigation. Witness transformation's repeated-last-class and limits need explicit retention fixtures. |
| M18 | `B:65,75,87,98,108,119,130,150`; `StatusFactRejects.scala:20,30,39`; `RuleRejects.scala:89`; `Entering.test.scala:117` pins absent writer. |
| C23 | `InputDefaults.test.scala:91` explicitly checks omitted, duplicate, foreign and computed tokens; generic shape fallback remains unmapped. |
| M19 | `TypedDeclarations.test.scala:13` pins compile-time crossed roots; historical request-scope rows `S:81,93` are held. New capture-location coverage is required. |
| C24 | Historical `S:110` pins duplicate literal field but is held; `Fixtures.test.scala:2869` checks typed protobuf constant mismatches/forged carriers. Remaining parser arms unmapped. |
| C25 | `Fixtures.test.scala:5153` asserts no roots/usage; selected missing IR-file cases need independent mapping. |
| L11 | `GapResolution.test.scala:315` explicitly asserts a gap on an unlifted declaration. |
| C26 | `DerivedIds.test.scala:21,34,52`; `Fixtures.test.scala:507` asserts duplicate Properties. Cross-file ownership must retain source provenance. |
| C27 | Snapshot `root ...noSuchRoot` and `R:28,1218` pin roots and unsupported IndexedSeq. IndexedSeq-of-Query is a legal-form deletion candidate, not a deletion proof for arbitrary Seq. |
| C28 | `DeclarationSugar.test.scala:106` and `DefinitionExpansion.test.scala:92` prove positive declaration expansion. No isolated negative for all four reconstruction/arity guards established. |
| L12 | `R:692` pins a stored never builder instead of a direct chain; keep source-located enforcement until an accepted legal-spelling proof changes the contract. |
| L13 | `R:1065,1161` pins used empty effects. Definition-site checks of an **unused invalid effect**, a **constant-folded-away invalid If branch**, and **all Match result branches** lack established dedicated fixtures. .4 adds those source-lint fixtures; .8 proves retained text/line independently of captures. |


Cross-cutting existing source assertions: `LintStages.test.scala:10,32` (lint before output),
`lint/Order.test.scala:7,72,124` (initialization/order), `TypedDeclarations.test.scala:13`
(compiler positions), `DerivedIds.test.scala:17–52` (identity).
`SugarFixtures.test.scala:82` and `DeclarationSugar.test.scala` are positive equivalence evidence,
not a complete negative matrix. `ModelBuild.test.scala:104–116` holds Scripts and ScriptRejects;
those held contracts must survive migration.

## Parser-shape retirement obligations

A C row may contain both a value invariant and a tree-reconstruction fallback. The invariant's
destination is settled as construction; **no factory may reject a computed legal value merely
because its former tree shape was unrecognized**. Until its owner supplies the following proof,
the existing exporter refusal stays active. This is a proof obligation at an assigned task, not
an unassigned outcome or a claim of implemented deletion. .8 reconciles each retired arm with an
accepted spelling and the surviving invariant's negative/location tests.

| Kinds | Surviving invariant and accepted-spelling proof required before matcher removal | Owner |
| --- | --- | --- |
| C02/C05/C08/C10/C23 | Action/input ownership, defaults, finite catalogs, channel policy and hole/assumption identity are value checks. Separate reconstruction/assignment-shape fallbacks from them; prove computed legal signature declarations export and wrong/foreign/duplicate/missing inputs still refuse at their origin. Do not impose literal policy constructor shapes on runtime values. | fn-174.3; .8 verifies |
| C03/C06/C09 | Monitor compatibility, derivation replacement/refinement/assumption preconditions survive. Validate operations when performed, not only their final result. Prove helper-computed headers/monitors/derivations export without preserving old constructor/builder matchers. | fn-174.4; .8 verifies |
| C11 | Claim kinds, bundle fields, ownership and names survive; generic fold, indexed-helper and builder-tree restrictions do not become runtime shape checks. In particular Claims.scala:675,683,688 needs helper-built bundle/comprehension positives, and the R:897 mixed-bundle negative must either retain its semantic refusal or have an explicit accepted-export counterpart. | fn-174.6; .8 verifies |
| C14/M15/L07 | Sync/member/action/replacement identity and authored section placement survive. Separate selector capture restrictions from composition declaration reconstruction; prove computed legal compositions/syncs export, preserving disabled/replaced-member validation. | fn-174.5; .8 verifies |
| C15/L06 | Typed facts, waiver reasons, bounds and optional-field consistency survive. Fn-174.6/.7 must preserve today's total values and validation in the constructed path; L06 retains authored-literal provenance in source lint. Fn-173.3 implements D8's later retirement of totals, after fn-174, and owns its accepted retirement/negative counterparts. Fn-174 receives no retirement credit and adds no new tree-countability restriction to otherwise legal computed declarations. None is counted deleted yet. | fn-174.6/.7 preserve; .8 verifies; fn-173.3 retires later |
| C17/C18/C19/C20/C21/C24 | Descriptor agreement, typed value kind, finite completeness, identity, exact Duration and field/assignment invariants survive. “Written out”/constructor decoding restrictions need accepted computed-value twins; literal-field duplicate/unknown-field restrictions need rejected twins. Do not replace these with a generic runtime prohibition on helpers or computed values. Source-only restrictions on selector/field spelling remain the M17/M19 capture's concern. Held ScriptRejects retain their sweep obligation. | fn-174.2; .8 verifies |
| C22 | Witness path, facts, repeated-last-class, names, owner and Limits are value invariants. Prove constructed/helper-built witnesses retain them without reimplementing tree extraction; retain negative/location fixtures for repeated-last-class and missing/invalid Limits. | fn-174.6; .8 verifies |
| C27 | Lifting.scala:58 / R:1218 currently rejects IndexedSeq[Query], though IrFile.scala:60–65,207–221 traverses Seq. Prove **that exact spelling** exports every named Query (.6), while .8 still refuses unsupported root/section values and a missing root. Acceptance of one sequence is not permission to accept arbitrary section contents. | fn-174.6/.8 |
| C28/L12 | Expansion.scala:119,137,145,149's cycle/core-parent/indexed-def/arity reconstruction disappears when Scala executes declarations. Preserve initialization cycles (.8 lint), function callback provenance (.1 macro), and each builder's own semantic preconditions (.6 construction). Prove declaration-sugar helper twins export before removing those matchers; R:692's direct-chain restriction stays a located source rule (L12) unless that exact legal stored-builder spelling is proved accepted. | fn-174.6; .1/.8 shared contracts |
| L01/L02/L03/L05/L08/L10 | Keep actual name/placement/vocabulary/order/cycle restrictions in source lint. A rule about the old lifter's ability to find a constructor is not automatically an authoring rule; each owner must show accepted legal runtime spellings before removing such a fallback. Source policy cannot be silently relaxed by declaration discovery. | Row owners; fn-174.8 verifies |

## Helpers, delegation and operational refusals

These paths do not add direct sites to the census. They expose alternative reasons and sequencing
that a literal-call inventory alone would miss.

| Path | Actual branches or callers | Ledger obligation |
| --- | --- | --- |
| `Context.misplacedBlock`, sink `Context.scala:72` | `Context.scala:62`; `Expressions.scala:123,493`; `Declarations.scala:1150` | M10. Four entry points must retain section/block placement checks. |
| `Expressions.wrongLevel`, sink `Expressions.scala:877` | `Expressions.scala:490` | M02. A called step producer inside a predicate/precondition is still refused. |
| `Expressions.unnamed`, sink `Expressions.scala:886` | `Expressions.scala:560,638` | M02. Both multi-step lists and list concatenation are checked. |
| `Expressions.namedCopy.refuse`, sink `Expressions.scala:827` | `Expressions.scala:838,843,845` | M02. Too many steps, a non-written single step, and opaque step production are separate branches. |
| `Capabilities.factFamilies.refused`, sink `Capabilities.scala:108` | `Capabilities.scala:117,131,141` | C15. Missing finite machine fact catalog, nonliteral family shape, and empty/incomplete/duplicate/noncontaining family are separate reasons. The last branch combines four value predicates. |
| `Gaps.resolve.refuse`, sink `Gaps.scala:420` | `Gaps.scala:433,434,539` | C16. Wrong realization placement, missing realized machine, and unsupported subject/placement pairing. |
| `Expansion.expanded` exception wrapper | `Expansion.scala:412` catches a nested LiftError and re-emits sugar definition context | M06. Preserve both sugar/capture context and the original refused tree location. |
| Capability typed-fact wrapper | `Capabilities.scala:110–113` catches `literalValue` refusal | C15. Retain the caller's constant typed-fact restriction; accepting a general literal is not equivalent. |
| Automatic-total wrapper | `Capabilities.scala:1234,1239` | C15. Missing Scenario and inability to count are separate; the latter wraps recursive/unknown/unsupported catalogs and overflow at 1264,1298,1303,1305. |
| Driver exception aggregation | `Lift.scala:140,142–144`; gap accumulation at 195–199; output abort at 355–360 | .8 must aggregate root/file refusals and write no IR after any failure. These catch/rethrow paths are plumbing, not new refusal kinds. |
| Source lint delegation | `Lift.scala:104–105` calls Order/Subset; Order calls Structure and Conventions at `lint/Order.scala:75–78` | L. Retain pre-construction ordering, structure, source conventions and admitted-source reach. |
| Marker delegation | `Lift.scala:135–139` calls Markers and unrefuted | L. Fault/negative-control markers remain source/metadata checks; marker contract is outside the direct exporter-call count. |
| Temporal module policy | `model/temporal/export/TemporalPlugin.scala:38–72` returns import-layer decisions; `testModule` at 75–86 feeds test ownership | L under .8, with the kit seam coordinated with .2. `lint/Conventions.scala:356–386,433` turns false decisions into located refusals. TemporalPlugin having zero direct fail calls does not mean it has no refusal policy. |


The first six helper families have 16 explicit calls to six counted sinks; their alternatives
exceed six branches. Preserve nested sugar error context and its original tree location, typed-fact
caller restrictions, and automatic-total wrapper checks through fn-174. Fn-173.3 later owns D8's
total retirement and the corresponding fixture disposition.

Other explicit operational checks: `Switches.scala:26` rejects unknown UMPIRE_LIFT_RULES names;
`Lift.scala:305–310` rejects CLI misuse, `326–328` empty roots and `367` an internal
nothing-lifted invariant. Fn-174.8 retains the appropriate driver/configuration failures.
`Ended(1)` at 360 transports recorded errors. Reflection/I/O exceptions and collection preconditions
are not additional authored direct refusal sites, but .8 must wrap construction failures with the
named root/declaration and source origin.

## Go predicates: retained independent admission

G01's outcome is construction, not Go-only. Its existing Go predicates and negatives are:

| Scala site | Go enforcing source | Go negative fixture |
| --- | --- | --- |
| Declarations.scala:1497, channel in two state fields | tools/umpire/ir/validate.go:397–402, typeDecl | tools/umpire/ir/admission_test.go:530–532 |
| Declarations.scala:1505, bound channel absent from state | tools/umpire/ir/validate.go:716–730, holds | tools/umpire/ir/admission_test.go:533–538 |
| Declarations.scala:1511, lossy channel without loss binding | tools/umpire/ir/validate.go:732–737, holds | tools/umpire/ir/admission_test.go:539–542 |
| Declarations.scala:1518, reliable channel with loss binding | tools/umpire/ir/validate.go:229–230, action | tools/umpire/ir/admission_test.go:543–545 |

`validate.go:25–87` invokes these admission families, including realization.Admit. Other read
predicates do not justify dropping Scala author rules: channel/catalog bounds at
`validate.go:892–921,1081–1113`, finite-member/collision checks at
`tools/umpire/interp/machine.go:19–75`, and exact Duration delegation at
`tools/umpire/realization/duration.go:17–28`. Go sees lowered IR, not unused Scala declarations,
authored spelling, discarded eager arguments or malformed disabled bindings omitted from IR.
Go text and location are not the current Scala diagnostic contract.

## Static and folded-path contract

| Retained obligation | Source evidence and current coverage / gap | Implementer; final verification |
| --- | --- | --- |
| Initialization lint before discovery | Lift.scala:101–108; lint/Order.scala:15–40,67–78,395–397,545–550. IrFile.discover at dsl/IrFile.scala:101 initializes owners. Order does not follow defs called during initialization (Order.scala:40–41), so null/half-made values still need a located construction failure. Order.test.scala and LintStages.test.scala are source evidence. | fn-174.8 |
| Every effect definition, including unused | Declarations.scala:553–557 calls givesNoEmpty for machine and facet effect defs. 1217–1236 visits **both If branches and every Match result**. R:1065,1161 covers used examples only; unused/folded-away/all-result fixtures remain gaps. A caller macro or runtime selected-path check cannot replace this. Source lint is L13's chosen home; it keeps the existing direct-result traversal scope (it does not newly follow helper bodies). | fn-174.4 definition-site lint; .8 text/line contract |
| Disabled target checks before suppression | Declarations.scala:712–720 and facetRules at 819 retain qualification, action resolution, facet ownership, duplicate and disabled/fired conflicts even though no executable step is emitted. Facets.test.scala:190–230 asserts ordinary/timer/fault/disabled forms. | fn-174.4; .8 |
| Every authored function branch and effect statement | Expressions.scala:537–553 lifts both If arms/all Match arms before remainder transformation. Syntax.scala:432–497 scans nesting/read-after-write before copy reconstruction. Folding an unreachable condition must not erase these checks; dedicated folded-away fixture coverage is not established here. | fn-174.1/.4; .8 |
| Strict sugar arguments before substitution | Expansion.scala:350–412 preserves by-value evaluation once, in caller order, even when an argument is unused, conditional or repeated. DefinitionExpansion.test.scala:290–337 asserts those Hole reads; 348 checks an opaque producer. Preserve by-name short circuit separately. | fn-174.1; .8 |
| Derived/intermediate operations | Declarations.scala:1311–1485 and Compositions.scala:292–327 validate replaced action identity, refinement/assumptions and resulting channels. A final object cannot recover a bad overwritten intermediate operation; construction must validate each operation. | fn-174.4/.5; .8 |
| Excepted capabilities and optional fields | Capabilities.scala:403–432,674–691,879–890 checks reasons/duplicates/bounds/optional combinations before expansion discards or replaces values. A removed Property or absent function does not waive author validation. | fn-174.7; .8 |
| Failure of one root suppresses all sibling output | Lift.scala:140–144,195–199,355–360 accumulates errors before writing. dsl/IrFile.scala:87–92 forces bindings but supplies no complete located root-failure wrapper. Collect failures, report root/declaration/origin, publish nothing after any failure; a dedicated multi-root construction/line fixture remains required. | fn-174.2 establishes driver; .8 full contract |

Current constructors only demonstrate partial coverage: `dsl/Action.scala:102–112` defaults;
`dsl/Channel.scala:87–94` capacity/duplicates; `dsl/Syntax.scala:695–748` targets/conflicts;
`dsl/Facet.scala:56–67` ownership; `dsl/IrFile.scala:41–45,75–84` instance/name distinctions.
None proves all exporter checks redundant. Paths beginning `dsl/` and `lint/` here resolve under
`model/framework/`.

The memory `.flow/memory/bug/integration/consolidated-extractor-dropped-a-2026-09-27.md` records a
Go extractor's lost nonempty-field check. It motivates comparing full rejection sets; it is not
evidence of a Scala folded-path fixture.

Fn-174.1 also adds the cross-compilation-unit `@lifted`/jar `@sugar` rule and warm-server helper
edit regression test. It has no existing direct-site equivalent and is not counted among 617.
Fn-174.8 migrates actual active/held contracts and checks their messages and positions, resolves
every gap or explicitly retains it as sweep-held, and updates the ledger with implementation
evidence. This census does not measure the plan's 331 fixtures or claim all branches covered.

## Verification of this inventory

The fn-141.6 verification independently reproduced the production census, compared every exact
inventory site against it, confirmed no omissions/duplicates/extras and reconciled the outcome
and per-file totals. Cited source and fixture paths were checked for existence; historical line
adaptations and coverage gaps remain explicit. `git diff --check` is the documentation gate.
No Scala/Go build, fixture execution, independent implementation review or source-batch closure
is claimed. Those gates remain with the owner at the complete Scala batch boundary.
