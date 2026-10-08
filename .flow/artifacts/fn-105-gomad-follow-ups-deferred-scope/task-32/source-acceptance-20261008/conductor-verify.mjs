import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';

const dir = '.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-32/source-acceptance-20261008';
const sha = value => crypto.createHash('sha256').update(value).digest('hex');
const read = path => fs.readFileSync(path);
const json = path => JSON.parse(read(path));
const binding = json(dir + '/source-binding.json');
execFileSync('git', ['merge-base', '--is-ancestor', binding.base_commit, 'HEAD']);
const oldInventory = execFileSync('git', ['show', binding.implementation + ':' + binding.inventory.path]);
assert(oldInventory.equals(read(binding.inventory.path)));
assert.equal(sha(oldInventory), binding.inventory.sha256);
const section = text => {
  const start = text.indexOf('Some reporting surfaces intentionally remain on host time.');
  const end = text.indexOf('For a fixed toolchain', start);
  assert(start > 0 && end > start);
  return text.slice(start, end);
};
const current = read(binding.contract.path);
assert.equal(sha(current), binding.contract.full_sha256);
const contract = section(current.toString());
assert.equal(contract, section(execFileSync('git', ['show', binding.implementation + ':' + binding.contract.path]).toString()));
assert.equal(Buffer.byteLength(contract), 2819);
assert.equal(sha(contract), binding.contract.sha256);
for (const entry of binding.policy_provenance) {
  const stored = read(entry.path);
  assert.equal(sha(stored), entry.sha256);
  const decoded = Buffer.from(JSON.parse(stored).value, 'utf8');
  assert.equal(decoded.length, entry.decoded_bytes);
  assert.equal(sha(decoded), entry.decoded_sha256);
}
const policy = json(binding.policy_provenance[0].path).value;
assert(policy.includes('declined the proposed `runtime/proc.go` overwrite'));
assert(policy.includes('stage: impl-review - skipped'));
assert.equal(sha(read(binding.retained_binding.path)), binding.retained_binding.sha256);
execFileSync('node', ['.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/conductor-verify.mjs'], {maxBuffer: 32 << 20});
assert.equal(binding.positive_inventory.clock_rows, 48);
for (const entry of binding.user_files) assert.equal(sha(read(entry.path)), entry.sha256);
const portable = json(dir + '/baseline-portable.json');
assert.equal(portable.exit, 0);
assert.deepEqual(portable.test_counts, {pass: 6, fail: 0, skip: 0});
const evidence = json(dir + '/evidence.json');
for (const entry of [evidence.source_binding, evidence.standards_attribution, evidence.policy_decision, ...evidence.observations]) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
for (const entry of evidence.observations) {
  const command = json(entry.path);
  assert.equal(command.exit, entry.exit);
  assert.deepEqual(command.source_changes, []);
  assert.equal(sha(read(entry.path.replace(/\.json$/, '.stdout'))), command.stdout_sha256);
  assert.equal(sha(read(entry.path.replace(/\.json$/, '.stderr'))), command.stderr_sha256);
}
assert.equal(json(dir + '/task-source-lint.json').exit, 0);
console.log(JSON.stringify({original_clock_inventory_preserved: true, contract_bytes: 2819, exact_runtime_bindings: 87, retained_clock_rows: 48, durable_policy_receipts: 2, worker_portable_pass: 6, fail: 0, skip: 0, task_lint_exit: 0, native_qualification: false, user_files_preserved: 2}));
