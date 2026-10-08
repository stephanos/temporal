import './conductor-verify.mjs';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {root, out, sha, git} from './capture.mjs';

const read = path => readFileSync(resolve(root, path));
const local = path => readFileSync(resolve(out, path));
const evidence = JSON.parse(local('evidence.json'));
const proof = JSON.parse(local('input-proof.json'));
assert.equal(sha(local(evidence.source_proof.path)), evidence.source_proof.sha256);
assert.equal(local(evidence.source_proof.path).length, evidence.source_proof.bytes);
assert.equal(evidence.source_ready, true);
assert.equal(evidence.task_status, 'in_progress');
for (const observation of evidence.observations) {
  for (const entry of [observation.receipt, observation.stdout, observation.stderr]) {
    assert.equal(local(entry.path).length, entry.bytes, entry.path);
    assert.equal(sha(local(entry.path)), entry.sha256, entry.path);
  }
  const receipt = JSON.parse(local(observation.receipt.path));
  assert.equal(receipt.exit, observation.exit, observation.receipt.path);
}
const bundle = JSON.parse(local('bundle-manifest.json'));
for (const entry of bundle.files) {
  assert(entry.path === 'conductor-verify.mjs' || !entry.path.startsWith('conductor-'), entry.path);
  assert.equal(local(entry.path).length, entry.bytes, entry.path);
  assert.equal(sha(local(entry.path)), entry.sha256, entry.path);
}
const testPath = 'tools/gomad3/runner/internal/campaign/retained_evidence_test.go';
const original = git(['show', proof.base_commit + ':' + testPath]).toString();
const current = read(testPath).toString();
const addedStart = current.indexOf('\nfunc TestOpenCampaignKeepsSameSignatureSuccessesDistinct(');
const addedEnd = current.indexOf('\nfunc TestResolveRetainedEvidenceAllowsSharedFailureIdentityWithinBatch(', addedStart);
assert(addedStart > 0 && addedEnd > addedStart);
const unaffected = (current.slice(0, addedStart) + current.slice(addedEnd))
  .replace('\t"fmt"\n', '').replace('\t"os"\n', '').replace('\t"strings"\n', '');
assert.equal(unaffected, original, 'all existing test bodies/comments remain exact');
const standards = JSON.parse(local('standards-attribution.json'));
for (const observation of Object.values(standards.observations)) {
  for (const issue of observation.issues) {
    const source = read(issue.path).toString();
    const code = source.split('\n')[issue.line - 1].trim();
    assert.equal(sha(source), issue.source_sha256, issue.path);
    assert.equal(sha(code), issue.code_sha256, issue.path);
    assert.equal(code, issue.code);
    assert.equal(issue.waiver, false);
    assert.equal(issue.original_native_task_surface, false);
    assert(!standards.original_native_task_paths.includes(issue.path));
    const blame = git(['blame', '-L', `${issue.line},${issue.line}`, '--porcelain', standards.worker_base, '--', issue.path]).toString();
    assert.equal(blame.split(' ')[0], issue.blame.commit);
  }
}
for (const label of ['conductor-portable', 'conductor-generated-validation']) {
  const receipt = JSON.parse(local(label + '.json'));
  assert.equal(receipt.exit, 0, label);
  assert.deepEqual(receipt.source_changes, []);
  assert.equal(receipt.source_before_sha256, proof.sources_sha256);
  assert.equal(receipt.source_after_sha256, proof.sources_sha256);
  assert.equal(sha(local(label + '.stdout')), receipt.stdout_sha256);
  assert.equal(sha(local(label + '.stderr')), receipt.stderr_sha256);
  if (label === 'conductor-portable') {
    assert.deepEqual(receipt.test_counts, {pass: 155, fail: 0, skip: 0});
    const events = local(label + '.stdout').toString().trim().split('\n').map(JSON.parse);
    assert.equal(events.filter(e => e.Action === 'fail' || e.Action === 'skip').length, 0);
    assert.equal(events.filter(e => !e.Test && e.Action === 'pass').length, 3);
  }
}
const lost = JSON.parse(local('failed-binding-observation.json'));
assert.equal(lost.exit, 1);
assert.equal(lost.classification, 'inconclusive input-proof attempt; never a pass');
assert(lost.metadata_integrity.includes('has not been reconstructed'));
console.log(JSON.stringify({independent_portable_pass: 155, fail: 0, skip: 0, generated_validation_exit: 0, original_test_bodies_preserved: true, source_bound_observations: evidence.observations.length, bundle_files: bundle.files.length, lint_attribution_observations: 80, task_lint_waiver: false, aggregate_lint_green: false, native_qualification: false, failed_binding_metadata_loss_disclosed: true}));
