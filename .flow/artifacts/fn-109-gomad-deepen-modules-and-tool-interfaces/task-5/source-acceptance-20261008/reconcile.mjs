import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {read,out,sha,write} from './capture.mjs';
import {reconstruct,named,ref,lineage} from './proof.mjs';
export function reconcile(){
 const r=reconstruct(),p='tools/gomad3/cmd/gomad/internal/cli/cli.go',current=read(p).toString(),proof=JSON.parse(read(out+'/lint-attribution-final.json')),oldPlan=named(r.images[p],'runPlanWithApplication'),nowPlan=named(current,'runPlanWith');
 const writerPath='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-4/source-acceptance-20261008/writer-equivalence.json',writers=JSON.parse(read(writerPath));assert.equal(named(current,'runDoctor').body,writers.current_doctor.body);
 const unmatched=proof.findings.filter(e=>e.inside_original_task5_edit_set&&!e.original_matches.task5_pre?.matched&&!e.original_matches.task5_post?.matched);
 const decisions=unmatched.map(e=>{
  if(e.current.signature.includes('runDoctor')){assert([369,374].includes(e.line));return {label:e.label,path:e.path,line:e.line,original_task4_writer_binding:ref(writerPath),current_doctor_sha256:sha(writers.current_doctor.body),branch_bindings:writers.branch_bindings.slice(0,2),disposition:'Existing exact task4 named doctor function and wrapper-bound writer branches unchanged by task5 restoration; finding remains open, no new exception.'};}
  assert(e.current.signature.includes('runPlanWith'));const argument=e.current.exact_line.includes('stderr, writeErr')?'writeErr':'err',prefix=e.current.exact_line.slice(0,e.current.exact_line.indexOf('fmt.'));
  const before=prefix+'if _, printErr := fmt.Fprintln(stderr, '+argument+'); printErr != nil {\n'+prefix+'\treturn 3\n'+prefix+'}\n'+prefix+'return 3';
  const after=prefix+'fmt.Fprintln(stderr, '+argument+')\n'+prefix+'return 3';assert(oldPlan.body.includes(before));assert(nowPlan.body.includes(after));
  return {label:e.label,path:e.path,line:e.line,original_function_sha256:sha(oldPlan.body),current_function_sha256:sha(nowPlan.body),original_branch:before,current_branch:after,successful_writer_status:3,failed_writer_status:3,stderr_write_attempts:1,same_write_expression:'fmt.Fprintln(stderr, '+argument+')',disposition:'Original task5 checked-print branch and integrated ignored-print branch attempt identical bytes once and return3 for either result. This is bounded observable source equivalence, not source-byte identity or lint cleanliness. Exact ignored-print correction remains open.',actual_owner_parent_hunks:lineage(p)};
 });
 assert.equal(new Set(decisions.map(e=>e.path+':'+e.line)).size,5);
 const outside=proof.findings.filter(e=>!e.inside_original_task5_edit_set&&!Object.values(e.original_matches).some(x=>x?.matched));
 return {all_final_findings:proof.counts,exact_match_rows:proof.findings.filter(e=>e.original_matches.task5_pre?.matched||e.original_matches.task5_post?.matched).length,bounded_unmatched:decisions,outside_original_task5_edit_set:outside.map(e=>({label:e.label,path:e.path,line:e.line,rule:e.rule,actual_owner_parent_hunks:e.current.owner_parent_hunks,current_function_sha256:e.current.sha256,disposition:'Separate correction/aggregate residual remains open; no blame-as-waiver or task5 ownership inferred.'})),restored_explore_helper_exact:sha(named(current,'reportExploreFailure').body)===sha(named(r.images[p],'reportExploreFailure').body),global_green:false,new_waivers:[],native:false};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){write('lint-reconciliation.json',reconcile());console.log('bounded task5 lint branches reconciled without waivers');}
