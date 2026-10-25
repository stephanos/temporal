# Gomad: general-purpose output predicates replace the hardwired softassert oracle

## Conversation Evidence

> user: "related to fn-191 maybe - it mentions softassert. but we don't want gomad to have tied to that temporal-specific package. can we provide some kind of general-purpose fail conditions to a gomad run? like undesired log lines or such? should that match on the JSON via regex or sth?"
> user: "yes file"

## Goal & Context

Gomad's only log-derived failure oracle is `campaign/soft_assertions.rs`, the Rust port of the Go softassert observer. It hardwires three facts about Temporal's `common/softassert` package: the `failed assertion: ` message prefix, the `failed-assertion` / `FailedAssertion` tag keys, and a gate that only evaluates the oracle when the prepared target's provenance kind is `go-test`. The one opt-out, `--soft-assertion-report-only`, is named after the package all the way into the campaign plan, the prepared-target recipe (`soft_assertion_report_only`) and the ledger's `soft_assertions` field. Gomad is a general deterministic-simulation host; a workload that is not Temporal has no way to say "this log line is a bug", and Temporal cannot add a second rule (for example an `ERROR`-level record from a named component) without a host change.

Everything else in that file is already generic and worth keeping: stream splitting, JSON-record detection behind the `source:line:` prefix, strict field parsing, message deduplication, the 4096-byte message bound, and the rule that truncated or non-UTF-8 output is inconclusive rather than a pass.

This spec replaces the hardwired oracle with a declared list of **output predicates** carried by the prepared target. Temporal's softassert rule becomes one predicate preset that the Temporal launcher declares; the host knows nothing about the package. Plan context: `.plans/GOMAD_FIND_BUGS.md` §"softassert is already `Always`".

## Architecture & Data Models

**Predicate schema (R1).** A prepared target carries `output_predicates: Vec<Predicate>` (typed per fn-245 discipline, `deny_unknown_fields`). Each predicate has:

- `name`: stable identifier, unique within the list, `[a-z0-9-]{1,64}`. Appears in reports and in the failure signature.
- `streams`: subset of `{stdout, stderr}`; default both.
- `format`: `json-line` or `text`.
  - `json-line` reuses the existing record detector (bare `{...}` or `source:line: {...}`) and strict field parser. Lines that are not JSON records are ignored by this predicate. A line that *looks* like a record for this predicate (contains any key or prefix the predicate names) but fails to parse is inconclusive, as today.
  - `text` applies to the trimmed raw line.
