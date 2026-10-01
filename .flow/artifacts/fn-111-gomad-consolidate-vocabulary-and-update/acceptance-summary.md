# fn-111 documentation acceptance summary

Derived from `guide-audit.json` (sha256 `93e9b2aec946e90e35da04e3b746519792eb227d98b4a1b3a5153f2f94e96ff5`), `vocabulary-audit.json`, and `documentation-audit.json` in this directory. Regenerate after any edit: the hashes below bind an uncommitted working tree.

## Binding

- Baseline revision: `29917069e089dc0739ec091b18e99161245b9bd5`. HEAD below the working tree: `d4d800fb473f008fa4a51a4d8575249f11c9cf3f`.
- Host of the help text and clock probe: `go version go1.27.1 darwin/arm64`, toolchain build key `c0661e38b4e001c8d86912d4ce9e265eb0f33e5f3298015da27a4c97ac4019b9`.
- Inputs read and hashed: 81 files. Differing from HEAD: `.plans/GOMAD_MILESTONES.md`, `tools/gomad3/ARCHITECTURE.md`, `tools/gomad3/CLI.md`, `tools/gomad3/Makefile`, `tools/gomad3/README.md`, `tools/gomad3/SPEC.md`, `tools/gomad3/TUTORIAL.md`.

| File checked | sha256 |
| --- | --- |
| `tools/gomad3/SPEC.md` | `cc96c283ff1131cba5e217ba4773cd4a8d196c90132ee6e47cce62cfe8da533d` |
| `tools/gomad3/ARCHITECTURE.md` | `3690db757882664d6009beac10796187afa6d5538e2a82ae5808bdbb0aa9d85e` |
| `tools/gomad3/CLI.md` | `256833cc3e85f13abcf05afa44ceb80010f8596045e19f1355e17012f82df10a` |
| `tools/gomad3/TUTORIAL.md` | `b56a32f109700b32e02c51b45a442f5831065ca5db6d7b39238fdbef29eeafb5` |
| `tools/gomad3/README.md` | `607ebac7ba7c586695d9095d754ed453a21c43c0383cec87040adf84da59cab7` |
| `.plans/GOMAD_MILESTONES.md` | `3d35c7a72c93e7de6e253bd2bc924f0a5ca6bf0a1a8e466ddf9e5a50bce66215` |

## Commands and results

| Command | Exit |
| --- | --- |
| `python3 verify-guides.py BIN_DIR FLOWCTL` | 0 |
| `python3.14 verify-vocabulary.py` | 0 |
| `python3.14 verify-documentation.py bin` | 0 |
| `git diff --check HEAD -- <six documents> <this directory>` | 0 |
| `flowctl validate --spec fn-111-gomad-consolidate-vocabulary-and-update` | 0 |
| `git diff --check 29917069e0 -- <six documents>` (baseline to current) | 0 |
| `git diff --check HEAD -- <six documents>` (head to current) | 0 |

## What was compared

- Terms and identifiers (task 1): 25 original glossary entries assessed, 24 current concepts retained, 123 semantic identifiers preserved in order (28 in command tables).
- Commands and flags: 29 indexed commands and 67 documented flags exist in the binaries; 135 command examples pass their own command's flag set, placement, value, and cross-flag rules; 31 make targets exist.
- Claim matrix: 88 guide sentences paired with their implementing source line (R4: 28, R5: 53, R6: 5, R7: 2).
- Navigation: 56 local links and fragments resolve; fences are typed and balanced (ARCHITECTURE.md 1, CLI.md 46, TUTORIAL.md 17, README.md 15).
- Platforms: `version.json` and the boundary manifest both name darwin/arm64 and linux/amd64, and each guide names exactly that set.
- Corpus: core 7 workloads; representative Temporal 28 (15 tier 2, 13 tier 3); generated `./tests` 147 workloads, 8 without a choice trace, 12 skipped subtests, all named in the milestones.
- Forward clock: probe built with the pinned toolchain matched every documented reading under `strict` and `forward` (state `observed`).

## Requirement mapping

| Requirement | Evidence |
| --- | --- |
| R1 | vocabulary-audit.json term_assessments (task 1); README Parity Case matrix row |
| R2 | vocabulary-audit.json identifier comparison (task 1); documentation-audit.json identifiers_preserved_in_order; stale_terms |
| R3 | vocabulary-audit.json distinctions (task 1); claim_matrix backend/fidelity and tape rows |
| R4 | claim_matrix R4 rows; clock_probe; platforms; corpus_inventory |
| R5 | examples; make_examples; flag_attribution; claim_matrix R5 rows; explore_classifications; actual_help |
| R6 | claim_matrix R6 rows; roadmap; platforms.guides.TUTORIAL |
| R7 | navigation; fences; stale_terms; claim_matrix R7 rows |
| R8 | head_revision; file_sha256; binary_sha256; whitespace; this mapping |

## fn-109 R9 and fn-105 D5 reuse

Reused, for the current-documentation portion only:

- both supported platforms named consistently with version.json and the boundary manifest (platforms)
- implemented choice replay and exploration and both backends described against source (claim_matrix R4 rows)
- capability support, repeatability, exact replay, and expectation matching kept separate (corpus_inventory, README corpus statements)
- current residual findings recorded: the milestones name every tests.json suite that runs without a choice trace, every suite whose expectation is not qualified, and every skipped subtest (corpus_inventory); skip reasons and owners stay in tests.generator.json and are not compared

Not covered, and still owned by fn-109 R9 and fn-105 D5:

- intentional Go interface changes from fn-109 R4-R7 and R11-R14, which are not implemented
- architecture fitness checks (fn-109 R8 / fn-105 D4)
- any documentation of interfaces that fn-109 has yet to change

## Still open and limits

- Open: fn-105 D12 and D14 replay fixes.
- Open: fn-105 D13 tracing policy and D15 trace capacity.
- Open: fn-109 R9 interface documentation.
- Limit: head_revision identifies the commit below an uncommitted tree; inputs and file_sha256 bind what was actually read, and inputs_differing_from_head lists the uncommitted ones.
- Limit: Statement patterns prove a sentence is present and its implementing line exists; the pairing of each row was judged by reading both.
- Limit: linux/amd64 behavior is taken from source, manifests, and the CI workflow; the probe and help text ran on the host recorded in clock_probe.
- Limit: No qualification workload was executed; manifest expectations are reported as expectations, not as observed qualification.
- Limit: the generated `tests.json` expects the F5/F6 suites `qualified` on linux/amd64 while `temporal.json` and `smoke.json` expect them `intermittent`; this audit changed no expectation.
