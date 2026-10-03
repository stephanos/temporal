#!/usr/bin/env python3
"""fn-112.9 behavior mapping check.

Usage: mapping-check.py BEFORE AFTER [BEFORE AFTER ...] > mapping.tsv

Each pair is one package before and after the consolidation: either `go test -json` output
named before-PACKAGE.json / after-PACKAGE.json, or a behaviors TSV (package, test, status)
given as PATH:PACKAGE. A behavior is one recorded test or subtest name. Every name recorded before and absent after
must map to a replacement name recorded after, or to a removal reason. The script prints one
row per removed or renamed behavior (old name, old status, replacement, new status, note),
then the retained rows count, and exits 1 when a removed behavior has no mapping, when a
replacement is missing after, or when a replacement's status is worse than the old status
(pass > skip > fail) without being a known harness-only failure listed in HARNESS_FAILS.
A removed validation behavior mapped to a TestCampaignOptionsLegacyCharacterization row must
also find its old wanted error text in that row's pinned error.
"""
import json
import os
import re
import sys

# Removed architecture checks: each pins a past refactor rather than a defect.
ARCHITECTURE_REASONS = {
    "TestCleanupRemovesSupersededFiles": "deleted-file list: pins the fn-102..fn-105 cleanup; a recreated script or directory is not a defect by itself, and any Go package an importer reaches must still pass TestPackageArchitecture's owner and import rules",
    "TestCurrentVocabularyHasNoLegacyCampaignBoundary": "banned words: pins a naming migration (batch/run -> campaign); wording is a review concern, not a behavior, and no test or gate consumes the vocabulary",
    "TestCanonicalJSONHasOnePrivateOwner": "required/deleted filenames: TestPackageArchitecture requires the canonicaljson owner to have a package, an importer of a recreated evidence package fails as an ownerless import, and a duplicate private helper in world/codec.go is a review concern",
    "TestRecordAndArtifactHaveSeparateOwners": "required/deleted filenames: TestPackageArchitecture requires record and artifact owner packages and forbids artifact/record importing runner (moduleMayImport); the file names and the removed evidence package pin the fn-105 split",
    "TestReadOnlyMountHasOneDeepOwner": "required/deleted filenames: readonlymount is imported by the runner build, so a missing package fails compilation; the absence of deterministicio/readonly_*.go pins a file layout",
    "TestDeveloperToolingIsNotOwnedByToolchain": "required/deleted filenames: TestPackageArchitecture requires the developer owner, make validate runs cmd/gomadtool and its generators, and the toolchain owner may import only canonicaljson, hostexec, hostfs; the absent toolchain/cmd paths pin a move",
    "TestWorldProcessSessionHasOneDeepOwner": "required/deleted filenames: world/process is compiled into the runner, and the absent world/host, world/target, world/internal/transport pin a past layout",
    "TestCompatibilityPackHasOneOwner": "required/deleted filenames: TestPackageArchitecture requires the compatibility owner and cmd/gomadtool compiles authoring; the absent target/internal/compatibility and target/packdev pin a move",
    "TestUpgradeOrchestrationIsAboveToolchain": "required/deleted filenames: TestExactModuleEdges keeps toolchain from importing qualification or upgrade and upgrade depending on qualification/set and toolchain/version; the file names pin a move",
    "TestQualificationUseCasesHaveExplicitOwners": "required/deleted filenames: TestExactModuleEdges requires upgrade to import qualification/set and the CLI compiles the other use cases; the absent suite_legacy.go/suite_previous.go pin a codec removal",
    "TestConformanceRuntimeIsGroupedByBehavior": "required/deleted filenames: pins the fn-112.6 file split of the conformance runtime; the runtime tests themselves (internal/gomadtool/conformance) cover its behavior",
}

DRIFT_CASES = {
    "upstream-file": "upstream-file", "missing-anchor": "missing-anchor", "ambiguous-anchor": "ambiguous-anchor",
    "replacement-digest": "replacement-digest", "upstream-edit": "upstream-file", "replacement-edit": "replacement-digest",
    "source": "upstream-file", "anchor": "missing-anchor", "ambiguous": "ambiguous-anchor", "replacement": "replacement-digest",
}

