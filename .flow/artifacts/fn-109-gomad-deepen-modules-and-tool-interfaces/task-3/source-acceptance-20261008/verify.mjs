import assert from 'node:assert/strict';
import {readdirSync} from 'node:fs';
import {read,out,sha,git,sources} from './capture.mjs';
import {finalProof} from './final-proof.mjs';
import {buildLint} from './lint-proof.mjs';
import {coverageProof} from './coverage-proof.mjs';
import {completionDelta} from './completion-additive.mjs';
import {functionSpans} from './proof.mjs';
const frozen=JSON.parse(read(out+'/freeze-manifest.json')),evidence=JSON.parse(read(out+'/evidence.json'));
assert.equal(evidence.status,'in_progress');assert.equal(evidence.review_verdict,null);assert.equal(evidence.native,false);assert.deepEqual(evidence.commits,[]);assert.deepEqual(evidence.prs,[]);
for(const f of frozen.files)assert.equal(sha(read(out+'/'+f.path)),f.sha256,'frozen '+f.path);
assert.deepEqual(readdirSync(out).filter(p=>p!=='freeze-manifest.json').sort(),frozen.files.map(f=>f.path).sort(),'no unmanifested capture added after freeze');
for(const f of frozen.external_helpers)assert.equal(sha(read(f.path)),f.sha256,'imported frozen helper '+f.path);
assert.deepEqual(sources(),frozen.current_sources,'whole current source including exact IP board; no normalization');
assert.equal(git(['show','-s','--format=%H',evidence.base_commit]).toString().trim(),evidence.base_commit);
const current=git(['rev-parse','HEAD']).toString().trim();
const commits=git(['rev-list','--reverse',evidence.base_commit+'..HEAD']).toString().trim().split('\n').filter(Boolean);
assert.equal(git(['merge-base',evidence.base_commit,'HEAD']).toString().trim(),evidence.base_commit,'base remains ancestor');
for(const c of commits){
 const paths=git(['diff-tree','--no-commit-id','--name-only','-r','--no-renames',c]).toString().trim().split('\n').filter(Boolean);
 for(const p of paths){if(p.startsWith('.flow/'))continue;if(p==='MILESTONES.md'){assert.equal(sha(git(['show',c+':'+p])),frozen.current_sources[p],'every intermediate committed board must equal captured IP bytes');continue;}const edit=evidence.admitted_test_edits.find(e=>e.path===p);assert(edit,'unadmitted ancestor path '+p);assert([edit.before_sha256,edit.after_sha256].includes(sha(git(['show',c+':'+p]))),'only exact pre/post test bytes in intermediate snapshot');}
}
assert.deepEqual(finalProof(),JSON.parse(read(out+'/accepted-source-proof.json')),'rederive originals, whole loop, owner chain, baseline670, raw72/runtime87/static/source bindings');
assert.deepEqual(buildLint(),JSON.parse(read(out+'/accepted-lint-proof.json')),'rederive exact function-scoped pre/post contexts/hunks and actual current blame');
const coverage=coverageProof(),saved=JSON.parse(read(out+'/accepted-coverage-proof.json'));
for(const p of [coverage,saved])for(const f of p.current_functions)delete f.actual_owner;
assert.deepEqual(coverage,saved,'rederive actual counts and current characterization function bytes');completionDelta();
const a=JSON.parse(read(out+'/additive-after.json')),before=read(out+'/runner-test-before.txt').toString(),after=read(a.path).toString(),old=functionSpans(before).find(f=>f.signature.startsWith('func '+a.name+'('));
assert.equal(after,before.replace(old.body,a.after_function.body));assert.equal(sha(after),a.after_sha256);
for(const receipt of evidence.receipts){const r=JSON.parse(read(out+'/'+receipt.path));assert.equal(sha(read(out+'/'+receipt.path)),receipt.sha256);assert.equal(sha(read(out+'/'+receipt.path.replace(/\.json$/,'.stdout'))),r.stdout_sha256);assert.equal(sha(read(out+'/'+receipt.path.replace(/\.json$/,'.stderr'))),r.stderr_sha256);assert.deepEqual(r.source_changes,[]);assert.equal(r.sources_before_sha256,r.sources_after_sha256);assert.equal(r.environment.GOENV,'off');assert.equal(r.environment.GOWORK,'off');assert.equal(r.environment.GOTOOLCHAIN,'local');assert.equal(r.environment.GOFLAGS,'');assert.equal(r.environment.GOEXPERIMENT,'nogreenteagc');}
for(const label of ['controller-missing-attempted','controller-wrong-success']){const r=JSON.parse(read(out+'/'+label+'.json')),raw=read(out+'/'+label+'.stdout').toString();assert.equal(r.exit,1);assert(raw.includes('controller_completion_test.go:132: statistics ='));assert(raw.includes('unclassified_attempt'));}
assert.equal(JSON.parse(read(out+'/canonical-controller-after-mutations.json')).exit,0);
assert.equal(JSON.parse(read(out+'/final-native-guarded-assertions.json')).exit,1);
assert.equal(JSON.parse(read(out+'/portable-runner-boundaries.json')).error.includes('ETIMEDOUT'),true);
console.log(JSON.stringify({verification:'retained source/evidence consistency only, not a review verdict or native qualification',source_identity_sha256:sha(JSON.stringify(sources())),evidence_sha256:sha(read(out+'/evidence.json')),base_commit:evidence.base_commit,current_head:current,permitted_conductor_commits:commits,portable_unique:coverage.unique_portable_top_level_tests,native:false,review_verdict:null,write_operations:0,go_invocations:0}));
