# fn-109.44 independent source-progress review

Assessment is `SOURCE_PROGRESS_COMMIT`, conditional on root retaining the actual original integrated lint result and the worker's final source-bound checks. No introduced P1, P2 or P3 finding was identified in the three-file source diff. This assessment authorizes only bounded source progress. Required lint remains red; formal implementation SHIP, merge and DONE are outside this review.

## Identity and scope

- Reviewer selection is `gpt-6.1-sol` at high under AGENTS.md, in a fresh context. Reviewer and writer are from the same model family intentionally.
- Review UTC start was `2026-10-05T19:24:56Z`; review UTC end and post-write source freeze check were `2026-10-05T19:29:13Z`. The final executable-hash check was `2026-10-05T19:28:00Z`.
- Branch is `gomad`. HEAD at both checks was `6f7631ac560a117f1300c1e64b0a9fd5d3e22e63`.
- Admitted source BASE is `13df4f16f90d49938ea123a29859da62ad2cab9f`. The three HEAD source files are byte-identical to that BASE.
- Source scope is `schema.go`, `mutation_test.go` and `schema_timezone_test.go` under `tools/gomad3/internal/compatibilitypack`. The root-owned MILESTONES.md change is outside this source assessment.

I read AGENTS.md, Gomad README/milestone guidance, `flowctl usage`, task 44 through `flowctl cat fn-109.44`, its anchor and full parent spec. I inspected the exact Git diff, destination/source declarations, production validation/digest/selection/copy paths and existing test controls. Read-only Git, Flow, file reads and Node in-memory comparisons supplied this review. I ran no test, lint, build, generator, download or Go/cache operation, and changed no source, index, commit, worktree or Flow state. This review artifact is the sole write.

## Exact source proof

At schema.go:292, `Source(source)` replaces `Source{Name: source.Name, SHA256: source.SHA256}`. PackSource at schema.go:101 has exactly the ordered string fields Name and SHA256, with JSON tags. Source at policy.go:48 has those same ordered string fields without tags. Go value conversion retains destination type Source and copies the same two string values. It introduces no pointer conversion, alias to an inventory element, JSON projection or field omission.

At mutation_test.go:98 the same conversion supplies Source. At mutation_test.go:102, `ForeignSource(source)` replaces the three-field literal. PackForeignSource at schema.go:106 and ForeignSource at policy.go:53 have exactly the ordered string fields Kind, Name and SHA256, again with tags only on the Pack type. Both expressions still produce the destination type that the original literal produced. Strings retain the same values and backing data semantics; mutable slice element storage remains separately allocated.

The schema.go:290 allocation retains length zero and capacity equal to both inventories' combined lengths. Its Go-source loop precedes the foreign-source loop. Schema.go:295 still constructs `source.Kind + ":" + source.Name` literally, preserving the digest namespace for foreign files. DigestSources at policy.go:215 retains the `gomad3.compatibility-source-set/v1\x00` prefix followed by each Name, NUL, SHA256 and NUL in input order. The changed expression supplies identical strings in identical order. DigestSources' handling of nil and empty input is unchanged because the range still writes no elements for either.

generatedExactPackages at mutation_test.go:93 retains both `make(..., len(...))` allocations and indexed loops. A zero-length inventory still becomes a nonnil empty destination slice, including a nil source slice passed to this helper. Conversion changes no slice length, capacity, append, activation completion, package order or detached element storage. The subsequent activation scan and generated adapter evidence remain byte-unchanged.

schema_timezone_test.go:3 changes only import order/grouping. Its test bodies, fixed canonical JSON vector, timestamp grammar and exact error assertions remain unchanged. All three ST1005 error strings remain exactly as before at schema.go:275,316,319, including uppercase `Go`.

I independently reversed only the three conversions and the single import regrouping in memory. Each replacement matched exactly once. Each reconstructed complete file equals `git show 13df4f16f90d49938ea123a29859da62ad2cab9f:<path>` byte-for-byte. This proves there is no additional comment, fixture, assertion, error, declaration or body edit hidden in these files. It also confirms the original destination types and every surrounding allocation/loop rather than relying on the handover description.

| File | Admitted BASE SHA-256 | Frozen candidate SHA-256 at first and final checks |
| --- | --- | --- |
| schema.go | 916b095eac3efd437982b3986beef5b712eb9c14d50552aca81930ebf3d29ae1 | 449fb233c61b6a3691c913dde04e1630cfb1f4e9b2fffea82dacfeb18251f0ed |
| mutation_test.go | bcbd1d5039bd04bcaffffe78273e954165ec022fbb26d1662b8cd5f68a83d070 | b6552e443ae5df2b1c6e3e8f97e52a96a06ea61a6afaead86103549c8b3ba185 |
| schema_timezone_test.go | 7bb4146b5872bdddedfd0e4fd2e44b47bc36a777c07eed6e96144c4f2ab272ca | 5a055e7ce61f18b9fbd3fb16f68daddbe7cb1a66c6fa039973aecebf9b53c2eb |

