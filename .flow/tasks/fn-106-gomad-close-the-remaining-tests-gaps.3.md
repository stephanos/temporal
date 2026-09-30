---
satisfies: [R3]
---
# fn-106-gomad-close-the-remaining-tests-gaps.3 Make the I/O transcript bound configurable and run the four excluded suites

## Description
Add a transcript byte limit to the profile, runner flag, qualify-set manifest field, and artifact identity; raise it for the four excluded suites and remove their exclusions if they qualify or classify.

## Acceptance
- the four suites run with a raised bound and are qualified or classified with a finding; exclusions removed


## Done summary
The I/O transcript bound is configurable (runtime reads the capacity from the produced header; --io-transcript-bytes / io_transcript_bytes up to 1 GiB, recorded in limits, evidence, and the plan when non-default). All four excluded suites left the exclusions: TestTaskQueueStats_Pri_Suite, TestVersioning3QueryFunctionalSuite, and TestWorkerDeploymentSuite qualify on seeds 11 and 17 with 512 MiB, TestVersioning3FunctionalSuite with 1 GiB. Their clusters also exceed the 64 MiB choice-trace maximum, so they run with choice_bytes 0 and no success replay: same-seed repeatability is proven, exact replay is not retained. No ./tests test is excluded now.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1cc1cce7d, f5a1741ce
- Tests: make test-runtime (darwin), gomad qualify-set four suites at 512 MiB: 3 qualified, gomad qualify-set Versioning3Functional at 1 GiB: qualified seeds 11 and 17
- PRs: