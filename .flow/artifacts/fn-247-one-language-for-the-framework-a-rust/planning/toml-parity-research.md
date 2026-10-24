> Disposable probe retained in `toml-parity-probe.tar.gz` (SHA-256 `64ff580590a17e147c549cb8da2944799046cad68ca392a062ca49e33768efdf`). This is research evidence; no project feature admission or compatibility implementation is claimed.

# fn-247.2 TOML acceptance parity research

The approved `toml 1.1.8+spec-1.1.0` has no strict TOML 1.0 mode. A lexical guard can reject the identified 1.1 syntax additions, but that alone cannot preserve Python 3.14.4 `tomllib` acceptance. The approved feature set also imposes a depth-80 limit that Python accepts beyond, and the usual Rust value conversion loses Python's large-integer domain. Full parity under the exact admission remains unproven. This report recommends no implementation or dependency change.

Research used the installed package source and official versioned specifications. The initial investigation used source inspection and Python probes. A subsequently authorized disposable Rust API experiment is recorded at the end of this report. No project source or Cargo manifest was changed, and no package was installed. Claims outside that experiment remain source evidence.

## Exact API and admission

The workspace lock identifies the requested version and checksum `20489e00e4d8741d6be680764cc12e270655e375a20d1011e844a9c3379e678d`. [workspace lock](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/Cargo.lock:776)

`Deserializer::parse(&str)` immediately calls `DeTable::parse`; deprecated `new` delegates to `parse`. Neither accepts options. `DeTable::parse` and `parse_recoverable` create the same source and parser; recovery collects errors rather than selecting a grammar. [Deserializer API](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/de/deserializer/mod.rs:29) [DeTable API](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/de/parser/detable.rs:16)

The complete feature list has `debug`, `default`, `display`, `fast_hash`, `parse`, `preserve_order`, `serde`, `std`, and `unbounded`. No version/extension feature exists. The approved `default-features=false`, `std/parse/serde` configuration uses `RecursionGuard` with `LIMIT=80`. `unbounded` removes this limit but is outside the stated feature admission and would still not select 1.0 or reproduce Python's own recursion behavior. The lower-level parser's exposed entrypoint takes tokens, an event receiver and an error sink, with no grammar version parameter. That transitive crate is also outside direct-library admission. [complete Cargo features](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/Cargo.toml:91) [fixed parser recursion limit](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/de/parser/mod.rs:28) [lower-level parse_document](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml_parser-1.1.5+spec-1.1.0/src/parser/document.rs:18)

## Complete identified grammar changes

