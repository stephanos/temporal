import {spawnSync} from 'node:child_process';
import assert from 'node:assert/strict';
import {root,out,stock,environment} from './capture.mjs';
for(const [label,overlay] of [['completion-original-cause-baseline','normal'],['completion-original-cause-mutant','mutant'],['completion-current-cause-mutant','current']]){
 const filename=overlay==='current'?'control-completion-current-mutant-overlay.json':'control-completion-original-'+overlay+'-overlay.json';
 const r=spawnSync('node',[out+'/capture.mjs',label,stock+'/go','-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-overlay',out+'/'+filename,'-run','^TestAssessCompletionProjectsCoverageInOrderAndClassifies$','./runner'],{cwd:root,env:environment().env,stdio:'inherit',timeout:610000});
 assert(r.status!==null&&r.signal===null);
}