ADAPTERS = {
    "Sprig": "sprig", "Validator": "validator", "Sentry": "sentry", "Pebble": "pebble",
    "CactusStatsD": "cactusstatsd", "Memberlist": "memberlist", "HashicorpMetrics": "hashicorp-metrics",
}

# Failing before and after on linux/arm64 only because no adapter is pinned for the host.
HARNESS_FAILS = set()

DRIFT = "TestRewrittenModuleRewritesPreserveCommentsAndRejectDrift"
GRPC_PROFILE = {
    "TestProfileRejectsUnsupportedGRPCVersion": "version",
    "TestProfileRejectsExistingGRPCReplacement": "replacement",
    "TestProfileRejectsExistingGRPCReplacementBlock": "replacement-block",
    "TestProfileRejectsChangedGRPCModuleSum": "sum",
    "TestProfileRejectsBuildModFileWithGRPCAdapter": "build-modfile",
}

# fn-109-dependent CLI and Runner part. Old name -> (replacement names, note). A subtest of
# an old name that is not listed maps through SUBTEST_PREFIXES.
GOLDEN = "TestCampaignOptionsLegacyCharacterization/"
QUALIFY = "TestRunQualifyForwardsFlagsAndClassifiesOutcome/"
ANALYZE = "TestRunAnalyzeForwardsTargetAndClassifiesReport/"
COMPLETION = "TestCompletionFaultsKeepReasonPrecedenceAndEvidence/"
DAMAGED = ["damaged_shared_target/" + damage + "/verify-only=" + mode for damage in ("altered", "truncated", "missing") for mode in ("false", "true")]
CANNED = "subprocess helper: SKIP in the parent run; TestCannedCoordinatorHelper serves the same canned response, selected by GOMAD3_CANNED_COORDINATOR"
EXPLICIT = {
    # cmd/gomad/internal/cli: forwarded-field and outcome tests, one table per command.
    "TestRunQualifySetUsesCurrentExecutableAndPublicPaths": (["TestRunQualifySetForwardsFlags/public_paths_and_executable"], "row: whole forwarded qualificationset.Spec (was field-by-field) and the JSON schema line"),
    "TestRunQualifySetPassesShardToTheSet": (["TestRunQualifySetForwardsFlags"], "same rows; each run row now compares the whole forwarded Spec"),
    "TestRunMinimizeUsesBoundedArtifactStoreAndCurrentInstallation": (["TestRunMinimizeForwardsFlags/bounded_store_and_installation"], "row: whole forwarded MinimizeSpec (was field-by-field) and accepted=1"),
    "TestRunMinimizeResumesOnlyOnRequest": (["TestRunMinimizeForwardsFlags/initial_run", "TestRunMinimizeForwardsFlags/resume"], "rows compare the whole forwarded MinimizeSpec, Resume included"),
    "TestRunMinimizeResumesOnlyOnRequest/initial_run": (["TestRunMinimizeForwardsFlags/initial_run"], "row: whole MinimizeSpec, Resume false"),
    "TestRunMinimizeResumesOnlyOnRequest/resume": (["TestRunMinimizeForwardsFlags/resume"], "row: whole MinimizeSpec, Resume true"),
    "TestRunQualifyRepeatsOneSeedAndRetainsJSONReport": ([QUALIFY + "repeat_one_seed_and_retain_the_JSON_report"], "row: same CampaignSpec fields per run, two runs, qualified report, requested toolchain root, five result-event fields"),
    "TestRunQualifyReportsNondeterministicEvidence": ([QUALIFY + "nondeterministic_evidence"], "row: status 1, first divergence stdout.full_sha256, nondeterministic event"),
    "TestRunQualifyReplaysRepeatedTargetFailure": ([QUALIFY + "replays_repeated_target_failure"], "row: two runs, two replays of failure-N, both replay results match, target_failure; stderr now also required empty"),
    "TestRunQualifyReplaysEveryRetainedSuccess": ([QUALIFY + "replays_every_retained_success"], "row: retention bounds per run, two replays of success-N, qualified"),
    "TestRunQualifyRequiresExplicitSuccessfulReplayBounds": ([QUALIFY + "successful_replay_requires_bounds", QUALIFY + "success_bounds_require_successful_replay"], "one row per rejected argument vector; no run and no report"),
    "TestRunQualifyRetainsMissingSuccessfulReplayArtifact": ([QUALIFY + "retains_missing_successful_replay_artifact"], "row: status 3, runner_failure with the same message"),
    "TestRunQualifyRetainsReplayCancellation": ([QUALIFY + "retains_replay_cancellation"], "row: status 3, cancelled, two executions, replay divergence recorded"),
    "TestRunQualifyRetainsUnsupportedBoundary": ([QUALIFY + "retains_unsupported_boundary"], "row: status 2, capability retained, unsupported_target event"),
    "TestRunQualifyRejectsUnboundedRepeat": ([QUALIFY + "rejects_unbounded_repeat"], "row: status 2, invalid_input, no run"),
    "TestRunResumeUsesStoredBatchAndReportsResult": (["TestRunResumeForwardsCampaignAndClassifiesResult/stored_campaign"], "row: whole forwarded ResumeSpec (was field-by-field), requested toolchain root, four result-event fields"),
    "TestRunResumeClassifiesInvalidJournalAsInputError": (["TestRunResumeForwardsCampaignAndClassifiesResult/invalid_journal_is_an_input_error"], "row: status 2, invalid_input; the forwarded ResumeSpec is now also compared"),
    "TestRunAnalyzeEmitsSupportedJSONWithoutExecutingTarget": ([ANALYZE + "emits_supported_JSON_without_executing_target"], "row: same reviewed target.Spec check, status 0, empty stderr, schema and classification"),
    "TestRunAnalyzeMapsUnsupportedInvalidAndInfrastructureStatuses": ([ANALYZE + name for name in ("unsupported", "opaque_executable", "invalid_package", "infrastructure")], "same four rows and base dependencies"),
    "TestRunAnalyzePreservesClassificationWhenCleanupFails": ([ANALYZE + "cleanup_failure_preserves_classification"], "row: status 1 and cleanup failed on stderr"),
    "TestRunAnalyzeSurfacesCleanupFailureAfterSupportedReport": ([ANALYZE + "cleanup_failure_after_supported_report"], "row: status 3 and cleanup failed on stderr"),
    "TestRunAnalyzeBuildsFromPreparedReview": ([ANALYZE + "builds_from_prepared_review"], "row: one inspection, one build from the prepared review, status 0, empty stderr"),
    "TestRunAnalyzeReportsOutputFailuresAsInfrastructure": ([ANALYZE + "output_failure_is_infrastructure"], "row: failing stdout, status 3, write capability analysis"),
    # runner: validation rejections pinned with their full error text by the golden table.
    "TestRunGuidanceRequiresCorpusAndSemanticCoverage": ([GOLDEN + "guidance_without_corpus", GOLDEN + "guidance_without_coverage"], "golden rows pin the error text; the old test went through exploreWith and asserted only a failure"),
    "TestRunRequiresExplicitSuccessRetentionBounds": ([GOLDEN + "all_retention_without_count", GOLDEN + "all_retention_without_bytes", GOLDEN + "novel_retention_without_coverage"], "golden rows; all_retention_without_bytes is new; the old test asserted only a failure through exploreWith"),
    "TestRunRequiresBoundedChoiceTraceCapacity": ([GOLDEN + "choice_trace_below_minimum", GOLDEN + "choice_trace_above_maximum"], "golden rows pin the capacity error (the old test required 'choice trace' through exploreWith)"),
    "TestValidateConfigRequiresBoundedSingleSeedChoiceExploration": ([GOLDEN + "choice_exploration"], "golden row accepts the bounded single-seed request; the rejections map per subtest"),
    "TestValidateConfigRequiresBoundedSingleSeedSimulationExploration": ([GOLDEN + "simulation_exploration"], "golden row accepts the bounded single-seed request; the rejections map per subtest"),
    "TestValidateConfigRejectsExplorationBoundsForSeedStrategy": ([GOLDEN + name for name in ("seed_max_executions", "seed_choice_depth", "seed_choice_start_ordinal", "seed_exploration_bytes")], "golden rows, one per exploration bound on a seed request"),
    "TestExecutionEvidenceRequiresOneSeedAndSemanticCoverage": ([GOLDEN + "execution_evidence_with_multiple_seeds", GOLDEN + "execution_evidence_without_semantic_coverage"], "golden rows pin the error text; the old test asserted only a failure through exploreWith"),
    # runner: completion and cancellation.
    "TestRunClassifiesInvalidChoiceTraceTerminalEvidence": ([COMPLETION + "choice_trace_rejected_by_supervision/seed", COMPLETION + "unterminated_choice_trace_rejected_by_supervision/seed"], "supervision-rejected trace rows, now for every strategy"),
    "TestRunClassifiesInvalidChoiceTraceTerminalEvidence/malformed": ([COMPLETION + "choice_trace_rejected_by_supervision/seed"], "existing row: same injected execution.ErrChoiceTraceMalformed, reason choice_trace_malformed"),
    "TestRunClassifiesInvalidChoiceTraceTerminalEvidence/unterminated": ([COMPLETION + "unterminated_choice_trace_rejected_by_supervision/" + strategy for strategy in ("seed", "choice-exploration", "simulation-exploration")], "new row: injected execution.ErrChoiceTraceUnterminated, reason choice_trace_unterminated"),
    "TestRunCancellationIsAHostFailure": (["TestCancellationIsAHostFailure/seed"], "row: same seed config, reason, resume plan and partial assertions"),
    "TestExplorationCancellationIsAHostFailure": (["TestCancellationIsAHostFailure"], "same exploration rows, joined by the seed row"),
    "TestAssessWorldValidatesTheRecordAgainstItsSeed/malformed_record": ([COMPLETION + "malformed_World/seed"], "same decode error text, pinned once, through Explore; the precedence row malformed_record_before_seed_mismatch keeps the private assessWorld pin"),
    "TestAssessWorldValidatesTheRecordAgainstItsSeed/seed_mismatch": ([COMPLETION + "World_seed_mismatch/seed"], "same error text (seed 7 there, 8 here), pinned once, through Explore"),
    # runner: resume, preparation and coordinator transport.
    "TestRunResumeRejectsChangedRunnerIdentity": (["TestRunResumeRejectsChangedEvidence/runner_build"], "row: same interruption and Runner build identity error"),
    "TestRunResumeRejectsTamperedRetainedSuccessArtifact": (["TestRunResumeRejectsChangedEvidence/tampered_retained_success"], "row: same interruption, tampered stdout, retained success error"),
    "TestRunPreparationFailureLeavesExplicitPartial": (["TestRunPreparationFailureLeavesClassifiedPartial/build_failure"], "row: same reason and partial fields"),
    "TestRunPreparationCancellationIsClassifiedSeparately": (["TestRunPreparationFailureLeavesClassifiedPartial/cancelled"], "row: same reason, context.Canceled cause and partial reason"),
    "TestRunPreparationOverallTimeoutIsClassifiedSeparately": (["TestRunPreparationFailureLeavesClassifiedPartial/overall_timeout"], "row: same reason, context.DeadlineExceeded cause and partial reason"),
    "TestIsolatedRunnerPreservesUnsupportedTargetError": (["TestIsolatedRunnerPreservesCoordinatorResponse/unsupported_target_error"], "row: same canned response and typed error"),
    "TestIsolatedRunnerPreservesMissingSemanticProbesError": (["TestIsolatedRunnerPreservesCoordinatorResponse/missing_semantic_probes_error"], "row: same canned response and typed error"),
    "TestIsolatedRunnerPreservesBoundedExecutionEvidence": (["TestIsolatedRunnerPreservesCoordinatorResponse/bounded_execution_evidence"], "row: same canned evidence"),
    "TestIsolatedRunnerTransportsChoiceTraceConfiguration": (["TestIsolatedRunnerPreservesCoordinatorResponse/choice_trace_configuration"], "row: the helper still echoes the transported limit"),
    "TestIsolatedRunnerBoundsCoordinatorOutput": (["TestIsolatedRunnerPreservesCoordinatorResponse/bounded_coordinator_output"], "row: same oversized output, coordinator_decode"),
    "TestFastCoordinatorHelper": (["TestCannedCoordinatorHelper"], CANNED),
    "TestUnsupportedTargetCoordinatorHelper": (["TestCannedCoordinatorHelper"], CANNED),
    "TestMissingSemanticProbesCoordinatorHelper": (["TestCannedCoordinatorHelper"], CANNED),
    "TestExecutionEvidenceCoordinatorHelper": (["TestCannedCoordinatorHelper"], CANNED),
    "TestChoiceTraceCoordinatorHelper": (["TestCannedCoordinatorHelper"], CANNED),
    "TestOversizedCoordinatorHelper": (["TestCannedCoordinatorHelper"], CANNED),
    "TestReplayVerifyOnlyDoesNotStartTarget": (["TestReplayDoesNotStartTarget/verify_only"], "row: verified, not matched, no target start"),
    "TestReplayRejectsUnavailableCompatibilityPackBeforeTargetStart": (["TestReplayDoesNotStartTarget/unavailable_compatibility_pack"], "row: error and no target start"),
    "TestReplayRejectsChangedPayloadBeforeTargetStart": (["TestReplayDoesNotStartTarget/changed_payload"], "row: error and no target start"),
    "TestReplayRejectsDamagedSharedTargetBeforeTargetStart": (["TestReplayDoesNotStartTarget/" + name for name in DAMAGED], "same six damage x verify-only rows"),
}
SUBTEST_PREFIXES = {
    "TestRunQualifySetPassesShardToTheSet/": "TestRunQualifySetForwardsFlags/",
    "TestRunAnalyzeMapsUnsupportedInvalidAndInfrastructureStatuses/": ANALYZE,
    "TestExplorationCancellationIsAHostFailure/": "TestCancellationIsAHostFailure/",
    "TestReplayRejectsDamagedSharedTargetBeforeTargetStart/": "TestReplayDoesNotStartTarget/damaged_shared_target/",
}
VALIDATION_SUBTESTS = {
    "TestValidateConfigRequiresBoundedSingleSeedChoiceExploration/": {
        "multiple_seeds": ("choice_multiple_seeds", "exactly one base seed"), "guidance": ("choice_guidance", "does not support guided exploration"),
        "missing_choice_trace": ("choice_without_trace", "requires an enabled choice trace"), "missing_execution_bound": ("choice_without_max_executions", "max executions"),
        "missing_depth_bound": ("choice_without_depth", "choice depth"), "missing_exploration_bound": ("choice_without_exploration_bytes", "exploration bytes"),
    },
    "TestValidateConfigRequiresBoundedSingleSeedSimulationExploration/": {
        "multiple_seeds": ("simulation_multiple_seeds", "exactly one base seed"), "guidance": ("simulation_guidance", "does not support guided exploration"),
        "missing_choice_trace": ("simulation_without_trace", "requires an enabled choice trace"), "missing_execution_bound": ("simulation_without_max_executions", "max executions"),
        "missing_forced-decision_bound": ("simulation_without_forced_decisions", "forced decisions"), "missing_exploration_bound": ("simulation_without_exploration_bytes", "exploration bytes"),
        "missing_result_bound": ("simulation_without_result_bytes", "result bytes"), "missing_dimension_bound": ("simulation_without_network_dimension", "network dimension"),
        "choice_start_ordinal": ("simulation_choice_start_ordinal", "choice start ordinal requires the choice-exploration strategy"),
    },
}
# Old error text each whole-test validation mapping asserted, checked against the golden rows.
VALIDATION_WANT = {
    "TestRunRequiresBoundedChoiceTraceCapacity": "choice trace",
    "TestValidateConfigRejectsExplorationBoundsForSeedStrategy": "choice-exploration strategy",
}
GOLDEN_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "..", "..", "tools", "gomad3", "runner", "testdata", "campaign_options_characterization.json")


