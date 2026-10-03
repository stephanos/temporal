---
title: Engine summary fields need the runner projection and both CLI exploration lines
date: "2026-10-03"
track: bug
category: integration
module: tools/gomad3/runner/runner.go
tags: [gomad3, choice-exploration, reporting, projection-drift]
problem_type: integration
symptoms: New omitted counter missing from gomad explore/inspect output and CampaignResult
root_cause: Engine Summary is mirrored by runner.ChoiceExplorationSummary and two printf lines that are updated by hand
resolution_type: fix
related_to: [bug/integration/go-mod-download-inside-a-target-module-2026-09-28, bug/integration/profile-adapter-changes-leave-libc-2026-10-01, bug/integration/shard-merge-and-prepared-target-cache-2026-09-29]
---

## Problem
A counter added to the choice-exploration engine's `Summary` reached the campaign journal record but not the Runner's public `ChoiceExplorationSummary`, its projection in `runner/runner.go`, or the `gomad explore` and `gomad inspect` exploration lines in `cmd/gomad/internal/cli`. The review caught it; no test did, because the runner test read the journal record through `campaign.OpenCampaign`.

## What Didn't Work
Treating the engine `Summary` as the reporting surface. The engine summary is published verbatim into the campaign record, which made the field look reported.

## Solution
Carry every new engine summary field through `runner.ChoiceExplorationSummary` + `projectChoiceExplorationSummary` (`runner/runner.go`), `formatExploration` (`cmd/gomad/internal/cli/explore_output.go`), and the `exploration:` printf in `cli.go`, and assert the field on the public `CampaignResult.ChoiceExploration` in the runner test, not only on the journal record.

## Prevention
When a `choiceengine.Summary` or `simulationengine.Summary` field is added, grep for the sibling field's name (`OmittedByCapacity`) across `runner/` and `cmd/gomad/` and update every hit; a runner test should read the public summary.
