import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {read,out,sha,git,write} from './capture.mjs';
import {reconstruct,lineage,named,name,ref} from './proof.mjs';
import {spans,reconstruct as four} from '../../task-4/source-acceptance-20261008/proof.mjs';
import {alignment} from '../../task-4/source-acceptance-20261008/attribution.mjs';
const cli='tools/gomad3/cmd/gomad/internal/cli/';
function oldFunction(path,current,images){
 let p=path,n=name(current),candidates=[n+'WithApplication',n];
 if(n==='validateCampaignRequest')candidates.push('validateConfig');
 if(n.endsWith('With'))candidates=[n.slice(0,-4)+'WithApplication',n,...candidates.slice(1),n.slice(0,-4)];
 if(['parseCampaignRequest','reportExploreFailure','runPlanWith'].includes(n))candidates.push('runExploreWithApplication','runExplore');
 if(p===cli+'application.go'&&n==='runPrivateMode'){p=cli+'cli.go';candidates=['Run'];}
 if(p===cli+'application.go'&&n==='digestRunner'&&!images[p]){p=cli+'doctor.go';candidates=['hashExecutable'];}
 if(n==='run')candidates.push('Run');
 for(const candidate of candidates){const old=named(images[p]??'',candidate);if(old)return {path:p,...old};}return null;
}
function match(path,current,line,images){
 const old=oldFunction(path,current,images);if(!old)return null;
 const pairs=alignment(old.body,current.body),offset=line-current.start,pair=pairs.find(e=>e.after===offset),lines=current.body.split('\n'),before=old.body.split('\n');
 let branch=null;if(pair){const left=Math.max(0,pair.before-2),right=Math.min(before.length,pair.before+4),block=before.slice(left,right),afterStart=offset-(pair.before-left);if(afterStart>=0&&block.every((s,i)=>s===lines[afterStart+i]))branch={old_start:old.start+left,current_start:current.start+afterStart,lines:block,sha256:sha(block.join('\n'))};}
 return {path:old.path,signature:old.signature,function_sha256:sha(old.body),body:old.body,matched:!!pair,original_line:pair?old.start+pair.before:null,exact_line:pair?before[pair.before]:null,exact_branch:branch};
}
export function lint(labels=['configured-fast-lint','unfiltered-task-lint']){
 const r=reconstruct(),f=four(),fourPre=Object.fromEntries(f.pre.map(e=>[e.path,read('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-4/preimage/'+e.path).toString()])),first={},baseline='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/baseline-reconstruction',be=JSON.parse(read(baseline+'/evidence.json'));
 for(const l of read(baseline+'/source.sha256').toString().trim().split('\n')){const p=l.slice(66);if(p.endsWith('.go'))first[p]=read(be.scratch_path+'/'+p).toString();}
 const rows=[];for(const label of labels)for(const m of read(out+'/'+label+'.stdout').toString().matchAll(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$/gm)){
  const path=m[1],line=Number(m[2]),source=read(path).toString(),fn=spans(source).find(f=>line>=f.start&&line<f.end);assert(fn,path+':'+line);const exact=source.split('\n')[line-1];
  const matches={task5_pre:match(path,fn,line,r.pre),task5_post:match(path,fn,line,r.images),task4_pre:match(path,fn,line,fourPre),task4_post:match(path,fn,line,f.images),first_fn109:match(path,fn,line,first)};
  for(const e of Object.values(matches))if(e?.matched)assert.equal(e.exact_line,exact);
  const blame=git(['blame','--porcelain','-L',line+','+line,'--',path]).toString(),owner=blame.split(' ')[0],dirty=/^0+$/.test(owner),effective=dirty?git(['rev-parse','HEAD']).toString().trim():owner,parents=git(['show','-s','--format=%P',effective]).toString().trim().split(' ').filter(Boolean),hunks=parents.map(parent=>({parent,patch:git(['diff','--no-ext-diff','--no-renames',parent,effective,'--',path]).toString()}));
  rows.push({label,path,line,column:Number(m[3]),message:m[4],rule:m[5],current:{signature:fn.signature,start:fn.start,end:fn.end,body:fn.body,sha256:sha(fn.body),exact_line:exact,blame,owner_commit:dirty?null:owner,owner_metadata:git(['show','-s','--format=%H%n%P%n%B',effective]).toString(),owner_parent_hunks:hunks,dirty_restoration:dirty?{parent:effective,patch:git(['diff','--no-ext-diff','--',path]).toString(),original_owner_parent_hunks:lineage(path)}:null},original_matches:matches,inside_original_task5_edit_set:Object.hasOwn(r.images,path),classification:matches.task5_pre?.matched?'Original task5 preimage exact statement; actual branch and owner-parent hunk retained':matches.task4_pre?.matched?'Original task4 preimage exact statement; exact task5 ownership not inferred':matches.first_fn109?.matched?'First fn109 baseline exact statement; later owners still retained':matches.task5_post?.matched?'Original task5 postimage statement; no preservation exception or cleanliness inferred':'Exact alignment absent; explicit branch reconciliation required; actual owner-parent hunks alone confer no waiver',exception_applied:false});
 }
 return {findings:rows,labels,method:'Whole named-function byte alignment, exact original finding line and bounded branch, actual local/rebased/merge owner-parent diffs. Unmatched statements remain explicit until separately reconciled.',counts:{configured_fast:rows.filter(e=>e.label===labels[0]).length,unfiltered_cli_runner:rows.filter(e=>e.label===labels[1]).length,unfiltered_cli:rows.filter(e=>e.label===labels[1]&&e.path.startsWith(cli)).length,unfiltered_runner:rows.filter(e=>e.label===labels[1]&&!e.path.startsWith(cli)).length},task4_bounded_unmatched_proof:ref('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-4/conductor-source-acceptance-20261008/audit.json'),global_green:false,new_waivers:[],correction_owners_open:true};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){write(process.argv[2]??'lint-attribution-corrected.json',lint(process.argv[3]?process.argv.slice(3):undefined));console.log('all configured and unfiltered findings retained');}
