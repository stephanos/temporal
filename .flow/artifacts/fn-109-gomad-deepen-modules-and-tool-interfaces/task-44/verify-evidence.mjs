import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const artifact = dirname(fileURLToPath(import.meta.url));
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const json = name => JSON.parse(readFileSync(resolve(artifact, name + '.json')));
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
const before = json('before'), final = json('final'), proof = json('preservation-proof');
assert.deepEqual(Object.keys(before.files).sort(), Object.keys(final.files).sort());
assert.deepEqual(before.binaries, final.binaries);
for (const [path, digest] of Object.entries(final.files)) assert.equal(hash(resolve(root, path)), digest, path);
for (const [path, digest] of Object.entries(final.binaries)) assert.equal(hash(path), digest, path);
assert.ok(Object.values(proof.checks).every(value => value === true));
assert.deepEqual(Object.keys(before.files).filter(path => before.files[path] !== final.files[path]).sort(), proof.changed.slice().sort());
assert.equal(proof.changed.length, 3);
assert.equal(proof.generated_or_pin_count, 51);
const labels = ['baseline-packages', 'baseline-target-consumers', 'baseline-target-controls', 'final-validate', 'final-packages', 'final-target-controls', 'architecture-purity', 'formatting', 'errortype', 'final-lint'];
const observations = [];
for (const label of labels) {
  const capture = json(label);
  assert.deepEqual(capture.before, capture.after, label);
  assert.ok(new Date(capture.finished_at) >= new Date(capture.started_at), label);
  assert.ok(capture.elapsed_seconds >= 0, label);
  const events = capture.stdout.split('\n').flatMap(line => { try { const event = JSON.parse(line); return event?.Action ? [event] : []; } catch { return []; } });
  const counts = Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, events.filter(event => event.Action === action && event.Test).length]));
  assert.deepEqual(capture.test_counts, counts, label);
  assert.equal(capture.exit_code, label === 'final-lint' || label === 'baseline-target-consumers' ? 1 : 0, label);
  observations.push({ label, exit_code: capture.exit_code, test_counts: counts, elapsed_seconds: capture.elapsed_seconds, skips: capture.skips });
}
assert.equal(json('formatting').stdout, '');
assert.deepEqual(json('baseline-packages').test_counts, json('final-packages').test_counts);
assert.deepEqual(json('baseline-target-controls').test_counts, json('final-target-controls').test_counts);
process.stdout.write(JSON.stringify({ checked_at: new Date().toISOString(), selected_files: Object.keys(final.files).length, protected_files: Object.keys(final.files).length - 3, selected_binaries: Object.keys(final.binaries).length, candidate_hashes: proof.candidate_hashes, generated_or_pin_count: 51, current_selected_inputs_match_final: true, observations }, null, 2) + '\n');