- `match`: one or more conditions, all required (conjunction):
  - `field_equals {key, value}` for a JSON boolean, number or string literal (today's tag `== true`).
  - `field_prefix {key, prefix}` for a JSON string field (today's `msg` prefix).
  - `field_regex {key, pattern}` for a JSON string field.
  - `line_contains {text}` and `line_regex {pattern}` for `text` format.
- `identity`: how the static failure message is derived from a hit. One of `field {key, strip_prefix?}`, `capture {group}` (a named group from the predicate's regex), or `literal {text}`. The derived message is bounded to 4096 bytes; an empty or over-long message makes the hit inconclusive, exactly as today's `static_message`.
- `severity`: `fail` (a hit is a target failure) or `report` (a hit is recorded but does not fail the execution). This replaces the single `soft_assertion_report_only` flag.

Regexes use the `regex` crate (linear-time, no backreferences) and are compiled once at preparation; a pattern that fails to compile or exceeds the crate's default size limit is refused before execution. Regexes match a parsed field value or a raw text line, never the serialized JSON record, so key order, escaping and the `source:line:` prefix cannot change a match.

**Report (R2).** The oracle result is a typed `OutputPredicateReport { hits: Vec<Hit>, inconclusive: bool }` with `Hit { predicate: name, severity, messages: sorted deduplicated Vec<String> }`. It replaces the `soft_assertions` field on execution receipts, ledger rows, `inspect` output and progress lines. The failure signature derived from a `fail` hit covers the predicate name and the sorted messages, so two seeds that trip the same rule with the same static message share a signature and the failure budget and corpus features behave as they do for soft assertions today. A `report` hit never changes the outcome class. `inconclusive` is set when any stream is truncated, not UTF-8, or a predicate-relevant line fails to parse; an inconclusive report with no hits is never a pass for that oracle.

**Identity binding (R3).** The predicate list is part of the prepared-target recipe and campaign plan, in place of `soft_assertion_report_only`. It stays outside the compilation key (as `soft_assertion_report_only` already is) and inside the full provenance digest, so changing a predicate re-prepares nothing but refuses resume, shard execution and replay against a plan prepared with a different list.

**Softassert preset (R4).** Temporal's launcher (the `cmd/gomad` side and the `go-test` recipe builder) declares the predicate the host used to hardwire:

```
name: soft-assertion, streams: [stdout, stderr], format: json-line,
match: any-of( field_equals{failed-assertion,true}, field_equals{FailedAssertion,true}, field_prefix{msg,"failed assertion: "}, field_prefix{message,"failed assertion: "} ),
identity: field{msg, strip_prefix "failed assertion: "}, severity: fail
```

Because `match` is a conjunction, the preset is expressed as the equivalent predicates sharing one `name`, or `match` gains a one-level `any_of`; the implementer picks whichever keeps the schema smaller and records the choice. Text-format fallback for a bare `failed assertion: ` line outside a JSON record is kept as a second preset entry. `--soft-assertion-report-only` becomes sugar on the Temporal launcher that sets the preset's severity to `report`; the host CLI gains `--output-predicates FILE` (campaign, explore, run-guest) and no package-specific flag.

## API Contracts

- Host CLI: `--output-predicates FILE` accepts a JSON file in the schema above; unknown fields, duplicate names and uncompilable regexes are refused with status 2 before any execution. Absent file means no log oracle.
- Temporal launcher: unchanged user-visible behaviour for `go-test` workloads; `--soft-assertion-report-only` keeps working and is documented as a preset shortcut.
- Receipts, ledger, `inspect --json` and progress JSON carry `output_predicates` in place of `soft_assertions`. CLI.md documents the schema and the report.
- `cargo test --workspace`, `make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism lint-rust` pass after every task.
- New dependency: `regex` (owner approval required; it is the only linear-time regex engine in the Rust ecosystem that is widely vetted).

## Edge Cases & Constraints

- Hostile or huge guest output must not change oracle complexity: `regex` is linear in input, the per-line size bound and the 4096-byte message bound stay, and the existing output-capacity classification (capacity errors precede finalization errors) is unchanged.
- A predicate must be able to express today's preset exactly; the retained scenario fixtures with handled assertion reports (`campaign-scenario-reference-check.py`, `TestSoftAssertion*` declarations in `cycle/plan/declarations.rs`) are the golden proof, regenerated once per the compatibility policy with the field rename.
- A `report`-severity hit on an otherwise successful run keeps outcome class `success`; a `fail` hit on a run that also trapped keeps the trap as the primary outcome and records the hit, as the current oracle does.
- Predicates never make a truncated or undecodable stream pass: inconclusive propagates to the ledger and to campaign summaries as it does today.
- A `must appear` (coverage) predicate is deliberately out of scope; that is fn-191's `Sometimes`, which needs the catalog and semantic channel to be meaningful.
- This spec touches `campaign/`, `cli/`, `evidence/record.rs` and the Temporal launcher. It must land after fn-240 (crate layout, `clap`) and fn-245 (typed receipts) so the report and recipe are typed once and in their final crate, and before fn-191.4, which adds campaign flags beside this oracle.
- No CI run, PR or push is authorized by this spec.

## Acceptance Criteria

- **R1:** The host evaluates only declared predicates: no string literal `failed assertion`, `failed-assertion`, `FailedAssertion` or `soft_assertion` remains in `tools/gomad/rust` outside the Temporal launcher preset and tests. An invalid predicate file (unknown field, duplicate name, bad regex, over-size pattern) is refused with status 2 before execution, with a message naming the predicate. Errors: a predicate that compiles but whose regex is applied to serialized JSON rather than a parsed field is a finding.
- **R2:** For the softassert preset, every retained scenario fixture and `TestSoftAssertion*` control produces the same hit set, the same inconclusive verdicts and the same failure grouping as before the change (field name aside). Two executions with the same predicate name and sorted messages share a failure signature; the same messages under a different predicate name do not. A `report`-severity hit leaves the outcome class unchanged. Errors: a dynamic field leaking into the identity, or an inconclusive report counted as a pass, is a finding.
- **R3:** Changing, adding or removing a predicate leaves the compilation key unchanged and changes the full provenance digest; resume, `execute-shard` and retained replay refuse a plan whose predicate list differs. Errors: a mismatch admitted silently is a finding.
- **R4:** A non-Temporal `go-run` fixture with a `text`/`line_regex` predicate and a `json-line`/`field_equals` predicate fails on a hit, reports on a `report` hit, dedups identical messages, and groups two seeds with the same message under one signature; the fixture is a golden under `testdata/`. Errors: any of the four behaviours missing is a finding.

## Boundaries

- No coverage ("sometimes") predicates; fn-191 owns coverage.
- No matching on evidence other than guest stdout/stderr (no network, storage or trace predicates).
- No change to outcome classes, failure policy semantics, retention bounds or output capacity rules.
- No new host flag that names a workload-specific package.
- No CI run, PR or push.

## Decision Context

Owner, 2026-10-25: Gomad must not be tied to Temporal's `softassert` package; fail conditions should be a general-purpose feature of a Gomad run. Design choices recorded here: match on parsed JSON fields (or raw text lines) rather than regex over serialized JSON, because serialized shape is brittle and the parser already exists; every predicate must declare how its static identity message is derived, because the failure signature feeds dedup, the failure budget and corpus features; the list is bound into plan identity because replay and resume already refuse any other oracle-affecting input change. Rejected: a "must appear" predicate as a cheap `Sometimes`, because fn-191 records that without the semantic channel a Sometimes is just a log line.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Declared predicate schema, validation and evaluation replace the hardwired oracle. | .1 | — |
| R2 | Typed report, signature grouping, severity, inconclusive preserved; softassert goldens unchanged. | .1, .2 | — |
| R3 | Predicate list bound into recipe/plan identity; mismatches refused. | .2 | — |
| R4 | Temporal preset on the launcher; non-Temporal golden fixture. | .3 | — |
