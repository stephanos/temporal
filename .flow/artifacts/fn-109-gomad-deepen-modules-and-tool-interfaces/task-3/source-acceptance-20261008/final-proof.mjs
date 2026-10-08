import assert from 'node:assert/strict';
import {writeFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {build} from './proof.mjs';
import {out,read,sha} from './capture.mjs';
import {completionDelta} from './completion-additive.mjs';
export function finalProof(){
 const p=build(),a=JSON.parse(read(out+'/additive-after.json'));
 assert.equal(sha(read(a.path)),a.after_sha256);
 p.early_characterization.missing_exact_success_trigger_whole_statistics_assertion=false;
 p.early_characterization.additive_restoration=a;
 p.early_characterization.current_native_guard='deterministicio/profile.go:283 refuses linux/arm64 before executor and before the error/statistics assertions';
 p.static_bindings.unchanged_closure=1220;
 p.static_bindings.changed_unreused=p.static_bindings.changed_unreused.map(e=>({...e,owner:e.path.endsWith('/retained_evidence_test.go')?'DONE fn112.16':'admitted task3 additive whole-statistics restoration'}));
 const c=completionDelta();
 p.early_characterization.two_row_restoration=c;
 p.admitted_test_edits=[a,c].map(e=>({path:e.path,before_sha256:e.before_sha256,after_sha256:e.after_sha256,scope:e.name}));
 return p;
}
if(process.argv[1]===fileURLToPath(import.meta.url)){const p=finalProof();writeFileSync(out+'/'+(process.argv[2]??'final-source-proof.json'),JSON.stringify(p,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify({postimages:5,call_sites:p.runner_chain.callsites.length,unchanged_closure:1220,additive_tests:2}));}
