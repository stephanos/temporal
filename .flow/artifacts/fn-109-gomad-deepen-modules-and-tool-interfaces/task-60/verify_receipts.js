const fs = require('fs');
const crypto = require('crypto');
const cp = require('child_process');
const path = require('path');
const root = '/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-60';
if (process.cwd() !== root) throw new Error('Wrong workspace');
const out = path.dirname(__filename);
const base = '7b75ae312a6641c6b00afbb63f44ae9a9dd0c2b2';
const file = 'tools/gomad3/runner/internal/execution/process_test.go';
const control = 'tools/gomad3/runner/internal/execution/process_fixture_output_test.go';
const sha = data => crypto.createHash('sha256').update(data).digest('hex');
const git = args => cp.execFileSync('git', args, {cwd: root});
const read = name => fs.readFileSync(path.join(out, name));
const check = (condition, message) => { if (!condition) throw new Error(message); };
const replacements = [
    ['\t\tfmt.Fprintln(os.Stdout, "target stdout")\n\t\tfmt.Fprintln(os.Stderr, "target stderr")', '\t\t_, stdoutErr := fmt.Fprintln(os.Stdout, "target stdout")\n\t\tfmt.Fprintln(os.Stderr, "target stderr")\n\t\tif stdoutErr != nil {\n\t\t\tos.Exit(7)\n\t\t}'],
    ['\t\tfmt.Fprintln(os.Stdout, "post-choice-marker")', '\t\tif _, err := fmt.Fprintln(os.Stdout, "post-choice-marker"); err != nil {\n\t\t\tos.Exit(0)\n\t\t}'],
    ['\t\tfmt.Fprintln(os.Stdout, "choice tape read-only")', '\t\tif _, err := fmt.Fprintln(os.Stdout, "choice tape read-only"); err != nil {\n\t\t\tos.Exit(0)\n\t\t}'],
    ['\tfmt.Fprintln(os.Stdout, <-results+<-results)', '\tif _, err := fmt.Fprintln(os.Stdout, <-results+<-results); err != nil {\n\t\treturn\n\t}'],
    ['\tfmt.Fprintln(os.Stdout, runChoiceSelectSequence(8))', '\tif _, err := fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8)); err != nil {\n\t\treturn\n\t}'],
    ['\tfmt.Fprintln(os.Stdout, runChoiceSelectSequence(8))', '\tif _, err := fmt.Fprintln(os.Stdout, runChoiceSelectSequence(8)); err != nil {\n\t\treturn\n\t}'],
];
const original = git(['show', base + ':' + file]);
let expected = original.toString();
for (const [before, after] of replacements) {
    check(expected.includes(before), 'Missing admitted preimage');
    expected = expected.replace(before, after);
}
check(fs.readFileSync(file).equals(Buffer.from(expected)), 'Non-admitted process source drift');
check(sha(fs.readFileSync(control)) === '94b7fb2073c75a46b0835cc2620fc3c522c7d945bf7b6e5d1686c84a58e16572', 'Frozen controls drift');
const names = ['baseline-output-controls', 'baseline-focused', 'baseline-configured-lint', 'baseline-original-lint',
    'final-focused', 'final-ordinary-execution', 'final-vet', 'final-standalone-errortype', 'final-format', 'final-configured-lint', 'final-fast-lint', 'final-original-lint',
    'sensitivity-output', 'sensitivity-choice-marker', 'sensitivity-choice-tape-readonly', 'sensitivity-choice-reorder', 'sensitivity-choice-select', 'sensitivity-choice-prefix-rng'];
