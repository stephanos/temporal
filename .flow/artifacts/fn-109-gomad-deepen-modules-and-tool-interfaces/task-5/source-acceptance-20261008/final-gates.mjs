import {spawnSync} from 'node:child_process';
import {root,out,stock,read} from './capture.mjs';
const load=n=>JSON.parse(read(out+'/'+n+'.json'));
const passing=n=>load(n).tests.filter(t=>t.action==='pass'&&!t.test.includes('/')).map(t=>t.test);
const go=stock+'/go',gci='/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',err='/tmp/fn109-lint-tools.ZdNe1t50/errortype';
const tests=(packages,pattern)=>[go,'-C','tools/gomad3','test','-count=1','-tags','test_dep','-json',...packages,...(pattern?['-run',pattern]:[]),'-timeout','90s'];
const commands=[
 ['restored-portable-cli',...tests(['./cmd/gomad/internal/cli'],'^('+[...passing('portable-cli-full'),'TestExploreErrorJoinsDiagnosticAndReporterWriterFailures'].join('|')+')$')],
 ['restored-portable-runner',...tests(['./runner'],'^('+passing('focused-runner').join('|')+')$')],
 ['restored-architecture-installation',...load('portable-architecture-installation').argv],
 ['restored-qualification-control',...load('portable-qualification-control').argv],
 ['restored-generators','make','-C','tools/gomad3','validate'],
 ['restored-errortype',go,'-C','tools/gomad3','vet','-tags','test_dep','-vettool='+err,'-style-check=false','./cmd/gomad/internal/cli','./runner'],
 ['restored-format-check',...load('task-format-check').argv],
 ['restored-diff-check',...load('product-diff-check').argv],
 ['restored-fast-lint','make','lint-code-fast','GOLANGCI_LINT_FIX=false','GOLANGCI_LINT_BASE_REV=d635e23f00d926a43b942f25a9d05bd0ccb72025','GOLANGCI_LINT='+gci,'ERRORTYPE='+err,'ALL_TEST_TAGS=test_dep'],
 ['restored-unfiltered-lint','make','lint-code','GOLANGCI_LINT_FIX=false','GOLANGCI_LINT_BASE_REV=','GOLANGCI_LINT='+gci,'ERRORTYPE='+err,'LINT_CODE_DIR='+root+'/tools/gomad3','LINT_CODE_TARGETS=./cmd/gomad/internal/cli ./runner','ALL_TEST_TAGS=test_dep'],
];
for(const args of commands){const r=spawnSync('node',[out+'/capture.mjs',...args],{cwd:root,timeout:620000,maxBuffer:8<<20});process.stdout.write(r.stdout??'');process.stderr.write(r.stderr??'');if(r.error||r.signal)throw new Error(JSON.stringify({label:args[0],status:r.status,signal:r.signal,error:r.error?.message}));}
console.log('post-restoration serialized gates completed');
