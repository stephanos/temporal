import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const git=(...args)=>{const result=spawnSync('git',args,{cwd:repo,encoding:'utf8'});if(result.status!==0)throw Error(result.stderr);return result.stdout;};
const baseline='b4602685b3184387cf2d713178095247f0c11d8f',findings=[];
for(const line of fs.readFileSync(path.join(out,'lint-affected-unfiltered.log'),'utf8').split('\n')){
 const match=line.match(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.*) \(([^()]+)\)$/);if(!match)continue;
 const [,file,row,column,message,rule]=match,current=fs.readFileSync(path.join(repo,file)),original=git('show',baseline+':'+file),blame=git('blame','--line-porcelain','-L',row+','+row,baseline,'--',file),commit=blame.split(' ')[0];
 const commitBody=git('show','-s','--format=%B',commit).trim();
 findings.push({path:file,line:Number(row),column:Number(column),message,rule,line_text:current.toString().split('\n')[Number(row)-1],source_sha256:hash(current),admission_source_sha256:hash(original),whole_file_unchanged:hash(current)===hash(original),origin_commit:commit,origin_subject:commitBody.split('\n')[0],origin_task_lines:commitBody.split('\n').filter(s=>/Task:|fn-\d+/.test(s)),disposition:'retained admission finding; outside task7 changed implementation and new test files; no waiver or global green claim'});
}
if(findings.length!==40||findings.some(f=>!f.whole_file_unchanged))throw Error('unattributed or changed lint source');
fs.writeFileSync(path.join(out,'lint-attribution.json'),JSON.stringify({command_receipt:'lint-affected-unfiltered-receipt.json',count:findings.length,new_task7_findings:0,findings},null,2)+'\n');
