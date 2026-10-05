# Task 38 independent source review

Assessment: SOURCE_PROGRESS_COMMIT_ONLY.

The eight admitted source repairs preserve the inspected capability projections,
digest framing and read-error behavior. No actionable introduced defect was
found. Root may commit this source progress and its controls/evidence; task 38
and original acceptance remain open.

## Review identity and conduct

Requested reviewer was `gpt-6.1-sol` at `high`. This fresh host reviewer is in
the Codex family, the same family as the writer. Execution-model metadata was
not exposed, so the requested exact model/effort pin cannot be independently
attested. This review does not satisfy a cross-family formal review requirement.

The host has no tool-enforced read-only mode. Conduct was prompt-enforced.
Only this report was written. No production source, Flow lifecycle, index,
Git history, configuration, worktree, stash, bridge or subagent was mutated or
created. Commands ran to terminal completion before return. The reviewer read
AGENTS.md, Gomad README, milestone guidance, flowctl usage, and the complete
fn-109 parent and fn-109.38 through `flowctl cat`.

## Strengths

- BASE and HEAD are `9f66d05bc8f7f49f18135a53525006198a675036`.
  The tracked diff contains exactly five import aliases and three SHA-256
  write replacements. The additional source is the assigned digest test.
  Existing production logic, comments, fixture bodies and assertions outside
  those eight lines are byte-preserved.
- `target/capability.go:12`, `capability_collection.go:17`,
  `capability_evaluation.go:11`, `capability_golden_test.go:13` and
  `capability_review_test.go:15` explicitly name the existing declared package
  `compatibility`. Import paths, uses, types and public projections stay intact.
- `target/prepared_cache.go:245`, `:257` and `:264` format one record at a time
  using `fmt.Appendf(nil, ...)` and write to the concrete SHA-256 implementation.
  Each prior format, argument and iteration order is unchanged. NUL/newline
  framing, lowercase hex, `absent`, basename selection and `sha256:` remain
  identical. No whole-stream accumulator or impossible error branch was added.
- `target/prepared_cache_digest_test.go:13` contains six top-level tests.
  Cases cover empty overlay; cleaned/sorted original names and both JSON orders;
  replacement files in fresh distinct directories; missing replacement;
  present/absent/empty go.sum; argument order; basename framing; empty file list;
  and nonregular module input. Success checks preserve input bytes and absence.
  `:56` retains ErrNotExist, direct PathError unwrap, operation/path and text;
  `:135` retains the nonregular-file wrapper and cause. Existing canonical and
  projection controls cover complete evidence, nil/empty state, order and
  detached nested storage.
- Retained evidence includes actual unfiltered baseline lint RED (17 issues),
  the superseded Sprintf attempt (12 issues, including three QF1012), and the
  corrected final result (nine inherited errcheck issues). Tool/configuration
  pins and enabled rules were preserved. Baseline literal controls are retained
  on unchanged production, with exactly the same test-file hash as final.

## Independent literal derivation

The reviewer computed SHA-256 with Python hashlib from explicit fixture bodies
and the basename/original-name + NUL + lowercase inner hex + newline frames,
without calling candidate code or its digest helpers. All seven literals match
the tests and literal-vectors.md.

| Input | Independently derived outer SHA-256 |
| --- | --- |
| Empty stream | e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 |
| Multiple overlay replacements | 1db3af338bd4366b249e4f3ae4bf2af122071bad27645a84d6f601516d1d25f6 |
| Present module and sum | cace9d14953fa84d0beac296d7937707921775c4c283615f329c0c4f23fb5674 |
| Absent sum | b9aa172d3a27d4116f21ae146eaf64403e30c4b9e19abb775fb48464dfee4d6d |
| Empty sum | c1c810346073420fefb509e1463b85b7d724297bbc13fca78fa1e71b72434c23 |
| Reversed arguments | 195265fcebde348a5b2bc830009fbab082ee679517c2ed47cd439af3e42c77d1 |
| module.txt basename | 512c10c73c8891cdbc87e147a83fad61c078da38cff9653a8d337404c598b9be |

## Source and evidence bindings