The official [1.1 changelog](https://raw.githubusercontent.com/toml-lang/toml/1.1.0/CHANGELOG.md) lists four syntax additions. Comparing the complete versioned [1.0 ABNF](https://raw.githubusercontent.com/toml-lang/toml/1.0.0/toml.abnf) and [1.1 ABNF](https://raw.githubusercontent.com/toml-lang/toml/1.1.0/toml.abnf) confirms the following acceptance changes.

| Change | Required compatibility check |
| --- | --- |
| `\e` escape | Reject active escapes in basic strings, multiline basic strings and basic quoted keys, including table-header/dotted keys. |
| `\xHH` escape | Same lexical contexts; do not reject the literal characters in literal strings, comments, or an escaped backslash. |
| Optional seconds | Require seconds in each local time, local datetime and offset datetime token. A quoted string resembling a datetime remains a string. |
| Inline-table structural newlines, comments and trailing commas | Reject these at each inline-table frame. Permit newlines/comments inside a nested array value and newlines inside multiline string values, both already legal in 1.0. Permit array trailing commas. |

The ABNF additionally corrects the comment upper ASCII bound from 7F to 7E. Both prose specifications prohibit DEL in comments, and actual Python rejects it, so this is no new Python incompatibility to permit. Other ABNF edits refactor equivalent productions. The changelog also clarifies comment independence, Unicode/UTF-8, dotted-key table creation, multiline-literal newline normalization, fractional precision, and numeric size flexibility. These need semantic verification; the changelog alone cannot prove two implementations equivalent.

A possible syntax guard therefore needs string/escape/comment states and a nesting stack that distinguishes arrays from inline tables, plus scalar-token boundaries. Searching globally for `\e`, newlines between braces, or `,}` would reject valid 1.0. Inspecting only the decoded tree loses escape spelling and inline trailing commas. The installed parser accepts inline comment/newline tokens directly and decodes dates with optional seconds. [inline-table token handling](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml_parser-1.1.5+spec-1.1.0/src/parser/document.rs:960) [datetime parsing and bounds](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml_datetime-1.1.2+spec-1.1.0/src/datetime.rs:310)

## Additional acceptance differences beyond the version delta

1. **Depth is a concrete counterexample.** Python accepted `v=` followed by 100 opening brackets, `0`, and 100 closing brackets. The approved Rust parser applies depth 80 to nested arrays/inline tables even through public `DeTable`. A reject-only guard cannot repair this lost acceptance. [fixed parser recursion limit](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/de/parser/mod.rs:28) [RecursionGuard](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml_parser-1.1.5+spec-1.1.0/src/parser/event.rs:363)
2. **Integer representation matters even in ignored fields.** Python accepted `v=184467440737095516160` and `v=1606938044258990275541962092341162602522202993782792835301376` (2**200). `toml::Value::Integer` is i64. Serde `deserialize_any` tries integer widths through u128, then errors. Public `toml::de::DeTable`/`DeValue` retain `DeInteger` textual digits and radix and can avoid that conversion for unused fields. This is a real existing API, not a proposed version selector. Python's decimal integer conversion also depends on its configured digit limit, locally 4300. [Value representation](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/value.rs:28) [Serde integer conversion](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/de/deserializer/value.rs:90) [public DeInteger representation](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml-1.1.8+spec-1.1.0/src/de/parser/devalue.rs:19) [Python number conversion](/usr/lib/python3.14/tomllib/_re.py:110)
3. **Date validation differs.** Python rejects year zero and second 60. Installed Rust datetime code parses a four-digit year without excluding zero and accepts seconds through 60. Rejecting those values throughout the parsed tree is needed even when helpers ignore them. Rust and Python both enforce offset hours <=23 and minutes <=59. Fractional seconds are truncated to six digits by Python and nine by Rust, which matters if values become observable. [datetime parsing and bounds](/home/agent/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/toml_datetime-1.1.2+spec-1.1.0/src/datetime.rs:310) [Python datetime regex and conversion](/usr/lib/python3.14/tomllib/_re.py:17)
4. **Input decoding is part of the helper contract.** All three helpers use `Path.read_text()` before `tomllib.loads`. Python text reading normalizes CR, CRLF and LF. Direct `tomllib.loads('v=1\rw=2')` refuses, while text-reader normalization followed by loads accepts it. A Rust `read_to_string` plus direct parse does not implement that preprocessing. Python `loads` itself also replaces CRLF. Encoding/error policy needs preservation for the actual runtime, rather than assuming `tomllib.load` binary semantics. [Python loads normalization](/usr/lib/python3.14/tomllib/_parser.py:130) [lock helper entry/error path](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-lock.py:56) [freshness helper reads](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-freshness.py:19) [rustc sysroot lock read](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-rustc.py:43)
5. **Error rendering is independent of grammar.** Python constructs line/column or end-of-document messages and the helpers add their own failure prefixes and return 1. Using Rust's `Display` error cannot be assumed byte-equivalent. For inputs with several faults, first-error ordering also matters if R2 requires exact diagnostic bytes. [Python error formatting](/usr/lib/python3.14/tomllib/_parser.py:100) [lock helper entry/error path](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-lock.py:56) [freshness helper reads](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-freshness.py:19) [rustc sysroot lock read](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-rustc.py:43)

These findings establish insufficiency of a simple guard; they are not an exhaustive proof of every implementation-level semantic difference. Dotted-key sealing, duplicate keys/tables, floating overflow and special values, Unicode, huge integers, and nesting/resource limits still need differential coverage.

## Actual helper domains

`guest-lock.py` parses the entire supplied workspace Cargo.lock, then reads package coordinates/checksums and lock version to hash closures. `guest-freshness.py` parses entire mise.toml and rust-toolchain.toml before comparing selected tool pins. `guest-rustc.py` parses the pinned sysroot library Cargo.lock only in its build-std registry-classification path. Unknown fields are still parsed. A typed struct that skips or narrows unknown values can silently alter full-document validation. Keeping ordinary generated lockfile shapes is insufficient for the requested whole-grammar contract. [lock helper entry/error path](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-lock.py:56) [freshness helper reads](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-freshness.py:19) [rustc sysroot lock read](/Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/guest-rustc.py:43)

## Bounded experiment results

Executed with installed Python 3.14.4, recursion limit 1000 and integer digit limit 4300, using stdlib only and no helper execution or files outside this report. `\n` below denotes a real newline in the tested input.

| Input | Observed tomllib result |
| --- | --- |
| `v="\e"`, `v="\x41"`, `"\x41"=1` | Refuse, unescaped backslash diagnostic. |
| `v={\na=1}`, `v={a=1,}` | Refuse, invalid initial key character. |
| `v={a=1 # comment\n}` | Refuse, unclosed inline table. |
| `v=14:15`, `v=2010-02-03T14:15`, `v=2010-02-03T14:15Z` | Refuse, expected newline/end after statement. |
| `v='\e\x41'`, `v="\\e\\x41"` | Accept literal backslashes. |
| `v={a=[1,\n2,]}` | Accept. |
| `v={a="""hello\nworld"""}` | Accept. |
| The two large integers and 100-deep array above | Accept. |
| `v=0000-01-01`, `v=23:59:60`, offset `+24:00` | Refuse. |
| DEL in comment, duplicate key `a=1\na=2` | Refuse. |

The initial Python results above are independent of the subsequently authorized Rust experiment below.

## Smallest supported route and remaining decision

For only the 1.1 grammar delta, a context-aware linear scan plus the existing full parser is a bounded approach. For the required **all-input Python acceptance/refusal parity**, that approach is insufficient under the exact approved features. Public `DeTable` can retain large integers; it cannot bypass depth 80. Date checks and read-text normalization are additional mandatory work. This is a compatibility layer with its own differential test obligation, beyond a dependency swap.

The lowest-risk immediate course is to retain the existing Python behavior until the compatibility route is resolved, and record the Rust conversion as unfinished. Under unchanged admissions, a faithful Rust port of the existing Python validator is conceptually possible but amounts to maintaining a full parser and error model; this research does not establish it as small or preferable. Splitting deep inputs into independently parsed fragments or rewriting values to fit the library would itself need semantic proof for table scopes, duplicate definitions, all value forms and diagnostics. Neither is an established API solution.

Before claiming completion, a selected route must pass a differential corpus covering every grammar form and extension, the concrete counterexamples above, ignored-field validation, input normalization, numeric/date edges, and the existing R2 diagnostic contract. The precise scope of byte-identical errors and Python resource-limit behavior must be established from the existing acceptance requirements. No such parity proof, feature expansion, source-copy licensing decision, or design approval is supplied by this report.


## Authorized disposable Rust API experiment

After the conductor sent BUILDGO, the scratch driver compiled and ran both configurations with Cargo 1.99.1 (`5f94df478`, 2026-08-27) and rustc 1.99.1 (`58cf8f9c3`, 2026-10-13), host `aarch64-unknown-linux-gnu`. Both builds and both runs exited 0. The version commands and metadata also exited 0. This is API behavior evidence, not a benchmark or a project feature admission.

The probe started with a copy of model/Cargo.lock, resolved offline, checked every resolved third-party name/version/source/checksum against the current project lock, and found no additions. Both subsequent builds used `--offline --locked`, `CARGO_BUILD_JOBS=1`, and the isolated `.flow/tmp/fn247-toml-probe-target`. Cargo warned that build metadata in the manifest's version constraint is ignored; the resolved lock and exact checksum subset check establish the actual `1.1.8+spec-1.1.0` package. The source uses the public `toml::de::DeTable::parse` API and reads public `DeInteger::as_str`/`radix` and datetime fields. No crate source was copied.

| Case | Approved std/parse/serde | Scratch plus unbounded | Python 3.14.4 |
| --- | --- | --- | --- |
| Array depth 80 | Accept | Accept | Accept |
| Array depth 100 | Refuse at column 83, maximum recursion depth | Accept | Accept |
| Integer 2**200 | Accept, exact raw decimal digits and radix 10 | Same | Accept |
| Year 0000 | Accept | Accept | Refuse |
| Second 60 | Accept | Accept | Refuse |
| Time/local datetime/offset datetime without seconds | Accept, `second: None` | Same | Refuse |
| Active `\e` and `\x41` | Accept, ESC and A respectively | Same | Refuse |
| Inline structural newline/comment/trailing comma | Accept | Accept | Refuse |
| Literal/escaped backslashes and multiline array in inline table | Accept | Accept | Accept |
| Duplicate key | Refuse, `duplicate key` with multiline excerpt | Same | Refuse, `Cannot overwrite a value (at end of document)` |
| Timestamp fraction .123456789 | Accept, nanoseconds 123456789 | Same | Accept, microseconds 123456 |

The experiment confirms that public DeTable avoids the large-integer conversion problem and that experimental `unbounded` removes the specific depth-100 obstacle. It does **not** establish full Python equivalence. Both Rust configurations continue to accept the identified 1.1 syntax and date values Python refuses. Removing the Rust limit also needs a defined policy for Python's own recursion/resource limits. Exact diagnostic parity remains separate work.

The smallest next technical route supported by these observations is a compatibility investigation around public DeTable, with an explicit decision about the additional `unbounded` feature before any project use. It would still need context-aware 1.0 syntax validation, Python date constraints and input normalization, lossless handling of observable integers, and differential validation of error semantics. This experiment supplies neither that implementation nor its approval. Under the unchanged approved features, the depth-100 counterexample remains a demonstrated blocker to a reject-only compatibility guard.

Evidence is retained in [probe source](fn247-toml-probe/src/main.rs), [scratch manifest](fn247-toml-probe/Cargo.toml), [runner](fn247-toml-probe/run_probe.py), [commands and exits](fn247-toml-probe/logs/commands.json), [resolver subset receipt](fn247-toml-probe/logs/resolver-subset.json), [approved run](fn247-toml-probe/logs/approved-run.stdout), [experimental run](fn247-toml-probe/logs/experimental-unbounded-run.stdout), [Python reference](fn247-toml-probe/logs/python-reference.json), and [input hashes](fn247-toml-probe/logs/input-hashes.json). Each command has separate stdout/stderr files beside those receipts. Tool versions are recorded in `cargo-version.stdout` and `rustc-version.stdout`.

Approved binary SHA-256 was `62db137fb8f03d8c840ebcafbe7f040d2b0d0cbce1079d2689756182d9f9d604`. Experimental binary SHA-256 was `8f4757daee46ced6ca05f7a2952afa7c3d6bd1dac88523766253ceb8d78bc965`. The target currently holds the experimental binary; both hashes and run outputs are retained.
