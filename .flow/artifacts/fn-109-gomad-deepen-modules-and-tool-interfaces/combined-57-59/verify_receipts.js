const fs = require('fs');
const crypto = require('crypto');
const cp = require('child_process');
const path = require('path');
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
if (process.cwd() !== root) throw new Error('Wrong workspace');
const out = path.dirname(__filename);
const base = 'f830467fcb712a83ba504c7dd43e2beb7f2cf223';
const sha = data => crypto.createHash('sha256').update(data).digest('hex');
const read = name => fs.readFileSync(path.join(out, name));
const git = args => cp.execFileSync('git', args, {cwd: root});
const check = (condition, message) => { if (!condition) throw new Error(message); };
const names = ['focused-combined', 'full-ordinary-runner', 'affected-vet', 'standalone-errortype',
    'architecture-source-sets', 'runner-ownership', 'generated-validation', 'format-check',
    'affected-configured-lint', 'make-fast-admission-base', 'make-gomad-original-base'];
const files = git(['ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration', '.github/workflows/gomad3.yml',
    '.github/.golangci.yml', 'Makefile', 'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode']).toString().split('\0').filter(Boolean);
const bindings = Object.fromEntries(files.map(file => [file, sha(fs.readFileSync(file))]));
const fingerprint = sha('{' + Object.keys(bindings).sort().map(key => JSON.stringify(key) + ': ' + JSON.stringify(bindings[key])).join(', ') + '}');
const observations = [];
const events = {};
for (const name of names) {
    const record = JSON.parse(read(name + '.json'));
    check(record.terminal && !record.timed_out && record.source_unchanged, name + ' terminal/source');
    check(record.source_before_sha256 === fingerprint && record.source_after_sha256 === fingerprint, name + ' fingerprint');
    check(record.gate_runner_sha256 === sha(read('run_gate.py')), name + ' runner');
    check(Number.isInteger(record.exit_code) && Number.isFinite(record.elapsed_seconds), name + ' numeric outcome');
    for (const suffix of ['stdout', 'stderr']) check(sha(read(name + '.' + suffix)) === record[suffix + '_sha256'], name + ' ' + suffix);
    for (const [tool, expected] of Object.entries(record.tools)) check(sha(fs.readFileSync(tool)) === expected, name + ' tool');
    const counts = {}, top = {}, identities = [];
    for (const line of read(name + '.stdout').toString().split('\n')) {
        let event;
        try { event = JSON.parse(line); } catch { continue; }
        if (event.Test && ['pass', 'fail', 'skip'].includes(event.Action)) {
            counts[event.Action] = (counts[event.Action] || 0) + 1;
            if (!event.Test.includes('/')) top[event.Action] = (top[event.Action] || 0) + 1;
            identities.push([event.Package, event.Test, event.Action].join('\0'));
        }
    }
    events[name] = identities;
    observations.push({name, exit: record.exit_code, elapsed_seconds: record.elapsed_seconds, counts, top,
        diagnostic_abort: name === 'full-ordinary-runner'});
}
const maps = new Map();
function oldLine(file, line) {
    if (!maps.has(file)) {
        const diff = git(['diff', '--no-ext-diff', '-U0', base, '--', file]).toString();
        maps.set(file, [...diff.matchAll(/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/gm)].map(match => {
            const oldCount = match[2] === undefined ? 1 : Number(match[2]);
            const newCount = match[4] === undefined ? 1 : Number(match[4]);
            return {old: Number(match[1]) + (oldCount === 0 ? 1 : 0), oldCount,
                next: Number(match[3]) + (newCount === 0 ? 1 : 0), newCount};
        }));
    }
    let offset = 0;
    for (const hunk of maps.get(file)) {
        if (line < hunk.next) break;
        if (line < hunk.next + hunk.newCount) throw new Error('Residual on changed line: ' + file + ':' + line);
        offset = hunk.old + hunk.oldCount - hunk.next - hunk.newCount;
    }
    return line + offset;
}
function diagnosticRows(text, remap) {
    return [...text.matchAll(/^([^\n]+\.go):(\d+):(\d+): ([^\n]+)\n([^\n]*)\n([^\n]*)/gm)].map(match => {
        const file = match[1].startsWith('tools/gomad3/') ? match[1] : 'tools/gomad3/' + match[1];
        return JSON.stringify([file, remap ? oldLine(file, Number(match[2])) : Number(match[2]), Number(match[3]), match[4], match[5], match[6]]);
    });
}
const before = diagnosticRows(fs.readFileSync(path.join(out, '../task-56/corrected-make-gomad-original-base.stdout'), 'utf8'), false);
const after = diagnosticRows(read('make-gomad-original-base.stdout').toString(), true);
const remaining = before.slice();
for (const row of after) {
    const index = remaining.indexOf(row);
    check(index !== -1, 'Introduced/residual mismatch: ' + row);
    remaining.splice(index, 1);
}
check(before.length === 80 && after.length === 60 && remaining.length === 20, 'Lint counts');
const removed = remaining.map(row => JSON.parse(row));
check(removed.filter(row => row[3].endsWith('(errcheck)')).length === 18, 'Errcheck removals');
check(removed.filter(row => row[3].endsWith('(exhaustive)')).length === 2, 'Exhaustive removals');
const allowed = new Set([...Object.keys(JSON.parse(fs.readFileSync(path.join(out, '../task-57/source-proof.json'))).files),
    'tools/gomad3/runner/inspect.go', 'tools/gomad3/internal/gomadtool/conformance/driver.go',
    'tools/gomad3/runner/choice_exploration_divergence_test.go', 'tools/gomad3/runner/completion_characterization_test.go',
    'tools/gomad3/runner/cleanup_test.go', 'tools/gomad3/runner/inspect_cleanup_test.go',
    'tools/gomad3/internal/gomadtool/conformance/driver_cleanup_test.go']);
const changed = git(['diff', '--name-only', base, '--', 'tools/gomad3']).toString().trim().split('\n');
check(changed.length === 18 && changed.every(file => allowed.has(file)), 'Product path scope');
for (const kind of ['inspect', 'driver']) {
    const encoded = JSON.parse(fs.readFileSync(path.join(out, '../task-58/initial-to-corrected-' + kind + '.diff.json')));
    check(sha(Buffer.from(encoded.unified_diff)) === encoded.decoded_sha256, 'Lossless diff ' + kind);
}
console.log(JSON.stringify({fingerprint, bound_files: files.length, product_paths: changed.length,
    receipts_verified: names.length, lint_before: before.length, lint_after: after.length, removed: removed.length,
    introduced: 0, all_residual_blocks_match: true, observations}, null, 2));
