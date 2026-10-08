import {spawnSync} from 'node:child_process';
import {out,root,stock} from './capture.mjs';
const commands=[
 ['campaign-state-machines',stock+'/go','-C','tools/gomad3','test','-json','-count=1','-tags','test_dep','./runner/internal/campaign'],
 ['portable-runner-pure-boundaries',stock+'/go','-C','tools/gomad3','test','-json','-count=1','-tags','test_dep','./runner','-run','^(TestShardedSeedControllerSchedulesPendingOrdinalsAndSynchronizesStatistics|TestAssessWorldValidatesTheRecordAgainstItsSeed|TestAssessCompletionProjectsCoverageInOrderAndClassifies|TestDecideSuccessRetentionJudgesNoveltyTranscriptAndBounds|TestSuccessPublicationFailureSeparatesCapacityFromPublication|TestSuccessRetentionAnnotatesTheJournalRecord|TestExecutionArtifactInputCarriesTheCapturedEvidence|TestChoiceTraceObservedExcludesWatchdogAndCancellation|TestDecodeCoordinatorMessages.*|TestCoordinatorGroupProbeRequiresExplicitESRCHForDisappearance)$'],
 ['portable-package-architecture',stock+'/go','-C','tools/gomad3','test','-json','-count=1','-tags','test_dep','-run','^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestRunnerExternalConsumerCompiles)$','.'],
 ['configured-errortype',stock+'/go','-C','tools/gomad3','vet','-tags','test_dep','-vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype','-style-check=false','./runner','./runner/internal/campaign'],
 ['generated-validation','make','-C','tools/gomad3','validate'],
 ['read-only-gofmt',stock+'/gofmt','-l','tools/gomad3/runner/internal/campaign/controller.go','tools/gomad3/runner/internal/campaign/controller_test.go','tools/gomad3/runner/internal/campaign/controller_completion_test.go','tools/gomad3/runner/runner.go','tools/gomad3/runner/campaign_test.go','tools/gomad3/runner/seed_completion_characterization_test.go','tools/gomad3/runner/completion_characterization_test.go','tools/gomad3/runner/runner_test.go'],
 ['diff-whitespace','git','diff','--check']
];
for(const args of commands){const r=spawnSync('node',[out+'/capture.mjs',...args],{cwd:root,stdio:'inherit',timeout:610000});console.log(JSON.stringify({capture:args[0],exit:r.status,error:r.error?.message??null}));if(r.error)break;}