Every one of the 1,038 baseline manifest entries was independently hashed from
`git show BASE:path`; all match. Baseline-plus-tests changes no tracked baseline
input. BASE-to-final changes exactly the six admitted existing files, adds the
digest test, and preserves the other 1,032 entries. The final 1,039-entry
manifest matches the current files. Root/nested module files, runtime overlays,
compatibility pins, profiles, source inventory, task 39 source, generated files
and lint configuration retain their recorded identities.

| Retained source state | Manifest SHA-256 |
| --- | --- |
| BASE | 1e7d577bf1127e061bf59c1f32fe9c156d2a61493a7cfc2421be49bc80342996 |
| BASE plus literal controls | 78485a0ed562cda56cc0f7c13cf4f11aa67d4f294b48de04423420361dcc9c0f |
| Superseded Sprintf candidate | 12449b3598149c3fe4290a270c27cc66c4706f8e45d9dfd9c680c3e874f10ad6 |
| Final Appendf candidate | a5eb12d3228b6fe5b829532d784229a2c8bdcfe7df61d53f81cba5646dbec7af |

All 12 command receipts were checked against their raw log SHA-256 and equal
before/after hashes mapped to these retained manifests. The seven final source
hashes in source-bindings.md match current files. Tool hashes independently
match `acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc`
for golangci-lint, `db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc`
for errortype, and `2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43`
for .github/.golangci.yml.

The Makefile generation input lists, protocol generation's explicit livecap
inputs and version descriptor exclude all changed sources. No generator,
schema, template, overlay or generated output changed. Generator execution was
therefore unnecessary for this bounded review.

## Fresh verification

Working directory was `tools/gomad3`. Cached stock Go 1.27.1 linux/arm64 was
first on PATH. Every execution used `GOENV=off GOWORK=off GOTOOLCHAIN=local
GOPROXY=off GOSUMDB=off GOFLAGS=` and unset `GOMADSEED`, `GOMAD3_CHILD_SEED`,
`GOMAD3_SEED`. No downloads, broad/full suites or native qualification ran.

| Command | Exit | Result |
| --- | --- | --- |
| `go test -v -count=1 -tags test_dep ./target -run 'TestPreparedCacheDigest\|TestCapabilityReviewGoldenCanonicalBytes\|TestCompatibilityPackProjectionPreserves'` | 0 | 10 top-level tests, 20 leaves pass; 0.042 package seconds |
| `go test -v -count=1 -tags test_dep . -run '^TestPackageArchitecture$'` | 0 | One test executes and passes; 0.645 package seconds |
| `/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target` | 1 | Exactly nine inherited errcheck findings; zero introduced findings |
| `go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype ./target` | 0 | No diagnostics |
| `gofmt -l` on all seven Touches, with empty-output check | 0 | No formatting differences |
| `git diff --check` | 0 | No whitespace defects |
| Repository-root `sha256sum -c --quiet` on final manifest | 0 | All 1,039 source/configuration entries match throughout review |

A checksum probe accidentally resolved repository-relative manifest paths
from the module directory and reported two mismatches plus no verified files.
The reviewer corrected the working directory. Repository-root verification
before the controls and after every gate matched the full manifest; this was
a probe-path error, not a candidate source mismatch. Every later gate used
repository-root before/after verification.

## Findings and required fixes

Critical: none introduced.

Important: none introduced.

Minor: none introduced.

Required fixes for this source-progress commit: none.

Actual target lint remains red at `target/adapter_source_set.go:36`,
`target/target.go:890`, `:913`, `:922`, `:926`, `:941`, `:945`, `:949` and
`target/target_test.go:518`. These exact nine unchanged cleanup findings belong
to task 39. They remain requirements and cannot be treated as a passing gate.

## Remaining qualifications

Root owns the source-progress commit and subsequent writer admission. Original
R18/R19, task 21, relevant tasks 9/10/11/19/23 and predecessor acceptance stay
required and open wherever unproved. Matched first-baseline identities and
fixed bytes, complete/full-host/default/native integration, functional smoke,
affected-consumer/qualification, 10-job/100-job bounded controls, static coverage
of both supported source sets, and formal review still need their appropriate
source-bound evidence. These developmental Linux/arm64 controls establish only
the observations above. Native Darwin qualification was not executed. Remaining
native Linux execution stays with fn-128 under the owner transfer and is
nonblocking for this source owner. No SHIP, task-done, merge-ready, full/native
qualification or whole-Gomad lint-green conclusion is supplied.
