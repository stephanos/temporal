import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {dirname} from 'node:path';
import {fileURLToPath} from 'node:url';

const out = dirname(fileURLToPath(import.meta.url));
const task5 = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008';
const implementation = 'bfb2bdb8ef136d3eb38cbd539735661d5d7c9af5';
const baseline = fs.readFileSync(out + '/base-commit', 'utf8').trim();
const sha = data => crypto.createHash('sha256').update(data).digest('hex');
const read = path => fs.readFileSync(path);
const json = path => JSON.parse(read(path));
const git = args => execFileSync('git', args, {maxBuffer: 32 << 20});
const receipt = path => ({path, bytes: read(path).length, sha256: sha(read(path))});
git(['merge-base', '--is-ancestor', baseline, 'HEAD']);
git(['merge-base', '--is-ancestor', implementation, 'HEAD']);
const originalBase = git(['rev-parse', implementation + '^']).toString().trim();
const inventory = 'tools/gomad3/toolchain/clock_inventory_test.go';
assert.deepEqual(read(inventory), git(['show', implementation + ':' + inventory]));
assert.deepEqual(git(['show', '--pretty=format:', '--name-only', implementation]).toString().trim().split('\n').sort(), ['tools/gomad3/README.md', inventory].sort());
const binding = json(task5 + '/source-binding.json');
git(['merge-base', '--is-ancestor', binding.head, 'HEAD']);
for (const entry of binding.exact_bindings) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
assert.equal(binding.exact_bindings.length, 87);
assert(binding.exact_bindings.some(entry => entry.path === inventory));
for (const entry of binding.input_closure) assert.equal(sha(read(entry.path)), entry.current_sha256, entry.path);
assert.equal(binding.input_closure.length, 1223);
assert.equal(binding.input_closure.filter(entry => entry.equal).length, 1218);
assert.equal(sha(read(binding.archive.path)), binding.archive.sha256);
assert.equal(read(binding.archive.path).length, 35109201);
assert.deepEqual(binding.allowlists, [20, 79]);
const positive = binding.positive_inventory;
assert.equal(sha(read(positive.receipt)), positive.receipt_sha256);
const positiveReceipt = json(positive.receipt);
assert.equal(positiveReceipt.exit, 0);
assert.equal(sha(read(positive.receipt.replace(/\.json$/, '.stdout'))), positive.log_sha256);
assert.equal(positive.clock_rows, 48);
assert.deepEqual(positive.platforms, ['darwin/arm64', 'linux/amd64']);
for (const entry of binding.user_files) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
const current = read('tools/gomad3/README.md').toString();
const section = text => {
  const start = text.indexOf('Some reporting surfaces intentionally remain on host time.');
  const end = text.indexOf('For a fixed toolchain', start);
  assert(start > 0 && end > start);
  return text.slice(start, end);
};
const contract = section(current);
assert.equal(contract, section(git(['show', implementation + ':tools/gomad3/README.md']).toString()));
assert.equal(Buffer.byteLength(contract), 2819);
const lines = current.slice(0, current.indexOf(contract)).split('\n').length;
const rawPolicy = [
  ['original-summary.utf8.json', 2364, '0aa4a6b4f16e6c50a1548f75fb0a044bdbcee55c91d8aa53c43b4dbb7614fccb'],
  ['original-evidence.utf8.json', 1296, 'a00334f212540bd32c30e2767bcaf31b574f4c746bc4662940985c37c0c2b60b'],
].map(([name, bytes, sha256]) => {
  const path = out + '/' + name;
  const container = json(path);
  assert.equal(container.encoding, 'utf8-json-string');
  const decoded = Buffer.from(container.value, 'utf8');
  assert.equal(decoded.length, bytes);
  assert.equal(sha(decoded), sha256);
  return {...receipt(path), encoding: container.encoding, original_path: container.original_path, decoded_bytes: bytes, decoded_sha256: sha256};
});
const summary = json(out + '/original-summary.utf8.json').value;
const policy = summary.slice(summary.indexOf('Patch-policy decision:'), summary.indexOf('\n\nFiles:'));
assert(policy.includes('declined the proposed `runtime/proc.go` overwrite'));
assert(policy.includes('scheduler-owned code would mutate collector-owned state'));
assert(policy.includes('leaving the underlying host read'));
fs.writeFileSync(out + '/policy-decision.md', '# Retained D27 patch-policy decision\n\n' + policy + '\n\n[Original worker summary](original-summary.utf8.json) and [evidence](original-evidence.utf8.json) retain the original bytes as lossless UTF-8 JSON strings. Decode each `value` as UTF-8; source-binding.json verifies the original byte counts and SHA256 values. This records the existing decision; it is neither new approval nor a fresh review receipt.\n');
const output = {
  base_commit: baseline, original_source_base: originalBase, implementation,
  product_edits: [], inventory: receipt(inventory), inventory_equal_implementation: true,
  contract: {path: 'tools/gomad3/README.md', full_sha256: sha(read('tools/gomad3/README.md')), start_line: lines, end_line: lines + contract.split('\n').length - 2, bytes: Buffer.byteLength(contract), sha256: sha(contract), equal_implementation_section: true},
  retained_binding: receipt(task5 + '/source-binding.json'), exact_inputs_checked: 87,
  closure_reference: {path: task5 + '/source-binding.json', paths_checked: 1223, unchanged_retained_inputs: 1218, changed_docs_excluded_from_reuse: 5},
  archive: binding.archive, allowlists: binding.allowlists,
  positive_inventory: positive,
  policy_provenance: rawPolicy,
  user_files: binding.user_files,
  native_qualification: {current_candidate_verified: false, darwin: 'deferred fn-149.1', linux: 'deferred fn-128.1/.4/.7', actual_host: 'linux/arm64 unsupported'},
  historical_review: 'Task text claims historical SHIP; original worker substrate records review skipped. Neither supplies a fresh source-review receipt.'
};
fs.writeFileSync(out + '/source-binding.json', JSON.stringify(output, null, 2) + '\n');
console.log(JSON.stringify({base: baseline, original_source_base: originalBase, exact_inputs_checked: 87, clock_rows: 48, platforms: positive.platforms, contract_bytes: 2819, inventory_equal_implementation: true, native_qualification: false}));