def golden_errors():
    rows = json.load(open(GOLDEN_PATH))
    return {GOLDEN + row["name"].replace(" ", "_"): row["error"] for row in rows}


def explicit_replacements(name):
    if name in EXPLICIT:
        return EXPLICIT[name]
    for prefix, replacement in SUBTEST_PREFIXES.items():
        if name.startswith(prefix):
            return [replacement + name[len(prefix):]], "same row"
    for prefix, rows in VALIDATION_SUBTESTS.items():
        if name.startswith(prefix) and name[len(prefix):] in rows:
            row, want = rows[name[len(prefix):]]
            return [GOLDEN + row], "golden row pins the full error; old wanted text: " + want
    return None


def golden_problem(name, target, errors):
    """Returns a problem when a validation mapping's old wanted text is not in the golden error."""
    if not target.startswith(GOLDEN):
        return ""
    error = errors.get(target)
    if error is None:
        return " GOLDEN ROW MISSING"
    top, _, sub = name.partition("/")
    prefix = top + "/"
    want = None
    if prefix in VALIDATION_SUBTESTS and sub in VALIDATION_SUBTESTS[prefix]:
        want = VALIDATION_SUBTESTS[prefix][sub][1]
    elif top in VALIDATION_WANT:
        want = VALIDATION_WANT[top]
    if target.endswith(("/choice_exploration", "/simulation_exploration")):
        return "" if error == "" else " GOLDEN ROW REJECTS"
    if error == "" or (want and want not in error):
        return " GOLDEN ERROR MISMATCH"
    return ""



