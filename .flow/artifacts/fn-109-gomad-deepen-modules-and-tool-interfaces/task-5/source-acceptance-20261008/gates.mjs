import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {read,root,out,stock} from './capture.mjs';
const load=n=>JSON.parse(read(out+'/'+n+'.json'));
const passing=n=>'^('+load(n).tests.filter(t=>t.action==='pass'&&!t.test.includes('/')).map(t=>t.test).join('|')+')$';
const go=stock+'/go',gci='/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',err='/tmp/fn109-lint-tools.ZdNe1t50/errortype';
const tests=(packages,pattern)=>[go,'-C','tools/gomad3','test','-count=1','-tags','test_dep','-json',...packages,...(pattern?['-run',pattern]:[]),'-timeout','90s'];
const commands=[
 ['patched-driver-prerequisite','bash','-c','test -x tools/gomad3/.toolchain/bin/go'],
 ['portable-cli-passing-set',...tests(['./cmd/gomad/internal/cli'],passing('portable-cli-full'))],
 ['portable-runner-passing-set',...tests(['./runner'],passing('focused-runner'))],
 ['portable-architecture-installation',...tests(['.','./toolchain'],'^(TestPackageArchitecture|TestArchitecture.*Fixtures|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestRunnerExecutionInjectionIsPrivate|TestCurrentVocabularyHasNoLegacyCampaignBoundary|TestMakeTargetsMatchTheirOwnership|TestExactModuleEdges|TestDomainModulesDoNotExportWireFraming|TestResolve.*|TestEveryResolutionSourceYieldsAValidatedDescription)$')],
 ['portable-qualification-control',...tests(['./qualification'])],
 ['configured-generators','make','-C','tools/gomad3','validate'],
 ['configured-task-errortype',go,'-C','tools/gomad3','vet','-tags','test_dep','-vettool='+err,'-style-check=false','./cmd/gomad/internal/cli','./runner'],
 ['task-format-check','bash','-c','result=$(/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt -l tools/gomad3/cmd/gomad/internal/cli/*.go tools/gomad3/runner/*.go); rc=$?; printf "%s" "$result"; test "$rc" -eq 0 && test -z "$result"'],
 ['product-diff-check','git','diff','--check','--','tools/gomad3','tools/gomad3sim','tools/gomad3integration','tests/gomadfunctional'],
 ['configured-fast-lint','make','lint-code-fast','GOLANGCI_LINT_FIX=false','GOLANGCI_LINT_BASE_REV=d635e23f00d926a43b942f25a9d05bd0ccb72025','GOLANGCI_LINT='+gci,'ERRORTYPE='+err,'ALL_TEST_TAGS=test_dep'],
 ['unfiltered-task-lint','make','lint-code','GOLANGCI_LINT_FIX=false','GOLANGCI_LINT_BASE_REV=','GOLANGCI_LINT='+gci,'ERRORTYPE='+err,'LINT_CODE_DIR='+root+'/tools/gomad3','LINT_CODE_TARGETS=./cmd/gomad/internal/cli ./runner','ALL_TEST_TAGS=test_dep'],
];
for(const args of commands){const r=spawnSync('node',[out+'/capture.mjs',...args],{cwd:root,timeout:620000,maxBuffer:8<<20});process.stdout.write(r.stdout??'');process.stderr.write(r.stderr??'');if(r.error||r.signal)throw new Error(JSON.stringify({label:args[0],status:r.status,signal:r.signal,error:r.error?.message}));}
console.log('serialized gate lane completed');
