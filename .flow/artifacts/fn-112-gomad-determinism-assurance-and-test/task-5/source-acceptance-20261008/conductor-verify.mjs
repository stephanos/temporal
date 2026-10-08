import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';

const dir = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008';
const retained = '.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/source-acceptance-20261007';
const sha = value => crypto.createHash('sha256').update(value).digest('hex');
const read = path => fs.readFileSync(path);
const json = path => JSON.parse(read(path));
const binding = json(dir + '/source-binding.json');
execFileSync('git', ['merge-base', '--is-ancestor', binding.head, 'HEAD']);
for (const entry of binding.exact_bindings) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
assert.equal(binding.exact_bindings.length, 87);
for (const entry of binding.input_closure) {
  assert.equal(sha(read(entry.path)), entry.current_sha256, entry.path);
  const old = execFileSync('git', ['show', binding.retained_ref + ':' + entry.path], {maxBuffer: 32 << 20});
  assert.equal(sha(old), entry.retained_sha256, entry.path);
  assert.equal(entry.equal, old.equals(read(entry.path)), entry.path);
}
assert.equal(binding.input_closure.filter(entry => !entry.equal).length, 5);
assert.equal(sha(read(binding.archive.path)), binding.archive.sha256);
assert.equal(read(binding.archive.path).length, 35109201);
const manifest = json(retained + '/raw-manifest.json');
let bytes = 0;
for (const entry of manifest) {
  const stored = read(retained + '/raw/' + (entry.retained_name ?? entry.name));
  if (entry.retained_name) assert.equal(sha(stored), entry.retained_sha256);
  const decoded = entry.encoding === 'utf8-json-string' ? Buffer.from(JSON.parse(stored).value, 'utf8') : stored;
  assert.equal(decoded.length, entry.bytes);
  assert.equal(sha(decoded), entry.sha256);
  bytes += decoded.length;
}
assert.equal(manifest.length, 72);
assert.equal(bytes, 521317);
for (const entry of binding.user_files) assert.equal(sha(read(entry.path)), entry.sha256);
for (const label of ['baseline-validate', 'portable-source-controls', 'task-source-lint', 'configured-vet', 'task-owned-format', 'generated-source-validation', 'source-binding-final']) {
  const observation = json(dir + '/' + label + '.json');
  assert.equal(sha(read(dir + '/' + label + '.stdout')), observation.stdout_sha256);
  assert.equal(sha(read(dir + '/' + label + '.stderr')), observation.stderr_sha256);
  assert.deepEqual(observation.source_changes, []);
}
const events = read(dir + '/portable-source-controls.stdout').toString().trim().split('\n').map(JSON.parse);
const verdicts = events.filter(event => event.Test && !event.Test.includes('/') && ['pass', 'fail', 'skip'].includes(event.Action));
assert.equal(verdicts.length, 21);
assert(verdicts.every(event => event.Action === 'pass'));
assert.equal(json(dir + '/task-source-lint.json').exit, 2);
const lint = read(dir + '/task-source-lint.stdout').toString() + read(dir + '/task-source-lint.stderr').toString();
assert(lint.includes('descriptor_dup_linux_test.go:15') && lint.includes('descriptor_dup_linux_test.go:16'));
assert.equal(read(dir + '/task-owned-format.stdout').toString().trim(), 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go');
console.log(JSON.stringify({head: binding.head, exact_inputs: 87, source_closure: binding.input_closure.length, changed_docs_excluded: 5, raw_files: 72, raw_bytes: bytes, portable_pass: 21, fail: 0, skip: 0, unrelated_descriptor_lint: 2, inherited_overlay_format_observation: true, user_files_preserved: 2, native_qualification: false}));