The retained before.json manifest contains 1,264 selected files. I recomputed current hashes for all 1,261 entries outside the three admitted candidates; none differed. The five recorded Go, gofmt, golangci-lint, errortype and Node executable hashes also still matched. These selected-input checks preserve generated/pin inputs covered by that manifest, including historical task evidence. They do not establish a complete repository/toolchain qualification closure.

## Existing controls inspected

- mutation_test.go:15,65,69,71,93,138 retains generated positive, Go-source, foreign-source, source-set and other exact-pack mutation controls. generatedRequireAllowed checks each rule's capabilities/linknames against the package built by the changed helper; changed source facts retain their denial assertions.
- policy_test.go:42 retains DecodePackV2 malformed/noncanonical, source-set mismatch, unknown-field/trailing-data and existing host-import refusals. policy_test.go:72 retains an exact complete-inventory grant and a changed Go-source denial. v2_selection.go:285 still compares complete inventories by the same fields.
- schema_timezone_test.go:12,35,62 retains timestamp grammar, literal complete canonical bytes/digest and governance error precedence.
- policy_exhaustive_test.go:208 retains input immutability, detached evidence and Pack copies, fixed identity and subsequent capability/linkname decisions after mutations. v2_selection.go:40 retains cloned inventory slices and nested copies. These tests support their named controls; the changed helper's allocation proof independently establishes its element-storage separation.
- schema_admission_test.go:9,34,84,104,128 retains all five forbidden imports, structural validation before refusal, rule/capability order, unselected-token revalidation, earlier-pack errors and unchanged other capabilities. external_test.go:154 retains exact external wrappers and nil partial results for all five. authoring/request_test.go:108,134 retains allowed-fact refusals and fact-validation priority. ValidatePack at schema.go:132 still performs complete structure validation first, and the five-element ban at schema.go:41 remains unchanged. Task 43 keeps R21 ownership.

## Retained execution observations and limits

I parsed raw retained JSON test events and independently recomputed their terminal counts. baseline-packages.json and final-packages.json both exit 0 with 469 passing tests, zero failures and one skip, TestHostPacksBindCurrentProfile. These are developmental-host observations. The unchanged skip supplies no profile or native qualification.

baseline-target-consumers.json separately retains exit 1 with 38 passing tests and one failure, TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests. Its output names the missing patched `.toolchain/bin/go`. That real failure remains evidence. The narrower baseline-target-controls.json and final-target-controls.json both exit 0 with 37 passing tests and no failures/skips under their recorded task-43-style regex. Their result does not turn the broader failed command into a pass.

final-validate.json records exit 0 for check-only generation/validation, beginning at 19:24:22.645Z and finishing at 19:24:28.796Z. Its output reports current compatibility packs and check-only version/protocol/boundary/qualification generation. It precedes the final package and target-control captures. The profile test's ordinary `ok` line in this Make output does not expose its skip; the full package JSON retains that limitation. architecture-purity.json has six passing named tests and no failures/skips. I did not re-execute these commands.

The actual scoped baseline-lint.json contains seven diagnostics, three S1016, one gci and three ST1005. The actual final-lint.json exits 1 with the same three ST1005 diagnostics. I split and compared their complete blocks; all three remaining blocks are byte-identical, and the four removed blocks are exactly the admitted sites. Scoped 7 to 3 is an observed capture result. The original integrated 323 to 319 result remains unproved by this review while root's actual gate is pending. The previous task-43 root-integrated-lint.md records actual red lint and an unreached integrated errortype stage; standalone errortype evidence cannot substitute for that stage.

## Remaining acceptance

Root must retain its actual original integrated gate with the unchanged comparison base, config, fix=false and tool identities, explain the exact diagnostic delta, preserve residual diagnostic bodies/counts and record Make exit plus integrated errortype reachability. Worker final checks and the independent evidence review must bind the same frozen source before root commits this progress separately. This review supplies no formal implementation-review receipt.

Original task 11/R17, task 21/R18/R19, predecessors, matched first-baseline identities, preservation/fixed-identity proof, full host/default/functional/affected-consumer qualification, native Darwin and static both-source-set requirements remain required and open wherever unproved. Task 42 keeps exhaustive ownership; task 43 keeps the five-import correction and its historical evidence. Native Linux remains unverified and nonblocking under fn-128. No conformance, native qualification, merge, DONE or formal SHIP claim follows from this bounded review.
