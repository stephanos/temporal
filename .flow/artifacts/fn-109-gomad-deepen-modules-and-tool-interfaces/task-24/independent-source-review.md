# Independent task24 source-progress review

Ready for SOURCE PROGRESS COMMIT: yes. No actionable introduced Critical, Important or Minor finding was found in the frozen configuration and regression test. Root and Gomad qualification remain failing, and task21, original R18/R19, formal review and both native gates remain open.

## Scope and method

This is one fresh manual independent source-progress review against fn-109 task24, the original fn-109 spec, source admission and MILESTONES. It carries no formal flow-next implementation-review, SHIP, native qualification, task-completion, merge or commit verdict.

BASE_SHA and HEAD are both `ce80d2425cf34da103939b5aa23f90bde1c2092f`. The committed range is empty. I reviewed the actual working diff in `.github/.golangci.yml` and the complete untracked `cmd/tools/lintcode/lint_policy_test.go`, plus the original criteria, AGENTS.md, full Gomad README, applicable milestone requirements, adjacent helper tests, task24 admission/handover/evidence, correction, inventory, meaningful final RED/GREEN logs, final helper/config/ownership/validate/product receipts, preservation log, source/tool manifests and exact finding inventories. Root's metadata changes remain separately owned.

I applied the requesting-code-review template, all six review-code criteria, verification-before-completion and the repository prose contract. The parent's explicit single-review/read-only instructions supersede the skills' ordinary additional-agent and worktree workflows. No additional reviewer, worktree, checkout, stash, bridge, download, broad lint/native/build/generation rerun or lifecycle mutation was used.

Requested reviewer and writer routing is the same family, `gpt-6.1-sol` at high. Actual execution model metadata is unavailable. Parent reported the single bounded judge result as `Tier: session (jev-unavailable(no_key))`; I read its state and did not repeat or reroute that judge. This report does not impersonate the configured formal backend.

## Strengths

- `.github/.golangci.yml:252` adds exactly `relative-path-mode: gitroot`. Every existing regex, global exclusion, linter, formatter, setting and forbid/text pattern remains byte-identical. Tracked product Go source, module manifests, Makefiles, workflows and historical task23 evidence match the task base.
- `lint_policy_test.go:17` parses the real YAML and exercises all fifteen rule path/path-except expressions and all three global paths using literal expected match/nonmatch cases. Expectations do not come from recompiling the same expression into an oracle. Directory near misses, wrong dot characters, non-Go names and anchored suffixes distinguish the intended cases. Existing unanchored/prefix behavior is retained.
- `lint_policy_test.go:83` executes the actual v2.13.0 binary, checks its version, logs its SHA-256, copies the actual config and initializes a private scratch Git root. Root and both nested module cwd cases test repository-relative filenames, unchanged exclusions and application findings. No test double supplies this policy proof.
- `lint_policy_test.go:125` expects no issue only for the inherited exclusions. The ordinary root panic and nested Gomad application panic each still require exactly one forbidigo finding and exit 1. The nested tests fixture deliberately preserves the inherited `tests/.+\.go` panic exemption for non-test files under tests.
- Fixture commands use argument arrays, contextual cancellation, `--fix=false`, explicit tags and the config's readonly module policy. File, format, version, process-exit and JSON errors are checked. Scratch state belongs to `t.TempDir`; all five reported real-tool fixture directories were absent after completion. The existing routing doubles remain evidence about command dispatch only.
- The corrected diagnosis follows the evidence. The final meaningful RED passes parsed-regex controls but fails real-tool root/nested exclusions and reports `../` paths. Raw byte evidence contains one backslash before slash/dot. GREEN restores the base without changing those expressions. The old task23 diagnosis stays immutable and its doubled-literal-escape claim is explicitly refuted.
- The helper contracts and actual tool/config checks passed freshly in this review. All 3,837 retained source-manifest entries, the final config/test snapshots and the three pinned executable hashes passed before and after the checks. HEAD and the index were stable.

## Issues

### Critical

None in the reviewed source change.

### Important

None in the reviewed source change.

### Minor

None that clears the review criteria.

The inherited source lint failures and unchanged global reporting limitation below are acceptance gaps with retained owners. They are not newly invented findings against the one-line repair.

## Verification and plan alignment

[The independent checks receipt](independent-source-review-checks.json) preserves exact commands, cwd, explicit environment, raw combined output, terminal exit codes, observed timestamps and timing limits. Fresh checks from the repository root were:

