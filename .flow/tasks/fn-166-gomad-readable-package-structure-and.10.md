---
satisfies: [R10, R6]
---
# fn-166-gomad-readable-package-structure-and.10 Move gomad3sim into gomad3 as internal/model/cluster (rotation batch)

## Description
Replace gomad3sim's single server common/collection use with the standard library, move tools/gomad3sim into tools/gomad3/model/cluster, and split it into the public cluster API and internal runtime-domain wire codecs and history records. Update WASM backend, native build and overlay references. Land in the same integrated candidate as fn-153, fn-154 and fn-166 tasks 6-8.

### Codec split binding

Establish lower-level internal record/identity DTO ownership before extracting wire codecs, so cluster API and codecs depend downward without a codec → parent cluster import. Include every moved semantic source in the cluster implementation identity. Integrate fn154 history removal before relocating these active write surfaces. Preserve the explicitly required native source and final WASM gates.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

tools/gomad3sim no longer exists; gomad3 core imports no server package; full WASM gate set and native source checks pass on the integrated candidate.

### Revival (2026-10-13)

Resume with fn166.6/.7 from a1f0be7e96, which includes the fn154 payload-free history owner. Establish lower DTO/codec ownership before moving the API and record all relocated semantic files in implementation identity. Root serializes integration with the independent oracle and network corpus lanes; final WASM/native-source gates remain mandatory.

## Done summary
No tools/gomad3sim tracked/untracked files remain. Main verified two remaining empty directories and removed them with rmdir; test ! -e tools/gomad3sim succeeds without deleting files. Cluster API, lower internal/history DTOs and internal/wire codecs import downward; implementation.go binds all relocated semantic sources. Core imports no server package; native-source/static/compile-only checks and full actual WASM qualification pass.

Full actual WASM qualification is frozen960c966cdc9fba84736a268d8949f30214c16ac4 at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE, source manifest8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278. All budgets/corpus dispositions are retained: five primary successes/four selections qualified; five expected-failure canaries not qualified. Raw stages.log terminal PASS includes ordinary Rust/Go, 800recordings/32replays, artifact/operator/storage/network/cluster, ten cells/aggregate, selections/catalogue/preservation. Exact three-file947 gate-only correction joins affected corrected-source checks under MILESTONES' bounded-fix rule; no retained result is relabelled current-HEAD execution.

Formal shared implementation receipt .flow/tmp/impl-review-rotation-integrated-20261014.json is SHIP, retained primarysession01a138cd-892e-74d3-96f1-0b60220f7c22, both prior findings fixed and no new blockers. Reviewer gpt-6.1-sol/high shares writer's GPT family. Native execution remains unverified/deferred fn128/fn149. Fn166.11 future atomic rename and complete post-rename qualification are excluded and remain mandatory; wholefn166 is not complete.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4, ea9ca7c35cc46db340b648126cb8edd8affbd571
- Tests: No tools/gomad3sim tracked/untracked files remain. Main verified two remaining empty directories and removed them with rmdir; test ! -e tools/gomad3sim succeeds without deleting files. Cluster API, lower internal/history DTOs and internal/wire codecs import downward; implementation.go binds all relocated semantic sources. Core imports no server package; native-source/static/compile-only checks and full actual WASM qualification pass., Complete frozen qualification stages.log PASS, retained source/corpus binding unchanged, Formal shared implementation SHIP .flow/tmp/impl-review-rotation-integrated-20261014.json, Corrected-source affected architecture/lint/validate-retry/clock/actionlint PASS; native execution deferred; fn166.11 excluded
- PRs: