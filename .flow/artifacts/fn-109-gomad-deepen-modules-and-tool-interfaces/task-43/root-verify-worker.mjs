import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = `${root}/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43`;
const read = name => JSON.parse(fs.readFileSync(`${artifacts}/${name}.json`, 'utf8'));
const digest = path => crypto.createHash('sha256').update(fs.readFileSync(path)).digest('hex');
const baseline = read('baseline'), final = read('final'), proof = read('preservation-proof');
assert.equal(Object.keys(baseline.files).length, 1219);
assert.deepEqual(Object.keys(baseline.files).sort(), Object.keys(final.files).sort());
for (const [path, expected] of Object.entries(final.files)) assert.equal(digest(`${root}/${path}`), expected, path);
assert.deepEqual(baseline.binaries, final.binaries);
for (const [path, expected] of Object.entries(final.binaries)) assert.equal(digest(path), expected, path);
for (const [path, expected] of Object.entries(proof.candidate_hashes)) assert.equal(digest(`${root}/${path}`), expected, path);
const changes = Object.keys(baseline.files).filter(path => baseline.files[path] !== final.files[path]);
assert.deepEqual(changes.sort(), proof.changed.slice().sort());
assert.equal(changes.length, 5);
assert.equal(Object.keys(baseline.files).length - changes.length, 1214);
assert.equal(proof.generated_or_pin_count, 51);
for (const path of proof.generated_or_pin_paths) assert.equal(baseline.files[path], final.files[path], path);
for (const [name, result] of Object.entries(proof.checks)) assert.equal(result, true, name);
const expectedExits = {
  'baseline-packages': 0, 'baseline-lint': 1, 'red-final': 1,
  'final-validate': 0, 'final-packages': 0, 'target-consumers': 0,
  'architecture-purity': 0, 'final-focused': 0, 'formatting': 0,
  'errortype': 0, 'final-lint': 1, 'source-diff-check': 0, 'preservation-final': 0,
};
const captures = [];
for (const [name, exit] of Object.entries(expectedExits)) {
  const capture = read(name);
  assert.equal(capture.exit_code, exit, name);
  assert.ok(Date.parse(capture.finished_at) >= Date.parse(capture.started_at), name);
  assert.ok(capture.elapsed_seconds >= 0, name);
  const events = capture.stdout.split('\n').filter(line => line.startsWith('{')).map(line => {
    try { return JSON.parse(line); } catch { return {}; }
  }).filter(event => typeof event.Test === 'string');
  const counts = { pass: 0, fail: 0, skip: 0 };
  for (const event of events) if (event.Action in counts) counts[event.Action]++;
  assert.deepEqual(counts, capture.test_counts, name);
  const skips = events.filter(event => event.Action === 'skip').map(event => ({ package: event.Package, test: event.Test }));
  assert.deepEqual(skips, capture.skips, name);
  captures.push({ name, exit, counts, sha256: digest(`${artifacts}/${name}.json`) });
}
assert.equal(read('formatting').stdout, '');
assert.match(read('baseline-lint').stdout, /7 issues:/);
assert.match(read('final-lint').stdout, /7 issues:/);
assert.equal(read('red-final').test_counts.fail, 25);
assert.equal(read('final-focused').test_counts.pass, 63);
const sourceLines = read('red-sources').stdout.trim().split('\n');
for (const line of sourceLines) {
  const [expected, path] = line.split(/\s+/);
  if (!path.endsWith('/schema.go')) assert.equal(digest(`${root}/${path}`), expected, path);
}
process.stdout.write(JSON.stringify({ verified: true, tracked_scoped_files: 1219, protected_files: 1214, generated_or_pin_files: 51, candidate_files: 6, captures }, null, 2) + '\n');
