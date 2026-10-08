import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {out,root,stock,environment} from './capture.mjs';
const go=stock+'/go',binary=out+'/gomad',moduleRoot=root+'/tools/gomad3';
const commands=[
 ['frozen-restored-causes-normal',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-run','^TestAssessCompletionProjectsCoverageInOrderAndClassifies$','./runner'],
 ['frozen-restored-causes-mutant',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-overlay',out+'/control-completion-current-mutant-overlay.json','-run','^TestAssessCompletionProjectsCoverageInOrderAndClassifies$','./runner'],
 ['frozen-root-failed-fixtures-recheck',go,'-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-run','^TestArchitectureEffectFixtures/(stored-callback|local-location)$','.'],
 ['frozen-built-cli-valid-explore',binary,'explore','--json','--working-dir',moduleRoot,'--artifacts',out+'/cli-valid-explore-artifacts','--seeds','1','go-run','./cmd/gomad/testdata/seedfree'],
 ['frozen-built-cli-valid-qualify',binary,'qualify','--json','--working-dir',moduleRoot,'--artifacts',out+'/cli-valid-qualify-artifacts','--seed','1','--repeat','2','go-run','./cmd/gomad/testdata/seedfree'],
 ['frozen-built-cli-valid-analyze',binary,'analyze','--format=json','--timeout','10s','--toolchain-root',moduleRoot+'/.toolchain','go-run','./tools/gomad3/cmd/gomad/testdata/seedfree'],
 ['frozen-size-detail','env','SIZE_COUNT_DETAIL=1','sh','.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh'],
 ['frozen-task-state','/home/agent/.codex/scripts/flowctl','show','fn-112-gomad-determinism-assurance-and-test.9','--json'],
 ['frozen-host-identity','uname','-a'],
 ['frozen-user-identity','id'],
];
for(const command of commands){const r=spawnSync('node',[out+'/capture.mjs',...command],{cwd:root,env:environment().env,stdio:'inherit',timeout:610000});assert(r.status!==null&&r.signal===null);}
