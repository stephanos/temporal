import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';

const root = process.cwd();
const packet = path.dirname(new URL(import.meta.url).pathname);
const hash = value => crypto.createHash('sha256').update(value).digest('hex');
const fileHash = name => hash(fs.readFileSync(name));
const git = (...args) => execFileSync('git', args, {cwd: root});
const evidence = JSON.parse(fs.readFileSync(path.join(packet, 'evidence.json')));
const base = evidence.base_commit;
const receiptNames = fs.readdirSync(packet).filter(name => name.endsWith('.json'));
let receipts = 0;
for (const name of receiptNames) {
  const record = JSON.parse(fs.readFileSync(path.join(packet, name)));
  if (!record.command || record.exit_code === undefined) continue;
  assert.equal(record.terminal, true, name);
  assert.equal(record.source_before_sha256, record.source_after_sha256, name);
  for (const stream of ['stdout', 'stderr']) {
    assert.equal(fileHash(path.join(packet, name.replace(/\.json$/, '.' + stream))), record[stream + '_sha256'], name);
  }
  for (const [name, digest] of Object.entries(record.tools)) assert.equal(fileHash(name), digest, name);
  receipts++;
}
for (const [name, values] of Object.entries(evidence.source_changes)) {
  assert.equal(hash(git('show', base + ':' + name)), values.before_sha256, name);
  assert.equal(fileHash(path.join(root, name)), values.after_sha256, name);
}
for (const [name, digest] of Object.entries(evidence.prior_evidence)) assert.equal(fileHash(name), digest, name);
for (const [name, values] of Object.entries(evidence.protected_user_files)) assert.equal(fileHash(name), values.before_sha256_reported_by_root, name);
assert.equal(fileHash(evidence.stdlib_rename_oracle.path), evidence.stdlib_rename_oracle.sha256);
const tracked = git('ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration', '.github/workflows/gomad3.yml', '.github/.golangci.yml', 'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode').toString().split('\0').filter(Boolean).sort();
const serialized = '{' + tracked.map(name => JSON.stringify(name) + ': ' + JSON.stringify(fileHash(name))).join(', ') + '}';
const aggregate = hash(serialized);
assert.equal(aggregate, '9a3b419eb5c42d484147ca4d8e447b3a3670500dad0a2ed69f5356f442caab00');
const issues = name => [...fs.readFileSync(path.join(packet, name + '.stdout'), 'utf8').matchAll(/^(.+\.go):(\d+):(\d+): (.+) \(([^)]+)\)$/gm)].map(match => JSON.stringify(match.slice(1)));
const before = issues('baseline-configured-lint'), after = issues('final-configured-lint');
assert.equal(before.length, 63); assert.equal(after.length, 59);
assert.deepEqual(after.filter(item => !before.includes(item)), []);
assert.deepEqual(before.filter(item => !after.includes(item)).sort(), evidence.lint.removed.map(item => JSON.stringify(item)).sort());
assert.equal(issues('make-gomad-original-base').length, 204);
const current = JSON.parse(fs.readFileSync(path.join(packet, 'root-independent-controls.json')));
assert.equal(current.exit_code, 0);
const testCounts = {pass: 0, fail: 0, skip: 0};
for (const event of current.test_events) if (event.Test) testCounts[event.Action]++;
assert.deepEqual(testCounts, {pass: 24, fail: 0, skip: 0});
console.log(JSON.stringify({receipts, aggregate, rootTestCounts: testCounts, scopedLint: {before: before.length, after: after.length}, originalBaseLint: 204, protectedFilesUnchanged: true}));
