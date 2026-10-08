import assert from 'node:assert/strict';
import {readdirSync,writeFileSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {root,out,read,sha,git,sources} from './capture.mjs';
import {finalProof} from './final-proof.mjs';
import {buildLint} from './lint-proof.mjs';
import {coverageProof} from './coverage-proof.mjs';
const source=finalProof(),lint=buildLint(),coverage=coverageProof();
assert.deepEqual(source,JSON.parse(read(out+'/accepted-source-proof.json')));assert.deepEqual(lint,JSON.parse(read(out+'/accepted-lint-proof.json')));
assert.equal(git(['rev-parse','HEAD']).toString().trim(),source.base_commit,'worker made no commits');
for(const e of source.admitted_test_edits)assert.equal(sha(git(['show',source.base_commit+':'+e.path])),e.before_sha256,'actual committed pre-edit test bytes');
const receipts=readdirSync(out).filter(p=>p.endsWith('.json')).flatMap(path=>{const r=JSON.parse(read(out+'/'+path));return r.argv?[{path,sha256:sha(read(out+'/'+path)),exit:r.exit,signal:r.signal,error:r.error,argv:r.argv}]:[];});
const evidence={task_id:'fn-109-gomad-deepen-modules-and-tool-interfaces.3',status:'in_progress',base_commit:source.base_commit,commits:[],prs:[],review_verdict:null,native:false,native_qualification:false,tier:'session (jev-unavailable(no_key))',baseline:'pre-edit broad stock-Go Runner gate red/refused/timeout on unsupported linux/arm64; portable Controller green; no BASELINE_HANDOFF asserted',source_identity_sha256:source.source_identity_sha256,admitted_test_edits:source.admitted_test_edits,tests:receipts.map(r=>r.argv.join(' ')),receipts,portable_unique_top_level_tests:coverage.unique_portable_top_level_tests,source_proof:'accepted-source-proof.json',coverage_proof:'accepted-coverage-proof.json',lint_proof:'accepted-lint-proof.json',fast_lint_findings:56,unfiltered_relevant_findings:17,global_lint_green:false,mutation_controls:'Controller only: two intended statistic mismatches RED, canonical GREEN',restored_runner_assertions:'three historical vectors restored additively; native preparation guard prevents executing their assertions here',native_owners:['fn-149','fn-128'],lane:'released at terminal return; no attributable process remains',worker_history_unchanged:true};
writeFileSync(out+'/evidence.json',JSON.stringify(evidence,null,2)+'\n',{flag:'wx'});
const files=readdirSync(out).filter(p=>p!=='freeze-manifest.json').sort().map(path=>({path,sha256:sha(read(out+'/'+path)),bytes:read(out+'/'+path).length}));
for(const f of files){const r=spawnSync('git',['diff','--no-index','--check','/dev/null',out+'/'+f.path],{cwd:root});assert.equal(r.stdout.length+r.stderr.length,0,'frozen artifact whitespace '+f.path);assert([0,1].includes(r.status));}
const external_helpers=['.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-2/source-acceptance-20261008/reconstruct.mjs','.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-2/source-acceptance-20261008/owner-chain.mjs'].map(path=>({path,sha256:sha(read(path))}));
writeFileSync(out+'/freeze-manifest.json',JSON.stringify({frozen_at:new Date().toISOString(),base_commit:source.base_commit,current_sources:sources(),files,external_helpers,all_artifacts_staged_style_whitespace:'no findings; per-file /dev/null no-index check, no staging or raw stripping'},null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({frozen_files:files.length,evidence_sha256:sha(read(out+'/evidence.json')),source_identity_sha256:source.source_identity_sha256,freeze_sha256:sha(read(out+'/freeze-manifest.json'))}));
