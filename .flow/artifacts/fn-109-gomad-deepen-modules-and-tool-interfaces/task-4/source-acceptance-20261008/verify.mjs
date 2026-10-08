import assert from 'node:assert/strict';
import {relative} from 'node:path';
import {root,out,read,sha,git,sources,stock} from './capture.mjs';
import {build,base,ref} from './proof.mjs';
import {lint,assertions} from './attribution.mjs';
import {gaps} from './gaps.mjs';
import {walk,receipts,counts} from './finish.mjs';
import {admission,writers} from './admission.mjs';
const json=p=>JSON.parse(read(out+'/'+p)),freeze=json('freeze.json'),expectedFiles=freeze.files.map(e=>e.path).concat('freeze.json').sort();assert.deepEqual(walk(out).map(p=>relative(out,p)).sort(),expectedFiles,'Strict frozen directory changed; write new conductor artifacts to a sibling');
for(const e of freeze.files){assert.equal(read(out+'/'+e.path).length,e.bytes,e.path);assert.equal(sha(read(out+'/'+e.path)),e.sha256,e.path);}
for(const e of [...json('original-records.json'),...json('import-bindings.json')])assert.deepEqual(ref(e.path),e,e.path);
assert.deepEqual(sources(),json('sources-current.json'),'Actual current source or exact IP milestone board changed');
assert.deepEqual(build(),json('source-proof-final.json'),'Re-derived original patch images/current owner chains/source bindings differ');
assert.deepEqual(lint(),json('lint-attribution-frozen.json'),'Re-derived function-scoped finding attribution differs');
assert.deepEqual(assertions(),json('host-assertions-frozen.json'),'Original/current assertion and host trigger binding differs');
assert.deepEqual(admission(),json('identity-admission.json'));assert.deepEqual(writers(),json('writer-equivalence.json'));
const initialGap=json('source-gaps.json'),nowGap=gaps();for(let i=0;i<2;i++)assert.equal(sha(read(out+'/initial-contract-'+i+'.md')),initialGap.contracts[i].sha256);nowGap.contracts=initialGap.contracts;assert.deepEqual(nowGap,initialGap,'Source gap mechanism changed; only explicit immutable initial contract snapshots are used');
const rs=receipts(),toolPaths={go:stock+'/go',gofmt:stock+'/gofmt',golangci_lint:'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',errortype:'/tmp/fn109-lint-tools.ZdNe1t50/errortype'};for(const r of rs)for(const [n,p] of Object.entries(toolPaths))assert.equal(sha(read(p)),r.tool_sha256[n],r.path+' tool '+n);
assert.deepEqual(counts(rs),json('coverage.json'));const e=json('evidence.json');assert.equal(e.status,'in_progress');assert.equal(e.review_verdict,null);assert.equal(e.native,false);assert.deepEqual(e.commits,[]);assert.deepEqual(e.prs,[]);
git(['merge-base','--is-ancestor',base,'HEAD']);const board=sha(read(out+'/milestones-ip.md')),commits=git(['rev-list','--reverse',base+'..HEAD']).toString().trim().split('\n').filter(Boolean);
for(const c of commits){
 const changed=git(['diff-tree','--root','-m','--no-commit-id','--name-only','-r',c]).toString().trim().split('\n').filter(Boolean);for(const p of changed)assert(p.startsWith('.flow/')||p==='MILESTONES.md',c+' changes unadmitted source '+p);assert.equal(sha(git(['show',c+':MILESTONES.md'])),board,c+' intermediate board differs from captured fn3Done/fn4IP board');
 const parents=git(['show','-s','--format=%P',c]).toString().trim().split(' ').filter(Boolean);if(parents.length>1)for(const p of parents)if(p!==base)assert.equal(sha(git(['show',p+':MILESTONES.md'])),board,c+' merge-parent board mismatch '+p);
}
console.log(JSON.stringify({task:e.task,status:'in_progress',verdict:null,readonly:true,no_go:true,no_writes:true,files:freeze.files.length,original_preimages:21,original_postimages:10,first_baseline_inputs:670,runtime_inputs:87,retained_raw_files:72,retained_decoded_bytes:521317,unique_pass:e.coverage.unique_pass,unique_fail:e.coverage.unique_fail,duplicate_observations:e.coverage.duplicated_observations,native:false,source_gap_chronology_preserved:true,evidence_sha256:sha(read(out+'/evidence.json')),freeze_sha256:sha(read(out+'/freeze.json'))}));
