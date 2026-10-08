import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {out,root,stock,environment} from './capture.mjs';
for(const command of [
 ['sealed-causes-normal',stock+'/go','-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-run','^TestAssessCompletionProjectsCoverageInOrderAndClassifies$','./runner'],
 ['sealed-causes-mutant',stock+'/go','-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-overlay',out+'/control-completion-current-mutant-overlay.json','-run','^TestAssessCompletionProjectsCoverageInOrderAndClassifies$','./runner'],
 ['sealed-disk-space','df','-h','/home/agent/.cache/go-build',root],
 ['sealed-task-state','/home/agent/.codex/scripts/flowctl','show','fn-112-gomad-determinism-assurance-and-test.9','--json'],
]){const r=spawnSync('node',[out+'/capture.mjs',...command],{cwd:root,env:environment().env,stdio:'inherit',timeout:610000});assert(r.status!==null&&r.signal===null);}