def load(path):
    results = {}
    if ":" in path:
        path, package = path.rsplit(":", 1)
        for line in open(path):
            fields = line.rstrip("\n").split("\t")
            if len(fields) == 3 and fields[0] == package:
                results[fields[1]] = fields[2]
        return results
    for line in open(path):
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if event.get("Test") and event.get("Action") in ("pass", "fail", "skip"):
            results[event["Test"]] = event["Action"]
    return results


def rewrite_paths(after, adapter):
    prefix = DRIFT + "/" + adapter + "/"
    return sorted({name[len(prefix):].rsplit("/", 1)[0] for name in after if name.startswith(prefix) and name.endswith("/comments")})


def replacements(name, after):
    """Returns (replacement names, note) for a removed behavior."""
    mapped = explicit_replacements(name)
    if mapped is not None:
        return mapped
    top, _, sub = name.partition("/")
    if top in ARCHITECTURE_REASONS:
        return [], "removed: " + ARCHITECTURE_REASONS[top]
    match = re.fullmatch(r"Test(Sprig|Validator|Sentry|Pebble|CactusStatsD|Memberlist|HashicorpMetrics)(.*)", top)
    if match:
        adapter, rest = ADAPTERS[match.group(1)], match.group(2)
        paths = rewrite_paths(after, adapter)
        if rest == "AdapterConsumer":
            return ["TestRewrittenModuleConsumers/" + adapter], "consumer template row"
        if rest == "RejectsChangedIdentity":
            return ["TestRewrittenModulesRejectChangedIdentity"], "row " + adapter + ": the same three changed identities (module, version, sum), same 'identity mismatch' check"
        if rest == "RejectsUnrewrittenInventoryDrift":
            return ["TestRewrittenModulesRejectUnrewrittenInventoryDrift/" + adapter], "same unrewritten file changed"
        if rest in ("RewritePreservesComments",):
            return [DRIFT + "/" + adapter + "/" + path + "/comments" for path in paths], "comments row per rewritten file"
        if rest == "RewritePreservesCommentsAndMemFS":
            return [DRIFT + "/" + adapter + "/" + path + "/comments" for path in paths] + ["TestRewrittenModuleInventoriesMatchPinnedModules"], "comments row per rewritten file; the MemFS byte-equality check is the table's unrewritten-file copy check (vfs/mem_fs.go)"
        if rest in ("RewriteRejectsDrift", "RewritePreservesCommentsAndRejectsDrift"):
            parts = sub.split("/") if sub else []
            if not parts:
                cases = ["comments"] if "Comments" in rest else []
                return [DRIFT + "/" + adapter + "/" + path + "/" + case for path in paths for case in cases + list(dict.fromkeys(DRIFT_CASES.values()))], "every drift case of every rewritten file"
            case = parts[-1]
            if case in DRIFT_CASES:
                path = "/".join(parts[:-1])
                targets = [path] if path else paths
                return [DRIFT + "/" + adapter + "/" + p + "/" + DRIFT_CASES[case] for p in targets], "drift case " + DRIFT_CASES[case]
            path = "/".join(parts)
            return [DRIFT + "/" + adapter + "/" + path + "/" + c for c in ["comments"] + list(dict.fromkeys(DRIFT_CASES.values()))], "rewritten file: comments and every drift case"
    for old, adapter in (("TestProfileRejectsHashicorpMetricsConfigurationDrift", "hashicorp-metrics"), ("TestProfileRejectsSentryConfigurationDrift", "sentry")):
        if top == old:
            prefix = "TestProfileRejectsAdapterConfigurationDrift/" + adapter
            if sub:
                return [prefix + "/" + sub], "configuration-drift row"
            return sorted(name for name in after if name.startswith(prefix + "/")), "configuration-drift rows of the adapter"
    if top in GRPC_PROFILE:
        return ["TestProfileRejectsAdapterConfigurationDrift/grpc/" + GRPC_PROFILE[top]], "configuration-drift row (same go.mod/go.sum shape, want text, and invalid-configuration check)"
    if top == "TestFilesystemManifestInterceptsProfileOperationsBeforeHostDispatch":
        return ["TestManifestInterceptsBeforeHostDispatch/filesystem_profile_operations"], "same four os functions and gomadIOEnabled guard"
    if top == "TestNetworkManifestInterceptsConcreteTCPMethodsBeforeHostDispatch":
        return ["TestManifestInterceptsBeforeHostDispatch/concrete_TCP_methods"], "same six TCPConn methods and guards"
    if top in ("TestPinnedGRPCModuleInventory", "TestPinnedXNetModuleInventory", "TestPinnedModerncMemoryModuleInventory"):
        return ["TestPinnedAdapterModuleInventories"], "inventory row with the same module directory and expected digest"
    return None, ""


