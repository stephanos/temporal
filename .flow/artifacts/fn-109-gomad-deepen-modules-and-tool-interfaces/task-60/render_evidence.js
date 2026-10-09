const fs = require('fs');
const cp = require('child_process');
const path = require('path');
const root = '/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-60';
if (process.cwd() !== root) throw new Error('Wrong workspace');
const out = path.dirname(__filename);
const base = fs.readFileSync('.flow/tmp/base_commit', 'utf8').trim();
const commits = cp.execFileSync('git', ['rev-list', '--reverse', base + '..HEAD'], {cwd: root, encoding: 'utf8'}).trim().split('\n').filter(Boolean);
const names = ['baseline-output-controls', 'baseline-focused', 'baseline-configured-lint', 'baseline-original-lint',
    'final-focused', 'final-ordinary-execution', 'final-vet', 'final-standalone-errortype', 'final-format', 'final-configured-lint', 'final-fast-lint', 'final-original-lint',
    'sensitivity-output', 'sensitivity-choice-marker', 'sensitivity-choice-tape-readonly', 'sensitivity-choice-reorder', 'sensitivity-choice-select', 'sensitivity-choice-prefix-rng'];
const tests = names.map(name => {
    const record = JSON.parse(fs.readFileSync(path.join(out, name + '.json')));
    if (!record.terminal || !record.source_unchanged || record.timed_out) throw new Error('Invalid gate ' + name);
    return record.command;
});
tests.push('node .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-60/verify_receipts.js');
fs.writeFileSync(path.join(out, 'evidence.json'), JSON.stringify({commits, base_commit: base, tests, prs: []}, null, 2) + '\n');
console.log(JSON.stringify({base_commit: base, commits, command_receipts: names.length}));
