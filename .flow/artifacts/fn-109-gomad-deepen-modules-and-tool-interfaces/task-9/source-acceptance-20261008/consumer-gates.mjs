import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const gates=[
 ['source-consumer-conformance',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./internal/gomadtool/conformance -run '^Test(RunUpstreamExecutesTypedGatesInOrder|RunBuilderExecutesTypedPackageGate|RunLiveCapabilityExecutesPinnedSemanticFixtures|RunAcceptsExecutableSymlink|RunUpstreamStopsAtFailedGateWithBoundedEvidence|RunRejectsUnknownModeAndMissingToolchain|RunReportsInfrastructureError|RunInterceptionExecutesManifestDrivenCompilerCases|RunInterceptionRejectsMissingCompiler|ExecWrapperIsStrictPOSIX)$'"],
 ['source-consumer-toolchain',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./toolchain -run '^Test(BuildPublishesAndReusesImmutableToolchain|BuildSerializesConcurrentSameKey|BuildInjectedFailuresLeaveNoTemporaryState|BuildRejectsOverlayCollisionBeforePatching|BuildRejectsUnguardedFailureInjection|SourceCleanup.*)$'"],
 ['source-consumer-resolver',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./upgrade/pinimpact -run '^TestGoResolver'"],
 ['source-consumer-adapterregen',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./upgrade/adapterregen -run '^Test(StagedCommandsKeepTheCallersModuleCache|DefaultPackCheckPassesInAStagedCopyOfTheModule|DefaultGeneratorsAreTheModuleLocalMakeGenerateSteps|GenerationFailureInStagingPublishesNothing|RegenerationLockReleaseComposition|PublicRegenerationReleasesLock|RunScratchCleanup.*)$'"],
 ['source-consumer-upgrade',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./upgrade -run '^Test(RunPublishesFailedGateEvidence|RunReplacesExistingDossierAfterFailedGate|RunReportsPublicationFailureAndKeepsPriorDossier|RunRenderingFailurePreservesGateEvidence|OverlayArchiveCleanupFailure)$'"],
 ['source-consumer-qualification',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./qualification/set ./qualification/soak -run '^Test(RunCountsRetainedRunnerFailureAsInfrastructure|AnalysisCommandBoundsAnalysisByWorkloadBudget|WorkloadCommandCarriesTheClockTick|RunClassifiesEveryWorkloadWhenAnalysisFails|ClassifyBatchKeepsOverflowAndInfrastructureApartFromDivergence|InfrastructureFailureIsNotAPassOrADivergence|ExhaustedBudgetIsReportedAsInfrastructure|FailedBaselineRetentionIsInfrastructureAndLeavesNoBaseline)$'"],
];
const results=[];
for(const [name,command]of gates) {
 const r=spawnSync(process.execPath,[path.join(out,'run.mjs'),name,command],{cwd:repo,stdio:'inherit'});
 results.push({name,exit:r.status,signal:r.signal});
 if(r.signal||r.status==null)break;
}
fs.writeFileSync(path.join(out,'consumer-gates.json'),JSON.stringify(results,null,2)+'\n',{flag:'wx'});
if(results.some(x=>x.exit!==0))process.exitCode=1;