RANK = {"pass": 2, "skip": 1, "fail": 0}


def main(arguments):
    failures = 0
    errors = golden_errors()
    print("package\told\told_status\treplacement\tnew_status\tnote")
    for index in range(0, len(arguments), 2):
        before, after = load(arguments[index]), load(arguments[index + 1])
        if ":" in arguments[index]:
            package = arguments[index].rsplit(":", 1)[1]
        else:
            package = arguments[index].rsplit("/", 1)[-1].replace("before-", "").replace(".json", "")
        retained = 0
        for name in sorted(before):
            if name in after:
                retained += 1
                if RANK[after[name]] < RANK[before[name]]:
                    print(f"{package}\t{name}\t{before[name]}\t(retained)\t{after[name]}\tSTATUS WORSE")
                    failures += 1
                continue
            targets, note = replacements(name, after)
            if targets is None:
                print(f"{package}\t{name}\t{before[name]}\t-\t-\tUNMAPPED")
                failures += 1
                continue
            if not targets:
                print(f"{package}\t{name}\t{before[name]}\t-\t-\t{note}")
                continue
            for target in targets:
                status = after.get(target, "MISSING")
                flag = ""
                if status == "MISSING":
                    flag, failures = " MISSING", failures + 1
                elif RANK[status] < RANK[before[name]] and target not in HARNESS_FAILS:
                    flag, failures = " STATUS WORSE", failures + 1
                golden = golden_problem(name, target, errors)
                if golden:
                    flag, failures = flag + golden, failures + 1
                print(f"{package}\t{name}\t{before[name]}\t{target}\t{status}\t{note}{flag}")
        added = sorted(set(after) - set(before))
        print(f"# {package}: {len(before)} behaviors before, {len(after)} after, {retained} retained by name, {len(added)} new names")
    print(f"# problems: {failures}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
