import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {root, out, sha, git, sources} from './capture.mjs';

const read = path => readFileSync(resolve(root, path));
const proof = JSON.parse(readFileSync(out + '/input-proof.json'));
git(['merge-base', '--is-ancestor', proof.head, 'HEAD']);
assert.deepEqual(sources(), proof.sources);
assert.equal(sha(JSON.stringify(proof.sources)), proof.sources_sha256);
for (const entry of [...proof.review_sources, ...proof.historical_receipts, ...proof.historical_integrated_receipts, proof.standards_config, ...proof.lint_route]) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
for (const entry of proof.standards_tools) assert.equal(sha(readFileSync(entry.path)), entry.sha256, entry.path);
for (const entry of proof.user_files) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
const runtime = proof.runtime_reuse;
const bindings = JSON.parse(read(runtime.binding.path));
assert.equal(sha(read(runtime.binding.path)), runtime.binding.sha256);
for (const [path, expected] of Object.entries(bindings.scoped_bindings)) assert.equal(sha(read(path)), expected, path);
assert.equal(Object.keys(bindings.scoped_bindings).length, 87);
assert.equal(sha(read(runtime.task5_binding.path)), runtime.task5_binding.sha256);
const task5 = JSON.parse(read(runtime.task5_binding.path));
assert.equal(task5.exact_bindings.length, 87);
for (const entry of task5.exact_bindings) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
const manifest = JSON.parse(read(runtime.raw_manifest.path));
assert.equal(sha(read(runtime.raw_manifest.path)), runtime.raw_manifest.sha256);
const runtimeRoot = runtime.raw_manifest.path.replace(/\/raw-manifest\.json$/, '');
let bytes = 0;
for (const entry of manifest) {
  const stored = read(runtimeRoot + '/raw/' + (entry.retained_name ?? entry.name));
  if (entry.retained_name) assert.equal(sha(stored), entry.retained_sha256);
  const decoded = entry.encoding === 'utf8-json-string' ? Buffer.from(JSON.parse(stored).value, 'utf8') : stored;
  assert.equal(decoded.length, entry.bytes);
  assert.equal(sha(decoded), entry.sha256);
  bytes += decoded.length;
}
assert.equal(manifest.length, runtime.raw_files);
assert.equal(bytes, runtime.decoded_bytes);
assert.equal(sha(read(proof.archive.path)), proof.archive.sha256);
assert.equal(read(proof.archive.path).length, 35109201);
const labels = ['baseline-artifact', 'baseline-validate', 'portable-source-controls', 'runner-portable-retention', 'runner-host-bound-regression', 'cli-host-bound-regression', 'configured-fast-lint', 'task-source-lint', 'new-campaign-regression', 'collapse-sensitivity', 'collapse-default-sensitivity', 'final-portable-source-controls', 'configured-vet', 'final-campaign-lint', 'final-generated-validation', 'source-proof', 'standards-proof', 'final-owned-format', 'final-diff-check', 'worker-evidence-verification'];
for (const label of labels) {
  const receipt = JSON.parse(readFileSync(out + '/' + label + '.json'));
  assert.equal(sha(readFileSync(out + '/' + label + '.stdout')), receipt.stdout_sha256, label);
  assert.equal(sha(readFileSync(out + '/' + label + '.stderr')), receipt.stderr_sha256, label);
  if (receipt.source_changes) assert.deepEqual(receipt.source_changes, [], label);
}
const final = JSON.parse(readFileSync(out + '/final-portable-source-controls.json'));
assert.equal(final.exit, 0);
assert.deepEqual(final.test_counts, {pass: 155, fail: 0, skip: 0});
assert.equal(final.source_after_sha256, proof.sources_sha256);
const events = readFileSync(out + '/final-portable-source-controls.stdout', 'utf8').trim().split('\n').map(JSON.parse);
const newCases = events.filter(e => e.Test?.startsWith('TestOpenCampaignKeepsSameSignatureSuccessesDistinct') && e.Action === 'pass');
assert.equal(newCases.length, 9);
for (const name of ['signature_with_record_fallback', 'execution']) {
  for (const control of ['seed', 'ordinal', 'reference']) assert(newCases.some(e => e.Test === 'TestOpenCampaignKeepsSameSignatureSuccessesDistinct/' + name + '/' + control));
}
for (const label of ['collapse-sensitivity', 'collapse-default-sensitivity']) {
  const receipt = JSON.parse(readFileSync(out + '/' + label + '.json'));
  assert.equal(receipt.exit, 1);
  assert.equal(receipt.test_counts.fail, 1);
  assert(readFileSync(out + '/' + label + '.stdout', 'utf8').includes('validate published success artifact 2: retained success artifact does not match its campaign execution'));
}
assert.equal(JSON.parse(readFileSync(out + '/runner-host-bound-regression.json')).exit, 1);
assert(readFileSync(out + '/runner-host-bound-regression.stdout', 'utf8').includes('host is linux/arm64'));
const cli = JSON.parse(readFileSync(out + '/cli-host-bound-regression.json'));
assert.equal(cli.exit, 1);
assert.deepEqual(cli.test_counts, {pass: 0, fail: 0, skip: 0});
const standards = JSON.parse(readFileSync(out + '/standards-attribution.json'));
assert.equal(standards.observations['configured-fast-lint'].issues.length, 68);
assert.equal(standards.observations['task-source-lint'].issues.length, 12);
assert.equal(standards.original_task_findings, 0);
assert.equal(standards.worker_introduced_findings, 0);
assert.equal(standards.aggregate_green, false);
assert.equal(JSON.parse(readFileSync(out + '/final-campaign-lint.json')).exit, 0);
assert.equal(JSON.parse(readFileSync(out + '/configured-vet.json')).exit, 0);
assert.equal(JSON.parse(readFileSync(out + '/final-generated-validation.json')).exit, 0);
console.log(JSON.stringify({source_count: proof.source_count, review_sources: proof.review_sources.length, exact_runtime_inputs: 87, runtime_raw_files: manifest.length, runtime_raw_bytes: bytes, portable_tests: 155, portable_fail: 0, portable_skip: 0, new_campaign_controls: 9, mutant_failures: 2, configured_fast_lint_exit: 2, configured_fast_lint_findings: 68, final_changed_package_lint_exit: 0, aggregate_lint_green: false, native_qualification: false, user_files_preserved: 2, failed_binding_metadata_lost: true}));
