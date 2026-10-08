import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {out,root,stock,read,environment} from './capture.mjs';
const runnerNames=[...new Set(JSON.parse(read(out+'/source-proof.json')).mapping_rows.filter(r=>r.package==='runner'&&r.current).map(r=>r.current.function))];
runnerNames.push('TestAssessWorldValidatesTheRecordAgainstItsSeed','TestAssessCompletionProjectsCoverageInOrderAndClassifies','TestIsolatedRunnerDrainsFastCoordinatorBeforeWaitClosesOutput','TestRunRetainsSuccessesOnlyWithinExplicitBounds');
const lint='/tmp/fn109-lint-tools.ZdNe1t50',go=stock+'/go',tags='ALL_TEST_TAGS=test_dep',common=['GOLANGCI_LINT_FIX=false','GOLANGCI_LINT='+lint+'/golangci-lint-v2.13.0','ERRORTYPE='+lint+'/errortype',tags];
const gates=[
 ['final-deterministicio',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=3m','-json','./deterministicio'],
 ['final-cli',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','./cmd/gomad/internal/cli'],
 ['final-runner-focused',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-run','^('+[...new Set(runnerNames)].join('|')+')$','./runner'],
 ['final-validate','make','-C','tools/gomad3','validate'],
 ['final-scoped-vet',go,'-C','tools/gomad3','vet','-tags','test_dep','.','./deterministicio','./cmd/gomad/internal/cli','./runner'],
 ['final-scoped-errortype',go,'-C','tools/gomad3','vet','-tags','test_dep','-vettool='+lint+'/errortype','-style-check=false','.','./deterministicio','./cmd/gomad/internal/cli','./runner'],
 ['final-unfiltered-scoped-lint','make','lint-code',...common,'GOLANGCI_LINT_BASE_REV=','LINT_CODE_DIR='+root+'/tools/gomad3','LINT_CODE_TARGETS=. ./deterministicio/... ./runner/... ./cmd/gomad/...'],
 ['final-configured-fast-lint','make','lint-code-fast',...common,'GOLANGCI_LINT_BASE_REV=59ca3d17395be5501906586006e2539d33bea28c'],
 ['final-focused-format','node',out+'/format-check.mjs','focused'],
 ['final-nested-format','node',out+'/format-check.mjs','nested'],
 ['final-size','sh','.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh'],
 ['final-checker-controls','python3','-B',out+'/checker-controls.py'],
 ['built-cli-doctor',out+'/gomad','doctor','--json','--artifacts',out+'/cli-doctor-artifacts'],
 ['built-cli-analyze',out+'/gomad','analyze','--format=json','--timeout','10s','go-run','./tools/gomad3/testdata/helloworld'],
 ['built-cli-explore',out+'/gomad','explore','--json','--artifacts',out+'/cli-explore-artifacts','--seeds','1','go-run','./tools/gomad3/testdata/helloworld'],
 ['built-cli-qualify',out+'/gomad','qualify','--json','--artifacts',out+'/cli-qualify-artifacts','--seeds','1','--repeat','0','go-run','./tools/gomad3/testdata/helloworld'],
 ['built-cli-resume',out+'/gomad','resume','--json',out+'/missing-campaign'],
 ['built-cli-minimize',out+'/gomad','minimize','--json','--artifacts',out+'/cli-minimize-artifacts',out+'/missing-artifact'],
 ['final-go-env',go,'env','-json'],
 ['final-task-state','/home/agent/.codex/scripts/flowctl','show','fn-112-gomad-determinism-assurance-and-test.9','--json'],
];
const prefix=process.argv[2]??'';
if(prefix)gates.unshift(['root',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=5m','-json','.']);
if(prefix)gates.splice(gates.findIndex(g=>g[0]==='built-cli-doctor'));
for(const [label,...command] of gates){const r=spawnSync('node',[out+'/capture.mjs',prefix+label,...command],{cwd:root,env:environment().env,stdio:'inherit',timeout:610000});assert.equal(r.signal,null);assert(r.status!==null);}
