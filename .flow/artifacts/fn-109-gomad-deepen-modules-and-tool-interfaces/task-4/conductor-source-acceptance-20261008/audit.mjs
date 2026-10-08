import assert from 'node:assert/strict';
import {readFileSync,writeFileSync} from 'node:fs';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {read,sha,sources} from '../source-acceptance-20261008/capture.mjs';
import {reconstruct,spans} from '../source-acceptance-20261008/proof.mjs';
import {alignment} from '../source-acceptance-20261008/attribution.mjs';
const out=dirname(fileURLToPath(import.meta.url)),worker=resolve(out,'../source-acceptance-20261008');
const json=(dir,name)=>JSON.parse(readFileSync(resolve(dir,name+'.json')));
const cli='tools/gomad3/cmd/gomad/internal/cli/',r=reconstruct();
const pre=Object.fromEntries(r.pre.map(e=>[e.path,read('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-4/preimage/'+e.path).toString()]));
const named=(s,n)=>spans(s).find(f=>new RegExp('^func (?:\\([^)]*\\) )?'+n+'\\(').test(f.signature));
const core=f=>{const a=f.body.split('\n'),end=a.indexOf('}',1);assert(end>0);return a.slice(1,end);};
const rows=json(worker,'lint-attribution-frozen').findings;
const unmatched=rows.filter(f=>f.label==='unfiltered-cli-lint'&&!f.original_task4_pre?.matched&&!f.original_task4_post?.matched);
assert.equal(unmatched.length,13);
const decisions=unmatched.map(f=>{
 const row={path:f.path,line:f.line,rule:f.rule,current_function_sha256:f.current.sha256,exception_applied:false};
 if(f.path.endsWith('qualify.go')||f.path.endsWith('resume.go')){
  const name=f.path.endsWith('qualify.go')?'runQualifyWith':'runResumeWith',offset=f.line-f.current.start;
  const matches=[pre,r.images].map(images=>{const old=named(images[f.path],name);assert(old);const match=alignment(old.body,f.current.body).find(m=>m.after===offset);assert(match);assert.equal(old.body.split('\n')[match.before],f.current.exact_line);const current=f.current.body.split('\n'),oldLines=old.body.split('\n');assert.deepEqual(oldLines.slice(match.before-1,match.before+3),current.slice(offset-1,offset+3));return {function:name,function_sha256:sha(old.body),line:old.start+match.before,branch:oldLines.slice(match.before-1,match.before+3)};});
  return {...row,disposition:'Original named dependency-helper error branch preserved exactly; frozen alias mapper selected its wrapper instead. Finding remains open, not waived.',original_pre_and_post:matches};
 }
 if(f.path===cli+'cli.go'&&(f.line===450||f.line===454)){
  const current=core(named(read(f.path).toString(),'reportExploreFailure'));
  const oldBranches=[['pre',pre,'runExplore'],['post',r.images,'runExploreWithApplication']].map(([label,images,name])=>{const old=named(images[f.path],name),lines=old.body.split('\n'),start=lines.indexOf('\t\tif summary.ChoiceTrace != nil {'),end=lines.indexOf('\t\treturn exploreErrorStatus(classification)',start);assert(start>=0&&end>start);const branch=lines.slice(start,end+1);assert(branch.every(l=>l.startsWith('\t')));assert.deepEqual(branch.map(l=>l.slice(1)),current);return {label,function_sha256:sha(old.body),start:old.start+start,end:old.start+end,branch,transform:'Exactly one leading tab removed from each line of the entire extracted error branch; no token or ordering change.'};});
  return {...row,disposition:'Whole original explore failure branch relocated with one indentation level removed; writer attempts and status/classification unchanged. Finding remains open.',original_pre_and_post:oldBranches};
 }
 if(f.path===cli+'cli.go'&&f.line===491){
  const current=named(read(f.path).toString(),'runPlanWith').body.split('\n'),start=current.indexOf('\t\tencoded, err := json.Marshal(planned)'),block=current.slice(start,start+6);assert(start>=0);
  const oldBranches=[['pre',pre,'runExplore'],['post',r.images,'runExploreWithApplication']].map(([label,images,name])=>{const old=named(images[f.path],name),lines=old.body.split('\n'),start=lines.indexOf('\t\t\tencoded, err := json.Marshal(planned)'),branch=lines.slice(start,start+6);assert(start>=0&&branch.every(l=>l.startsWith('\t')));assert.deepEqual(branch.map(l=>l.slice(1)),block);return {label,function_sha256:sha(old.body),start:old.start+start,branch,transform:'Exactly one leading tab removed from each of the six marshal/error branch lines.'};});
  return {...row,disposition:'Original plan JSON marshal/error branch extracted by task5; same stderr attempt and status3, existing finding remains open.',original_pre_and_post:oldBranches};
 }
 assert((f.path===cli+'application.go'&&f.line===88)||(f.path===cli+'cli.go'&&(f.line===369||f.line===374)));
 return {...row,disposition:'Source-equivalent relocated writer attempt: one stderr write, status3 for successful or failed writer. Executable discovery/path prefixes and wrapping bound separately by writer-equivalence.json; not an old-runtime reproduction or a lint waiver.',source_equivalence_receipt_sha256:sha(read(worker+'/writer-equivalence.json'))};
});
const exact=rows.filter(f=>f.original_task4_pre?.matched||f.original_task4_post?.matched);
const boundedIds=new Set(decisions.map(f=>f.path+':'+f.line));
const outside=rows.filter(f=>!f.original_task4_pre?.matched&&!f.original_task4_post?.matched&&!boundedIds.has(f.path+':'+f.line));
assert(outside.every(f=>!Object.hasOwn(r.images,f.path)), 'An unclassified finding lies in the original task4 edit set');
const labels=['worker-integrity','portable-cli-passing-set','portable-architecture-installation','portable-qualification-control','configured-generators','configured-cli-errortype','cli-format-check','configured-fast-lint','unfiltered-cli-lint'];
const receipts=labels.map(label=>{const x=json(out,label);assert.equal(x.exit,label.endsWith('lint')?2:0);assert.equal(x.signal,null);assert.equal(x.error,null);assert.deepEqual(x.source_changes,[]);assert.equal(x.stdout_sha256,sha(read(out+'/'+label+'.stdout')));assert.equal(x.stderr_sha256,sha(read(out+'/'+label+'.stderr')));assert.equal(x.sources_before_sha256,x.sources_after_sha256);return {label,sha256:sha(read(out+'/'+label+'.json')),exit:x.exit};});
const terminal=x=>x.tests.map(t=>JSON.stringify(t)).sort();
for(const label of ['portable-architecture-installation','portable-qualification-control'])assert.deepEqual(terminal(json(out,label)),terminal(json(worker,label)));
const full=json(worker,'portable-cli-full-temp-retry'),rootCli=json(out,'portable-cli-passing-set'),selected=new Set(full.tests.filter(t=>t.action==='pass'&&!t.test.includes('/')).map(t=>t.test));
assert.equal(selected.size,89);assert.deepEqual(terminal(rootCli),terminal({...full,tests:full.tests.filter(t=>selected.has(t.test.split('/')[0]))}));
const top=labels.flatMap(l=>json(out,l).tests.filter(t=>!t.test.includes('/')));assert.equal(top.length,129);assert(top.every(t=>t.action==='pass'));assert.equal(new Set(top.map(t=>t.package+' '+t.test)).size,129);
const findings=b=>b.toString().split('\n').filter(l=>/^tools\/gomad3\/[^:]+:\d+:\d+: .+ \([^)]+\)$/.test(l));
for(const label of ['configured-fast-lint','unfiltered-cli-lint'])assert.deepEqual(findings(read(out+'/'+label+'.stdout')),findings(read(worker+'/'+label+'.stdout')));
const result={readonly:true,no_go:true,task:'fn-109-gomad-deepen-modules-and-tool-interfaces.4',worker_evidence_sha256:sha(read(worker+'/evidence.json')),worker_freeze_sha256:sha(read(worker+'/freeze.json')),source_identity_sha256:sha(JSON.stringify(sources())),receipts,unique_fresh_portable_pass:129,fail:0,skip:0,full_cli_worker_observation:{pass:89,fail:3,all_cli_green:false},lint:{configured_fast:68,unfiltered_cli:53,exact_pre_or_post_rows:exact.length,bounded_source_dispositions:decisions,outside_original_task4_edit_set:outside.map(f=>({label:f.label,path:f.path,line:f.line,rule:f.rule,owner:f.current.owner_commit,disposition:'Outside the exact ten-file original task4 edit set; separate correction/aggregate obligations remain open, not accepted through blame alone.'})),new_exceptions:[],global_green:false},native:false};
if(process.argv[2])writeFileSync(resolve(out,process.argv[2]),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({readonly:true,unique_fresh_portable_pass:129,skip:0,unchanged_fast_findings:68,unchanged_cli_findings:53,classified_cli_unmatched:13,outside_edit_set:outside.length,global_green:false,native:false}));