const observations = [];
const baselineBindings = JSON.parse(read('baseline-output-controls-sources.json'));
const verdicts = {};
for (const name of names) {
    const record = JSON.parse(read(name + '.json'));
    check(record.terminal && !record.timed_out && record.source_unchanged, name + ' terminal/source');
    check(record.gate_runner_sha256 === sha(read('run_gate.py')), name + ' runner');
    check(Number.isInteger(record.exit_code), name + ' numeric outcome');
    for (const suffix of ['stdout', 'stderr']) check(sha(read(name + '.' + suffix)) === record[suffix + '_sha256'], name + ' raw ' + suffix);
    for (const [tool, expectedHash] of Object.entries(record.tools)) check(sha(fs.readFileSync(tool)) === expectedHash, name + ' tool');
    const manifest = read(name + '-sources.json');
    check(sha(manifest) === record.source_manifest_sha256, name + ' manifest');
    const bindings = JSON.parse(manifest);
    check(JSON.stringify(Object.keys(bindings).sort()) === JSON.stringify(Object.keys(baselineBindings).sort()), name + ' bound path set');
    for (const [filename, expectedHash] of Object.entries(baselineBindings)) {
        if (filename !== file) check(bindings[filename] === expectedHash, name + ' non-admitted source ' + filename);
        if (!name.startsWith('baseline')) check(bindings[filename] === sha(fs.readFileSync(filename)), name + ' current source ' + filename);
    }
    const fingerprint = sha('{' + Object.keys(bindings).sort().map(key => JSON.stringify(key) + ': ' + JSON.stringify(bindings[key])).join(', ') + '}');
    check(record.source_before_sha256 === fingerprint && record.source_after_sha256 === fingerprint, name + ' fingerprint');
    check(bindings[file] === (name.startsWith('baseline') ? sha(original) : sha(Buffer.from(expected))), name + ' process binding');
    check(bindings[control] === sha(fs.readFileSync(control)), name + ' controls binding');
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
    observations.push({name, exit: record.exit_code, elapsed_seconds: record.elapsed_seconds, counts, top});
    verdicts[name] = identities.sort();
}
check(JSON.stringify(verdicts['baseline-focused']) === JSON.stringify(verdicts['final-focused']), 'Focused verdict drift');
const hunks = [...git(['diff', '--no-ext-diff', '-U0', base, '--', file]).toString().matchAll(/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/gm)].map(match => ({old: +match[1], oldCount: match[2] === undefined ? 1 : +match[2], next: +match[3], newCount: match[4] === undefined ? 1 : +match[4]}));
function oldLine(line) {
    let offset = 0;
    for (const hunk of hunks) {
        if (line < hunk.next) break;
        check(line >= hunk.next + hunk.newCount, 'Residual on an admitted changed line');
        offset = hunk.old + hunk.oldCount - hunk.next - hunk.newCount;
    }
    return line + offset;
}
function diagnosticRows(text, remap) {
    return [...text.matchAll(/^([^\n]+\.go):(\d+):(\d+): ([^\n]+)\n([^\n]*)\n([^\n]*)/gm)].map(match => {
        const filename = match[1].startsWith('tools/gomad3/') ? match[1] : 'tools/gomad3/' + match[1];
        return JSON.stringify([filename, remap && filename === file ? oldLine(+match[2]) : +match[2], +match[3], match[4], match[5], match[6]]);
    });
}
const before = diagnosticRows(read('baseline-original-lint.stdout').toString(), false);
const after = diagnosticRows(read('final-original-lint.stdout').toString(), true);
const remaining = before.slice();
for (const row of after) {
    const index = remaining.indexOf(row);
    check(index !== -1, 'Introduced/residual mismatch ' + row);
    remaining.splice(index, 1);
}
check(before.length === 60 && after.length === 54 && remaining.length === 6, 'Actual lint counts');
const removed = remaining.map(row => JSON.parse(row));
check(removed.every(row => row[0] === file && row[3].endsWith('(errcheck)')), 'Wrong lint removals');
const productPaths = git(['diff', '--name-only', base, '--', 'tools/gomad3']).toString().trim().split('\n').filter(Boolean);
check(productPaths.every(filename => [file, control].includes(filename)), 'Product scope drift');
const sensitivity = JSON.parse(read('sensitivity-inputs.json'));
check(sensitivity.candidate_sha256 === sha(Buffer.from(expected)) && sensitivity.mutations.length === 6, 'Sensitivity candidate/count');
for (const row of sensitivity.mutations) {
    check(row.exit_code === 1 && row.inputs_unchanged, row.name + ' sensitivity outcome');
    check(sha(fs.readFileSync(row.source)) === row.source_sha256 && sha(fs.readFileSync(row.overlay)) === row.overlay_sha256, row.name + ' overlay inputs');
    check(read('sensitivity-' + row.name + '.stdout').toString().includes('status/stderr = 99/'), row.name + ' wrong reason');
}
console.log(JSON.stringify({base_commit: base, original_sha256: sha(original), candidate_sha256: sha(Buffer.from(expected)), frozen_control_sha256: sha(fs.readFileSync(control)), reconstructed_non_admitted_bytes: true, lint_before: before.length, lint_after: after.length, removed, introduced: 0, all_residual_blocks_match: true, sensitivity_rejections: sensitivity.mutations.length, observations}, null, 2));
