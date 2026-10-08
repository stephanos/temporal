import assert from 'node:assert/strict';
import {existsSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {root,out,read,sha,git,write} from './capture.mjs';
import {reconstruct,spans,original,ref} from './proof.mjs';
const cli='tools/gomad3/cmd/gomad/internal/cli/';
const name=f=>/^func (?:\([^)]*\) )?(\w+)/.exec(f.signature)[1];
export function alignment(before,after){
 const a=before.split('\n'),b=after.split('\n'),matrix=Array.from({length:a.length+1},()=>new Uint32Array(b.length+1));
 for(let i=a.length-1;i>=0;i--)for(let j=b.length-1;j>=0;j--)matrix[i][j]=a[i]===b[j]?1+matrix[i+1][j+1]:Math.max(matrix[i+1][j],matrix[i][j+1]);
 let i=0,j=0;const matches=[];while(i<a.length&&j<b.length){if(a[i]===b[j])matches.push({before:i++,after:j++});else if(matrix[i+1][j]>=matrix[i][j+1])i++;else j++;}return matches;
}
function locate(s,line){return spans(s).find(f=>line>=f.start&&line<f.end);}
function candidate(path,f,images){
 let p=path,n=name(f);
 if(path===cli+'application.go'&&n==='runPrivateMode'){p=cli+'cli.go';n='Run';}
 else if(path===cli+'application.go'&&n==='digestRunner'){p=images[cli+'application.go']?cli+'application.go':cli+'doctor.go';n='hashExecutable';}
 else if(path===cli+'cli.go'&&n==='run')n='Run';
 if(n==='parseCampaignRequest'||n==='reportExploreFailure'||n==='runPlanWith')n='runExplore';
 if(n.endsWith('With'))n=n.slice(0,-4);
 const fs=spans(images[p]??'');let found=fs.find(x=>name(x)===n+'WithApplication');found??=fs.find(x=>name(x)===n);return found?{path:p,...found}:null;
}
function oldMatch(path,f,offset,images){
 const old=candidate(path,f,images);if(!old)return null;
 const m=alignment(old.body,f.body).find(m=>m.after===offset);return {path:old.path,signature:old.signature,source_function_sha256:sha(old.body),start:old.start,end:old.end,body:old.body,line:m?old.start+m.before:null,exact_line:m?old.body.split('\n')[m.before]:null,matched:!!m};
}
export function lint(){
 const r=reconstruct(),pre=Object.fromEntries(r.pre.map(e=>[e.path,read(original+'/preimage/'+e.path).toString()])),baseline=JSON.parse(read('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/baseline-reconstruction/evidence.json')),first={};
 const manifest=read('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/baseline-reconstruction/source.sha256').toString();for(const l of manifest.trim().split('\n')){const p=l.slice(66);if(p.endsWith('.go'))first[p]=read(baseline.scratch_path+'/'+p).toString();}
 const rows=[];for(const label of ['configured-fast-lint','unfiltered-cli-lint']){
  const log=read(out+'/'+label+'.stdout').toString();for(const m of log.matchAll(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$/gm)){
   const path=m[1],line=Number(m[2]),source=read(path).toString(),f=locate(source,line);assert(f,path+':'+line+' has function');const offset=line-f.start,currentLine=source.split('\n')[line-1];
   const post=oldMatch(path,f,offset,r.images),before=oldMatch(path,f,offset,pre),firstMatch=oldMatch(path,f,offset,first);
   for(const v of [post,before,firstMatch])if(v?.matched)assert.equal(v.exact_line,currentLine);
   const blame=git(['blame','--porcelain','-L',line+','+line,'--',path]).toString(),commit=blame.split(' ')[0];assert(/^[a-f0-9]{40}$/.test(commit));
   const parents=git(['show','-s','--format=%P',commit]).toString().trim().split(' ').filter(Boolean),hunks=parents.map(parent=>({parent,patch:git(['diff','--no-ext-diff','--no-renames',parent,commit,'--',path]).toString()}));
   rows.push({label,path,line,column:Number(m[3]),message:m[4],rule:m[5],current:{signature:f.signature,start:f.start,end:f.end,body:f.body,sha256:sha(f.body),exact_line:currentLine,blame,owner_commit:commit,owner_metadata:git(['show','-s','--format=%H%n%P%n%B',commit]).toString(),owner_parent_hunks:hunks},original_task4_post:post,original_task4_pre:before,first_fn109_baseline:firstMatch,classification:before?.matched?'original task4 pre-existing exact statement':post?.matched?'present in original task4 postimage; inspect retained preimage and actual owner hunk':firstMatch?.matched?'first fn109 baseline exact statement':'exact pre/post alignment absent; current owner and parent hunks bound; conductor must classify disposition',exception_applied:false});
  }
 }
 assert.equal(rows.filter(r=>r.label==='configured-fast-lint').length,68);assert.equal(rows.filter(r=>r.label==='unfiltered-cli-lint').length,53);
 return {method:'Whole named-function exact-byte LCS alignment, explicit relocated function mappings, exact finding lines, original pre/post bodies, original 670-input baseline and current line blame plus actual parent diffs. No includes/trimmed-string classification.',findings:rows,correction_owners_remain_open:['fn109.21','fn109 correction owners'],global_green:false,new_waivers:[],task4_source_edited:false};
}
export function assertions(){
 const r=reconstruct(),tests=[['cli_test.go','TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput'],['cli_test.go','TestRunDoctorReportsAvailableContractAsJSON'],['doctor_test.go','TestCheckReportsAvailableContract']],rows=tests.map(([file,n])=>{
  const path=cli+file,current=spans(read(path).toString()).find(f=>name(f)===n);assert(current);const old=spans(r.images[path]??read(original+'/preimage/'+path).toString()).find(f=>name(f)===n);assert(old);return {test:n,path,current,original:old,assertion_alignment:alignment(old.body,current.body),current_sha256:sha(current.body),original_sha256:sha(old.body),current_blame:git(['blame','--porcelain','-L',current.start+','+current.end,'--',path]).toString(),trigger:n.includes('Analyze')?'real target.ReviewCapabilities starts absent .toolchain/bin/go before the expected missing go.sum diagnostic':'runtime.GOOS/runtime.GOARCH supplies actual linux/arm64; hostCheck rejects it before Available can be true',portable_pass_claimed:false};
 });
 const functions=[['doctor.go','Check'],['doctor.go','hostCheck'],['cli.go','runDoctor'],['analyze.go','runAnalyzeWith']].map(([file,n])=>{const path=cli+file,f=spans(read(path).toString()).find(f=>name(f)===n);assert(f);return {path,...f,sha256:sha(f.body)};});
 const doctorPost=spans(r.images[cli+'cli.go']).find(f=>name(f)==='runDoctorWithApplication'),doctorPre=spans(r.preimages[cli+'cli.go']).find(f=>name(f)==='runDoctor'),doctorCurrent=functions.find(f=>name(f)==='runDoctor');
 const ignored='\t\tfmt.Fprintf(stdout, "%s\\n", encoded)';for(const f of [doctorPre,doctorPost,doctorCurrent])assert(f.body.split('\n').some(l=>l===ignored));
 return {host_bound_failures:rows,source_triggers:functions,doctor_json_ignored_writer:{statement:ignored,original_pre:doctorPre,original_post:doctorPost,current:doctorCurrent,original_status_when_available:0,current_status_when_available:0,pre_existing:true,preserved:true,new_test_or_waiver_added:false},assertion_edits:[]};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){write('lint-attribution-frozen.json',lint());write('host-assertions-frozen.json',assertions());console.log('function-scoped evidence written; absent exact alignment remains explicit and no owner disposition or exception is inferred');}
