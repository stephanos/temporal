import {spawnSync} from 'node:child_process';
import {root,out,stock} from './capture.mjs';
const commands=[
 ['final-native-guarded-assertions',stock+'/go','-C','tools/gomad3','test','-json','-count=1','-tags','test_dep','./runner','-run','^(TestCompletionFaultsKeepReasonPrecedenceAndEvidence|TestRunRejectsSuccessfulRetentionWithoutReplayTranscript)$/^(World_seed_mismatch|malformed_semantic_coverage)$/^seed$'],
 ['final-portable-runner-boundaries',stock+'/go','-C','tools/gomad3','test','-json','-count=1','-tags','test_dep','./runner','-run','^(TestShardedSeedControllerSchedulesPendingOrdinalsAndSynchronizesStatistics|TestAssessWorldValidatesTheRecordAgainstItsSeed|TestAssessCompletionProjectsCoverageInOrderAndClassifies|TestDecideSuccessRetentionJudgesNoveltyTranscriptAndBounds|TestSuccessPublicationFailureSeparatesCapacityFromPublication|TestSuccessRetentionAnnotatesTheJournalRecord|TestExecutionArtifactInputCarriesTheCapturedEvidence|TestChoiceTraceObservedExcludesWatchdogAndCancellation|TestDecodeCoordinatorMessages.*|TestCoordinatorGroupProbeRequiresExplicitESRCHForDisappearance)$'],
 ['final-configured-fast-lint','make','lint-code-fast','GOLANGCI_LINT_FIX=false','GOLANGCI_LINT_BASE_REV=48a2c8cd9eafe45e435022838c72864be16453f3','GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0','ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype','ALL_TEST_TAGS=test_dep'],
 ['final-relevant-unfiltered-lint','make','lint-code','GOLANGCI_LINT_FIX=false','GOLANGCI_LINT_BASE_REV=','GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0','ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype','LINT_CODE_DIR='+root+'/tools/gomad3','LINT_CODE_TARGETS=./runner ./runner/internal/campaign','ALL_TEST_TAGS=test_dep'],
 ['final-errortype',stock+'/go','-C','tools/gomad3','vet','-tags','test_dep','-vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype','-style-check=false','./runner','./runner/internal/campaign'],
 ['final-generated-validation','make','-C','tools/gomad3','validate'],
 ['final-gofmt',stock+'/gofmt','-l','tools/gomad3/runner/runner_test.go','tools/gomad3/runner/completion_characterization_test.go'],
 ['final-diff-check','git','diff','--check']
];
for(const args of commands){const r=spawnSync('node',[out+'/capture.mjs',...args],{cwd:root,stdio:'inherit',timeout:610000});console.log(JSON.stringify({capture:args[0],exit:r.status,error:r.error?.message??null}));if(r.error)break;}
