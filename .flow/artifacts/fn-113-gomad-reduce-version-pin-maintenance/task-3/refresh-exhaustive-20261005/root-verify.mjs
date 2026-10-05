import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const directory = dirname(fileURLToPath(import.meta.url));
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const base = '765547c32d2f674993026743fadceb4319e641d8';
const source = 'tools/gomad3/cmd/gomadtool/compatibility_pack_refresh.go';
const sha = bytes => createHash('sha256').update(bytes).digest('hex');
const read = name => readFileSync(resolve(directory, name), 'utf8');
const git = args => {
  const result = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout;
};
const before = git(['show', base + ':' + source]);
const after = readFileSync(resolve(root, source), 'utf8');
const insertion = '\t\t\tcase pinimpact.StatusUnaffected, pinimpact.StatusNotSelected:\n';
assert.equal(after.split(insertion).length, 2);
assert.equal(after.replace(insertion, ''), before);
assert.deepEqual(git(['diff', '--name-only', base, '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'Makefile', '.github/.golangci.yml']).trim().split('\n'), [source]);
const receipt = JSON.parse(read('checks.json'));
assert.equal(receipt.freezes.BASE.head, base);
assert.equal(receipt.freezes.BASE.files[source], sha(before));
assert.equal(receipt.freezes.FINAL.files[source], sha(after));
for (const run of receipt.runs) {
  assert.equal(run.source_and_tool_inputs_stable, true, run.label);
  assert.equal(run.signal, null, run.label);
  assert.equal(sha(read(run.label + '.stdout.log')), run.stdout_sha256, run.label);
  assert.equal(sha(read(run.label + '.stderr.log')), run.stderr_sha256, run.label);
}
const events = name => read(name + '.stdout.log').trim().split('\n').map(line => JSON.parse(line));
const identities = name => events(name).filter(event => event.Test && ['run', 'pass', 'fail', 'skip'].includes(event.Action)).map(({ Action, Package, Test }) => ({ Action, Package, Test }));
for (const suite of ['refresh', 'pinimpact']) assert.deepEqual(identities('baseline-' + suite), identities('final-' + suite));
const errors = name => events(name).filter(event => event.OutputType === 'error').map(({ Test, Output }) => ({ Test, Output }));
assert.deepEqual(errors('baseline-pinimpact'), errors('final-pinimpact'));
assert.equal(errors('final-pinimpact').length, 1);
assert.match(errors('final-pinimpact')[0].Output, /host is linux\/arm64, want rejection containing "unsupported github.com\/getsentry\/sentry-go version"/);
const blocks = text => {
  const matches = [...text.matchAll(/^tools\/gomad3\/.+\.go:\d+:\d+: .+$/gm)];
  const footer = text.search(/^\d+ issues:$/m);
  assert.ok(footer > 0);
  return matches.map((match, index) => text.slice(match.index, matches[index + 1]?.index ?? footer));
};
const compare = (oldText, newText, oldCount, newCount) => {
  const oldBlocks = blocks(oldText), newBlocks = blocks(newText);
  assert.equal(oldBlocks.length, oldCount);
  assert.equal(newBlocks.length, newCount);
  const removed = oldBlocks.filter(block => !newBlocks.includes(block));
  assert.equal(removed.length, 1);
  assert.match(removed[0], /^tools\/gomad3\/cmd\/gomadtool\/compatibility_pack_refresh.go:328:4: missing cases in switch of type pinimpact.Status: pinimpact.StatusUnaffected, pinimpact.StatusNotSelected \(exhaustive\)/);
  assert.deepEqual(oldBlocks.filter(block => block !== removed[0]), newBlocks);
  return { before: oldCount, after: newCount, unchanged_complete_blocks: newCount, introduced: 0, removed: removed[0].split('\n')[0] };
};
const scoped = compare(read('baseline-lint.stdout.log'), read('final-lint.stdout.log'), 136, 135);
const integrated = compare(readFileSync(resolve(root, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-11/inventory-format-20261005/root-integrated-lint/stdout.log'), 'utf8'), read('root-integrated-lint.stdout.log'), 318, 317);
const lint = receipt.runs.find(run => run.label === 'root-integrated-lint');
assert.equal(lint.exit_code, 2);
assert.match(read('root-integrated-lint.stdout.log'), /Lint module tools\/gomad3: 55 host packages/);
assert.match(read('root-integrated-lint.stdout.log'), /--new-from-rev=951c5516e9e7b3066e7e069adda9565cfd68844c/);
assert.match(read('root-integrated-lint.stdout.log'), /--fix=false/);
assert.doesNotMatch(read('root-integrated-lint.stdout.log'), / -vettool=/);
console.log(JSON.stringify({ exact_one_line_insertion: true, matched_test_identities: true, inherited_failure_unchanged: true, retained_log_bindings: receipt.runs.length, scoped, integrated, integrated_errortype: 'UNREACHED' }, null, 2));