| Fresh check | Outcome |
| --- | --- |
| all lintcode contracts with `-count=1 -tags test_dep -v` and the verified real binary | exit 0; actual tool test 3.26 s, package 6.770 s |
| unfiltered helper golangci using the actual root config and `--fix=false` | exit 0; 0 issues |
| helper errortype vet with `test_dep` and `-style-check=false` | exit 0 |
| pinned binary config verification | exit 0 |
| before/after retained source, tool and final-policy SHA checks | exit 0 |
| exact Gomad raw-log/finding/owner accounting | exit 0; both equality checks true |
| tracked source/manifests/Make/workflow/old-task23 preservation | exit 0; no diff |
| authored config whitespace check and new-test gofmt | exit 0; no diagnostics |
| new-test `git diff --no-index --check` | exit 1 with empty output; added-file difference, no whitespace diagnostic |
| real fixture cleanup | exit 0; all five reported directories absent |

The fresh test's complete output contains all four real-tool subtests and no skip. Its command yielded once; 17.429 s is the start-to-terminal-observation bound, while 6.770 s is the Go-reported package elapsed time. I do not treat summed tool wait durations as total command execution time.

The final retained ownership and `make -C tools/gomad3 validate` logs/receipts pass on source-bound inputs. They were inspected and reused, not rerun. The meaningful policy RED/GREEN logs predate the latest binary-hash logging addition to the test; the current full helper run supplies fresh execution of the final frozen file. No broad product result is inferred from scratch fixtures.

Task24's bounded repair and evidence requirements are satisfied for source progress. The acceptance item requiring a progress commit remains root's action. Original fn-109 criteria are unchanged, and neither task24 source admission nor this review closes qualification. Correcting the new task's refuted cause narrows the repair hypothesis; it does not weaken a gate.

## Qualification limits and recommendations

| Retained gate | Actual outcome | Remaining work |
| --- | --- | --- |
| root fast | Make exit 2 / golangci exit 1; one exhaustive finding at `tools/gomad3sim/controller.go:159:3` | bounded source owner must preserve the switch's semantics; root vet and later dispatch scopes were unreached |
| ordinary nested Gomad | Make exit 2 / golangci exit 1; 419 findings across 112 files and 31 directories | bounded owners from `gomad-finding-owners.json`; nested vet was unreached |
| mixedbrain | exit 0 including vet under existing comparison filter | does not establish unfiltered whole-module cleanliness |
| exact tagged integration batch | exit 0 including vet under existing comparison filter | does not establish unfiltered whole-module cleanliness |
| native darwin/arm64 and linux/amd64 | open | required native evidence; stock Linux aarch64 is developmental |

The independently reconciled 419 findings comprise 319 errcheck, 11 exhaustive, 12 forbidigo, 2 gci, 14 goimports and 61 staticcheck. The exact path/line/column/message entries match the retained raw log, and every owner aggregate matches those entries. Their full provenance or remediation validity is outside this source review; preserved source alone does not prove the origin of every issue.

`.github/.golangci.yml:173` retains `^.git`, whose wildcard dot includes `.github`. Git-root matching activates that existing reporting limitation. The retained root log records 27 suppressed findings, and the fresh actual hidden-action fixture is excluded. Hidden-action cleanliness has not been established. Changing this rule would require its own bounded policy owner.

Root can commit this coherent source progress, retaining the red receipts and original acceptance gaps, then assign bounded source fixes. Error-handling fixes must preserve cleanup/publication failure precedence and transactions. New suppressions, blanket discards, weaker assertions, changed comparisons or a fabricated green baseline would violate the retained contract. Formal implementation review still requires the actual green qualification tree.

## Assessment

Correctness is correct within the bounded reviewed scope. Security is secure for this trusted test/config scope. API usage is correct against the locally inspected pinned golangci source and existing root dependencies. Consistency is consistent, simplicity is clean, and test coverage is adequate for the requested path-base repair with the explicit real-tool input supplied. There is no separate peer-review verdict because this dispatch expressly authorizes one reviewer.

Ready for SOURCE PROGRESS COMMIT: yes. The demonstrated path-base repair is minimal and the real-tool positive/negative controls pass on frozen inputs. Qualification remains red, so this assessment cannot authorize formal SHIP, native acceptance or Flow completion.

Live command handles at return: none. Delegated agents: none. Reviewer writes are limited to this report and its checks receipt. Root owns fix decisions, Git, Flow, lifecycle metadata and commit.

