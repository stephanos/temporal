import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const artifact = dirname(fileURLToPath(import.meta.url));
const baselinePath = resolve(artifact, '../task-43/root-integrated.log');
const candidatePath = resolve(artifact, 'root-integrated.json');
const baseline = readFileSync(baselinePath, 'utf8');
const capture = JSON.parse(readFileSync(candidatePath, 'utf8'));
assert.equal(capture.exit_code, 2);
assert.deepEqual(capture.before, capture.after);
const candidate = capture.stdout;
const hash = text => createHash('sha256').update(text).digest('hex');
function blocks(raw) {
  const lines = raw.split('\n');
  const starts = lines.flatMap((line, i) => /^\S[^:]*\.go:\d+:\d+: .+ \([^)]+\)$/.test(line) ? [i] : []);
  return starts.map((start, i) => {
    const end = starts[i + 1] ?? lines.findIndex((line, j) => j > start && /^\d+ issues:/.test(line));
    assert.ok(end > start);
    return { header: lines[start], raw: lines.slice(start, end).join('\n') + '\n' };
  });
}
const old = blocks(baseline);
const current = blocks(candidate);
assert.equal(old.length, 323);
assert.equal(current.length, 319);
const removed = old.filter(block => !current.some(entry => entry.header === block.header));
const expected = [
  'tools/gomad3/internal/compatibilitypack/mutation_test.go:98:23: S1016:',
  'tools/gomad3/internal/compatibilitypack/mutation_test.go:102:28: S1016:',
  'tools/gomad3/internal/compatibilitypack/schema.go:292:29: S1016:',
  'tools/gomad3/internal/compatibilitypack/schema_timezone_test.go:5:1: File is not properly formatted (gci)',
];
assert.equal(removed.length, expected.length);
for (const prefix of expected) assert.ok(removed.some(block => block.header.startsWith(prefix)), prefix);
for (const block of current) assert.equal(block.raw, old.find(entry => entry.header === block.header)?.raw, block.header);
assert.deepEqual(current.map(block => block.header).sort(), old.filter(block => !removed.includes(block)).map(block => block.header).sort());
const counts = {};
for (const block of current) {
  const analyzer = block.header.match(/\(([^)]+)\)$/)[1];
  counts[analyzer] = (counts[analyzer] ?? 0) + 1;
}
assert.deepEqual(counts, { errcheck: 252, exhaustive: 3, forbidigo: 11, staticcheck: 53 });
process.stdout.write(JSON.stringify({ baseline: { path: baselinePath, sha256: hash(baseline), count: old.length }, candidate: { path: candidatePath, stdout_sha256: hash(candidate), count: current.length }, counts, removed: removed.map(block => block.header), added: [], exact_unchanged_blocks: current.length, before_after_capture_inputs_identical: true }, null, 2) + '\n');
